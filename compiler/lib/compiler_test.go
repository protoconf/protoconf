package lib

import (
	"io/ioutil"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	c := NewCompiler("testdata", true)
	dir, err := ioutil.TempDir("", "compiler_output")
	if err != nil {
		t.Fatal("cannot create tmpdir", err)
	}
	t.Log("Test results written to", dir)
	c.MaterializedDir = dir
	assert.NoError(t, c.CompileFile("test.pconf"))
	assert.Error(t, c.CompileFile("validator_test.pconf"))
	assert.Error(t, c.CompileFile("validator_repeated_test.pconf"))
	assert.Error(t, c.CompileFile("validator_map_test.pconf"))
	assert.NoError(t, c.CompileFile("validator_passing_test.pconf"))
	assert.NoError(t, c.CompileFile("enum_test.pconf"))
	assert.Error(t, c.CompileFile("enum_wrong_types_test.pconf"))
	assert.NoError(t, c.CompileFile("multioutputs_test.mpconf"))
	assert.NoError(t, c.CompileFile("include_pinc_test.pconf"))
	assert.NoError(t, c.CompileFile("load_mutable_test.pconf"))
	assert.NoError(t, c.CompileFile("field_type_any_test.pconf"))
	assert.NoError(t, c.CompileFile("uninitialized_msg_test.pconf"))
	assert.NoError(t, c.CompileFile("test_hashable.pconf"))
	assert.NoError(t, c.CompileFile("output.json.pconf"))
	assert.NoError(t, c.CompileFile("multioutputs.mpconf"))
}
