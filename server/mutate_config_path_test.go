package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// buildMutationRequest constructs a ConfigMutationRequest carrying a
// resolved test.v1.TestMessage Any with the given path and stringValue,
// mirroring mutate_config_race_test.go's request construction.
func buildMutationRequest(t *testing.T, srv *ProtoconfMutationServer, path, stringValue string) *protoconf_pb.ConfigMutationRequest {
	t.Helper()
	mt, err := srv.parser.TypeResolver.FindMessageByURL("type.googleapis.com/test.v1.TestMessage")
	require.NoError(t, err)
	stringValueField := mt.Descriptor().Fields().ByName("stringValue")
	require.NotNil(t, stringValueField)

	msg := dynamicpb.NewMessage(mt.Descriptor())
	msg.Set(stringValueField, protoreflect.ValueOfString(stringValue))
	anyVal, err := anypb.New(msg)
	require.NoError(t, err)

	return &protoconf_pb.ConfigMutationRequest{
		Path: path,
		Value: &protoconf_pb.ProtoconfValue{
			ProtoFile: "test.proto",
			Value:     anyVal,
		},
	}
}

// TestMutateConfigRejectsTraversalPath proves a caller-supplied in.Path
// cannot write a file outside protoconfRoot/mutable_config. The escape
// target is computed under filepath.Dir(root) -- a temp directory outside
// the server's own root -- and a two-level "../../escaped" traversal is
// used because a single ".." only cancels consts.MutableConfigPath and
// lands back inside protoconfRoot.
//
// The file-absence assertion is the one that matters: an error-only
// assertion would pass for the wrong reason, which is the exact defect
// 14-VERIFICATION.md recorded against TestGetRejectsTraversalKey. If a
// file IS present, its content is included in the failure message so the
// leak is visible in the test output.
func TestMutateConfigRejectsTraversalPath(t *testing.T) {
	root := testdata.SmallTestDir()
	srv, err := NewProtoconfMutationServer(root)
	require.NoError(t, err)

	escapeTarget := filepath.Join(filepath.Dir(root), "escaped"+consts.CompiledConfigExtension)
	t.Cleanup(func() { _ = os.Remove(escapeTarget) })

	const secretPayload = "PWNED-OUTSIDE-ROOT-14-09"
	req := buildMutationRequest(t, srv, "../../escaped", secretPayload)

	resp, err := srv.MutateConfig(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)

	_, statErr := os.Stat(escapeTarget)
	if statErr == nil {
		content, _ := os.ReadFile(escapeTarget)
		t.Fatalf("traversal path escaped protoconfRoot: file written at %s, content=%q", escapeTarget, content)
	}
	require.True(t, errors.Is(statErr, os.ErrNotExist), "expected os.ErrNotExist, got: %v", statErr)
}

// TestMutateConfigAllowsNestedPath pins that the containment fix rejects
// escapes, not separators: a legitimate nested Path containing an interior
// separator must still succeed and land at
// protoconfRoot/mutable_config/<path>.materialized_JSON exactly as before.
// A containment check written as a separator ban would break real callers
// (the existing race test already writes paths like "mutate_race_0"; this
// test additionally exercises an interior separator).
func TestMutateConfigAllowsNestedPath(t *testing.T) {
	root := testdata.SmallTestDir()
	srv, err := NewProtoconfMutationServer(root)
	require.NoError(t, err)

	wantFile := filepath.Join(root, consts.MutableConfigPath, "traversal_probe/nested"+consts.CompiledConfigExtension)
	t.Cleanup(func() { _ = os.Remove(wantFile) })

	req := buildMutationRequest(t, srv, "traversal_probe/nested", "nested-value")

	resp, err := srv.MutateConfig(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	_, statErr := os.Stat(wantFile)
	require.NoError(t, statErr)
}
