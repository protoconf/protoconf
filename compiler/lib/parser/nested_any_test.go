package parser

import (
	"os"
	"path/filepath"
	"testing"

	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// nestedAnyProto declares a nested symbol (nested.v1.Outer.Middle.Inner) and
// a sibling top-level symbol (nested.v1.Leaf) that Inner's own "nested"
// field carries as a second, depth-2 google.protobuf.Any. Neither symbol is
// reachable from the registry's construction-time snapshot -- nothing loads
// this file until the scan tier does.
const nestedAnyProto = `syntax = "proto3";
package nested.v1;

import "google/protobuf/any.proto";

message Outer {
  message Middle {
    message Inner {
      string value = 1;
      google.protobuf.Any nested = 2;
    }
  }
}

message Leaf {
  string name = 1;
}
`

// writeNestedAnySrc writes nestedAnyProto under a fresh src/ tree rooted at
// t.TempDir(), at the root-relative path ParseOne and the scan tier both
// expect (package-prefix-derived directory, per D-05).
func writeNestedAnySrc(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	dir := filepath.Join(src, "nested", "v1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested.proto"), []byte(nestedAnyProto), 0o644))
	return src
}

// newLazyParser builds a Parser over a fresh, lazy DescriptorRegistry rooted
// at src -- the deadlockCorpus/canonicalRaceRoot idiom (utils package), no
// Compiler needed.
func newLazyParser(src string) (*utils.DescriptorRegistry, *Parser) {
	registry := utils.NewDescriptorRegistry()
	registry.ImportPaths = []string{src}
	return registry, NewParserWithDescriptorRegistry(registry)
}

// writeMaterializedJSON writes a materialized ProtoconfValue JSON envelope
// whose value is a nested google.protobuf.Any, itself carrying a second,
// depth-2 Any.
func writeMaterializedJSON(t *testing.T, dir, name, outerType, outerValue, innerType, innerName string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	body := `{
  "value": {
    "@type": "` + outerType + `",
    "value": "` + outerValue + `",
    "nested": {
      "@type": "` + innerType + `",
      "name": "` + innerName + `"
    }
  }
}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

// TestReadConfigNestedAny pins TYPE-03/TYPE-09: a nested symbol declared in a
// file nothing has loaded resolves through parser.ReadConfig, and its own
// inner Any resolves at depth 2 -- provably answered by the scan tier, not a
// pre-seeded snapshot or the eager ParseAll fallback.
func TestReadConfigNestedAny(t *testing.T) {
	src := writeNestedAnySrc(t)
	registry, p := newLazyParser(src)

	jsonPath := writeMaterializedJSON(t, t.TempDir(), "config.materialized_JSON",
		"type.googleapis.com/nested.v1.Outer.Middle.Inner", "outer-value",
		"type.googleapis.com/nested.v1.Leaf", "leaf-value")

	protoconfValue := &protoconf_pb.ProtoconfValue{}
	require.NoError(t, p.ReadConfig(jsonPath, protoconfValue))

	require.Equal(t, "type.googleapis.com/nested.v1.Outer.Middle.Inner", protoconfValue.GetValue().GetTypeUrl())

	mt, err := p.TypeResolver.FindMessageByURL(protoconfValue.GetValue().GetTypeUrl())
	require.NoError(t, err)
	inner := dynamicpb.NewMessage(mt.Descriptor())
	require.NoError(t, protoconfValue.GetValue().UnmarshalTo(inner))

	valueField := inner.Descriptor().Fields().ByName("value")
	require.NotNil(t, valueField)
	require.Equal(t, "outer-value", inner.Get(valueField).String())

	nestedField := inner.Descriptor().Fields().ByName("nested")
	require.NotNil(t, nestedField)
	nestedRaw, err := proto.Marshal(inner.Get(nestedField).Message().Interface())
	require.NoError(t, err)
	nestedAny := &anypb.Any{}
	require.NoError(t, proto.Unmarshal(nestedRaw, nestedAny))
	require.Equal(t, "type.googleapis.com/nested.v1.Leaf", nestedAny.GetTypeUrl())

	leafType, err := p.TypeResolver.FindMessageByURL(nestedAny.GetTypeUrl())
	require.NoError(t, err)
	leaf := dynamicpb.NewMessage(leafType.Descriptor())
	require.NoError(t, nestedAny.UnmarshalTo(leaf))
	nameField := leaf.Descriptor().Fields().ByName("name")
	require.NotNil(t, nameField)
	require.Equal(t, "leaf-value", leaf.Get(nameField).String())

	// A pass here must be provably the scan tier's doing, not a pre-seeded
	// snapshot or a vacuous short-circuit.
	require.Equal(t, 1, registry.ScanResolutionCount(),
		"the scan tier, not the eager ParseAll fallback, must be the tier that answered")
	require.GreaterOrEqual(t, registry.LoadedFileCount(), 1)
}

// TestResolveTiersAgreeAcrossEntryPoints pins TYPE-08: FindMessageByURL and
// FindMessageByName share the one resolveTiers chain and must return message
// types backed by the identical descriptor pointer for the same symbol.
func TestResolveTiersAgreeAcrossEntryPoints(t *testing.T) {
	src := writeNestedAnySrc(t)
	_, p := newLazyParser(src)

	byURL, err := p.TypeResolver.FindMessageByURL("type.googleapis.com/nested.v1.Outer.Middle.Inner")
	require.NoError(t, err)
	byName, err := p.TypeResolver.FindMessageByName("nested.v1.Outer.Middle.Inner")
	require.NoError(t, err)

	require.Same(t, byURL.Descriptor(), byName.Descriptor(),
		"FindMessageByURL and FindMessageByName must return the same descriptor pointer for the same symbol")
}

// TestScanNeverAnswersWithoutParse pins D-05's non-negotiable: a symbol name
// that only appears lexically -- inside a block comment, never a real
// declaration -- must never resolve, and must never be counted as a scan
// resolution.
func TestScanNeverAnswersWithoutParse(t *testing.T) {
	src := t.TempDir()
	dir := filepath.Join(src, "ghost", "v1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	ghostProto := `syntax = "proto3";
package ghost.v1;

message Something {
  string value = 1;
}

/*
message NeverDeclared {
  string value = 1;
}
*/
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ghost.proto"), []byte(ghostProto), 0o644))

	registry, p := newLazyParser(src)

	_, err := p.TypeResolver.FindMessageByURL("type.googleapis.com/ghost.v1.NeverDeclared")
	require.Error(t, err)
	require.ErrorIs(t, err, protoregistry.NotFound)
	require.Equal(t, 0, registry.ScanResolutionCount(),
		"a lexical hit inside a comment must never be counted as a scan resolution")
}
