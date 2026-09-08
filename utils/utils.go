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

// ErrNoGrowableResolver is returned by FindFileByPath/RangeFiles when the
// registry has no growable filesResolver — i.e. it is on the eager,
// D-03-excluded path (ImportPaths empty). Growth is opt-in per D-03/D-04, so
// this is the expected signal for every non-compiler consumer, not a bug.
var ErrNoGrowableResolver = errors.New("registry has no growable files resolver")

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

	mu         sync.RWMutex // guards FileRegistry, lazyLoaded on the lazy path
	group      singleflight.Group
	lazyLoaded map[string]struct{}

	// filesResolver, when non-nil, is the growable *protoregistry.Files view
	// for a lazy registry (ImportPaths non-empty). Built once by the first
	// GetFilesResolver call and grown incrementally by registerFileLocked
	// thereafter — never rebuilt. Guarded by mu: a locally-constructed
	// *protoregistry.Files gates its only internal lock on r == GlobalFiles,
	// so it has zero synchronization of its own once it starts growing. nil
	// for every eager registry (D-03), which keeps GetFilesResolver's
	// fresh-build-per-call behavior unchanged for them.
	filesResolver *protoregistry.Files
	// registrationCount is the D-05 observable: incremented once per
	// successful RegisterFile call, i.e. once per distinct file recorded —
	// never once per demand. Guarded by mu. Test-only (D-06): no log line,
	// no CLI surface.
	registrationCount int
	// registrationErrors tallies RegisterFile failures (e.g. a duplicate
	// registration that would otherwise only be logged via slog.Error),
	// resolving research Open Question 1 as yes: a count-only assertion
	// cannot see a FileRegistry/filesResolver divergence hiding behind a
	// swallowed error, but a non-zero error tally can. Guarded by mu.
	// Test-only (D-06).
	registrationErrors int

	// scanResolutions counts how many times the D-01 lexical scan tier
	// (LoadSymbolByScan, utils/symbol_scan.go) has answered a lookup by
	// confirming a candidate via a real ParseOne plus a MessageRegistry
	// re-check. Guarded by mu. Test-only observable, mirroring the Phase 12
	// D-05/D-06 counter precedent: no log line, no CLI surface.
	scanResolutions int

	// CacheDir, when non-empty, is where the symbol index (Tier 3,
	// utils/symbol_index.go) persists under symbolIndexCacheFile. When
	// empty, the symbol index is built in memory and never persisted --
	// every eager registry and every test that does not care.
	CacheDir string

	// symbolIndex is the built symbol -> declaring-file-path map (13-02).
	// nil until ensureSymbolIndex's first successful build or cache load.
	// Guarded by mu.
	symbolIndex map[string]string
	// indexBuilds counts how many times the symbol index has been built
	// from a parse (as opposed to served from CacheDir). Guarded by mu.
	// Test-only observable: no log line, no CLI surface.
	indexBuilds int
	// indexCacheHits counts how many times the symbol index was served
	// from CacheDir instead of rebuilt. Guarded by mu. Test-only
	// observable: no log line, no CLI surface.
	indexCacheHits int
	// indexState records the outcome of the most recent ensureSymbolIndex
	// call: "rebuilt", "cache hit", or "unavailable: <reason>" on a
	// persistence failure. The empty string (its zero value, set by no
	// code path) reads as "not consulted" via IndexState(). Guarded by mu.
	// Test-only observable: no log line, no CLI surface.
	indexState string

	// afterParseHook, when non-nil, runs inside ParseOne's singleflight
	// closure after parser.ParseFiles returns and before d.mu is taken for
	// the insert. It exists so a test can deterministically force the
	// interleaving where some other writer inserts the same path into
	// FileRegistry first, which would otherwise hand callers a
	// non-canonical descriptor pointer; nil in every production path, so
	// it costs one nil check.
	afterParseHook func()
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

// fileDescriptorSetLocked builds a FileDescriptorSet from every entry
// currently in FileRegistry. Callers must already hold d.mu (read or write).
func (d *DescriptorRegistry) fileDescriptorSetLocked() *descriptorpb.FileDescriptorSet {
	fileDescriptors := []*desc.FileDescriptor{}
	for _, fd := range d.FileRegistry {
		fileDescriptors = append(fileDescriptors, fd)
	}
	return desc.ToFileDescriptorSet(fileDescriptors...)
}

func (d *DescriptorRegistry) GetFileDescriptorSet() *descriptorpb.FileDescriptorSet {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.fileDescriptorSetLocked()
}

