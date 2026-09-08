package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// writeIndexProto writes body verbatim at root/relPath, creating parent
// directories as needed.
func writeIndexProto(t *testing.T, root, relPath, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
}

// TestSymbolIndexNestedType pins Pitfall 3: a nested symbol must map to its
// fully-scoped name (nested.v1.Outer.Middle.Inner), never to the
// package-level name (nested.v1.Inner).
func TestSymbolIndexNestedType(t *testing.T) {
	root := t.TempDir()
	body := "syntax = \"proto3\";\npackage nested.v1;\n\n" +
		"message Outer {\n  message Middle {\n    message Inner {\n      string x = 1;\n    }\n  }\n}\n\n" +
		"message Leaf {\n  string y = 1;\n}\n"
	writeIndexProto(t, root, "nested/v1/nested.proto", body)

	index, err := buildSymbolIndex([]string{root})
	require.NoError(t, err)

	for _, sym := range []string{"nested.v1.Outer", "nested.v1.Outer.Middle", "nested.v1.Outer.Middle.Inner", "nested.v1.Leaf"} {
		require.Equal(t, "nested/v1/nested.proto", index[sym], "symbol %s", sym)
	}
	_, ok := index["nested.v1.Inner"]
	require.False(t, ok, "must not index a nested type under just the package name")
	_, ok = index["nested.v1.Middle"]
	require.False(t, ok, "must not index a nested type under just the package name")
}

// TestSymbolIndexIncludesEnums pins symbol-kind coverage: top-level and
// nested enums are both indexed under their correctly-scoped names.
func TestSymbolIndexIncludesEnums(t *testing.T) {
	root := t.TempDir()
	body := "syntax = \"proto3\";\npackage enums.v1;\n\n" +
		"enum Top {\n  TOP_UNSPECIFIED = 0;\n}\n\n" +
		"message Holder {\n  enum Inner {\n    INNER_UNSPECIFIED = 0;\n  }\n}\n"
	writeIndexProto(t, root, "enums/v1/e.proto", body)

	index, err := buildSymbolIndex([]string{root})
	require.NoError(t, err)
	require.Equal(t, "enums/v1/e.proto", index["enums.v1.Top"])
	require.Equal(t, "enums/v1/e.proto", index["enums.v1.Holder.Inner"])
}

// TestIndexBuildLinksZeroFiles pins TYPE-04: the build must never link, so
// LoadedFileCount and FileRegistry are untouched by it.
func TestIndexBuildLinksZeroFiles(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "links/v1/l.proto", "syntax = \"proto3\";\npackage links.v1;\n\nmessage Thing {\n  string x = 1;\n}\n")

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{root}

	beforeLoaded := d.LoadedFileCount()
	beforeRegistry := len(d.FileRegistry)

	require.NoError(t, d.ensureSymbolIndex())

	require.Equal(t, 0, beforeLoaded)
	require.Equal(t, 0, d.LoadedFileCount())
	require.Len(t, d.FileRegistry, beforeRegistry)
}

// TestIndexBuildSkipsBrokenFile pins verified_facts fact 4: a single broken
// file fails the whole batch parse, so the build must fall back to a
// per-file loop rather than losing every valid file's symbols.
func TestIndexBuildSkipsBrokenFile(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "ok/v1/a.proto", "syntax = \"proto3\";\npackage ok.v1;\n\nmessage A {\n  string x = 1;\n}\n")
	writeIndexProto(t, root, "ok/v1/b.proto", "syntax = \"proto3\";\npackage ok.v1;\n\nmessage B {\n  string x = 1;\n}\n")
	writeIndexProto(t, root, "ok/v1/c.proto", "syntax = \"proto3\";\npackage ok.v1;\n\nmessage C {\n  string x = 1;\n}\n")
	writeIndexProto(t, root, "ok/v1/broken.proto", "this is not { valid proto at all")

	index, err := buildSymbolIndex([]string{root})
	require.NoError(t, err)
	require.Equal(t, "ok/v1/a.proto", index["ok.v1.A"])
	require.Equal(t, "ok/v1/b.proto", index["ok.v1.B"])
	require.Equal(t, "ok/v1/c.proto", index["ok.v1.C"])
}

