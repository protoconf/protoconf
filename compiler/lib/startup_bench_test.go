package lib

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestGeneratedCorpusCompiles is a fixture test: it proves the generator
// emits a corpus the real compiler accepts, end-to-end through NewCompiler +
// CompileFile, with no lock file, no git repo, and no CONFIGSPACE marker. It
// passes before and after the lazy-loading milestone and is NOT a
// performance regression guard — see TestCompilerStartupScaling for that.
func TestGeneratedCorpusCompiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 50))

	start := time.Now()
	c, err := NewCompiler(dir, false)
	elapsed := time.Since(start)
	require.NoError(t, err)

	require.NoError(t, c.CompileFile("main.mpconf"))

	entries, err := os.ReadDir(filepath.Join(dir, "materialized_config", "main"))
	require.NoError(t, err)
	require.NotEmpty(t, entries, "expected materialized output under materialized_config/main/")

	fileCount := len(c.ModuleService.GetProtoRegistry().FileRegistry)
	// Sanity-check the generator against BASELINE.md: a wildly
	// unrepresentative corpus (e.g. NewCompiler finishing in microseconds
	// with a handful of files) should be visible here immediately, rather
	// than surfacing later as a milestone planned against a fake number.
	t.Logf("NewCompiler took %s for corpus n=50, FileRegistry has %d files", elapsed, fileCount)
	require.GreaterOrEqual(t, fileCount, 50, "generated corpus should be reflected in the proto registry")
}
