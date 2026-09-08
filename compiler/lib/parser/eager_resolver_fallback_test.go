package parser

import (
	"path/filepath"
	"regexp"
	"testing"

	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/utils"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// TestParseFilesXResolvesEagerHandRegisteredFile pins the failed
// 12-VERIFICATION.md must-have: a file present only in the parser's raw
// FilesResolver (hand-registered by a consumer the way server/server.go:291-297
// registers its six well-known files) and genuinely absent from FileRegistry
// must resolve through ParseFilesX on an eager registry (ImportPaths empty,
// D-03). This also proves RSLV-03 adjacency/ordering/empty behavior survives
// the fix. It exists as a distinct file from growable_resolver_test.go because
// its evidence path is ParseFilesX specifically: the previously cited
// TestDiscoveryScanDoesNotBackReflection (server/init_prohibitions_test.go)
// calls FilesResolver.FindFileByPath directly and cannot observe this
// regression at all.
func TestParseFilesXResolvesEagerHandRegisteredFile(t *testing.T) {
	// Eager shape (D-03): ImportPaths left empty, so d.filesResolver is nil
	// forever and GetFilesResolver always builds fresh.
	dr := utils.NewDescriptorRegistry()
	require.NoError(t, dr.Import(dr.Parse, []*regexp.Regexp{}, filepath.Join(testdata.SmallTestDir(), consts.SrcPath)))

	p := NewParserWithDescriptorRegistry(dr)
	require.NotNil(t, p.FilesResolver)

	// Hand-register the external file AFTER construction, mirroring
	// server.go's NewProtoconfMutationServer.
	require.NoError(t, p.FilesResolver.RegisterFile(grpc_health_v1.File_grpc_health_v1_health_proto))

	// Setup precondition only, not the proof: confirm the path is NOT in
	// FileRegistry, so a later reader cannot mistake a branch-1 hit for a
	// fallback hit.
	_, inRegistry := dr.FileDescriptor("grpc/health/v1/health.proto")
	require.False(t, inRegistry, "grpc/health/v1/health.proto must be absent from FileRegistry for this test to be meaningful")

	t.Run("the failed truth: eager hand-registered file resolves through ParseFilesX", func(t *testing.T) {
		results, err := p.ParseFilesX("grpc/health/v1/health.proto")
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "grpc/health/v1/health.proto", results[0].GetName())
	})

	t.Run("RSLV-03 adjacency: present in both sources returns the FileRegistry canonical entry", func(t *testing.T) {
		results, err := p.ParseFilesX("test.proto")
		require.NoError(t, err)
		require.Len(t, results, 1)
		canonical, ok := dr.FileDescriptor("test.proto")
		require.True(t, ok)
		require.Same(t, canonical, results[0], "ParseFilesX must return the registry's canonical descriptor, not a second wrapper")
	})

	t.Run("RSLV-03 ordering: mixed branches in one call return in argument order", func(t *testing.T) {
		results, err := p.ParseFilesX("test.proto", "grpc/health/v1/health.proto")
		require.NoError(t, err)
		require.Len(t, results, 2)
		require.Equal(t, "test.proto", results[0].GetName())
		require.Equal(t, "grpc/health/v1/health.proto", results[1].GetName())
	})

	t.Run("RSLV-03 empty/absent: no args, empty string, and unknown paths behave safely", func(t *testing.T) {
		empty, err := p.ParseFilesX()
		require.NoError(t, err)
		require.Empty(t, empty)

		_, err = p.ParseFilesX("does/not/exist.proto")
		require.Error(t, err)
		require.ErrorIs(t, err, utils.ErrLazyParseDisabled)

		require.NotPanics(t, func() {
			_, _ = p.ParseFilesX("")
		})
		_, err = p.ParseFilesX("")
		require.Error(t, err)
	})
}
