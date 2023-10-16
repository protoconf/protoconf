package lib

import (
	"os"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	c := NewCompiler("testdata", true)
	dir, err := os.MkdirTemp("", "compiler_output")
	if err != nil {
		t.Fatal("cannot create tmpdir", err)
	}
	t.Log("Test results written to", dir)
	c.MaterializedDir = dir
	assert.Error(t, c.CompileFile("validator_ext.pconf"))
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
