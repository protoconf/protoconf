package utils

import (
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestParseAllRegistersIntoFilesResolver pins D-02: files that reach
// FileRegistry through the ParseAll whole-tree eager fallback must register
// into the growable filesResolver through the same before/after diff that
// already drives lazyLoaded — not a second, independent pass — so the two
// sets never diverge (12-RESEARCH.md Pitfall 3).
func TestParseAllRegistersIntoFilesResolver(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 10))
	src := filepath.Join(dir, "src")

	dr := NewDescriptorRegistry()
	dr.ImportPaths = []string{src}
	require.NotNil(t, dr.GetFilesResolver())

	_, err := dr.ParseOne("pkg7/msg7.proto")
	require.NoError(t, err)

	require.NoError(t, dr.ParseAll())

	for key := range dr.FileRegistry {
		_, err := dr.FindFileByPath(key)
		require.NoError(t, err, "FileRegistry key %s must resolve through the growable resolver after ParseAll", key)
	}
	require.Equal(t, 0, dr.FilesResolverRegistrationErrorCount(),
		"a non-zero error count is the Pitfall-3 signature: a file already registered by ParseOne being re-registered by the fallback")

	countBeforeSecondCall := dr.FilesResolverRegistrationCount()
	errorsBeforeSecondCall := dr.FilesResolverRegistrationErrorCount()
	require.NoError(t, dr.ParseAll())
	require.Equal(t, countBeforeSecondCall, dr.FilesResolverRegistrationCount(),
		"a second ParseAll call must be a no-op (eagerFallback guard)")
	require.Equal(t, errorsBeforeSecondCall, dr.FilesResolverRegistrationErrorCount())
}

// TestFilesResolverRegistrationErrorsStayZero runs the ParseOne/ParseAll
// sequence in the opposite order: ParseAll first, then ParseOne for a path
// the fallback already loaded. ParseOne's own early FileRegistry hit means
// it never reaches recordFileLocked for an already-present path, so the
// error count must stay 0 in this ordering too.
func TestFilesResolverRegistrationErrorsStayZero(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 10))
	src := filepath.Join(dir, "src")

	dr := NewDescriptorRegistry()
	dr.ImportPaths = []string{src}
	require.NotNil(t, dr.GetFilesResolver())

	require.NoError(t, dr.ParseAll())
	require.Equal(t, 0, dr.FilesResolverRegistrationErrorCount())

	_, err := dr.ParseOne("pkg7/msg7.proto")
	require.NoError(t, err)
	require.Equal(t, 0, dr.FilesResolverRegistrationErrorCount())
}

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
