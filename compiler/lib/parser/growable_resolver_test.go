package parser

import (
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestParserFilesResolverGrowsAfterConstruction is the phase-12 tracer: it
// pins RSLV-01 (a file parsed after construction is findable through the
// same FilesResolver field a consumer already holds), D-02 (the growable
// view never diverges from FileRegistry), D-03 (eager registries keep
// today's fresh-build-per-call behavior and get ErrNoGrowableResolver from
// the locked accessor), and the empty/nil edge cases.
func TestParserFilesResolverGrowsAfterConstruction(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 10))
	src := filepath.Join(dir, "src")

	dr := utils.NewDescriptorRegistry()
	dr.ImportPaths = []string{src}
	p := NewParserWithDescriptorRegistry(dr)

	require.NotNil(t, p.FilesResolver)

	// Test 1 (RSLV-01): the view starts small — pkg7/msg7.proto was not
	// seeded at construction time.
	_, err := p.FilesResolver.FindFileByPath("pkg7/msg7.proto")
	require.Error(t, err, "pkg7/msg7.proto must not be resolvable before it has been demanded")

	results, err := p.ParseFilesX("pkg7/msg7.proto")
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Re-read the SAME field — not a new resolver obtained some other way —
	// and assert it now resolves the just-parsed file. This is the whole
	// phase in one assertion: the object a consumer already holds grew.
	fd, err := p.FilesResolver.FindFileByPath("pkg7/msg7.proto")
	require.NoError(t, err)
	require.NotNil(t, fd)

	// Test 2 (D-02 non-divergence): every key FileRegistry now holds must
	// also resolve through the growable view.
	for key := range dr.FileRegistry {
		_, err := dr.FindFileByPath(key)
		require.NoError(t, err, "FileRegistry key %s must resolve through the growable resolver", key)
	}

	// Test 3 (D-03 boundary): an eager registry (ImportPaths empty) keeps
	// today's fresh-build-per-call behavior and has no growable view.
	eager := utils.NewDescriptorRegistry()
	r1 := eager.GetFilesResolver()
	r2 := eager.GetFilesResolver()
	require.NotSame(t, r1, r2, "an eager registry must keep building a fresh resolver on every call")

	_, err = eager.FindFileByPath("test.proto")
	require.ErrorIs(t, err, utils.ErrNoGrowableResolver)

	// Test 4 (empty/nil edge cases).
	empty, err := p.ParseFilesX()
	require.NoError(t, err)
	require.Empty(t, empty)

	emptyFd, err := dr.FindFileByPath("")
	require.Error(t, err)
	require.Nil(t, emptyFd)
}