// TestIndexDuplicateSymbolPicksFirstPath pins the deterministic tie-break:
// two files declaring the same symbol resolve to the lexicographically
// smaller path, and two builds over the unchanged tree agree.
func TestIndexDuplicateSymbolPicksFirstPath(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "dup/z.proto", "syntax = \"proto3\";\npackage dup;\n\nmessage Thing {\n  string x = 1;\n}\n")
	writeIndexProto(t, root, "dup/a.proto", "syntax = \"proto3\";\npackage dup;\n\nmessage Thing {\n  string y = 1;\n}\n")

	index1, err := buildSymbolIndex([]string{root})
	require.NoError(t, err)
	require.Equal(t, "dup/a.proto", index1["dup.Thing"])

	index2, err := buildSymbolIndex([]string{root})
	require.NoError(t, err)
	require.Equal(t, index1, index2)
}

// TestIndexEmptyTree pins the degenerate zero-file case: no error, empty
// index, every lookup misses without panicking.
func TestIndexEmptyTree(t *testing.T) {
	root := t.TempDir()
	index, err := buildSymbolIndex([]string{root})
	require.NoError(t, err)
	require.Empty(t, index)

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{root}

	path, ok, err := d.SymbolFile("")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, path)

	path, ok, err = d.SymbolFile("anything")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, path)
}

// TestIndexLookupIsExactBytes pins the exact-match contract: no case
// folding, no normalization, no prefix/suffix matching.
func TestIndexLookupIsExactBytes(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "pkg/x.proto", "syntax = \"proto3\";\npackage pkg;\n\nmessage Thing {\n  string x = 1;\n}\n")

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{root}

	path, ok, err := d.SymbolFile("pkg.Thing")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "pkg/x.proto", path)

	for _, miss := range []string{"pkg.thing", "PKG.THING", "Thing", "pkg.Thing."} {
		_, ok, err := d.SymbolFile(miss)
		require.NoError(t, err)
		require.False(t, ok, "must miss on %q", miss)
	}
}

// TestIndexBuildIsSingleflighted pins the Claude's-Discretion decision to
// reuse d.group for the index build: N concurrent callers pay exactly one
// build.
func TestIndexBuildIsSingleflighted(t *testing.T) {
	root := t.TempDir()
	writeIndexProto(t, root, "sf/x.proto", "syntax = \"proto3\";\npackage sf;\n\nmessage Thing {\n  string x = 1;\n}\n")

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{root}

	const n = 16
	g := new(errgroup.Group)
	for i := 0; i < n; i++ {
		g.Go(func() error {
			return d.ensureSymbolIndex()
		})
	}
	require.NoError(t, g.Wait())
	require.Equal(t, 1, d.IndexBuildCount())
}

// TestIndexBuildRacesParseOne mirrors
// TestLoadSymbolByScanRacesParseOne's structure exactly (symbol_scan_test.go),
// swapping the scan tier for the index tier: the lock-discipline contract
// (Phase 11) is that d.mu is never held across buildSymbolIndex or ParseOne,
// so racing the two must never deadlock and must preserve ParseOne's
// pointer-identity contract.
func TestIndexBuildRacesParseOne(t *testing.T) {
	const fileCount = 6
	src, paths := deadlockCorpus(t, fileCount)

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{src}

	done := make(chan error, 1)
	go func() {
		g := new(errgroup.Group)

		for i := 0; i < fileCount; i++ {
			i := i
			g.Go(func() error {
				d.LoadSymbolByIndex(fmt.Sprintf("dl.v%d.Msg%d", i, i))
				return nil
			})
		}
		for _, p := range paths {
			p := p
			g.Go(func() error {
				_, err := d.ParseOne(p)
				return err
			})
		}

		done <- g.Wait()
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("LoadSymbolByIndex concurrent with ParseOne did not complete within 30s -- lock-discipline regression: LoadSymbolByIndex or buildSymbolIndex most likely holds d.mu across a ParseOne/protoparse call, which sync.RWMutex does not allow")
	}

	// Guard against a vacuous pass: the run must actually have loaded files.
	require.Greater(t, d.LoadedFileCount(), 0,
		"the run must actually have loaded files, or this test proved nothing")

	for _, p := range paths {
		fd, err := d.ParseOne(p)
		require.NoError(t, err)

		d.mu.RLock()
		canonical := d.FileRegistry[p]
		d.mu.RUnlock()
		require.Same(t, canonical, fd, "ParseOne must return the canonical pointer for %s", p)
	}
}
