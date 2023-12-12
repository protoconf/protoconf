package utils

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"

	_ "github.com/protoconf/proto-validate-reflect/validate"
	"github.com/protoconf/protoconf/consts"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	localResolver *protoregistry.Types
)

type fileDescriptorSorter []*descriptorpb.FileDescriptorProto

var _ sort.Interface = (*fileDescriptorSorter)(nil)

func (s fileDescriptorSorter) Len() int {
	return len(s)
}

func (s fileDescriptorSorter) Less(i, j int) bool {
	return string(*s[i].Name) < string(*s[j].Name)
}

func (s fileDescriptorSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type DescriptorRegistry struct {
	fileDescriptors fileDescriptorSorter
}

func NewDescriptorRegistry() *DescriptorRegistry {
	return &DescriptorRegistry{
		fileDescriptors: fileDescriptorSorter{},
	}
}

func (d *DescriptorRegistry) GetFileDescriptorSet() *descriptorpb.FileDescriptorSet {

	sort.Stable(d.fileDescriptors)
	return &descriptorpb.FileDescriptorSet{
		File: d.fileDescriptors,
	}
}

func (d *DescriptorRegistry) GetFilesResolver() *protoregistry.Files {
	fds := d.GetFileDescriptorSet()
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(fd.Path(), "google") {
			fds.File = append(fds.File,
				protodesc.ToFileDescriptorProto(fd),
			)
		}
		return true
	})
	files, err := protodesc.FileOptions{AllowUnresolvable: true}.NewFiles(fds)
	if err != nil {
		log.Fatalln(err)
	}
	return files
}

func (d *DescriptorRegistry) GetTypesResolver(regs ...*protoregistry.Files) *protoregistry.Types {
	localTypes := new(protoregistry.Types)
	for _, reg := range regs {
		reg.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			if fd.Messages().Len() > 0 {
				for i := 0; i < fd.Messages().Len(); i++ {
					fdm := fd.Messages().Get(i)
					msg := dynamicpb.NewMessageType(fdm)
					localTypes.RegisterMessage(msg)
				}
			}
			if fd.Enums().Len() > 0 {
				for i := 0; i < fd.Enums().Len(); i++ {
					enum := fd.Enums().Get(i)
					localTypes.RegisterEnum(dynamicpb.NewEnumType(enum))
				}
			}
			return true
		})
	}
	return localTypes
}

func (d *DescriptorRegistry) Import(parse ParserFunc, excludes []*regexp.Regexp, paths ...string) error {
	files := []string{}
	for _, path := range paths {

		localFiles := find(path, ".proto")
		for _, f := range localFiles {
			skip := false
			for _, r := range excludes {
				if r.MatchString(f) {
					skip = true
				}
			}
			if skip {
				continue
			}

			files = append(files, strings.TrimPrefix(strings.TrimPrefix(f, path), "/"))
		}
	}
	parser := &protoparse.Parser{
		ImportPaths:                     paths,
		InterpretOptionsInUnlinkedFiles: true,
		LookupImport:                    desc.LoadFileDescriptor,
	}

	fds, err := parse(parser, files, d.fileDescriptors)
	d.fileDescriptors = fds
	return err
}

func (d *DescriptorRegistry) MergeFileDescriptorSet(fds *descriptorpb.FileDescriptorSet) {
	d.fileDescriptors = append(d.fileDescriptors, fds.File...)
}

func (d *DescriptorRegistry) Store(path string) (string, error) {
	b, err := proto.Marshal(d.GetFileDescriptorSet())
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h), os.WriteFile(path, b, 0666)
}

func (d *DescriptorRegistry) Load(path, checksum string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if checksum != fmt.Sprintf("%x", sha256.Sum256(b)) {
		return fmt.Errorf("failed to validate file content: %s (expected: %s, got: %x)", path, checksum, sha256.Sum256(b))
	}
	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(b, fds)
	if err != nil {
		return err
	}
	d.MergeFileDescriptorSet(fds)
	return nil
}

func (d *DescriptorRegistry) ReadConfig(filename string, msg proto.Message) error {
	configReader, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return protojson.UnmarshalOptions{Resolver: d.GetTypesResolver()}.Unmarshal(configReader, msg)

}

