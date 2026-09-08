package utils

import (
	"io"
	"os"
	"path/filepath"

	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/types/descriptorpb"
)

// symbolIndexCacheFile is the filename the persisted symbol index cache is
// written under, inside CacheDir (13-02 Task 2).
const symbolIndexCacheFile = "symbol_index.v1"

// symbolIndexHeader is the first line of a persisted symbol index cache
// file: symbolIndexHeader, a space, the content key, a space, and the
// entry count (13-02 Task 2).
const symbolIndexHeader = "protoconf-symbol-index/v1"

// symbolIndexGroupKey is the fixed singleflight key for the index build,
// reusing the existing d.group field (no second singleflight.Group).
// NUL-prefixed so it cannot collide with a real root-relative proto path:
// ParseOne rejects anything failing filepath.IsLocal, which a
// NUL-prefixed string always does.
const symbolIndexGroupKey = "\x00symbol-index"

// buildSymbolIndex parses every .proto under each root without linking
// (TYPE-02, verified_facts fact 1/3) and maps every message and enum
// symbol -- nested types included -- to the root-relative path of the file
// that declares it. A single broken file fails the whole batch call
// (verified_facts fact 4), so a batch error falls back to a per-file loop
// that keeps every file that parses and skips the ones that error. On a
// duplicate symbol across files, the lexicographically-smaller path wins,
// keeping the result -- and its serialization -- deterministic across runs.
func buildSymbolIndex(roots []string) (map[string]string, error) {
	index := map[string]string{}
	for _, root := range roots {
		files := find(root, ".proto")
		if len(files) == 0 {
			continue
		}
		fdProtos, err := parseUnlinked(root, files)
		if err != nil {
			fdProtos = nil
			for _, f := range files {
				one, oneErr := parseUnlinked(root, []string{f})
				if oneErr != nil {
					continue
				}
				fdProtos = append(fdProtos, one...)
			}
		}
		for _, fd := range fdProtos {
			indexFileDescriptorProto(index, fd)
		}
	}
	return index, nil
}

// parseUnlinked runs protoparse.ParseFilesButDoNotLink over files, resolved
// relative to root via Accessor. ImportPaths and ValidateUnlinkedFiles are
// deliberately left unset: this call never links (nothing to import), and
// turning validation on would make a broken proto nobody's config loads
// fail the index build, contradicting D-02's "a degraded path must never
// become a hard failure for files nobody asked about".
func parseUnlinked(root string, files []string) ([]*descriptorpb.FileDescriptorProto, error) {
	parser := protoparse.Parser{
		Accessor: func(f string) (io.ReadCloser, error) {
			return os.Open(filepath.Join(root, f))
		},
	}
	return parser.ParseFilesButDoNotLink(files...)
}

// joinSymbol joins a scope prefix and a name with a "." unless prefix is
// empty (no proto package declared), in which case it returns name alone.
func joinSymbol(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// indexFileDescriptorProto walks every message and enum in fd -- nested
// types included, using the newly-formed full name as the recursion
// prefix rather than the package (Pitfall 3) -- and installs
// symbol -> fd.GetName() into index. On a duplicate symbol the
// lexicographically-smaller path wins.
func indexFileDescriptorProto(index map[string]string, fd *descriptorpb.FileDescriptorProto) {
	path := fd.GetName()
	install := func(name string) {
		if existing, ok := index[name]; ok && existing <= path {
			return
		}
		index[name] = path
	}

	var walkMessage func(prefix string, m *descriptorpb.DescriptorProto)
	walkMessage = func(prefix string, m *descriptorpb.DescriptorProto) {
		full := joinSymbol(prefix, m.GetName())
		install(full)
		for _, nested := range m.GetNestedType() {
			walkMessage(full, nested)
		}
		for _, e := range m.GetEnumType() {
			install(joinSymbol(full, e.GetName()))
		}
	}

	pkg := fd.GetPackage()
	for _, m := range fd.GetMessageType() {
		walkMessage(pkg, m)
	}
	for _, e := range fd.GetEnumType() {
		install(joinSymbol(pkg, e.GetName()))
	}
}

// ensureSymbolIndex builds -- or, once Task 2's persistence lands, loads --
// the symbol index exactly once per registry, collapsing concurrent
// callers via singleflight (d.group, the same field ParseOne uses -- no
// second Group). d.mu is never held across buildSymbolIndex, which calls
// into protoparse (Phase 11 lock discipline; utils/parse_all_deadlock_test.go
// guards this class of inversion).
func (d *DescriptorRegistry) ensureSymbolIndex() error {
	d.mu.RLock()
	built := d.symbolIndex != nil
	d.mu.RUnlock()
	if built {
		return nil
	}

	_, err, _ := d.group.Do(symbolIndexGroupKey, func() (interface{}, error) {
		d.mu.RLock()
		alreadyBuilt := d.symbolIndex != nil
		roots := d.ImportPaths
		d.mu.RUnlock()
		if alreadyBuilt {
			return nil, nil
		}

		index, buildErr := buildSymbolIndex(roots)
		if buildErr != nil {
			return nil, buildErr
		}

		d.mu.Lock()
		d.symbolIndex = index
		d.indexBuilds++
		d.indexState = "rebuilt"
		d.mu.Unlock()
		return nil, nil
	})
	return err
}

// SymbolFile answers "which file declares fullName", building (or loading)
// the index on first need. The lookup is an exact match on the
// fully-qualified name's bytes: no case folding, no Unicode normalization,
// no suffix or prefix matching.
func (d *DescriptorRegistry) SymbolFile(fullName string) (string, bool, error) {
	if err := d.ensureSymbolIndex(); err != nil {
		return "", false, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	path, ok := d.symbolIndex[fullName]
	return path, ok, nil
}

// IndexBuildCount reports how many times the symbol index has been built
// from a parse (as opposed to served from CacheDir). Test-only observable:
// no log line, no CLI surface. Safe to call concurrently.
func (d *DescriptorRegistry) IndexBuildCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.indexBuilds
}

// IndexCacheHitCount reports how many times the symbol index was served
// from CacheDir instead of rebuilt. Test-only observable: no log line, no
// CLI surface. Safe to call concurrently.
func (d *DescriptorRegistry) IndexCacheHitCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.indexCacheHits
}

// IndexState reports the outcome of the most recent ensureSymbolIndex
// call: "not consulted" (never yet run), "rebuilt", "cache hit", or
// "unavailable: <reason>" on a persistence failure. Test-only observable:
// no log line, no CLI surface. Safe to call concurrently.
func (d *DescriptorRegistry) IndexState() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.indexState == "" {
		return "not consulted"
	}
	return d.indexState
}
