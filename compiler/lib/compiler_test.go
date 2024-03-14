package lib

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	assert "github.com/stretchr/testify/require"
)

func TestCompiler_CompileFile(t *testing.T) {
	c := NewCompiler(testdata.SmallTestDir(), false)
	assert.NoError(t, c.ModuleService.Init(context.Background(), "CONFIGSPACE"))
	assert.NoError(t, c.SyncModules(context.Background()))

	t.Run("pinc_load_remote.pconf", compilerTest(c, nil))
	t.Run("with_config_rollout_value_is_string.pconf", compilerTest(c, ErrStarlarkEval))
	t.Run("with_config_rollout_value_is_none.pconf", compilerTest(c, ErrStarlarkEval))
	t.Run("with_config_rollout.pconf", compilerTest(c, nil))
	t.Run("load_remote_with_load_local.pconf", compilerTest(c, nil))
	t.Run("load_remote.pconf", compilerTest(c, nil))
	t.Run("validator_ext.pconf", compilerTest(c, ErrInvalidConfig))
	t.Run("test.pconf", compilerTest(c, nil))
	t.Run("validator_test.pconf", compilerTest(c, ErrInvalidConfig))
	t.Run("validator_repeated_test.pconf", compilerTest(c, ErrInvalidConfig))
	t.Run("validator_map_test.pconf", compilerTest(c, ErrInvalidConfig))
	t.Run("validator_passing_test.pconf", compilerTest(c, nil))
	t.Run("enum_test.pconf", compilerTest(c, nil))
	t.Run("enum_top_test.pconf", compilerTest(c, nil))
	t.Run("enum_wrong_types_test.pconf", compilerTest(c, ErrStarlarkEval))
	t.Run("multioutputs_test.mpconf", compilerTest(c, nil))
	t.Run("include_pinc_test.pconf", compilerTest(c, nil))
	t.Run("load_mutable_test.pconf", compilerTest(c, nil))
	t.Run("field_type_any_test.pconf", compilerTest(c, nil))
	t.Run("uninitialized_msg_test.pconf", compilerTest(c, nil))
	t.Run("test_hashable.pconf", compilerTest(c, nil))
	t.Run("output.json.pconf", compilerTest(c, nil))
	t.Run("multioutputs.mpconf", compilerTest(c, nil))
	t.Run("wrong_file_err.name", compilerTest(c, ErrBadConfigExtension))
	t.Run("not_a_dict_err.mpconf", compilerTest(c, ErrNotADictionary))
	t.Run("not_a_string_key_err.mpconf", compilerTest(c, ErrNotStringKey))
	t.Run("not_a_proto_value_err.mpconf", compilerTest(c, ErrNoProtobufValue))
	t.Run("not_a_proto_value_err.pconf", compilerTest(c, ErrNoProtobufValue))
	t.Run("loading_err.pconf", compilerTest(c, ErrLoadStarlark))
	t.Run("eval_err.pconf", compilerTest(c, ErrStarlarkEval))
	t.Run("no_main_err.pconf", compilerTest(c, ErrMainNotFound))
	t.Run("main_not_a_function_err.pconf", compilerTest(c, ErrMainNotCallable))
	t.Run("mutable_not_exists_err.pconf", compilerTest(c, ErrLoadMutable))
	t.Run("mutable_bad_json_err.pconf", compilerTest(c, ErrLoadStarlark))
	t.Run("mutable_bad_proto_filename_err.pconf", compilerTest(c, ErrLoadStarlark))

}

func compilerTest(c *Compiler, wantErr error) func(*testing.T) {
	return func(t *testing.T) {
		err := c.CompileFile(filepath.Base(t.Name()))
		assert.Truef(t, errors.Is(err, wantErr), "err = %v, want = %v", err, wantErr)
	}
}
