package utils

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

	"google.golang.org/protobuf/encoding/protojson"
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
	localFiles      map[string]struct{}
}

func NewDescriptorRegistry() *DescriptorRegistry {
	fds := []protoreflect.FileDescriptor{}
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if globalRegexMatcher.MatchString(fd.Path()) {
			fds = append(fds, fd)
		}
		return true
	})
	descs, err := desc.WrapFiles(fds)
	if err != nil {
		slog.Error("failed to initialize descriptor registry", "error", err)
	}
	fr := map[string]*desc.FileDescriptor{}
	for _, fd := range descs {
		fr[fd.GetName()] = fd
	}
	return &DescriptorRegistry{
		MessageRegistry: *msgregistry.NewMessageRegistryWithDefaults(),
		FileRegistry:    fr,
		localFiles:      map[string]struct{}{},
	}
}

func (d *DescriptorRegistry) Merge(other *DescriptorRegistry) {
	for k, v := range other.FileRegistry {
		d.FileRegistry[k] = v
		d.MessageRegistry.AddFile("type.googleapis.com", v)
	}
}

var globalRegexMatcher = regexp.MustCompile(`(google|google/rpc|google/type|buf/validate|validate|protoconf/v1)/(.*)\.proto`)

func (d *DescriptorRegistry) GetFileDescriptorSet() *descriptorpb.FileDescriptorSet {
	fileDescriptors := []*desc.FileDescriptor{}
	for _, fd := range d.FileRegistry {
		fileDescriptors = append(fileDescriptors, fd)
	}
	return desc.ToFileDescriptorSet(fileDescriptors...)
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
				if _, present := d.FileRegistry[f]; present {
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
		InferImportPaths:                true,
		Accessor: func(filename string) (io.ReadCloser, error) {
			return os.Open(filename)
		},
		LookupImport: func(s string) (*desc.FileDescriptor, error) {
			if fd, ok := d.FileRegistry[s]; ok {
				return fd, nil
			}
			fd, err := desc.LoadFileDescriptor(s)
			if err == nil {
				d.FileRegistry[s] = fd
				return fd, nil
			}

			return nil, fmt.Errorf("failed to find descriptor for file: %s", s)
		},
	}

	err := parse(parser, files)
	if err != nil {
		return errors.Join(errors.New("failed to import files"), err)
	}
	return nil
}

func (d *DescriptorRegistry) Parse(parser *protoparse.Parser, files []string) error {
	d.localFiles = map[string]struct{}{}
	descriptors, err := parser.ParseFiles(files...)
	for _, fd := range descriptors {
		d.MessageRegistry.AddFile("type.googleapis.com", fd)
		d.FileRegistry[fd.GetName()] = fd
		d.localFiles[fd.GetName()] = struct{}{}
	}
	if err != nil {
		return errors.Join(errors.New("failed to parse files"), err)
	}
	return nil
}

type ParserFunc func(parser *protoparse.Parser, files []string) error

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
	keys := []string{}
	for filename := range d.localFiles {
		keys = append(keys, filename)
	}
	sort.Strings(keys)
	fileDescriptors := []*desc.FileDescriptor{}
	for _, k := range keys {
		fileDescriptors = append(fileDescriptors, d.FileRegistry[k])
	}
	fds := desc.ToFileDescriptorSet(fileDescriptors...)
	b, err := proto.Marshal(fds)
	if err != nil {
		return "", err
	}
	h := fileDescriptorSetSum(fds)
	return h, os.WriteFile(path, b, 0644)
}

type fdsSorter struct {
	*descriptorpb.FileDescriptorSet
}

var _ sort.Interface = (*fdsSorter)(nil)

func (f fdsSorter) Len() int {
	return len(f.File)
}

func (f fdsSorter) Less(i, j int) bool {
	return f.File[i].GetName() < f.File[j].GetName()
}

func (f fdsSorter) Swap(i, j int) {
	f.File[i], f.File[j] = f.File[j], f.File[i]
}

func fileDescriptorSetSum(fds *descriptorpb.FileDescriptorSet) string {
	sorted := fdsSorter{fds}
	sort.Stable(sorted)
	b, _ := protojson.Marshal(sorted)

	return fmt.Sprintf("%x", md5.Sum(b))
}

func (d *DescriptorRegistry) Load(path, checksum string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(b, fds)
	if err != nil {
		return err
	}
	md5sum := fileDescriptorSetSum(fds)
	if checksum != md5sum {
		return fmt.Errorf("failed to validate file content: %s (expected: %s, got: %s)", path, checksum, md5sum)
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
