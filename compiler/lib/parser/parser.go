package parser

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/protoconf/protoconf/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Parser provides a wrapper around jhump/protoreflect/protoparse that will keep a cache of dpd.FileDescriptor
type Parser struct {
	LocalResolver *protoregistry.Types
	FilesResolver *protoregistry.Files
}

func NewParser(fileRegs ...*protoregistry.Files) *Parser {
	dr := utils.NewDescriptorRegistry()
	fileRegs = append([]*protoregistry.Files{protoregistry.GlobalFiles}, fileRegs...)
	resolver := dr.GetTypesResolver(fileRegs...)

	p := &Parser{
		LocalResolver: resolver,
		FilesResolver: &protoregistry.Files{},
	}
	for _, reg := range fileRegs {
		reg.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			p.FilesResolver.RegisterFile(fd)
			return true
		})
	}

	return p
}

func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
	for _, filename := range filenames {
		fd, err := p.FilesResolver.FindFileByPath(filename)
		if err != nil {
			return nil, err
		}
		d, err := desc.WrapFile(fd)
		if err != nil {
			return nil, err
		}
		results = append(results, d)

	}
	return results, nil
}
