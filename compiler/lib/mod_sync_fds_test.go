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
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
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
