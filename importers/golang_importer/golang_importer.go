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
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/protoconf/protoconf/importers"
	"golang.org/x/tools/go/packages"
)

// GolangImporter go through a package and extract its structs to proto files
type GolangImporter struct {
	Importer           *importers.Importer
	pkgs               []*packages.Package
	pkgID              string
	protoFilesRegistry *protoFilesRegistry
	errors             []error
}

// NewGolangImporter creates a new GolangImporter
func NewGolangImporter(pkg, outputDir, goSrcPath string, env ...string) (*GolangImporter, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Dir:  goSrcPath,
		Mode: packages.LoadAllSyntax,
		Fset: fset,
		Env:  env,
	}
	packages, err := packages.Load(cfg, filepath.Join(goSrcPath, pkg))
	if err != nil {
		return nil, err
	}
	log.Println(packages)
	return &GolangImporter{
		pkgID:              pkg,
		pkgs:               packages,
		Importer:           importers.NewImporter("go_imported.proto", outputDir),
		protoFilesRegistry: newProtoFilesRegistry(),
	}, nil
}

func (p *GolangImporter) goFieldToProtoField(f *types.Var, tagstr string) *builder.FieldBuilder {
	t1 := f.Type()
	pkg := f.Pkg()
	b := builder.NewField(f.Name(), builder.FieldTypeString())

	if tag, err := structtag.Parse(tagstr); err == nil {
		if y, e := tag.Get("yaml"); e == nil {
			b.SetJsonName(y.Name)
			if y.Name != "" {
				b.SetName(y.Name)
			}
		}
		if j, e := tag.Get("json"); e == nil {
			// b.SetName(j.Name)
			b.SetJsonName(j.Name)
		}
	}
	if strings.HasPrefix(f.Type().String(), "func") {
		return nil
	}
	t := goFieldTypeToProtoFieldType(t1.Underlying().String())

	// When a type is map
	if s, ok := t1.Underlying().(*types.Map); ok {
		t1 = s.Elem()
		key := goFieldTypeToProtoFieldType(s.Key().Underlying().String())
		val := goFieldTypeToProtoFieldType(s.Elem().Underlying().String())
		if key != nil && val != nil && isValidKeyType(key) {
			log.Println("new map field for", key.GetTypeName(), val.GetTypeName())
			b = builder.NewMapField(f.Name(), key, val)
		}
		log.Println(pkg.Name(), f.Name(), "detected as map of", t1.String())
	}

	// When a type is an Slice
	if s, ok := t1.Underlying().(*types.Slice); ok {
		t1 = s.Elem()
		t = goFieldTypeToProtoFieldType(s.Elem().String())
		b.SetRepeated()
		log.Println(pkg.Name(), f.Name(), "detected as slice of", t1.String())
	}
	// When a type is an array
	if s, ok := t1.Underlying().(*types.Array); ok {
		t1 = s.Elem()
		t = goFieldTypeToProtoFieldType(s.Elem().String())
		b.SetRepeated()
		log.Println(pkg.Name(), f.Name(), "detected as array of", t1.String())
	}

	// When a type is a pointer
	if s, ok := t1.Underlying().(*types.Pointer); ok {
		t1 = s.Elem()
		log.Println(pkg.Name(), f.Name(), "detected as pointer of", t1.String())
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
		log.Println(pkg.Name(), f.Name(), "detected as struct of", t1.String())
	}

	if b.IsMap() {
		return b
	}
	if t != nil {
		log.Println(pkg.Name(), f.Name(), "detected as", t1.String())
		b.SetType(t)
		return b
	}
	p.errors = append(p.errors, fmt.Errorf("could not understand field for %v %v, underlying: %v", f.Name(), f.Type(), t1.Underlying()))
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
	log.Println("preparing", file.GetName())
	if pkg.TypesInfo == nil {
		log.Println("TypesInfo is empty :(")
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
				log.Println("failed to add message", file.GetName(), msg.GetName(), "error:", err)
			}
		}
	}
	return true
}

func (p *GolangImporter) post(pkg *packages.Package) {
	file := p.protoFilesRegistry.GetProtoFile(pkg.Name, pkg.ID)
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
			log.Println("*******", pkg.ID, t.Id(), "******")
			msg := file.GetMessage(t.Name())
			if msg == nil {
				log.Println("could not find message for", t.Name(), file.GetName())
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
	log.Println("visiting packages")
	packages.Visit(p.pkgs, p.pre, p.post)
	for _, err := range p.errors {
		log.Println(err)
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
	log.Println("getting importer")
	p.Visit()
	return p.Importer
}
