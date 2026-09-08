package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
)

// nestedAnyOutput mirrors the JSON shape the nested_any_mutation fixture
// carries: a top-level TestMessage whose any_field unpacks to a depth-2
// nested google.protobuf.Any (test.v1.MessageWithSubMessage.SubMessage).
type nestedAnyOutput struct {
	StringValue string `json:"stringValue"`
	AnyField    struct {
		Type  string `json:"@type"`
		Value string `json:"value"`
	} `json:"anyField"`
}

// TestLoadMutableResolvesNestedAny proves loadMutable resolves the mutable
// value's type -- and the depth-2 nested google.protobuf.Any inside it --
// entirely through l.parser.TypeResolver (the shared chain), never through
// l.moduleService's own registry (CONS-05).
//
// The divergence is forced deterministically, in isolation from the rest of
// the compile: the loader's parser keeps the compiler's own, correctly-wired
// TypeResolver (test.proto resolves fine through it, via the shared
// scan/index chain), but its moduleService is swapped for a cold, empty one
// that has never parsed anything and never will (it points at an empty temp
// directory with no src/ tree at all). Before the fix, loadMutable's second
// lookup reached directly into
// l.moduleService.GetProtoRegistry().MessageRegistry -- a registry the cold
// ModuleService can never populate -- even though l.parser.TypeResolver,
// two lines above, already resolved the identical type URL correctly (this
// produced a nil *desc.MessageDescriptor silently accepted by the missing
// nil-check at the bypassed call site, then a nil-pointer panic deep inside
// dynamic.Message.Unmarshal). After the fix
// (desc.WrapMessage(mt.Descriptor())), loadMutable never touches
// l.moduleService for message resolution at all, so this succeeds
// regardless of what the moduleService happens to point at.
func TestLoadMutableResolvesNestedAny(t *testing.T) {
	dir := testdata.SmallTestDir()
	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	coldMS, err := NewLazyModuleService(t.TempDir())
	require.NoError(t, err)

	loader := &starlarkLoader{
		cache:         make(map[string]*cacheEntry),
		mutableDir:    filepath.Join(dir, consts.MutableConfigPath),
		srcDir:        filepath.Join(dir, consts.SrcPath),
		parser:        c.parser,
		moduleService: coldMS,
	}

	_, err = loader.loadMutable(consts.MutableConfigPrefix + "nested_any_mutation")
	require.NoError(t, err, "loadMutable must resolve the value's type entirely through l.parser.TypeResolver, never through l.moduleService's own registry")

	// Full-pipeline regression, proving the nested Any value round-trips
	// correctly end to end: the same fixture compiles through the
	// compiler's own (consistent) parser and moduleService, with no
	// load("//test.proto") anywhere in the .pconf.
	require.NoError(t, c.CompileFile("load_mutable_nested_any_test.pconf"))
	outputFile := filepath.Join(dir, consts.CompiledConfigPath, "load_mutable_nested_any_test"+consts.CompiledConfigExtension)
	b, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var out struct {
		Value nestedAnyOutput `json:"value"`
	}
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, "outer", out.Value.StringValue)
	require.Equal(t, "type.googleapis.com/test.v1.MessageWithSubMessage.SubMessage", out.Value.AnyField.Type)
	require.Equal(t, "deep", out.Value.AnyField.Value)
}

// TestLoadMutableEmptyValueFailsLoudly asserts that a mutable config whose
// "value" field is absent produces an error naming the file -- never a
// panic, and never a silently-empty Starlark value.
func TestLoadMutableEmptyValueFailsLoudly(t *testing.T) {
	dir := testdata.SmallTestDir()
	c, err := NewCompiler(dir, false)
	require.NoError(t, err)

	emptyValueFile := filepath.Join(dir, consts.MutableConfigPath, "empty_value_test"+consts.CompiledConfigExtension)
	require.NoError(t, os.WriteFile(emptyValueFile, []byte(`{}`), 0o644))

	loader := c.GetLoader()
	require.NotPanics(t, func() {
		_, err = loader.loadMutable(consts.MutableConfigPrefix + "empty_value_test")
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty_value_test"+consts.CompiledConfigExtension)
}
