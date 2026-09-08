package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils"
	"github.com/stretchr/testify/require"
)

// writeLoadedFileCountProto writes a minimal single-message proto3 file at
// root/relPath declaring pkg and a top-level message named msgName. Mirrors
// utils/symbol_scan_test.go's writeSymbolScanProto, which is unexported in
// package utils and therefore not importable here.
func writeLoadedFileCountProto(t *testing.T, root, relPath, pkg, msgName string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	body := fmt.Sprintf("syntax = \"proto3\";\npackage %s;\n\nmessage %s {\n  string x = 1;\n}\n", pkg, msgName)
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
}

// newLoadedFileCountRegistry builds a lazy *utils.DescriptorRegistry the way
// GetProtoRegistry's lazy branch does (ImportPaths set to root, CacheDir set
// to a separate temp dir) and a *Parser over it. The returned *utils.DescriptorRegistry
// IS the serving registry handle -- every count assertion in this file reads
// it directly. LoadedFileCount returns len(d.lazyLoaded) under d.mu.RLock:
// the count is per-*DescriptorRegistry instance, never global, and no
// assertion in this file may read any other registry.
func newLoadedFileCountRegistry(t *testing.T, root string) (*utils.DescriptorRegistry, *Parser) {
	t.Helper()
	d := utils.NewDescriptorRegistry()
	d.ImportPaths = []string{root}
	d.CacheDir = t.TempDir()
	p := NewParserWithDescriptorRegistry(d)
	return d, p
}

// writeCandidateLimitFixture writes scanCandidateLimit+1 (33) files under
// many/v1/, all declaring package many.v1 and message Thing -- the committed
// bail fixture shape from utils/symbol_scan_test.go's
// TestSymbolScanRespectsCandidateLimit. many.v1.Thing's package prefix maps
// to an existing directory (many/v1) under the root, so the scan narrows to
// that subtree and finds all 33 candidates, exceeding scanCandidateLimit=32
// and bailing to the index tier.
func writeCandidateLimitFixture(t *testing.T) (root string, fileCount int) {
	t.Helper()
	root = t.TempDir()
	const n = 33
	for i := 0; i < n; i++ {
		writeLoadedFileCountProto(t, root, fmt.Sprintf("many/v1/f%d.proto", i), "many.v1", "Thing")
	}
	return root, n
}

// TestLoadedFileCountEscalationGuard proves D-08(a): resolving a symbol that
// only the index tier can answer (the scan tier bails at the candidate
// limit) grows the loaded-file count by exactly the resolved file's own
// closure (1 -- these fixture files declare no imports), never by the 33
// candidates the scan tier considered.
func TestLoadedFileCountEscalationGuard(t *testing.T) {
	root, _ := writeCandidateLimitFixture(t)
	d, p := newLoadedFileCountRegistry(t, root)

	before := d.LoadedFileCount()

	mt, err := p.TypeResolver.FindMessageByURL("type.googleapis.com/many.v1.Thing")
	require.NoError(t, err)
	require.NotNil(t, mt)

	require.Equal(t, 0, d.ScanResolutionCount(), "scan tier must have bailed at the candidate limit, not answered")
	require.GreaterOrEqual(t, d.IndexBuildCount(), 1, "index tier must have answered")
	require.Equal(t, before+1, d.LoadedFileCount(), "escalation must load only the resolved file's closure, not all 33 candidates")
}

// TestIndexBuildDoesNotRegisterTheTree proves D-08(a)'s other half: the
// index build itself (ParseFilesButDoNotLink over the whole root) must not
// register every file it walked into the serving registry -- only ParseOne
// on the one resolved path does that.
func TestIndexBuildDoesNotRegisterTheTree(t *testing.T) {
	root, _ := writeCandidateLimitFixture(t)
	d, p := newLoadedFileCountRegistry(t, root)

	_, err := p.TypeResolver.FindMessageByURL("type.googleapis.com/many.v1.Thing")
	require.NoError(t, err)

	var protoFileCount int
	require.NoError(t, filepath.WalkDir(root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !de.IsDir() && filepath.Ext(de.Name()) == ".proto" {
			protoFileCount++
		}
		return nil
	}))

	require.Less(t, d.LoadedFileCount(), protoFileCount, "index build must not register the whole tree")
}

// TestEmptyTypeURLResolvesNothingAndBuildsNoIndex proves the SAFE-03 empty
// edge: an empty type URL, and a name with no resolvable package-and-message
// shape (splitSymbolPackage finds no PascalCase segment), both fail fast
// with zero cost -- neither one ever reaches ParseOne or triggers a symbol
// index build.
func TestEmptyTypeURLResolvesNothingAndBuildsNoIndex(t *testing.T) {
	root, _ := writeCandidateLimitFixture(t)
	d, p := newLoadedFileCountRegistry(t, root)

	before := d.LoadedFileCount()

	_, err := p.TypeResolver.FindMessageByURL("")
	require.Error(t, err)

	_, err = p.TypeResolver.FindMessageByURL("type.googleapis.com/all.lowercase.name")
	require.Error(t, err)

	require.Equal(t, before, d.LoadedFileCount())
	require.Equal(t, 0, d.IndexBuildCount())
}

// TestDuplicateSymbolResolutionIsStable proves the SAFE-03 ordering edge:
// when two files declare the same fully-qualified symbol, resolution is
// deterministic and stable across repeated calls, and the loaded-file count
// grows only once -- the second resolution hits the registry's own
// MessageRegistry (Tier 1) directly, never re-parsing.
func TestDuplicateSymbolResolutionIsStable(t *testing.T) {
	root := t.TempDir()
	writeLoadedFileCountProto(t, root, "dup/v1/a.proto", "dup.v1", "Dup")
	writeLoadedFileCountProto(t, root, "dup/v1/b.proto", "dup.v1", "Dup")
	d, p := newLoadedFileCountRegistry(t, root)

	before := d.LoadedFileCount()

	var firstPath string
	for i := 0; i < 10; i++ {
		mt, err := p.TypeResolver.FindMessageByURL("type.googleapis.com/dup.v1.Dup")
		require.NoError(t, err)
		require.NotNil(t, mt)
		path := mt.Descriptor().ParentFile().Path()
		if i == 0 {
			firstPath = path
		} else {
			require.Equal(t, firstPath, path, "resolution must be stable across repeated calls")
		}
	}

	require.Equal(t, before+1, d.LoadedFileCount(), "duplicate symbol resolution must grow the count only once")
}
