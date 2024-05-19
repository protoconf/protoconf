package utils

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic/msgregistry"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type DescriptorRegistry struct {
	MessageRegistry msgregistry.MessageRegistry
	FileRegistry    map[string]*desc.FileDescriptor
}

func NewDescriptorRegistry() *DescriptorRegistry {
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{}}
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if globalRegexMatcher.MatchString(fd.Path()) {
			fds.File = append(fds.File, protodesc.ToFileDescriptorProto(fd))
		}
		return true
	})
	fr, _ := desc.CreateFileDescriptorsFromSet(fds)
	return &DescriptorRegistry{
		MessageRegistry: *msgregistry.NewMessageRegistryWithDefaults(),
		FileRegistry:    fr,
	}
}

var globalRegexMatcher = regexp.MustCompile(`(google/protobuf|google/api|google/rpc|google/type|buf/validate|validate|protoconf/v1)/(.*)\.proto`)

func (d *DescriptorRegistry) GetFileDescriptorSet() *descriptorpb.FileDescriptorSet {
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{}}
	for _, fd := range d.FileRegistry {
		fds.File = append(fds.File, fd.AsFileDescriptorProto())
	}
	return fds
}

func (d *DescriptorRegistry) GetFilesResolver() *protoregistry.Files {
	fds := d.GetFileDescriptorSet()
	files, err := protodesc.FileOptions{AllowUnresolvable: true}.NewFiles(fds)
	if err != nil {
		slog.Error("failed to generate files resolver", "error", err.Error())
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
					slog.Debug("evaluating regexp", "file", f, "regexp", r)
					skip = true
				}
			}
			if !skip {
				files = append(files, strings.TrimPrefix(strings.TrimPrefix(f, path), "/"))
				continue
			}
			slog.Debug("skipping file", "file", f)
		}
	}
	slog.Debug("files", "files", files)
	parser := &protoparse.Parser{
		ImportPaths:                     paths,
		InterpretOptionsInUnlinkedFiles: true,
		ValidateUnlinkedFiles:           true,
		InferImportPaths:                false,
		LookupImport: func(s string) (*desc.FileDescriptor, error) {
			if fd, ok := d.FileRegistry[s]; ok {
				return fd, nil
			}
			if d, err := desc.LoadFileDescriptor(s); err == nil {
				return d, nil
			}

			return nil, fmt.Errorf("failed to find descriptor for file: %s", s)
		},
	}

	err := parse(parser, files)
	return err
}

func (d *DescriptorRegistry) MergeFileDescriptorSet(fds *descriptorpb.FileDescriptorSet) {
	fff, err := desc.CreateFileDescriptorsFromSet(fds)
	if err != nil {
		slog.Error("error creating file descriptors", "error", err)
	}
	for name, fd := range fff {
		d.FileRegistry[name] = fd
	}
}

func (d *DescriptorRegistry) Store(path string) (string, error) {
	b, err := proto.Marshal(d.GetFileDescriptorSet())
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h), os.WriteFile(path, b, 0644)
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

func (d *DescriptorRegistry) Parse(parser *protoparse.Parser, files []string) error {
	descriptors, err := parser.ParseFiles(files...)
	for _, fd := range descriptors {
		d.MessageRegistry.AddFile("type.googleapis.com", fd)
		d.FileRegistry[fd.GetName()] = fd
	}
	return err
}

type ParserFunc func(parser *protoparse.Parser, files []string) error
