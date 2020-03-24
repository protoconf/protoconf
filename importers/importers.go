package importers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/jhump/protoreflect/desc/protoprint"
)

// Importer will be used as an API to import config structs as proto files
type Importer struct {
	MasterFile     *builder.FileBuilder
	MasterFileName string
	ProtoconfPath  string
	Files          map[string]*builder.FileBuilder
}

// NewImporter returns a new ImporterInstance
func NewImporter(masterFileName, protoconfPath string) *Importer {
	if !strings.HasSuffix(masterFileName, ".proto") {
		masterFileName = fmt.Sprintf("%s.proto", masterFileName)
	}
	return &Importer{
		MasterFile:     builder.NewFile(masterFileName).SetProto3(true),
		MasterFileName: masterFileName,
		ProtoconfPath:  protoconfPath,
		Files:          map[string]*builder.FileBuilder{},
	}
}

// RegisterFile will add a file to fileRegistry
func (i *Importer) RegisterFile(f *builder.FileBuilder) {
	i.Files[f.GetName()] = f
}

func getFilesForBuilder(b builder.Builder, registry map[string]*builder.FileBuilder) map[string]*builder.FileBuilder {
	file := b.GetFile()
	registry[file.GetName()] = file

	for _, child := range b.GetChildren() {
		log.Println(child.GetName())
		registry = getFilesForBuilder(child, registry)
		// if field, ok := child.(*builder.FieldBuilder); ok {
		// 	ft := field.GetType()
		// 	if ft.localMsgType != nil {
		// 		registry = getFilesForBuilder(ft.localMsgType, registry)
		// 	}
		// }
	}

	return registry
}

// SaveAll will write all the proto files
func (i *Importer) SaveAll() error {
	i.RegisterFile(i.MasterFile)
	descriptors := []*desc.FileDescriptor{}
	log.Println(i.Files)
	for _, b := range i.Files {
		d, err := b.Build()
		if err != nil {
			return err
		}
		descriptors = append(descriptors, d)
	}
	// // build all children files
	// for _, child := range i.MasterFile.GetChildren() {
	// 	if file, ok := child.(*builder.FileBuilder); ok {
	// 		log.Println(child.GetName(), "is a file")
	// 		if d, err := file.Build(); err != nil {
	// 			return err
	// 		} else {
	// 			descriptors = append(descriptors, d)
	// 		}
	// 	} else {
	// 		log.Println(child.GetName(), "is not a file")
	// 	}
	// }

	// // Build master file
	// if d, err := i.MasterFile.Build(); err != nil {
	// 	return err
	// } else {
	// 	descriptors = append(descriptors, d)
	// }

	pr := &protoprint.Printer{}
	return pr.PrintProtoFiles(descriptors, i.opener)
}

func (i *Importer) opener(name string) (io.WriteCloser, error) {
	filePath := filepath.Join(i.ProtoconfPath, name)
	dirName := filepath.Dir(filePath)
	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	return &myWriteCloser{f, bufio.NewWriter(f)}, nil
}

type myWriteCloser struct {
	f *os.File
	*bufio.Writer
}

// https://stackoverflow.com/questions/43115699/how-to-get-a-bufio-writer-that-implements-io-writecloser
func (mwc *myWriteCloser) Close() error {
	if err := mwc.Flush(); err != nil {
		return err
	}
	return mwc.f.Close()
}
