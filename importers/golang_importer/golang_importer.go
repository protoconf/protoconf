package golangimporter

import (
	"fmt"
	"go/token"
	"go/types"
	"log"
	"path/filepath"
	"strings"

	"github.com/fatih/structtag"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/protoconf/protoconf/importers"
	zap "go.uber.org/zap"
	"golang.org/x/tools/go/packages"
)

// GolangImporter go through a package and extract its structs to proto files
type GolangImporter struct {
	Importer           *importers.Importer
	pkgs               []*packages.Package
	pkgID              string
	protoFilesRegistry *protoFilesRegistry
	targetTags         []string
	errors             []error
	logger             *zap.Logger
	InterfacesAsAny    bool
}

// NewGolangImporter creates a new GolangImporter
func NewGolangImporter(pkg, outputDir, goSrcPath string, env ...string) (*GolangImporter, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	logger.Debug(fmt.Sprintf("%v", env))
	cfg := &packages.Config{
		Dir:  filepath.Join(goSrcPath, pkg),
		Mode: packages.LoadAllSyntax,
		Fset: fset,
		Env:  env,
		Logf: logger.Sugar().Debugf,
	}
	packages, err := packages.Load(cfg, filepath.Join(goSrcPath, pkg))
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %v", err)
	}
	logger.Sugar().Debugw("packages", "packages", packages)
	return &GolangImporter{
		pkgID:              pkg,
		pkgs:               packages,
		Importer:           importers.NewImporter("go_imported.proto", outputDir),
		protoFilesRegistry: newProtoFilesRegistry(),
		targetTags:         []string{"json", "yaml", "mapstructure"},
		logger:             logger,
	}, nil
}

// SetTargetTags allows setting the go tag names in struct to be used as `json_name` and protobuf key name
func (p *GolangImporter) SetTargetTags(tags []string) {
	p.logger.Debug("setting target tags", zap.String("tags", fmt.Sprintf("%v", tags)))
	p.targetTags = tags
}

