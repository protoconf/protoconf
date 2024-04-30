package utils

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	MessageRegistry msgregistry.MessageRegistry
	FileRegistry    map[string]*desc.FileDescriptor
}

func NewDescriptorRegistry() *DescriptorRegistry {
	return &DescriptorRegistry{
		fileDescriptors: fileDescriptorSorter{},
		MessageRegistry: *msgregistry.NewMessageRegistryWithDefaults(),
		FileRegistry:    make(map[string]*desc.FileDescriptor),
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
		ValidateUnlinkedFiles:           true,
		LookupImport: func(s string) (*desc.FileDescriptor, error) {
			fds, err := desc.CreateFileDescriptorsFromSet(d.GetFileDescriptorSet())
			if err != nil {
				slog.Default().Debug("failed to create file descriptors from set", slog.Any("err", err))
			}
			if fd, ok := fds[s]; ok {
				return fd, nil
			}
			if d, err := desc.LoadFileDescriptor(s); err == nil {
				return d, nil
			}
			if fd, err := protoregistry.GlobalFiles.FindFileByPath(s); err == nil {
				return desc.WrapFile(fd)
			}

			return desc.LoadFileDescriptor(s)
		},
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

func (d *DescriptorRegistry) Parse(parser *protoparse.Parser, files []string, fds fileDescriptorSorter) (fileDescriptorSorter, error) {
	descriptors, err := parser.ParseFiles(files...)
	if err != nil {
		return fds, fmt.Errorf("parser: %v", err)
	}
	for _, fd := range descriptors {
		d.MessageRegistry.AddFile("type.googleapis.com", fd)
		d.FileRegistry[fd.GetName()] = fd
		fds = append(fds, fd.AsFileDescriptorProto())
	}
	return fds, nil
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
