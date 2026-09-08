package lib

import (
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// TestLoadedFileCount asserts LAZY-05 through the exported accessor, not by
// scraping log text: after compiling main.mpconf on a 50-proto corpus, the
// compiler should have loaded only the config's own transitive dependency
// closure (five demanded protos plus whatever lower-indexed pkgs they
// import, per pickDeps's construction), never the whole corpus, and no
// resolution tier past the growable MessageRegistry should ever have fired
// — the corpus never resolves an unknown type URL.
func TestLoadedFileCount(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 50))

	c, err := NewCompiler(dir, false)
	require.NoError(t, err)
	require.NoError(t, c.CompileFile("main.mpconf"))

	registry := c.ModuleService.GetProtoRegistry()
	count := registry.LoadedFileCount()
	t.Logf("compile loaded %d proto files out of a 50-proto corpus", count)
	require.GreaterOrEqual(t, count, 5)
	require.LessOrEqual(t, count, 15)
	require.Equal(t, 0, registry.ScanResolutionCount())
	require.Equal(t, 0, registry.IndexBuildCount())
}
