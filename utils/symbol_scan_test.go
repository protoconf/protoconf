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

// writeSymbolScanProto writes a minimal single-message .proto file at
// root/relPath declaring pkg and a top-level message named msgName.
func writeSymbolScanProto(t *testing.T, root, relPath, pkg, msgName string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relPath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	body := fmt.Sprintf("syntax = \"proto3\";\npackage %s;\n\nmessage %s {\n  string x = 1;\n}\n", pkg, msgName)
	require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
}

// TestSymbolScanCandidatesNarrowsByPackage pins D-05's package-prefix ->
// directory narrowing: two files in different package directories both
// declare the same top-level symbol name, and only the one whose directory
// matches the queried symbol's package prefix is returned.
func TestSymbolScanCandidatesNarrowsByPackage(t *testing.T) {
	root := t.TempDir()
	writeSymbolScanProto(t, root, "a/v1/x.proto", "a.v1", "Thing")
	writeSymbolScanProto(t, root, "b/v1/x.proto", "b.v1", "Thing")

	candidates := symbolScanCandidates([]string{root}, "a.v1.Thing")
	require.Equal(t, []string{"a/v1/x.proto"}, candidates)
}

// TestSymbolScanCandidatesToleratesNesting pins D-05's non-negotiable: the
// scan pattern must match an indented (nested) declaration. An
// ^message-anchored pattern with no leading-whitespace tolerance finds only
// 3.1% of symbols on the benchmark corpus (D-05) because 96.9% of message
// declarations are nested and indented.
func TestSymbolScanCandidatesToleratesNesting(t *testing.T) {
	root := t.TempDir()
	body := "syntax = \"proto3\";\npackage nest.v1;\n\nmessage Outer {\n  message Thing {\n    string x = 1;\n  }\n}\n"
	require.NoError(t, os.MkdirAll(filepath.Join(root, "nest", "v1"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "nest", "v1", "outer.proto"), []byte(body), 0o644))

	candidates := symbolScanCandidates([]string{root}, "nest.v1.Thing")
	require.Equal(t, []string{"nest/v1/outer.proto"}, candidates)

	// Pin the regression an ^message anchor (no [ \t]* prefix) would miss.
	require.Regexp(t, `(?m)^[ \t]*message[ \t]+Thing[ \t]*\{`, body)
	require.NotRegexp(t, `(?m)^message[ \t]+Thing[ \t]*\{`, body)
}

// TestSymbolScanCandidatesEmptyInputs pins the three degenerate shapes the
// scan must handle without panicking: a root that does not exist, an empty
// fullName, and a fullName with no non-package (type) segment.
func TestSymbolScanCandidatesEmptyInputs(t *testing.T) {
	require.NotPanics(t, func() {
		require.Nil(t, symbolScanCandidates([]string{filepath.Join(t.TempDir(), "does-not-exist")}, "a.v1.Thing"))
	})
	require.NotPanics(t, func() {
		require.Nil(t, symbolScanCandidates([]string{t.TempDir()}, ""))
	})
	require.NotPanics(t, func() {
		require.Nil(t, symbolScanCandidates([]string{t.TempDir()}, "all.lowercase.name"))
	})
}

// TestSymbolScanCandidatesStableOrder pins the ordering guarantee the
// scan-index escalation rule depends on: two scans of an unchanged tree
// return an identical candidate slice.
func TestSymbolScanCandidatesStableOrder(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		writeSymbolScanProto(t, root, fmt.Sprintf("pkg/v1/f%d.proto", i), "pkg.v1", "Thing")
	}

	first := symbolScanCandidates([]string{root}, "pkg.v1.Thing")
	second := symbolScanCandidates([]string{root}, "pkg.v1.Thing")
	require.NotEmpty(t, first)
	require.Equal(t, first, second)
}

// TestSymbolScanRespectsCandidateLimit pins T-13-04: a pathological
// many-declarations shape (more than scanCandidateLimit files declaring the
// same symbol) must yield zero candidates, not a burst of parses, so
// LoadSymbolByScan returns false and the caller escalates rather than
// paying for a pathological number of parses.
func TestSymbolScanRespectsCandidateLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < scanCandidateLimit+1; i++ {
		writeSymbolScanProto(t, root, fmt.Sprintf("many/v1/f%d.proto", i), "many.v1", "Thing")
	}

	require.Nil(t, symbolScanCandidates([]string{root}, "many.v1.Thing"))

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{root}
	require.False(t, d.LoadSymbolByScan("many.v1.Thing"))
	require.Equal(t, 0, d.ScanResolutionCount())
}

// TestLoadSymbolByScanRacesParseOne mirrors
// TestIndexBuildRacesParseOne's structure exactly
// (utils/index_build_deadlock_test.go), swapping the index tier for
// LoadSymbolByScan: the lock-discipline contract (Phase 11) is that d.mu is
// never held across symbolScanCandidates or ParseOne, so racing the two
// tiers must never deadlock and must preserve ParseOne's pointer-identity
// contract.
func TestLoadSymbolByScanRacesParseOne(t *testing.T) {
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
				d.LoadSymbolByScan(fmt.Sprintf("dl.v%d.Msg%d", i, i))
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
		t.Fatal("LoadSymbolByScan concurrent with ParseOne did not complete within 30s — lock-discipline regression: LoadSymbolByScan or symbolScanCandidates most likely holds d.mu across a ParseOne call, which sync.RWMutex does not allow")
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
