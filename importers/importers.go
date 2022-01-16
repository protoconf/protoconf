package importers

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
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
		MasterFile:     builder.NewFile(masterFileName).SetProto3(true).SetPackageName(getPackageNameForProtoFile(masterFileName)),
		MasterFileName: masterFileName,
		ProtoconfPath:  protoconfPath,
		Files:          map[string]*builder.FileBuilder{},
	}
}

func getPackageNameForProtoFile(s string) string {
	return strings.ReplaceAll(path.Dir(s), "/", ".")
}

// RegisterFile will add a file to fileRegistry
func (i *Importer) RegisterFile(f *builder.FileBuilder) {
	i.Files[f.GetName()] = f
}

func getFilesForBuilder(b builder.Builder, registry map[string]*builder.FileBuilder) map[string]*builder.FileBuilder {
	file := b.GetFile()
	registry[file.GetName()] = file

	for _, child := range b.GetChildren() {
		registry = getFilesForBuilder(child, registry)
	}

	return registry
}

type requiredMessage struct {
	File    string
	Message string
}

func (r *requiredMessage) String() string {
	return fmt.Sprintf("{File: %s, Message: %s}", r.File, r.Message)
}

// GetMessageFromFile returns a message from Files
func (i *Importer) GetMessageFromFile(fileName, msgName string) *builder.MessageBuilder {
	if !strings.HasSuffix(fileName, ".proto") {
		fileName = fileName + ".proto"
	}
	file, ok := i.Files[fileName]
	if !ok {
		return nil
	}
	return file.GetMessage(msgName)
}

var visited = map[string]bool{}

// findRequiredMessages returns a list of messages required by the top message
func (i *Importer) findRequiredMessages(fileName, msgName string) []*requiredMessage {
	rmsgs := []*requiredMessage{}
	msg := i.GetMessageFromFile(fileName, msgName)
	if msg == nil {
		return rmsgs
	}
	file := msg.GetFile()
	if file != nil {
		rmsgs = append(rmsgs, &requiredMessage{File: file.GetName(), Message: msg.GetName()})
	}

	for _, child := range msg.GetChildren() {
		if field, ok := child.(*builder.FieldBuilder); ok {
			fTypeName := field.GetType().GetTypeName()
			if _, exists := visited[fTypeName]; exists {
				continue
			}
			visited[fTypeName] = true
			if fTypeName != "" {
				ret := strings.Split(fTypeName, ".")
				if len(ret) == 1 {
					rmsgs = append(rmsgs, &requiredMessage{File: msg.GetFile().GetName(), Message: ret[0]})
				} else if len(ret) == 2 {
					for _, myMsg := range i.findRequiredMessages(ret[0], ret[1]) {
						rmsgs = append(rmsgs, myMsg)
					}
				}
			}
		}
	}

	return rmsgs
}

// FilterFilesAndMessages filter out messages which are not requird by the top config strunt
func (i Importer) FilterFilesAndMessages(fileName, msgName string) map[string]*builder.FileBuilder {
	rmsgs := i.findRequiredMessages(fileName, msgName)
	outcome := map[string]*builder.FileBuilder{}

	for _, rmsg := range rmsgs {
		oldFile := i.Files[rmsg.File]
		file := builder.NewFile(rmsg.File).SetProto3(true).SetOptions(oldFile.Options).SetPackageName(oldFile.Package)
		if newFile, ok := outcome[rmsg.File]; ok {
			file = newFile
		} else {
			outcome[rmsg.File] = file
		}
		msg := i.GetMessageFromFile(rmsg.File, rmsg.Message)
		if msg != nil {

			file.TryAddMessage(msg)
		}
	}

	return outcome
}

// FilterFiles filter out files which are not required by the top config struct
func (i Importer) FilterFiles(fileName, msgName string) map[string]*builder.FileBuilder {

	rmsgs := i.findRequiredMessages(fileName, msgName)
	outcome := map[string]*builder.FileBuilder{}

	for _, rmsg := range rmsgs {
		outcome[rmsg.File] = i.Files[rmsg.File]
	}

	return outcome
}

// SaveAll will write all the proto files
func (i *Importer) SaveAll() error {
	i.RegisterFile(i.MasterFile)
	descriptors := []*desc.FileDescriptor{}
	for _, b := range i.Files {
		d, err := b.Build()
		if err != nil {
			return err
		}
		descriptors = append(descriptors, d)
	}

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
