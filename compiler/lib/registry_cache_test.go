package lib

import (
	"sync"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestGetProtoRegistryIsSingletonUnderConcurrency covers code review finding
// WR-04 (UAT item 6).
//
// GetProtoRegistry caches into m.cachedRegistry. Before the fix that guard was
// an unsynchronized check-then-act: two callers racing an unprimed service
// would both build a registry and both assign, and the loser would return an
// ORPHAN registry that is not the one stored in the field. The orphan carries
// its own FileRegistry, so its descriptors are not pointer-identical to the
// winner's and LAZY-02's memoisation guarantee silently stops holding across
// those callers.
//
// The service is constructed directly rather than through NewCompiler on
// purpose: NewCompiler primes cachedRegistry synchronously before any
// concurrency exists, which is exactly what keeps this latent in production
// today — and would hide it from this test.
//
// Two independent detectors: the require.Same assertion below is deterministic
// without the race detector, and under `go test -race` the unsynchronized
// write/write on the field is flagged directly.
func TestGetProtoRegistryIsSingletonUnderConcurrency(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 10))

	ms, err := NewLazyModuleService(dir)
	require.NoError(t, err)

	const goroutines = 16

	// Release every goroutine at once so they collide inside the slow path
	// rather than trickling through the fast path one at a time.
	var start sync.WaitGroup
	start.Add(1)

	var done sync.WaitGroup
	registries := make([]interface{}, goroutines)

	for i := 0; i < goroutines; i++ {
		done.Add(1)
		go func(idx int) {
			defer done.Done()
			start.Wait()
			registries[idx] = ms.GetProtoRegistry()
		}(i)
	}

	start.Done()
	done.Wait()

	first := registries[0]
	require.NotNil(t, first, "GetProtoRegistry must not return nil")

	for i := 1; i < goroutines; i++ {
		require.Same(t, first, registries[i],
			"goroutine %d received a different *utils.DescriptorRegistry — GetProtoRegistry built more than one registry, so callers hold divergent descriptor caches (WR-04)", i)
	}

	// A later caller taking the fast path must observe that same instance.
	require.Same(t, first, ms.GetProtoRegistry(),
		"a post-priming call must return the cached registry, not a rebuilt one")
}
