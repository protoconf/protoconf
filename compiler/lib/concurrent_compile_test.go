package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// TestConcurrentCompile proves the production concurrency shape —
// compiler/service.go's CompileFiles handler and compiler/command.go's
// runLocally both fan out errgroup.Go per file against one shared
// *lib.Compiler — is safe under -race. Before plan 11-03's pointer fix,
// config.messageRegistry copied msgregistry.MessageRegistry by value in
// Compiler.load, giving each config its own zero-value mutex over maps that
// stayed shared with the registry AddFile writes on every on-demand parse;
// that is an unsynchronized concurrent map access. Each goroutine here
// demands one proto only it needs plus one proto every goroutine shares, so
// both the ParseOne singleflight path and the MessageRegistry AddFile path
// are exercised concurrently.
func TestConcurrentCompile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 30))

	srcDir := filepath.Join(dir, "src")
	const n = 8
	for k := 0; k < n; k++ {
		body := fmt.Sprintf(
			"load(\"//pkg0/msg0.proto\", \"Msg0\")\nload(\"//pkg%d/msg%d.proto\", \"Msg%d\")\n\ndef main():\n    return Msg%d(name=\"c%d\")\n",
			k+10, k+10, k+10, k+10, k,
		)
		path := filepath.Join(srcDir, fmt.Sprintf("concurrent%d.pconf", k))
		require.NoError(t, os.WriteFile(path, []byte(body), 0644))
	}

	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	g := new(errgroup.Group)
	for k := 0; k < n; k++ {
		k := k
		g.Go(func() error {
			return c.CompileFile(fmt.Sprintf("concurrent%d.pconf", k))
		})
	}
	require.NoError(t, g.Wait())

	for k := 0; k < n; k++ {
		outputFile := filepath.Join(dir, "materialized_config", fmt.Sprintf("concurrent%d.materialized_JSON", k))
		_, err := os.Stat(outputFile)
		require.NoError(t, err, "expected materialized output for concurrent%d.pconf", k)
	}

	loadedCount := c.ModuleService.GetProtoRegistry().LoadedFileCount()
	require.Greater(t, loadedCount, n, "each goroutine's own proto plus the shared pkg0 proto and their closures should be loaded")
	require.Less(t, loadedCount, 30, "the corpus should not have been fully parsed")
}
