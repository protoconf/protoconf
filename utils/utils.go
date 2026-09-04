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
	"sync"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic/msgregistry"

	"golang.org/x/sync/singleflight"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ErrLazyParseDisabled is returned by ParseOne when the registry has no
// ImportPaths configured — on-demand parsing is opt-in (D-01), so every
// eager consumer's behavior stays byte-identical: ParseOne is simply never
// reachable for them.
var ErrLazyParseDisabled = errors.New("on-demand parsing requires import paths")

// ErrUnsafeProtoPath is returned by ParseOne when the requested path is not
// local to the import root (T-11-02): on-demand parsing newly lets a
// load()'d path name a file the eager, pre-populated registry could not
// reach outside src/.
var ErrUnsafeProtoPath = errors.New("proto path escapes the import root")

type DescriptorRegistry struct {
	MessageRegistry msgregistry.MessageRegistry
	FileRegistry    map[string]*desc.FileDescriptor
	localFiles      map[string]struct{}

	// ImportPaths, when non-empty, enables on-demand parsing via ParseOne:
	// those roots are the import paths a requested path resolves against.
	// When empty (the default, and every eager consumer's registry) the
	// registry is eager-only and ParseOne always returns
	// ErrLazyParseDisabled.
	ImportPaths []string

	mu            sync.RWMutex // guards FileRegistry, lazyLoaded, eagerFallback on the lazy path
	group         singleflight.Group
	lazyLoaded    map[string]struct{}
	eagerFallback bool
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
		lazyLoaded:      map[string]struct{}{},
	}
}

func (d *DescriptorRegistry) Merge(other *DescriptorRegistry) {
	for k, v := range other.FileRegistry {
		d.FileRegistry[k] = v
		d.MessageRegistry.AddFile("type.googleapis.com", v)
	}
}

var globalRegexMatcher = regexp.MustCompile(`(google|google/rpc|google/type|buf/validate|validate|protoconf/v1)/(.*)\.proto`)

// FileDescriptor returns the file descriptor registered under name, if any.
// Safe for concurrent use with ParseOne — a snapshot taken while another
// goroutine is inside ParseOne cannot observe a torn map.
func (d *DescriptorRegistry) FileDescriptor(name string) (*desc.FileDescriptor, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	fd, ok := d.FileRegistry[name]
	return fd, ok
}

