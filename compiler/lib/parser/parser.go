package parser

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/protoconf/protoconf/utils"
	"github.com/scylladb/go-set/strset"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Parser provides a wrapper around jhump/protoreflect/protoparse that will keep a cache of dpd.FileDescriptor
type Parser struct {
	LocalResolver *protoregistry.Types
}

func NewParser(protoconfRoot ...string) *Parser {

	p := &Parser{
		LocalResolver: utils.LocalLinkedResolver(strset.New(protoconfRoot...).List()...),
	}
	return p
}

func (p *Parser) ParseFilesX(filenames ...string) (results []*desc.FileDescriptor, err error) {
	for _, filename := range filenames {
		fd, err := protoregistry.GlobalFiles.FindFileByPath(filename)
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
