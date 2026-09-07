package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/reflect/protoreflect"
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
//
// Since phase 12, the same shared registry also owns a growable
// *protoregistry.Files. The assertions after g.Wait() prove the concurrent
// registrations all landed and that FileRegistry and the resolver did not
// diverge under concurrency (SAFE-01, D-02).
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

	reg := c.ModuleService.GetProtoRegistry()
	loadedCount := reg.LoadedFileCount()
	require.Greater(t, loadedCount, n, "each goroutine's own proto plus the shared pkg0 proto and their closures should be loaded")
	require.Less(t, loadedCount, 30, "the corpus should not have been fully parsed")

	// D-02 non-divergence: every key the registry holds must resolve
	// through the growable resolver. A divergence here would mean a
	// concurrent RegisterFile call was lost or landed under a different
	// lock than the FileRegistry write it accompanies.
	for key := range reg.FileRegistry {
		_, err := reg.FindFileByPath(key)
		require.NoError(t, err, "FileRegistry key %s must resolve through the growable resolver after concurrent compiles", key)
	}

	// The resolver holds at least every file the registry holds — not
	// necessarily exactly the same count, since the one-time seed build can
	// legitimately pull in a transitive dependency FileRegistry does not key
	// separately.
	resolverCount := 0
	reg.RangeFiles(func(protoreflect.FileDescriptor) bool {
		resolverCount++
		return true
	})
	require.GreaterOrEqual(t, resolverCount, len(reg.FileRegistry),
		"the resolver must hold at least every file the registry holds")

	// No lost or duplicated registration: every RegisterFile call landed
	// under the same d.mu write lock, so no registration was attempted
	// twice.
	require.Equal(t, 0, reg.FilesResolverRegistrationErrorCount())
	require.Greater(t, reg.FilesResolverRegistrationCount(), 0)
}
