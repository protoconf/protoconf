package parser

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Parser provides a wrapper around jhump/protoreflect/protoparse that will keep a cache of dpd.FileDescriptor
type Parser struct {
	mutex sync.Mutex
	Cache map[string]*desc.FileDescriptor
	protoparse.Parser
	LocalResolver *protoregistry.Types
}

func NewParser(protoconfRoot string) *Parser {
	p := &Parser{
		mutex:         sync.Mutex{},
		Cache:         make(map[string]*desc.FileDescriptor),
		LocalResolver: utils.LocalResolver(protoconfRoot),
	}
	p.ImportPaths = []string{filepath.Join(protoconfRoot, consts.SrcPath)}
	p.Accessor = p.accessor
	p.LookupImport = p.lookupImport
	return p
}

func (p *Parser) accessor(filename string) (io.ReadCloser, error) {
	if _, exists := p.Cache[strings.TrimPrefix(filename, consts.SrcPath)]; exists {
		// log.Println("accessor found in cache", filename)
		return nil, fmt.Errorf("file %s already parsed", filename)
	}
	if !strings.HasPrefix(filename, p.ImportPaths[0]) {
		return nil, fmt.Errorf("file %s is not under %s", filename, p.ImportPaths[0])
	}
	// log.Println("accessor opening", filename)
	return os.Open(filename)
}

func (p *Parser) lookupImport(filename string) (*desc.FileDescriptor, error) {
	if d, ok := p.Cache[strings.TrimPrefix(filename, consts.SrcPath)]; ok {
		// log.Println("lookupImport receives from cache", filename, d)
		return d, nil
	}
	return nil, fmt.Errorf("could not find %s in cache", filename)
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
