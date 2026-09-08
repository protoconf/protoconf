package utils

import (
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestParseAllRegistersIntoFilesResolver and
// TestFilesResolverRegistrationErrorsStayZero (D-02 disposition, 13-03):
// deleted rather than re-pointed. Both pinned that files reaching
// FileRegistry through the ParseAll whole-tree eager fallback's own
// before/after diff loop registered into the growable filesResolver — a
// mechanism that lived entirely inside ParseAll itself (utils.go's deleted
// diff loop), not in Import/Parse. Every production caller of Import/Parse
// (compiler/lib/module_service.go, server/server.go) always runs on an
// EAGER registry (ImportPaths empty): GetFilesResolver's early branch
// rebuilds a fresh resolver from FileRegistry on every call for that case,
// so growth-through-registerFileLocked never applies to it. The combination
// these two tests exercised — ImportPaths set (lazy) AND Import/Parse called
// directly — has no production caller anywhere in the tree; it only ever
// happened inside ParseAll. Re-pointing at "Import/Parse driven directly"
// would therefore only test test-only glue reimplementing ParseAll's own
// deleted diff loop, asserting nothing a real code path exercises.
// TestRegistrationCountIsIncremental below already fully pins the
// registration-diff invariant via ParseOne, the sole remaining writer into a
// lazy/growable registry after this deletion.

// TestRegistrationCountIsIncremental is RSLV-02's measurable gate: it goes
// red under either regression shape a "fix" for staleness might reintroduce.
//
//   - The count-equality assertion catches re-registering already-present
//     files: if a regression re-registered everything loaded so far on each
//     demand, cumulative registrations would grow quadratically against
//     cumulative distinct files, and the equality below would fail.
//   - The zero-error assertion catches a FileRegistry/filesResolver
//     divergence hiding behind a logged duplicate-registration error, which
//     the count alone cannot see.
//   - The require.Same on the resolver pointer catches a wholesale rebuild:
//     "fixing" staleness by rebuilding the whole FileDescriptorSet on a miss
//     returns a new pointer and never calls RegisterFile at all, so the
//     counter alone would not notice.
func TestRegistrationCountIsIncremental(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 20))
	src := filepath.Join(dir, "src")

	dr := NewDescriptorRegistry()
	dr.ImportPaths = []string{src}

	resolver0 := dr.GetFilesResolver()
	require.NotNil(t, resolver0)

	baseRegistrations := dr.FilesResolverRegistrationCount()
	baseFiles := len(dr.FileRegistry)

	paths := []string{"pkg19/msg19.proto", "pkg18/msg18.proto", "pkg17/msg17.proto"}
	for i, path := range paths {
		_, err := dr.ParseOne(path)
		require.NoError(t, err)

		registrationDelta := dr.FilesResolverRegistrationCount() - baseRegistrations
		fileDelta := len(dr.FileRegistry) - baseFiles
		require.Equal(t, fileDelta, registrationDelta,
			"cumulative registrations must track cumulative distinct files exactly, not grow quadratically (step %d, %s)", i, path)
		require.Equal(t, 0, dr.FilesResolverRegistrationErrorCount())
		require.Same(t, resolver0, dr.GetFilesResolver(),
			"the resolver object must never be replaced (step %d, %s)", i, path)

		if i == 0 {
			require.Greater(t, registrationDelta, 0,
				"the first load's registration delta must be > 0, or growth never fired")
		}
	}

	// A repeat demand is a map lookup and must cost zero registrations.
	countBeforeRepeat := dr.FilesResolverRegistrationCount()
	errorsBeforeRepeat := dr.FilesResolverRegistrationErrorCount()
	_, err := dr.ParseOne("pkg19/msg19.proto")
	require.NoError(t, err)
	require.Equal(t, countBeforeRepeat, dr.FilesResolverRegistrationCount(),
		"re-requesting an already-loaded path must not register anything again")
	require.Equal(t, errorsBeforeRepeat, dr.FilesResolverRegistrationErrorCount())
}
