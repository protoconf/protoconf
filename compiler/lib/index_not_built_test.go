package lib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValueOnlyMutableNeverBuildsIndex pins TYPE-07 (ROADMAP criterion 3):
// compiling a config whose only mutable value is a google.protobuf.Value
// must never build or read the symbol index, and must never fire the D-01
// scan tier -- the unconditional globalRegexMatcher seed (utils/utils.go)
// answers google.protobuf.Value at Tier 0, before either new tier could
// ever be reached. This is RESEARCH.md Pitfall 6's exact hazard: a test
// that only checks IndexBuildCount() would pass while the scan tier fired
// on every compile, so all three counters are asserted.
func TestValueOnlyMutableNeverBuildsIndex(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	mutableDir := filepath.Join(root, "mutable_config")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(mutableDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "main.pconf"), []byte(
		"load(\"mutable:value_only\", \"value\")\n\ndef main():\n    return value\n",
	), 0o644))

	// Mirrors utils/testdata/small/mutable_config/mutation_test.materialized_JSON's
	// shape, but the wrapped type is google.protobuf.Value: protojson's
	// well-known-type special case represents that as the raw JSON scalar
	// under "value", once wrapped in an Any envelope carrying "@type".
	require.NoError(t, os.WriteFile(filepath.Join(mutableDir, "value_only.materialized_JSON"), []byte(
		`{"value": {"@type": "type.googleapis.com/google.protobuf.Value", "value": "hello"}}`,
	), 0o644))

	c, err := NewCompiler(root, false)
	require.NoError(t, err)
	require.NoError(t, c.CompileFile("main.pconf"))

	registry := c.ModuleService.GetProtoRegistry()
	require.Equal(t, 0, registry.IndexBuildCount())
	require.Equal(t, 0, registry.IndexCacheHitCount())
	require.Equal(t, 0, registry.ScanResolutionCount())
}
