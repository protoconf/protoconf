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
// lock behaviour this test guards.
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

// TestParseAllConcurrentWithParseOneDoesNotDeadlock covers the one genuinely
// testable half of code review finding WR-01 (UAT item 5).
//
// WR-01 observes that ParseAll holds d.mu across its entire whole-tree parse,
// unlike ParseOne, which deliberately parses outside the lock. That lock
// DURATION is a performance property, not a correctness one, and was accepted
// as documented risk: ParseAll is the latched D-03 fallback, guarded by
// d.eagerFallback, so the expensive path runs at most once per registry.
//
// What IS a correctness property — and what this test pins — is that the
// arrangement does not deadlock. It holds today only because Import's
// LookupImport and Parse closures touch FileRegistry directly and never
// re-enter d.mu; Go's sync.RWMutex is not reentrant, so a future change that
// made either closure take RLock while ParseAll holds the write lock would
// hang every caller. This test turns that latent hang into a red build.
func TestParseAllConcurrentWithParseOneDoesNotDeadlock(t *testing.T) {
	const fileCount = 6
	src, paths := deadlockCorpus(t, fileCount)

	d := NewDescriptorRegistry()
	d.ImportPaths = []string{src}

	done := make(chan error, 1)
	go func() {
		g := new(errgroup.Group)

		// One goroutine drives the whole-tree fallback...
		g.Go(func() error { return d.ParseAll() })

		// ...while others demand individual paths. Whether a given ParseOne
		// lands before, during, or after ParseAll is exactly the contention
		// this test wants; all orderings must terminate.
		for _, p := range paths {
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
		// closure inside Import/Parse re-entering d.mu while ParseAll holds it.
		t.Fatal("ParseAll concurrent with ParseOne did not complete within 30s — lock-discipline regression (WR-01): a closure reached from ParseAll most likely re-enters d.mu, which sync.RWMutex does not allow")
	}

	// Guard against a vacuous pass: the run must actually have loaded the tree,
	// not returned early from a short-circuit.
	require.True(t, d.eagerFallback, "ParseAll must have run its whole-tree parse, or this test proved nothing")
	require.GreaterOrEqual(t, d.LoadedFileCount(), fileCount,
		"every corpus file should be loaded after ParseAll plus the per-path demands")

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
