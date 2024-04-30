package parser

import (
	"os"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/jhump/protoreflect/desc"
	"github.com/protoconf/protoconf/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Parser provides a wrapper around jhump/protoreflect/protoparse that will keep a cache of dpd.FileDescriptor
type Parser struct {
	LocalResolver   *protoregistry.Types
	FilesResolver   *protoregistry.Files
	FileDescriptors map[string]*desc.FileDescriptor
}

func NewParser(fileRegs ...*protoregistry.Files) *Parser {
	dr := utils.NewDescriptorRegistry()
	fileRegs = append([]*protoregistry.Files{protoregistry.GlobalFiles}, fileRegs...)
	resolver := dr.GetTypesResolver(fileRegs...)

	p := &Parser{
		LocalResolver:   resolver,
		FilesResolver:   &protoregistry.Files{},
		FileDescriptors: dr.FileRegistry,
	}

	fds := &descriptorpb.FileDescriptorSet{}
	for _, reg := range fileRegs {
		reg.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			fds.File = append(fds.File, protodesc.ToFileDescriptorProto(fd))
			p.FilesResolver.RegisterFile(fd)
			return true
		})
	}

	fileDescriptors, _ := desc.CreateFileDescriptorsFromSet(fds)
	// if err != nil {
	// 	panic(err)
	// }
	p.FileDescriptors = fileDescriptors

	return p
}

func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
	for _, filename := range filenames {
		if fd, ok := p.FileDescriptors[filename]; ok {
			results = append(results, fd)
			continue
		}
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

func (p *Parser) ReadConfig(filename string, msg proto.Message) error {
	configReader, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return protojson.UnmarshalOptions{Resolver: p.LocalResolver}.Unmarshal(configReader, msg)

}
