package lib

import (
	"context"
	"errors"
	"testing"

	proto_validate_reflect "github.com/protoconf/proto-validate-reflect"
	"github.com/protoconf/protoconf/utils/testdata"
	assert "github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	c := NewCompiler(testdata.SmallTestDir(), false)
	assert.NoError(t, c.moduleService.Init(context.Background(), "init.pinc"))
	assert.NoError(t, c.SyncModules(context.Background()))
	assert.NoError(t, c.CompileFile("load_remote_with_load_local.pconf"))
	assert.NoError(t, c.CompileFile("load_remote.pconf"))
	assert.ErrorAs(t, c.CompileFile("validator_ext.pconf"), &proto_validate_reflect.ErrorInvalidEmail)
	assert.NoError(t, c.CompileFile("test.pconf"))
	assert.Error(t, c.CompileFile("validator_test.pconf"))
	assert.Error(t, c.CompileFile("validator_repeated_test.pconf"))
	assert.Error(t, c.CompileFile("validator_map_test.pconf"))
	assert.NoError(t, c.CompileFile("validator_passing_test.pconf"))
	assert.NoError(t, c.CompileFile("enum_test.pconf"))
	assert.NoError(t, c.CompileFile("enum_top_test.pconf"))
	assert.Error(t, c.CompileFile("enum_wrong_types_test.pconf"))
	assert.NoError(t, c.CompileFile("multioutputs_test.mpconf"))
	assert.NoError(t, c.CompileFile("include_pinc_test.pconf"))
	assert.NoError(t, c.CompileFile("load_mutable_test.pconf"))
	assert.NoError(t, c.CompileFile("field_type_any_test.pconf"))
	assert.NoError(t, c.CompileFile("uninitialized_msg_test.pconf"))
	assert.NoError(t, c.CompileFile("test_hashable.pconf"))
	assert.NoError(t, c.CompileFile("output.json.pconf"))
	assert.NoError(t, c.CompileFile("multioutputs.mpconf"))
	assert.Error(t, c.CompileFile("wrong_file_err.name"))
	assert.Error(t, c.CompileFile("not_a_dict_err.mpconf"))
	assert.Error(t, c.CompileFile("not_a_string_key_err.mpconf"))
	assert.Error(t, c.CompileFile("not_a_proto_value_err.mpconf"))
	assert.Error(t, c.CompileFile("not_a_proto_value_err.pconf"))
	assert.Error(t, c.CompileFile("loading_err.pconf"))
	assert.Error(t, c.CompileFile("eval_err.pconf"))
	assert.Error(t, c.CompileFile("no_main_err.pconf"))
	assert.Error(t, c.CompileFile("main_not_a_function_err.pconf"))
	assert.Error(t, c.CompileFile("mutable_not_exists_err.pconf"))
	assert.Error(t, c.CompileFile("mutable_bad_json_err.pconf"))
	assert.Error(t, c.CompileFile("mutable_bad_proto_filename_err.pconf"))
}

func TestCompiler_CompileFile(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
	}{
		// TODO: Add test cases.
		{"load_remote_with_load_local.pconf", nil},
		{"load_remote.pconf", nil},
		{"validator_ext.pconf", proto_validate_reflect.ErrorInvalidEmail},
		{"test.pconf", nil},
		{"validator_test.pconf", nil},
		{"validator_repeated_test.pconf", nil},
		{"validator_map_test.pconf", nil},
		{"validator_passing_test.pconf", nil},
		{"enum_test.pconf", nil},
		{"enum_top_test.pconf", nil},
		{"enum_wrong_types_test.pconf", nil},
		{"multioutputs_test.mpconf", nil},
		{"include_pinc_test.pconf", nil},
		{"load_mutable_test.pconf", nil},
		{"field_type_any_test.pconf", nil},
		{"uninitialized_msg_test.pconf", nil},
		{"test_hashable.pconf", nil},
		{"output.json.pconf", nil},
		{"multioutputs.mpconf", nil},
		{"wrong_file_err.name", nil},
		{"not_a_dict_err.mpconf", nil},
		{"not_a_string_key_err.mpconf", nil},
		{"not_a_proto_value_err.mpconf", nil},
		{"not_a_proto_value_err.pconf", nil},
		{"loading_err.pconf", nil},
		{"eval_err.pconf", nil},
		{"no_main_err.pconf", nil},
		{"main_not_a_function_err.pconf", nil},
		{"mutable_not_exists_err.pconf", nil},
		{"mutable_bad_json_err.pconf", nil},
		{"mutable_bad_proto_filename_err.pconf", nil},
	}
	c := NewCompiler(testdata.SmallTestDir(), false)
	assert.NoError(t, c.moduleService.Init(context.Background(), "init.pinc"))
	assert.NoError(t, c.SyncModules(context.Background()))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.CompileFile(tt.name); !errors.Is(err, tt.wantErr) {
				t.Errorf("Compiler.CompileFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
