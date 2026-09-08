package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIndexCacheReusedOnUnchangedTree pins TYPE-05: a second registry over
// the same tree and cache dir serves the index from disk without rebuilding.
func TestIndexCacheReusedOnUnchangedTree(t *testing.T) {
	root := t.TempDir()
	cacheDir := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")

	a := NewDescriptorRegistry()
	a.ImportPaths = []string{root}
	a.CacheDir = cacheDir
	path, ok, err := a.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)
	require.Equal(t, 1, a.IndexBuildCount())

	b := NewDescriptorRegistry()
	b.ImportPaths = []string{root}
	b.CacheDir = cacheDir
	path, ok, err = b.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)
	require.Equal(t, 0, b.IndexBuildCount())
	require.Equal(t, 1, b.IndexCacheHitCount())
	require.Equal(t, "cache hit", b.IndexState())
}

// TestIndexCacheInvalidatedOnProtoEdit pins TYPE-06: editing a .proto under
// the import root changes the content key, so the next registry rebuilds.
func TestIndexCacheInvalidatedOnProtoEdit(t *testing.T) {
	root := t.TempDir()
	cacheDir := t.TempDir()
	protoPath := filepath.Join(root, "pkg", "x.proto")
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")

	a := NewDescriptorRegistry()
	a.ImportPaths = []string{root}
	a.CacheDir = cacheDir
	_, _, err := a.SymbolFile("pkg.Thing")
	require.NoError(t, err)

	edited := "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n\nmessage Other {\n  string y = 1;\n}\n"
	require.NoError(t, os.WriteFile(protoPath, []byte(edited), 0o644))

	b := NewDescriptorRegistry()
	b.ImportPaths = []string{root}
	b.CacheDir = cacheDir
	path, ok, err := b.SymbolFile("pkg.Other")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)
	require.Equal(t, 1, b.IndexBuildCount())
	require.Equal(t, 0, b.IndexCacheHitCount())
}

// TestIndexCacheRefusedOnKeyMismatch pins T-13-06: a cache file whose stored
// content key does not match the recomputed key is refused and rebuilt, and
// the symbol still resolves correctly.
func TestIndexCacheRefusedOnKeyMismatch(t *testing.T) {
	root := t.TempDir()
	cacheDir := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")

	a := NewDescriptorRegistry()
	a.ImportPaths = []string{root}
	a.CacheDir = cacheDir
	_, _, err := a.SymbolFile("pkg.Thing")
	require.NoError(t, err)

	cachePath := filepath.Join(cacheDir, symbolIndexCacheFile)
	raw, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	corrupted := strings.Replace(string(raw), "h1:", "h1:CORRUPT", 1)
	require.NotEqual(t, string(raw), corrupted)
	require.NoError(t, os.WriteFile(cachePath, []byte(corrupted), 0o644))

	b := NewDescriptorRegistry()
	b.ImportPaths = []string{root}
	b.CacheDir = cacheDir
	path, ok, err := b.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)
	require.Equal(t, 1, b.IndexBuildCount())
	require.Equal(t, 0, b.IndexCacheHitCount())
}

// TestIndexCacheRefusedOnTruncation pins T-13-08: a cache file truncated
// mid-entry, so its header's entry count no longer matches, is refused and
// rebuilt rather than served short.
func TestIndexCacheRefusedOnTruncation(t *testing.T) {
	root := t.TempDir()
	cacheDir := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")
	writeIndexProto(t, root, "pkg/y.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Other {\n  string x = 1;\n}\n")

	a := NewDescriptorRegistry()
	a.ImportPaths = []string{root}
	a.CacheDir = cacheDir
	_, _, err := a.SymbolFile("pkg.Thing")
	require.NoError(t, err)

	cachePath := filepath.Join(cacheDir, symbolIndexCacheFile)
	raw, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	lines := strings.Split(string(raw), "\n")
	require.GreaterOrEqual(t, len(lines), 4) // header + 2 entries + trailing empty
	truncated := strings.Join(lines[:len(lines)-2], "\n") + "\n"
	require.NoError(t, os.WriteFile(cachePath, []byte(truncated), 0o644))

	b := NewDescriptorRegistry()
	b.ImportPaths = []string{root}
	b.CacheDir = cacheDir
	path, ok, err := b.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)
	require.Equal(t, 1, b.IndexBuildCount())
	require.Equal(t, 0, b.IndexCacheHitCount())
}

// TestIndexCacheRefusedOnBadHeader pins the refuse-and-rebuild contract on
// an unrecognised first line.
func TestIndexCacheRefusedOnBadHeader(t *testing.T) {
	root := t.TempDir()
	cacheDir := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")

	require.NoError(t, os.MkdirAll(cacheDir, 0o755))
	cachePath := filepath.Join(cacheDir, symbolIndexCacheFile)
	require.NoError(t, os.WriteFile(cachePath, []byte("not-a-recognised-header\n"), 0o644))

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{root}
	d.CacheDir = cacheDir
	path, ok, err := d.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)
	require.Equal(t, 1, d.IndexBuildCount())
}

// TestIndexCacheIsByteIdenticalAcrossBuilds pins that two builds over the
// same tree, persisted to different paths, write byte-identical files --
// required for the sorted-entry, deterministic-tie-break contract to be
// verifiable at all.
func TestIndexCacheIsByteIdenticalAcrossBuilds(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\nmessage Zeta {\n  string y = 1;\n}\n")

	cacheDir1 := t.TempDir()
	a := NewDescriptorRegistry()
	a.ImportPaths = []string{root}
	a.CacheDir = cacheDir1
	_, _, err := a.SymbolFile("pkg.Thing")
	require.NoError(t, err)

	cacheDir2 := t.TempDir()
	b := NewDescriptorRegistry()
	b.ImportPaths = []string{root}
	b.CacheDir = cacheDir2
	_, _, err = b.SymbolFile("pkg.Thing")
	require.NoError(t, err)

	c1, err := os.ReadFile(filepath.Join(cacheDir1, symbolIndexCacheFile))
	require.NoError(t, err)
	c2, err := os.ReadFile(filepath.Join(cacheDir2, symbolIndexCacheFile))
	require.NoError(t, err)
	require.Equal(t, c1, c2)
}

// TestIndexCacheDisabledWithoutCacheDir pins that an empty CacheDir builds
// in memory only: nothing is written to disk, and a second registry
// rebuilds rather than finding anything to load.
func TestIndexCacheDisabledWithoutCacheDir(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")

	a := NewDescriptorRegistry()
	a.ImportPaths = []string{root}
	_, _, err := a.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.Equal(t, 1, a.IndexBuildCount())

	var found []string
	require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			found = append(found, path)
		}
		return nil
	}))
	require.Len(t, found, 1, "no cache artifact should be written anywhere under root when CacheDir is empty")

	b := NewDescriptorRegistry()
	b.ImportPaths = []string{root}
	_, _, err = b.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.Equal(t, 1, b.IndexBuildCount())
	require.Equal(t, 0, b.IndexCacheHitCount())
}
