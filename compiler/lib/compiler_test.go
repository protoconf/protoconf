package lib

import (
	"context"
	"errors"
	"testing"

	"github.com/protoconf/protoconf/utils/testdata"
	assert "github.com/stretchr/testify/require"
)

func TestCompiler_CompileFile(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
	}{
		{"with_config_rollout_value_is_string.pconf", ErrStarlarkEval},
		{"with_config_rollout_value_is_none.pconf", ErrStarlarkEval},
		{"with_config_rollout.pconf", nil},
		{"load_remote_with_load_local.pconf", nil},
		{"load_remote.pconf", nil},
		{"validator_ext.pconf", ErrInvalidConfig},
		{"test.pconf", nil},
		{"validator_test.pconf", ErrInvalidConfig},
		{"validator_repeated_test.pconf", ErrInvalidConfig},
		{"validator_map_test.pconf", ErrInvalidConfig},
		{"validator_passing_test.pconf", nil},
		{"enum_test.pconf", nil},
		{"enum_top_test.pconf", nil},
		{"enum_wrong_types_test.pconf", ErrStarlarkEval},
		{"multioutputs_test.mpconf", nil},
		{"include_pinc_test.pconf", nil},
		{"load_mutable_test.pconf", nil},
		{"field_type_any_test.pconf", nil},
		{"uninitialized_msg_test.pconf", nil},
		{"test_hashable.pconf", nil},
		{"output.json.pconf", nil},
		{"multioutputs.mpconf", nil},
		{"wrong_file_err.name", ErrBadConfigExtension},
		{"not_a_dict_err.mpconf", ErrNotADictionary},
		{"not_a_string_key_err.mpconf", ErrNotStringKey},
		{"not_a_proto_value_err.mpconf", ErrNoProtobufValue},
		{"not_a_proto_value_err.pconf", ErrNoProtobufValue},
		{"loading_err.pconf", ErrLoadStarlark},
		{"eval_err.pconf", ErrStarlarkEval},
		{"no_main_err.pconf", ErrMainNotFound},
		{"main_not_a_function_err.pconf", ErrMainNotCallable},
		{"mutable_not_exists_err.pconf", ErrLoadMutable},
		{"mutable_bad_json_err.pconf", ErrLoadStarlark},
		{"mutable_bad_proto_filename_err.pconf", ErrLoadStarlark},
	}
	c := NewCompiler(testdata.SmallTestDir(), false)
	assert.NoError(t, c.ModuleService.Init(context.Background(), "CONFIGSPACE"))
	assert.NoError(t, c.SyncModules(context.Background()))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.CompileFile(tt.name); !errors.Is(err, tt.wantErr) {
				t.Errorf("Compiler.CompileFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
