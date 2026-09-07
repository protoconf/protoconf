package lib

// TestModSyncFdsByteIdentical protects LAZY-04: `Store` serializes exactly
// the set `Parse` populated, so a lazy path that ever reused `Parse` — or a
// scoping change that made the `mod` CLI's registry lazy — would write a
// `.fds` that is missing entries, unmarshals cleanly, passes its own
// checksum, and is silently wrong in whatever process loads it next.
//
// grep -rn --include='*_test.go' -E 'GenFileDescriptorSet|\.fds' . (recorded
// in the plan summary) found utils/utils_test.go:39-40 (TestNewDescriptorRegistry),
// which writes a data.fds and checks the checksum returned by Store, but
// asserts nothing about byte content across two independent construction
// paths — so this is not a duplicate guard.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestModSyncFdsByteIdentical(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 40))
	src := filepath.Join(dir, consts.SrcPath)

	// Golden — the shape Sync/GenFileDescriptorSet builds directly.
	golden := utils.NewDescriptorRegistry()
	require.NoError(t, golden.Import(golden.Parse, []*regexp.Regexp{}, src))
	goldenPath := filepath.Join(t.TempDir(), "golden.fds")
	goldenSum, err := golden.Store(goldenPath)
	require.NoError(t, err)
	goldenBytes, err := os.ReadFile(goldenPath)
	require.NoError(t, err)

	// Subject — the mod CLI's construction path: an eager ModuleService.
	ms, err := NewModuleService(dir)
	require.NoError(t, err)
	_ = ms.LoadFromLockFile() // no lock file in the generated corpus; NewCompiler already tolerates this
	reg := ms.GetProtoRegistry()
	subjectPath := filepath.Join(t.TempDir(), "subject.fds")
	sum, err := reg.Store(subjectPath)
	require.NoError(t, err)
	subjectBytes, err := os.ReadFile(subjectPath)
	require.NoError(t, err)

	require.Equal(t, goldenSum, sum)
	require.Equal(t, goldenBytes, subjectBytes)

	// Distinguishability: a lazy ModuleService must not have walked src/, so
	// this guard cannot pass vacuously if GetProtoRegistry ever became lazy
	// for every consumer. Compared against the eager registry's own count
	// (not a hardcoded corpus-relative number): NewDescriptorRegistry seeds
	// ~65 well-known types (google/, buf/validate, protoconf/v1) before any
	// src/ parsing happens, which already exceeds the 40-file corpus size —
	// so the meaningful, dependency-version-agnostic assertion is "lazy
	// loaded strictly fewer files than eager", not "lazy loaded fewer than
	// the corpus size".
	lazyMs, err := NewLazyModuleService(dir)
	require.NoError(t, err)
	lazyCount := len(lazyMs.GetProtoRegistry().FileRegistry)
	require.Less(t, lazyCount, len(reg.FileRegistry), "the lazy path must not have walked src/")

	// Idempotency: serializing the same registry twice writes the same file.
	secondPath := filepath.Join(t.TempDir(), "subject2.fds")
	sum2, err := reg.Store(secondPath)
	require.NoError(t, err)
	secondBytes, err := os.ReadFile(secondPath)
	require.NoError(t, err)
	require.Equal(t, sum, sum2)
	require.Equal(t, subjectBytes, secondBytes)
}

// TestModSyncFdsUnaffectedByConcurrentLazyCompile closes UAT item 3's
// backstop truth (11-UAT.md, test 3): mod sync's serialized `.fds` bytes are
// immune to a concurrent in-process lazy compile. G-11-3's nil-map panic
// fired while building this exact fixture, which is why the truth was never
// actually observed before now — see 11-UAT.md's "note" on test 3 and
// 11-04-PLAN.md's Task 2.
//
// This test stands in for `mod sync` using the eager GetProtoRegistry()
// instance rather than ModuleService.Sync itself, because Sync constructs
// its own utils.NewDescriptorRegistry() behind a Download step that needs
// the network. The instance-separation property under test is identical in
// both cases, but only the stand-in is executed here — which is why the
// corresponding must_haves entry in 11-04-PLAN.md stays tagged
// `verification: backstop`.
func TestModSyncFdsUnaffectedByConcurrentLazyCompile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 40))
	srcDir := filepath.Join(dir, "src")

	const n = 6
	for k := 0; k < n; k++ {
		body := fmt.Sprintf(
			"load(\"//pkg0/msg0.proto\", \"Msg0\")\nload(\"//pkg%d/msg%d.proto\", \"Msg%d\")\n\ndef main():\n    return Msg%d(name=\"c%d\")\n",
			k+10, k+10, k+10, k+10, k,
		)
		path := filepath.Join(srcDir, fmt.Sprintf("conc%d.pconf", k))
		require.NoError(t, os.WriteFile(path, []byte(body), 0644))
	}

	// Eager side, mirroring TestModSyncFdsByteIdentical's subject.
	ms, err := NewModuleService(dir)
	require.NoError(t, err)
	_ = ms.LoadFromLockFile()
	reg := ms.GetProtoRegistry()

	// Golden, taken BEFORE any concurrency starts.
	goldenPath := filepath.Join(t.TempDir(), "golden.fds")
	goldenSum, err := reg.Store(goldenPath)
	require.NoError(t, err)
	goldenBytes, err := os.ReadFile(goldenPath)
	require.NoError(t, err)

	// Lazy side: a distinct *Compiler over the same protoconf root.
	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	// Instance separation is the mechanism the truth relies on — assert it
	// directly rather than only inferring it from unmoved bytes.
	require.NotSame(t, reg, c.ModuleService.GetProtoRegistry())

	storeDir := t.TempDir()
	g := new(errgroup.Group)
	for k := 0; k < n; k++ {
		k := k
		g.Go(func() error {
			return c.CompileFile(fmt.Sprintf("conc%d.pconf", k))
		})
		g.Go(func() error {
			for i := 0; i < 3; i++ {
				storePath := filepath.Join(storeDir, fmt.Sprintf("store-%d-%d.fds", k, i))
				sum, err := reg.Store(storePath)
				if err != nil {
					return err
				}
				if sum != goldenSum {
					return fmt.Errorf("Store checksum drifted: got %s, want %s", sum, goldenSum)
				}
				b, err := os.ReadFile(storePath)
				if err != nil {
					return err
				}
				if string(b) != string(goldenBytes) {
					return fmt.Errorf("Store bytes drifted from golden at %s", storePath)
				}
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())

	// Non-vacuity, both directions: the lazy side really parsed files during
	// the window (so the concurrent Store calls were not racing an idle
	// registry), and the corpus was not fully parsed (so the two registries
	// are genuinely in different states while this ran).
	lazyLoaded := c.ModuleService.GetProtoRegistry().LoadedFileCount()
	require.Greater(t, lazyLoaded, n)
	require.Less(t, lazyLoaded, 40)
}
