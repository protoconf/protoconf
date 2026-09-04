package lib

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/bufbuild/protovalidate-go"
	_ "github.com/bufbuild/protovalidate-go/legacy"
	_ "github.com/protoconf/protoconf/pb/protoconf/v1"

	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupValidatorTestDir mirrors TestCompiler_CompileFile's opening sequence
// (SmallTestDir, NewModuleService, Init, Sync) so fixtures written into the
// returned dir before NewCompiler is called are picked up identically to how
// the compiler test suite exercises this package.
func setupValidatorTestDir(t *testing.T) string {
	t.Helper()
	dir := testdata.SmallTestDir()

	ms, err := NewModuleService(dir)
	require.NoError(t, err)
	require.NoError(t, ms.Init(context.Background(), "CONFIGSPACE"))
	require.NoError(t, ms.Sync(context.Background()))

	return dir
}

func TestLoadValidators_OrphanValidatorIsDiscovered(t *testing.T) {
	dir := setupValidatorTestDir(t)

	// no_such_proto.proto does not exist anywhere in this fixture, so this
	// validator is unreachable through the descriptor registry under any
	// loading strategy. It must be found by walking the filesystem instead.
	//
	// Named with a "zz_" prefix so it sorts lexically after the fixture's
	// pre-existing src/test.proto-validator: both files register a
	// validator for the same ValidateMe message, and add_validator's map
	// keeps only the most recently registered function per message
	// (starlark_functions.go). Walk order is now deterministic (that is
	// this task's whole point), so putting this fixture last guarantees
	// its always-failing rule is the one actually enforced, rather than
	// being silently overwritten by test.proto-validator's own validators.
	orphan := `load("//test.proto", "ValidateMe")

def always_fails_validator(v):
    fail("always fails")

add_validator(ValidateMe, always_fails_validator)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "zz_no_such_proto.proto-validator"), []byte(orphan), 0644))

	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	err = c.CompileFile("validator_passing_test.pconf")
	assert.ErrorIs(t, err, ErrInvalidConfig)
}

func TestLoadValidators_DirectoryShapedLikeValidator(t *testing.T) {
	dir := setupValidatorTestDir(t)

	require.NoError(t, os.Mkdir(filepath.Join(dir, "src", "oops.proto-validator"), 0755))

	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	err = c.CompileFile("test.pconf")
	assert.ErrorContains(t, err, "expected validator file, got directory")
}