// GetFilesResolver returns the registry's file-resolution view. For an eager
// registry (ImportPaths empty, D-03) this builds a fresh *protoregistry.Files
// from the current FileRegistry on every call — byte-identical to the
// pre-Phase-12 behavior every non-compiler consumer already relies on. For a
// lazy registry it builds that same way exactly once, caches the result on
// d.filesResolver, and returns the SAME object on every later call: the
// object grows in place via registerFileLocked instead of being rebuilt, so
// any file recorded before the first call here is still present afterwards
// because the one-time build reads the then-current FileRegistry. A nil
// build result (protodesc.NewFiles failure) leaves growth disabled rather
// than panicking — the registry falls back to eager-style fresh builds.
func (d *DescriptorRegistry) GetFilesResolver() *protoregistry.Files {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.ImportPaths) == 0 {
		fds := d.fileDescriptorSetLocked()
		files, err := protodesc.FileOptions{AllowUnresolvable: true}.NewFiles(fds)
		if err != nil {
			slog.Error("failed to generate files resolver", "error", err.Error())
		}
		return files
	}
	if d.filesResolver != nil {
		return d.filesResolver
	}
	fds := d.fileDescriptorSetLocked()
	files, err := protodesc.FileOptions{AllowUnresolvable: true}.NewFiles(fds)
	if err != nil {
		slog.Error("failed to generate files resolver", "error", err.Error())
	}
	if files != nil {
		d.filesResolver = files
	}
	return d.filesResolver
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

		d.mu.RLock()
		hook := d.afterParseHook
		d.mu.RUnlock()
		if hook != nil {
			hook()
		}

		d.mu.Lock()
		d.recordFileLocked(fds[0])
		// Return the CANONICAL registry entry, not the descriptor this
		// goroutine just parsed. recordFileLocked is a no-op when the path
		// is already present, so if anything inserted it while we parsed
		// outside the lock — another writer can land exactly here —
		// returning fds[0] would hand this caller a different pointer than
		// every map lookup sees, breaking ParseOne's own pointer-identity
		// contract.
		// Keyed by GetName(), which is what recordFileLocked stores under;
		// it normally equals path, and the fallback keeps a divergence from
		// turning into a nil return.
		canonical, ok := d.FileRegistry[fds[0].GetName()]
		d.mu.Unlock()
		if !ok {
			canonical = fds[0]
		}

		return canonical, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*desc.FileDescriptor), nil
}

// registerFileLocked registers fd into the growable filesResolver, if one is
// armed, and increments registrationCount on success. A no-op when
// d.filesResolver is nil (no growable view armed yet, or an eager registry).
// A RegisterFile error (e.g. a duplicate) is logged and tallied into
// registrationErrors, matching this file's best-effort slog.Error idiom for
// background bookkeeping (see GetFilesResolver) — it must never abort the
// calling parse. Callers must hold d.mu for writing.
func (d *DescriptorRegistry) registerFileLocked(fd *desc.FileDescriptor) {
	if d.filesResolver == nil {
		return
	}
	if err := d.filesResolver.RegisterFile(fd.UnwrapFile()); err != nil {
		slog.Error("failed to register file in growable resolver", "file", fd.GetName(), "error", err.Error())
		d.registrationErrors++
		return
	}
	d.registrationCount++
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
	d.registerFileLocked(fd)
	for _, dep := range fd.GetDependencies() {
		d.recordFileLocked(dep)
	}
}

// FindFileByPath delegates to the growable filesResolver, returning
// ErrNoGrowableResolver when none is armed (eager registry, D-03). The read
// lock is mandatory: a locally-constructed *protoregistry.Files gates its
// only internal lock on r == GlobalFiles and therefore has no
// synchronization of its own, so this is the only safe way to read it
// concurrently with registerFileLocked's writes.
func (d *DescriptorRegistry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.filesResolver == nil {
		return nil, ErrNoGrowableResolver
	}
	return d.filesResolver.FindFileByPath(path)
}

// RangeFiles delegates to the growable filesResolver, and is a no-op when
// none is armed (eager registry, D-03). The read lock is mandatory for the
// same reason as FindFileByPath.
func (d *DescriptorRegistry) RangeFiles(fn func(protoreflect.FileDescriptor) bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.filesResolver == nil {
		return
	}
	d.filesResolver.RangeFiles(fn)
}

// FilesResolverRegistrationCount reports how many distinct files have been
// successfully registered into the growable filesResolver (D-05). Test-only
// (D-06): no log line, no CLI surface. Safe to call concurrently.
func (d *DescriptorRegistry) FilesResolverRegistrationCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.registrationCount
}

// FilesResolverRegistrationErrorCount reports how many RegisterFile calls
// have failed (D-05, research Open Question 1: paired with the count above
// so a duplicate-registration error that is only logged cannot mask a
// FileRegistry/filesResolver divergence). Test-only (D-06). Safe to call
// concurrently.
func (d *DescriptorRegistry) FilesResolverRegistrationErrorCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.registrationErrors
}

// LoadedFileCount reports how many proto files have been loaded on the lazy
// path via ParseOne (directly, or indirectly through the scan/index tiers,
// both of which call it) — the operator-visible count for LAZY-05. Safe to
// call concurrently with ParseOne.
func (d *DescriptorRegistry) LoadedFileCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.lazyLoaded)
}

// LocalFileCount reports the size of the set Store will serialize -- the
// set Parse (mod sync's whole-tree path) last populated into localFiles.
// This is the emptiness signal GenFileDescriptorSet's guard checks before
// ever calling Store: GetFileDescriptorSet cannot serve this purpose, since
// it ranges FileRegistry, which NewDescriptorRegistry seeds with ~65
// well-known types before any parsing happens and so is never zero.
//
// NOT safe to call concurrently with Import/Parse on the same registry.
// localFiles is deliberately absent from mu's guarantee (see the field
// declaration): Parse writes it -- and resets it, per call -- with no lock
// at all, so an RLock here would advertise a synchronisation that does not
// exist. Every caller today runs on the goroutine that just finished
// Import/Parse, which is what makes the read correct.
//
// ponytail: safe only because Sync's walk is serial and Parse resets
// localFiles per dependency. Parallelising that walk needs a registry per
// dependency, not a lock here -- a lock would silence the race detector
// while leaving the reset to clobber a sibling's entries, which would make
// the G-11-7 guard read 0 for a dependency that parsed fine.
func (d *DescriptorRegistry) LocalFileCount() int {
	return len(d.localFiles)
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
