package terraformimporter

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/zap"

	"github.com/protoconf/protoconf/importers/terraform_importer/meta"
)

var metaFile *builder.FileBuilder = meta.MetaFile()

// NewFile returns a FileBuilder prepared with the required filename format
func NewFile(name, postfix string) *builder.FileBuilder {
	file := builder.NewFile(fmt.Sprintf("terraform/%s-%s.proto", name, postfix))
	file.SetProto3(true)
	file.SetPackageName(fmt.Sprintf("terraform.%s.%s", name, postfix))
	return file
}

// Print prints a FileBuilder to stderr
func Print(b *builder.FileBuilder) {
	p := &protoprint.Printer{}
	desc, err := b.Build()
	if err != nil {
		log.Fatal(err)
	}
	str, err := p.PrintProtoToString(desc)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(str)
}

func (p *ProviderImporter) schemaToProtoMessage(name string, schema providers.Schema) *builder.MessageBuilder {
	log := p.logger.With(zap.String("message", name))
	log.Info("handling message")

	m := p.msgBuilderFromBlock(name, schema.Block)
	c := builder.Comments{LeadingComment: fmt.Sprintf("%s version is %d", name, schema.Version)}
	m.SetComments(c)

	// Adding meta fields
	metaMsg := metaFile.GetMessage("MetaFields")
	for _, field := range metaMsg.GetChildren() {
		f := metaMsg.GetField(field.GetName())
		fBuilder := builder.NewField(f.GetName(), f.GetType())
		if f.IsRepeated() {
			fBuilder.SetRepeated()
		}
		m.AddField(fBuilder)
	}
	fieldLifecycle := builder.NewField("lifecycle", builder.FieldTypeMessage(metaFile.GetMessage("Lifecycle")))
	m.AddField(fieldLifecycle)
	return m
}

func (p *ProviderImporter) msgBuilderFromBlock(name string, b *configschema.Block) *builder.MessageBuilder {
	m := builder.NewMessage(name)
	attrs := b.Attributes
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, fieldName := range keys {
		p.attributeToProtoField(m, fieldName, attrs[fieldName])
	}
	for n, nb := range b.BlockTypes {
		nm := p.msgBuilderFromBlock(capitalizeMessageName(n), &nb.Block)
		f := builder.NewField(n, builder.FieldTypeMessage(nm))
		f.SetJsonName(n)
		m.TryAddField(f)
		m.TryAddNestedMessage(nm)
	}
	return m
}

func (p *ProviderImporter) attributeToProtoField(msg *builder.MessageBuilder, name string, attr *configschema.Attribute) *builder.MessageBuilder {
	p.logger.With(zap.String("attr", name)).Debug("got attr")
	t := attr.Type
	p.handleCty(msg, name, t, attr.Description)
	return msg
}

func (p *ProviderImporter) handleCty(parent *builder.MessageBuilder, fieldName string, t cty.Type, description string) error {
	f := p.ctyTypeToProtoField(fieldName, t)

	if t.IsListType() || t.IsSetType() {
		t = t.ElementType()
	}
	if t.IsObjectType() {
		p.handleObject(fieldName, t, f, parent)
	}

	c := builder.Comments{LeadingComment: description}
	f.SetComments(c)
	parent.TryAddField(f)
	return nil
}

func (p *ProviderImporter) handleObject(name string, t cty.Type, f *builder.FieldBuilder, msg *builder.MessageBuilder) {
	log := p.logger.With(zap.String("object_name", name))
	m := builder.NewMessage(capitalizeMessageName(name))
	keys := []string{}
	for n := range t.AttributeTypes() {
		keys = append(keys, n)
	}
	sort.Strings(keys)

	for _, n := range keys {
		t2 := t.AttributeType(n)
		f2 := p.ctyTypeToProtoField(n, t2)
		c := builder.Comments{LeadingComment: fmt.Sprintf("%v: %s", n, t2.FriendlyName())}
		if t2.IsListType() || t2.IsSetType() {
			t2 = t2.ElementType()
			f2.SetRepeated()
		}
		if t2.IsObjectType() {
			p.handleObject(n, t2, f2, m)
		}
		f2.SetComments(c)
		m.TryAddField(f2)
	}

	log.Info("trying to add nested message", zap.String("submessage", capitalizeMessageName(name)))
	if err := msg.TryAddNestedMessage(m); err != nil {
		log.Fatal("failed to add message", zap.Error(err))
	}
	f.SetType(builder.FieldTypeMessage(m))
}

func (p *ProviderImporter) ctyTypeToProtoField(name string, t cty.Type) *builder.FieldBuilder {
	log := p.logger
	jsonName := name
	var validFieldName = regexp.MustCompile(`^[a-z]`)
	if !validFieldName.MatchString(name) {
		name = "_" + name
	}
	f := builder.NewField(name, builder.FieldTypeString())
	f.SetJsonName(jsonName)

	if t.IsListType() || t.IsSetType() {
		log.Info("detected as list", zap.String("type", t.ElementType().FriendlyName()))
		t = t.ElementType()
		f.SetRepeated()
	}
	if t.IsMapType() {
		log.Info("detected as map", zap.String("type", t.ElementType().FriendlyName()))
		f = builder.NewMapField(name, builder.FieldTypeString(), builder.FieldTypeString())
		return f
	}
	if t.IsObjectType() {
		log.Info("detected as object, returning")
		return f
	}
	f.SetType(ctyTypeToProtoFieldType(t))
	return f
}

func ctyTypeToProtoFieldType(t cty.Type) *builder.FieldType {
	var ft *builder.FieldType
	switch x := t.FriendlyName(); x {
	case "string":
		return builder.FieldTypeString()
	case "number":
		return builder.FieldTypeInt64()
	case "bool":
		return builder.FieldTypeBool()
	default:
		log.Fatalf("unknown type: %v", x)
	}
	return ft
}

func capitalizeMessageName(s string) string {
	out := []string{}
	a := strings.Split(s, "_")
	for _, item := range a {
		out = append(out, strings.Title(item))
	}
	return strings.Join(out, "")
}
