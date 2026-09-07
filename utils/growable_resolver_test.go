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
