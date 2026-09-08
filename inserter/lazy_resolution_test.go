package inserter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// TestInserterResolvesTypeAbsentFromConstructionSnapshot proves CONS-02
// on-demand: type.googleapis.com/test.v1.TestMessage is genuinely absent
// from i.parser's construction-time snapshot (*protoregistry.Types, written
// only at construction and never after) both before and after an insert
// that resolves it, while i.parser.TypeResolver — the tiered resolver —
// resolves it once the lazy tiers have actually run.
func TestInserterResolvesTypeAbsentFromConstructionSnapshot(t *testing.T) {
	kvStore, err := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	require.NoError(t, err)
	testDir := testdata.SmallTestDir()
	i := NewProtoconfInserter(testDir, kvStore)
	require.NotNil(t, i)

	const url = "type.googleapis.com/test.v1.TestMessage"

	// Before insert: absent from the construction snapshot.
	_, err = i.parser.LocalResolver.FindMessageByURL(url)
	require.Error(t, err)
	assert.True(t, errors.Is(err, protoregistry.NotFound), "expected protoregistry.NotFound, got: %v", err)

	// Drive an insert so the lazy tiers actually run for this type.
	require.NoError(t, i.InsertConfigFile("field_type_any_test.materialized_JSON"))

	// After insert: the snapshot object is fixed at construction and never
	// grows — still absent.
	_, err = i.parser.LocalResolver.FindMessageByURL(url)
	require.Error(t, err)
	assert.True(t, errors.Is(err, protoregistry.NotFound), "expected protoregistry.NotFound after insert, got: %v", err)

	// The tiered resolver, in contrast, resolves it.
	mt, err := i.parser.TypeResolver.FindMessageByURL(url)
	require.NoError(t, err)
	require.NotNil(t, mt)
}

// TestInserterUnresolvableTypeReturnsDiagnostic proves that a type URL
// naming a symbol declared nowhere under src/ surfaces the Phase 13 D-02
// diagnostic (naming the symbol, the import roots and the index state)
// rather than being swallowed into a successful insert.
func TestInserterUnresolvableTypeReturnsDiagnostic(t *testing.T) {
	kvStore, err := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	require.NoError(t, err)
	testDir := testdata.SmallTestDir()

	badJSON := `{"protoFile":"test.proto","value":{"@type":"type.googleapis.com/absent.v1.NoSuchMessage","stringValue":"x"}}`
	path := filepath.Join(testDir, "materialized_config", "unresolvable.materialized_JSON")
	require.NoError(t, os.WriteFile(path, []byte(badJSON), 0644))

	// GatherMetadata needs the fixture to have git history, matching how the
	// rest of testdata/small's fixtures are provided (testdata.SmallTestDir
	// commits once at creation; a file written afterward needs its own
	// commit for `git log` to find it).
	repo, err := git.PlainOpen(testDir)
	require.NoError(t, err)
	w, err := repo.Worktree()
	require.NoError(t, err)
	_, err = w.Add(".")
	require.NoError(t, err)
	_, err = w.Commit("add unresolvable fixture", &git.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	require.NoError(t, err)

	i := NewProtoconfInserter(testDir, kvStore)
	require.NotNil(t, i)

	err = i.InsertConfigFile("unresolvable.materialized_JSON")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "absent.v1.NoSuchMessage")
	assert.Contains(t, err.Error(), "symbol index:")
}

// TestInserterCLIExitsZeroOnPerFileFailure pins the inserter CLI's
// skip-and-log-continue contract: one good file and one nonexistent file
// together still exit 0.
func TestInserterCLIExitsZeroOnPerFileFailure(t *testing.T) {
	testDir := testdata.SmallTestDir()
	cmd, err := Command()
	require.NoError(t, err)

	code := cmd.Run([]string{testDir, "test.materialized_JSON", "does_not_exist.materialized_JSON"})
	require.Equal(t, 0, code)
}