// ReadConfig reads a materialized config
func ReadConfig(protoconfRoot string, configName string) (*protoconfvalue.ProtoconfValue, error) {
	filename := filepath.Join(protoconfRoot, consts.CompiledConfigPath, configName+consts.CompiledConfigExtension)

	configReader, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file, file=%s", filename)
	}

	if localResolver == nil {
		localResolver = LocalResolver(protoconfRoot)
	}

	val := &protoconfvalue.ProtoconfValue{}
	err = protojson.UnmarshalOptions{Resolver: localResolver}.Unmarshal(configReader, val)
	if err != nil {
		return nil, err
	}
	return val, nil

}

func LocalResolver(protoconfRoot ...string) *protoregistry.Types {
	return localLinkedResolver(false, protoconfRoot...)
}

func LocalLinkedResolver(protoconfRoot ...string) *protoregistry.Types {
	return localLinkedResolver(true, protoconfRoot...)
}

func localLinkedResolver(link bool, protoconfRoot ...string) *protoregistry.Types {

	localTypes := new(protoregistry.Types)
	localFiles, err := LoadLocalProtoFiles(link, protoconfRoot...)
	if err != nil {
		log.Fatal("LocalResolver:", err)
	}
	localFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		_, err := protoregistry.GlobalFiles.FindFileByPath(file.Path())
		if err != nil {
			protoregistry.GlobalFiles.RegisterFile(file)
		}
		return true
	})
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Messages().Len() > 0 {
			for i := 0; i < fd.Messages().Len(); i++ {
				fdm := fd.Messages().Get(i)
				msg := dynamicpb.NewMessageType(fdm)
				localTypes.RegisterMessage(msg)
			}
		}
		if fd.Enums().Len() > 0 {
			for i := 0; i < fd.Enums().Len(); i++ {
				enum := fd.Enums().Get(i)
				localTypes.RegisterEnum(dynamicpb.NewEnumType(enum))
			}
		}
		return true
	})
	return localTypes
}

// LoadAnyResolver is a util that helps resolve `Any` messages
func LoadAnyResolver(rootPath string, parseFiles ...string) (jsonpb.AnyResolver, error) {
	parser := &protoparse.Parser{ImportPaths: []string{rootPath}}
	descriptors, err := parser.ParseFiles(parseFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%v", parseFiles, err)
	}
	return dynamic.AnyResolver(nil, descriptors...), nil
}

func find(root, ext string) []string {
	var a []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, strings.TrimPrefix(s, root+"/"))
		}
		return nil
	})
	return a
}

type ParserFunc func(parser *protoparse.Parser, files []string, fds fileDescriptorSorter) (fileDescriptorSorter, error)

func Parse(parser *protoparse.Parser, files []string, fds fileDescriptorSorter) (fileDescriptorSorter, error) {
	descriptors, err := parser.ParseFiles(files...)
	if err != nil {
		return fds, fmt.Errorf("parser: %v", err)
	}
	for _, fd := range descriptors {
		fds = append(fds, fd.AsFileDescriptorProto())
	}
	return fds, nil
}

func ParseFilesButDoNotLink(parser *protoparse.Parser, files []string, fds fileDescriptorSorter) (fileDescriptorSorter, error) {
	descriptors, err := parser.ParseFilesButDoNotLink(files...)
	if err != nil {
		return fds, fmt.Errorf("parser: %v", err)
	}
	fds = append(fds, descriptors...)
	return fds, nil
}

func LoadLocalProtoFiles(link bool, protoPaths ...string) (*protoregistry.Files, error) {
	files := []string{}
	for _, protoPath := range protoPaths {
		files = append(files, find(protoPath, ".proto")...)
	}
	parser := &protoparse.Parser{
		ImportPaths:                     protoPaths,
		InterpretOptionsInUnlinkedFiles: true,
		LookupImport:                    desc.LoadFileDescriptor,
	}
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{}}

	if link {
		descriptors, err := parser.ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parser: %v", err)
		}
		for _, fd := range descriptors {
			fds.File = append(fds.File, fd.AsFileDescriptorProto())
		}

	} else {
		descriptors, err := parser.ParseFilesButDoNotLink(files...)
		if err != nil {
			return nil, fmt.Errorf("parser: %v", err)
		}
		fds.File = descriptors

	}

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(fd.Path(), "google") {
			fds.File = append(fds.File,
				protodesc.ToFileDescriptorProto(fd),
			)
		}
		return true
	})
	fileoptions := &protodesc.FileOptions{AllowUnresolvable: true}
	protoFiles, err := fileoptions.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("new files: %v", err)
	}
	return protoFiles, err

}