func (d *DescriptorRegistry) GetFileDescriptorSet() *descriptorpb.FileDescriptorSet {
	d.mu.RLock()
	fileDescriptors := []*desc.FileDescriptor{}
	for _, fd := range d.FileRegistry {
		fileDescriptors = append(fileDescriptors, fd)
	}
	d.mu.RUnlock()
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

// ParseOne parses and links a single file by path on first request, memoising
// the result — and its whole transitive dependency closure — in FileRegistry
// so a second request for the same path, or a later request for one of its
// imports, is a map lookup (LAZY-02), not a re-parse. Concurrent requests for
// the same not-yet-parsed path are collapsed into one parse via singleflight.
//
// ParseOne never touches localFiles (LAZY-03): Store() serializes exactly
// that map, and Parse (mod sync's whole-tree path) is its only writer.
//
// ParseOne requires ImportPaths to be set; when empty it returns
// ErrLazyParseDisabled, which is what keeps every eager consumer's behavior
// unchanged (D-01).
func (d *DescriptorRegistry) ParseOne(path string) (*desc.FileDescriptor, error) {
	d.mu.RLock()
	importPaths := d.ImportPaths
	d.mu.RUnlock()
	if len(importPaths) == 0 {
		return nil, errors.Join(ErrLazyParseDisabled, fmt.Errorf("path=%s", path))
	}

	if !filepath.IsLocal(filepath.FromSlash(path)) {
		return nil, errors.Join(ErrUnsafeProtoPath, fmt.Errorf("path=%s", path))
	}

	d.mu.RLock()
	if fd, ok := d.FileRegistry[path]; ok {
		d.mu.RUnlock()
		return fd, nil
	}
	d.mu.RUnlock()

	v, err, _ := d.group.Do(path, func() (interface{}, error) {
		// Another goroutine may have finished parsing this path while we
		// waited to enter the group.
		d.mu.RLock()
		if fd, ok := d.FileRegistry[path]; ok {
			d.mu.RUnlock()
			return fd, nil
		}
		d.mu.RUnlock()

		// Lock discipline: never hold d.mu while calling parser.ParseFiles.
		// protoparse calls LookupImport from inside ParseFiles, and that
		// closure takes RLock — holding the write lock across the call
		// would self-deadlock. Read, unlock, parse, then take the write
		// lock only for the inserts below.
		parser := &protoparse.Parser{
			ImportPaths: importPaths,
			Accessor: func(filename string) (io.ReadCloser, error) {
				return os.Open(filename)
			},
			LookupImport: func(s string) (*desc.FileDescriptor, error) {
				d.mu.RLock()
				fd, ok := d.FileRegistry[s]
				d.mu.RUnlock()
				if ok {
					return fd, nil
				}
				fd, err := desc.LoadFileDescriptor(s)
				if err == nil {
					return fd, nil
				}
				return nil, fmt.Errorf("failed to find descriptor for file: %s", s)
			},
		}

		fds, err := parser.ParseFiles(path)
		if err != nil {
			return nil, errors.Join(errors.New("failed to parse file on demand"), err)
		}

		d.mu.Lock()
		d.recordFileLocked(fds[0])
		d.mu.Unlock()

		return fds[0], nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*desc.FileDescriptor), nil
}

// recordFileLocked records fd and its whole transitive dependency closure
// into FileRegistry and lazyLoaded, skipping names already present, and
// registers every newly recorded file's messages with MessageRegistry
// (already internally mutex-protected, so this needs no lock of its own).
// Callers must hold d.mu for writing.
func (d *DescriptorRegistry) recordFileLocked(fd *desc.FileDescriptor) {
	if _, ok := d.FileRegistry[fd.GetName()]; ok {
		return
	}
	d.FileRegistry[fd.GetName()] = fd
	d.lazyLoaded[fd.GetName()] = struct{}{}
	d.MessageRegistry.AddFile("type.googleapis.com", fd)
	for _, dep := range fd.GetDependencies() {
		d.recordFileLocked(dep)
	}
}

// ParseAll performs the D-03 whole-tree eager fallback exactly once per
// registry: fired only when a type-URL lookup misses both the
// construction-time snapshot and the growable MessageRegistry, so a registry
// that a lazy-by-path lookup structurally cannot answer (no protoFile hint,
// nothing parsed yet) still resolves correctly instead of failing the
// compile. A no-op if the fallback already fired or the registry is not
// configured for on-demand parsing.
func (d *DescriptorRegistry) ParseAll() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.eagerFallback || len(d.ImportPaths) == 0 {
		return nil
	}
	before := make(map[string]struct{}, len(d.FileRegistry))
	for k := range d.FileRegistry {
		before[k] = struct{}{}
	}
	// Import's own LookupImport closure touches FileRegistry without
	// locking, which is safe here only because this goroutine already
	// holds the write lock for the whole call.
	err := d.Import(d.Parse, []*regexp.Regexp{}, d.ImportPaths...)
	for k := range d.FileRegistry {
		if _, ok := before[k]; !ok {
			d.lazyLoaded[k] = struct{}{}
		}
	}
	d.eagerFallback = true
	return err
}

// LoadedFileCount reports how many proto files have been loaded on the lazy
// path (either via ParseOne or, once, via the ParseAll fallback) — the
// operator-visible count for LAZY-05. Safe to call concurrently with ParseOne.
func (d *DescriptorRegistry) LoadedFileCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.lazyLoaded)
}

// FellBackToEager reports whether the D-03 whole-tree eager fallback has
// fired for this registry.
func (d *DescriptorRegistry) FellBackToEager() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.eagerFallback
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
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(sorted.FileDescriptorSet)

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