func (p *GolangImporter) goFieldToProtoField(f *types.Var, tagstr string) *builder.FieldBuilder {
	t1 := f.Type()
	pkg := f.Pkg()
	b := builder.NewField(f.Name(), builder.FieldTypeString())

	if tag, err := structtag.Parse(tagstr); err == nil {
		for _, targetTag := range p.targetTags {
			if y, e := tag.Get(targetTag); e == nil {
				b.SetJsonName(y.Name)
				if y.Name != "" && y.Name != "-" {
					b.TrySetName(y.Name)
				}
			}
		}
	}
	if strings.HasPrefix(f.Type().String(), "func") {
		return nil
	}
	t := p.goFieldTypeToProtoFieldType(t1.Underlying().String())

	// When a type is a pointer
	if s, ok := t1.Underlying().(*types.Pointer); ok {
		t1 = s.Elem()
		p.logger.Debug("detected as pointer",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
	}
	// When a type is map
	if s, ok := t1.Underlying().(*types.Map); ok {
		t1 = s.Elem()
		key := p.goFieldTypeToProtoFieldType(s.Key().Underlying().String())
		val := p.goFieldTypeToProtoFieldType(s.Elem().Underlying().String())
		if key != nil && val != nil && isValidKeyType(key) {
			p.logger.Debug("new map field", zap.String("key", key.GetTypeName()), zap.String("value", val.GetTypeName()))
			b = builder.NewMapField(f.Name(), key, val)
		}
		p.logger.Debug("detected as map",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
	}

	// When a type is an Slice
	if s, ok := t1.Underlying().(*types.Slice); ok {
		t1 = s.Elem()
		t = p.goFieldTypeToProtoFieldType(s.Elem().String())
		b.SetRepeated()
		p.logger.Debug("detected as slice",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
	}
	// When a type is an array
	if s, ok := t1.Underlying().(*types.Array); ok {
		t1 = s.Elem()
		t = p.goFieldTypeToProtoFieldType(s.Elem().String())
		b.SetRepeated()
		p.logger.Debug("detected as array",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
	}

	// When a type is a pointer
	if s, ok := t1.Underlying().(*types.Pointer); ok {
		t1 = s.Elem()
		p.logger.Debug("detected as pointer",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
	}

	// When a type is another struct
	if _, ok := t1.Underlying().(*types.Struct); ok {
		a := strings.Split(t1.String(), ".")
		structName := a[len(a)-1]
		pkgID := strings.Join(a[:len(a)-1], ".")
		a = strings.Split(pkgID, "/")
		pkgName := a[len(a)-1]
		file := p.protoFilesRegistry.GetProtoFile(pkgName, pkgID)
		msg := file.GetMessage(structName)
		if msg == nil {
			p.errors = append(p.errors, fmt.Errorf("could not load message for %v %v %v", pkgID, structName, t1))
			return nil
		}
		t = builder.FieldTypeMessage(msg)
		p.logger.Debug("detected as struct",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
	}

	if b.IsMap() {
		return b
	}
	if t != nil {
		p.logger.Debug("detected as",
			zap.String("pkg", pkg.Name()),
			zap.String("field", f.Name()),
			zap.String("type", t1.String()),
		)
		b.SetType(t)
		return b
	}
	err := fmt.Errorf("could not understand field for %v %v, underlying: %v", f.Name(), f.Type(), t1.Underlying())
	p.logger.Info("could not understand field", zap.String("name", f.Name()), zap.String("type", f.Type().String()), zap.String("underlying", t1.Underlying().String()))
	p.errors = append(p.errors, err)
	return nil
}

func isValidKeyType(keyTyp *builder.FieldType) bool {
	switch keyTyp.GetType() {
	case dpb.FieldDescriptorProto_TYPE_BOOL,
		dpb.FieldDescriptorProto_TYPE_STRING,
		dpb.FieldDescriptorProto_TYPE_INT32, dpb.FieldDescriptorProto_TYPE_INT64,
		dpb.FieldDescriptorProto_TYPE_SINT32, dpb.FieldDescriptorProto_TYPE_SINT64,
		dpb.FieldDescriptorProto_TYPE_UINT32, dpb.FieldDescriptorProto_TYPE_UINT64,
		dpb.FieldDescriptorProto_TYPE_FIXED32, dpb.FieldDescriptorProto_TYPE_FIXED64,
		dpb.FieldDescriptorProto_TYPE_SFIXED32, dpb.FieldDescriptorProto_TYPE_SFIXED64:
		return true
	default:
		return false
	}
}

func (p *GolangImporter) pre(pkg *packages.Package) bool {
	file := p.protoFilesRegistry.GetProtoFile(pkg.Name, pkg.ID)
	log := p.logger.With(zap.String("file", file.GetName()), zap.String("func", "pre"))
	log.Info("preparing")
	if pkg.TypesInfo == nil {
		log.Debug("TypesInfo is empty :(")
		return true
	}
	for _, t := range pkg.TypesInfo.Defs {
		if t == nil {
			continue
		}
		if !t.Exported() {
			continue
		}
		if _, ok := t.Type().Underlying().(*types.Struct); ok {
			msg := builder.NewMessage(t.Name())
			err := file.TryAddMessage(msg)
			if err != nil {
				log.Error("failed to add message", zap.String("msg", msg.GetName()), zap.Error(err))
			}
		}
	}
	return true
}

func (p *GolangImporter) post(pkg *packages.Package) {
	file := p.protoFilesRegistry.GetProtoFile(pkg.Name, pkg.ID)
	log := p.logger.With(zap.String("file", file.GetName()), zap.String("func", "post"))
	if pkg.TypesInfo == nil {
		return
	}
	for _, t := range pkg.TypesInfo.Defs {
		if t == nil {
			continue
		}
		if !t.Exported() {
			continue
		}
		if s, ok := t.Type().Underlying().(*types.Struct); ok {
			log = log.With(zap.String("pkg", pkg.ID), zap.String("t", t.Id()))
			msg := file.GetMessage(t.Name())
			if msg == nil {
				log.Error("could not find message")
				continue
			}
			for i := 0; i < s.NumFields(); i++ {
				f := s.Field(i)
				tag := s.Tag(i)
				if f.IsField() && f.Exported() {
					bf := p.goFieldToProtoField(f, tag)
					if bf != nil {
						msg.TryAddField(bf)
					}
				}
			}
		}
	}
}

// Visit will run the packages visit logic
func (p *GolangImporter) Visit() {
	p.logger.Info("visiting packages")
	packages.Visit(p.pkgs, p.pre, p.post)
	for _, err := range p.errors {
		p.logger.Error("", zap.Error(err))
	}
	for _, fbuilder := range p.protoFilesRegistry.GetAllFiles() {
		p.Importer.RegisterFile(fbuilder)
	}
}

// GetMessageFromFile is a facility that helps filter the files to import
func (p *GolangImporter) GetMessageFromFile(filename, messageName string) *builder.MessageBuilder {
	my := map[string]*builder.FileBuilder{}
	for _, b := range p.protoFilesRegistry.GetAllFiles() {
		my[b.GetName()] = b
	}
	return my[filename+".proto"].GetMessage(messageName)
}

// GetFileNameFor will assist the command line to detect the file of the top config struct
func (p *GolangImporter) GetFileNameFor(pkgName, pkgID string) string {
	return p.protoFilesRegistry.GetProtoFile(pkgName, pkgID).GetName()
}

// func (p *GolangImporter) ImportMessageFromFile(fileName, msgName string) {
// 	msg := p.GetMessageFromFile(fileName, msgName)
// 	if msg == nil {
// 		log.Println("cannot load msg", fileName, msgName)
// 		return
// 	}
// 	log.Println(msg.GetFile().GetName(), msg.GetName())
// 	p.Importer.RegisterFile(msg.GetFile())
// 	for _, child := range msg.GetChildren() {
// 		if field, ok := child.(*builder.FieldBuilder); ok {
// 			fTypeName := field.GetType().GetTypeName()
// 			if fTypeName != "" {
// 				ret := strings.Split(fTypeName, ".")
// 				if ret[0] != fileName && ret[1] != msgName {
// 					p.ImportMessageFromFile(ret[0], ret[1])
// 				}
// 			}
// 		}
// 	}
// }

// GetImporter returns the importers.Importer generated by GolangImporter
func (p *GolangImporter) GetImporter() *importers.Importer {
	p.logger.Info("getting importer")
	p.Visit()
	return p.Importer
}

var wkt = make(map[string]*builder.FileBuilder)

func init() {
	durationFile, err := desc.LoadFileDescriptor("google/protobuf/duration.proto")
	if err != nil {
		log.Fatal(err)
	}
	durationBuilder, err := builder.FromFile(durationFile)
	if err != nil {
		log.Fatal(err)
	}
	wkt["duration"] = durationBuilder

	structFile, err := desc.LoadFileDescriptor("google/protobuf/struct.proto")
	if err != nil {
		log.Fatal(err)
	}
	structBuilder, err := builder.FromFile(structFile)
	if err != nil {
		log.Fatal(err)
	}
	wkt["struct"] = structBuilder

	anyFile, err := desc.LoadFileDescriptor("google/protobuf/any.proto")
	if err != nil {
		log.Fatal(err)
	}
	anyBuilder, err := builder.FromFile(anyFile)
	if err != nil {
		log.Fatal(err)
	}
	wkt["any"] = anyBuilder
}

func (p *GolangImporter) goFieldTypeToProtoFieldType(x string) *builder.FieldType {
	x = strings.Replace(x, "*", "", -1)
	switch x {
	case "bool":
		return builder.FieldTypeBool()
	case "byte":
		return builder.FieldTypeBytes()
	case "float":
		return builder.FieldTypeFloat()
	case "float64":
		return builder.FieldTypeFloat()
	case "int":
		return builder.FieldTypeInt32()
	case "int8":
		return builder.FieldTypeInt32()
	case "int32":
		return builder.FieldTypeInt32()
	case "int64":
		return builder.FieldTypeInt64()
	case "uint":
		return builder.FieldTypeUInt32()
	case "uint8":
		return builder.FieldTypeUInt32()
	case "uint16":
		return builder.FieldTypeUInt32()
	case "uint32":
		return builder.FieldTypeUInt32()
	case "uint64":
		return builder.FieldTypeUInt64()
	case "string":
		return builder.FieldTypeString()
	case "time.Duration":
		return builder.FieldTypeMessage(wkt["duration"].GetMessage("Duration"))
	case "interface{}":
		if p.InterfacesAsAny {
			return builder.FieldTypeMessage(wkt["any"].GetMessage("Any"))
		}
		return builder.FieldTypeMessage(wkt["struct"].GetMessage("Value"))
	case "error":
		return nil
	}
	return nil
}
