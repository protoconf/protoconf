package parser

import (
	"os"
	"path/filepath"
	"testing"

	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

func readConfigFixture(t *testing.T, protoFile string) (*utils.DescriptorRegistry, error) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, testdata.GenerateCorpus(dir, 10))

	registry := utils.NewDescriptorRegistry()
	registry.ImportPaths = []string{filepath.Join(dir, "src")}
	p := NewParserWithDescriptorRegistry(registry)

	cfg := filepath.Join(dir, "cfg.materialized_JSON")
	body := `{"protoFile":"` + protoFile + `","value":{"@type":"type.googleapis.com/corpus.pkg7.Msg7","name":"x"}}`
	require.NoError(t, os.WriteFile(cfg, []byte(body), 0o644))

	return registry, p.ReadConfig(cfg, &protoconf_pb.ProtoconfValue{})
}

// TestReadConfigUsesProtoFileHint pins the insert-at-scale fix: the declaring
// file named by protoFile is loaded by exact path, so the Any @type resolves
// from the registry (Tier 1) and neither the per-symbol lexical scan (which
// reads every .proto in the package dir per distinct type) nor the whole-tree
// symbol index runs.
func TestReadConfigUsesProtoFileHint(t *testing.T) {
	registry, err := readConfigFixture(t, "pkg7/msg7.proto")
	require.NoError(t, err)
	require.Zero(t, registry.ScanResolutionCount(), "scan tier must not run when protoFile names the declaring file")
	require.Zero(t, registry.IndexBuildCount(), "symbol index must not be built when protoFile names the declaring file")
}

// TestReadConfigBadProtoFileHintFallsBack proves the hint is best-effort: a
// wrong or escaping protoFile is ignored and the tier chain still resolves
// the type.
func TestReadConfigBadProtoFileHintFallsBack(t *testing.T) {
	for _, hint := range []string{"", "no/such.proto", "../escape.proto"} {
		registry, err := readConfigFixture(t, hint)
		require.NoError(t, err, "hint %q", hint)
		require.Positive(t, registry.ScanResolutionCount()+registry.IndexBuildCount(), "hint %q: tier chain should have resolved the type", hint)
	}
}
