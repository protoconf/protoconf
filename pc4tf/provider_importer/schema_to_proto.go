package provider_importer

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/zclconf/go-cty/cty"

	"github.com/protoconf/protoconf/pc4tf/meta"
)

var metaFile *builder.FileBuilder = meta.MetaFile()

func NewFile(name, postfix string) *builder.FileBuilder {
	file := builder.NewFile(fmt.Sprintf("terraform/%s-%s.proto", name, postfix))
	file.SetProto3(true)
	file.SetPackageName(fmt.Sprintf("terraform.%s.%s", name, postfix))
	return file
}

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

func schemaToProtoMessage(name string, schema providers.Schema) *builder.MessageBuilder {
	m := builder.NewMessage(name)
	c := builder.Comments{LeadingComment: fmt.Sprintf("%s version is %d", name, schema.Version)}
	m.SetComments(c)

	attrs := schema.Block.Attributes
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, fieldName := range keys {
		attributeToProtoField(m, fieldName, attrs[fieldName])
	}

	// Adding meta fields
	metaMsg := metaFile.GetMessage("MetaFields")
	for _, field := range metaMsg.GetChildren() {
		f := metaMsg.GetField(field.GetName())
		fBuilder := builder.NewField(f.GetName(), f.GetType())
		m.AddField(fBuilder)
	}
	fieldLifecycle := builder.NewField("lifecycle", builder.FieldTypeMessage(metaFile.GetMessage("Lifecycle")))
	m.AddField(fieldLifecycle)
	return m
}

func attributeToProtoField(msg *builder.MessageBuilder, name string, attr *configschema.Attribute) *builder.MessageBuilder {
	// f := builder.NewField(name, builder.FieldTypeString())
	t := attr.Type
	f := ctyTypeToProtoField(name, t)

	if t.IsListType() || t.IsSetType() {
		t = t.ElementType()
	}
	if t.IsObjectType() {
		m := builder.NewMessage(capitalizeMessageName(name))
		for n, t1 := range t.AttributeTypes() {
			m.TryAddField(ctyTypeToProtoField(n, t1))
		}
		log.Println("trying to add nested message", name)
		if err := msg.TryAddNestedMessage(m); err != nil {
			log.Fatal(err)
		}
		f.SetType(builder.FieldTypeMessage(m))
	}

	c := builder.Comments{LeadingComment: attr.Description}
	f.SetComments(c)
	msg.TryAddField(f)
	return msg
}

func ctyTypeToProtoField(name string, t cty.Type) *builder.FieldBuilder {
	f := builder.NewField(name, builder.FieldTypeString())
	f.SetJsonName(name)

	if t.IsListType() || t.IsSetType() {
		log.Println(name, "is a list of", t.ElementType().FriendlyName())
		t = t.ElementType()
		f.SetRepeated()
	}
	if t.IsMapType() {
		log.Println(name, "is a map of", t.ElementType().FriendlyName())
		f = builder.NewMapField(name, builder.FieldTypeString(), builder.FieldTypeString())
		return f
	}
	if t.IsObjectType() {
		log.Println(name, "is an object, returning")
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
		log.Fatalf("unkown type: %v", x)
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
