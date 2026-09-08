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

// deadlockCorpus writes a small tree where later files import earlier ones, so
// protoparse actually exercises the LookupImport closure — the closure whose
// lock behaviour this test guards. Shared with symbol_scan_test.go and
// symbol_index_test.go.
func deadlockCorpus(t *testing.T, n int) (string, []string) {
	t.Helper()

	src := t.TempDir()
	paths := make([]string, 0, n)

	for i := 0; i < n; i++ {
		name := fmt.Sprintf("dep%d.proto", i)
		body := fmt.Sprintf("syntax = \"proto3\";\npackage dl.v%d;\n", i)
		if i > 0 {
			// Import the previous file so LookupImport is driven for real.
			body += fmt.Sprintf("import \"dep%d.proto\";\n", i-1)
		}
		body += fmt.Sprintf("\nmessage Msg%d {\n    string value = 1;\n}\n", i)

		require.NoError(t, os.WriteFile(filepath.Join(src, name), []byte(body), 0o644))
		paths = append(paths, name)
	}

	return src, paths
}

// TestIndexBuildRacesParseOne covers the D-02 replacement for code review
// finding WR-01 (UAT item 5): WR-01 originally pinned that the whole-tree
// eager fallback (ParseAll, deleted by this plan) did not deadlock against
// concurrent ParseOne calls despite holding d.mu across its entire parse.
// That fallback is gone; the tier that now does filesystem work followed by
// a parse is the symbol index build (Tier 3, 13-02), so this test drives
// LoadSymbolByIndex concurrently with ParseOne instead.
//
// What IS a correctness property — and what this test pins — is that the
// arrangement does not deadlock. Lock discipline (Phase 11) requires that
// d.mu is never held across buildSymbolIndex or a parser.ParseFiles call;
// a future regression that violated that would hang every caller, since
// Go's sync.RWMutex is not reentrant. This test turns that latent hang into
// a red build, and also preserves ParseOne's pointer-identity contract under
// that contention.
//
// This folds what would otherwise be a near-duplicate of
// utils/symbol_index_test.go's own TestIndexBuildRacesParseOne (added by
// 13-02 Task 3, before this plan existed) into the one home this renamed
// file already provides for deadlockCorpus — see 13-03-SUMMARY.md.
func TestIndexBuildRacesParseOne(t *testing.T) {
	const fileCount = 6
	src, paths := deadlockCorpus(t, fileCount)

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{src}

	done := make(chan error, 1)
	go func() {
		g := new(errgroup.Group)

		// One goroutine per corpus symbol drives the index build via the
		// index tier...
		for i := 0; i < fileCount; i++ {
			i := i
			g.Go(func() error {
				d.LoadSymbolByIndex(fmt.Sprintf("dl.v%d.Msg%d", i, i))
				return nil
			})
		}
		// ...while others demand individual paths directly. Whether a given
		// ParseOne lands before, during, or after the index build is exactly
		// the contention this test wants; all orderings must terminate.
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
		// A deadlock here means a lock-discipline regression, most likely a
		// closure inside buildSymbolIndex/ParseOne re-entering d.mu.
		t.Fatal("LoadSymbolByIndex concurrent with ParseOne did not complete within 30s — lock-discipline regression: LoadSymbolByIndex or buildSymbolIndex most likely holds d.mu across a ParseOne/protoparse call, which sync.RWMutex does not allow")
	}

	// Guard against a vacuous pass: the run must actually have loaded the
	// tree, not returned early from a short-circuit.
	require.GreaterOrEqual(t, d.LoadedFileCount(), fileCount,
		"every corpus file should be loaded after the index build plus the per-path demands")

	// And the pointer-identity contract still holds under that contention.
	for _, p := range paths {
		fd, err := d.ParseOne(p)
		require.NoError(t, err)

		d.mu.RLock()
		canonical := d.FileRegistry[p]
		d.mu.RUnlock()
		require.Same(t, canonical, fd, "ParseOne must return the canonical pointer for %s", p)
	}
}
