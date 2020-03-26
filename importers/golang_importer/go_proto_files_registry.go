package golangimporter

import (
	"log"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc/builder"
)

type protoFilesRegistry struct {
	reg map[string]map[string]*builder.FileBuilder
}

func newProtoFilesRegistry() *protoFilesRegistry {
	return &protoFilesRegistry{
		reg: make(map[string]map[string]*builder.FileBuilder),
	}
}

func (p *protoFilesRegistry) GetNameFor(pkgName, pkgID string) string {
	return p.nameFor(pkgName, pkgID)
}

func (p *protoFilesRegistry) nameFor(pkgName, pkgID string) string {
	if pkg, ok := p.reg[pkgName]; ok {
		pkgLen := len(pkg)
		if pkgLen > 0 {
			a := strings.Split(pkgID, "/")
			ret := strings.Join(a, "_")
			if len(a) > pkgLen {
				ret = strings.Join(a[len(a)-pkgLen:], "_")
			}
			return ret
		}
	}
	return pkgName
}

func (p *protoFilesRegistry) GetProtoFile(pkgName, pkgID string) *builder.FileBuilder {
	if pkg, ok := p.reg[pkgName]; ok {
		if file, ok := pkg[pkgID]; ok {
			return file
		}
		for _pkgID, file := range pkg {
			file.SetName(p.nameFor(pkgName, _pkgID) + ".proto")
			file.SetPackageName(p.nameFor(pkgName, _pkgID))
		}
	} else {
		p.reg[pkgName] = make(map[string]*builder.FileBuilder)
	}
	file := builder.NewFile(p.nameFor(pkgName, pkgID) + ".proto")
	file.SetPackageName(p.nameFor(pkgName, pkgID))
	file.SetOptions(&descriptor.FileOptions{GoPackage: &pkgID})
	file.SetProto3(true)
	log.Println("file name:", file.GetName())
	p.reg[pkgName][pkgID] = file
	return file
}

func (p *protoFilesRegistry) GetAllFiles() (result []*builder.FileBuilder) {
	for pkgName, pkgs := range p.reg {
		for pp, _ := range pkgs {
			file := p.GetProtoFile(pkgName, pp)
			if len(file.GetChildren()) > 0 {
				result = append(result, file)
			}
		}
	}
	return result
}
