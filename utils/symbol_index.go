package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jhump/protoreflect/desc/protoparse"
	"golang.org/x/mod/sumdb/dirhash"
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

// symbolIndexContentKey hashes every root with dirhash.HashDir(root, "",
// dirhash.Hash1) and joins the per-root results together with the root
// path itself, so a multi-root registry cannot collide with a single-root
// one. dirhash hashes every file under the root, not only .proto files --
// a strict superset of TYPE-06's "any .proto edit invalidates" contract,
// which is intentional: the staleness check must be one-way, and hashing
// more can only ever cause more rebuilding, never a stale serve.
func symbolIndexContentKey(roots []string) (string, error) {
	var b strings.Builder
	for _, root := range roots {
		h, err := dirhash.HashDir(root, "", dirhash.Hash1)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "%s=%s;", root, h)
	}
	return b.String(), nil
}

// writeSymbolIndexCache writes index to path as a versioned, content-keyed,
// line-oriented text file: line 1 is symbolIndexHeader, the content key,
// and the entry count; each following line is "<symbol>\t<path>", sorted
// by symbol so the file is byte-identical across builds of the same tree.
// The write goes to path+".tmp" and is renamed into place, so a partial
// write is never visible as a cache file.
func writeSymbolIndexCache(path, contentKey string, index map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	symbols := make([]string, 0, len(index))
	for s := range index {
		symbols = append(symbols, s)
	}
	sort.Strings(symbols)

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s %d\n", symbolIndexHeader, contentKey, len(index))
	for _, s := range symbols {
		fmt.Fprintf(&b, "%s\t%s\n", s, index[s])
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// loadSymbolIndexCache reads and validates a cache file written by
// writeSymbolIndexCache, refusing it -- returning an error, never a
// partial map -- on any of: a first line that does not start with
// symbolIndexHeader; a stored content key that differs from contentKey; an
// entry count that differs from the number of parsed entries; or a line
// without exactly one tab. This is registry.Load's checksum gate re-applied
// to the symbol index: validate before trusting, refuse and rebuild on any
// mismatch, never serve stale or partial data.
func loadSymbolIndexCache(path, contentKey string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(raw), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("symbol index cache %s: empty file", path)
	}

	header := lines[0]
	first := strings.IndexByte(header, ' ')
	last := strings.LastIndexByte(header, ' ')
	if first == -1 || last == -1 || first == last || header[:first] != symbolIndexHeader {
		return nil, fmt.Errorf("symbol index cache %s: unrecognised header %q", path, header)
	}
	storedKey := header[first+1 : last]
	if storedKey != contentKey {
		return nil, fmt.Errorf("symbol index cache %s: content key mismatch", path)
	}
	wantCount, err := strconv.Atoi(header[last+1:])
	if err != nil {
		return nil, fmt.Errorf("symbol index cache %s: bad entry count: %w", path, err)
	}

	entries := lines[1:]
	if len(entries) != wantCount {
		return nil, fmt.Errorf("symbol index cache %s: entry count mismatch: header says %d, found %d", path, wantCount, len(entries))
	}

	index := make(map[string]string, wantCount)
	for _, line := range entries {
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return nil, fmt.Errorf("symbol index cache %s: malformed entry line %q", path, line)
		}
		index[parts[0]] = parts[1]
	}
	return index, nil
}

// ensureSymbolIndex builds or loads the symbol index exactly once per
// registry, collapsing concurrent callers via singleflight (d.group, the
// same field ParseOne uses -- no second Group). When CacheDir is set, a
// content-key-validated cache hit installs the index without a parse; a
// miss or refusal builds and then persists, and a persistence failure
// leaves indexState recording why but never fails the build -- an
// in-memory index is still a correct index. d.mu is never held across
// buildSymbolIndex, which calls into protoparse (Phase 11 lock discipline;
// utils/parse_all_deadlock_test.go guards this class of inversion).
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
		cacheDir := d.CacheDir
		d.mu.RUnlock()
		if alreadyBuilt {
			return nil, nil
		}

		var contentKey string
		var keyErr error
		if cacheDir != "" {
			contentKey, keyErr = symbolIndexContentKey(roots)
			if keyErr == nil {
				cachePath := filepath.Join(cacheDir, symbolIndexCacheFile)
				if index, loadErr := loadSymbolIndexCache(cachePath, contentKey); loadErr == nil {
					d.mu.Lock()
					d.symbolIndex = index
					d.indexCacheHits++
					d.indexState = "cache hit"
					d.mu.Unlock()
					return nil, nil
				}
			}
		}

		index, buildErr := buildSymbolIndex(roots)
		if buildErr != nil {
			return nil, buildErr
		}

		state := "rebuilt"
		if cacheDir != "" {
			switch {
			case keyErr != nil:
				state = fmt.Sprintf("unavailable: %s", keyErr)
			default:
				cachePath := filepath.Join(cacheDir, symbolIndexCacheFile)
				if writeErr := writeSymbolIndexCache(cachePath, contentKey, index); writeErr != nil {
					state = fmt.Sprintf("unavailable: %s", writeErr)
				}
			}
		}

		d.mu.Lock()
		d.symbolIndex = index
		d.indexBuilds++
		d.indexState = state
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
