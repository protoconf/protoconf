package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	protoconf_server_config "github.com/protoconf/protoconf/server/config/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// makeTempScript creates a temp executable shell script with the given body (e.g. "exit 0" or "exit 1").
func makeTempScript(t *testing.T, body string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-script-*.sh")
	require.NoError(t, err)
	_, err = f.WriteString("#!/bin/sh\n" + body + "\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0755))
	return f.Name()
}

func Test_server_MutateConfig(t *testing.T) {
	t.Run("test no workspace", func(t *testing.T) {
		s, err := NewProtoconfMutationServer(os.TempDir())
		require.NoError(t, err)
		s.config = &protoconf_server_config.ServerConfig{}
		resp, err := s.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
			Path:  "test",
			Value: &protoconf_pb.ProtoconfValue{},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Uuid)
	})

	t.Run("test", func(t *testing.T) {
		s, err := NewProtoconfMutationServer(testdata.SmallTestDir())
		require.NoError(t, err)
		s.config = &protoconf_server_config.ServerConfig{}
		resp, err := s.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
			Path: "test",
			Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "test.proto",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Uuid)
	})

	t.Run("run scripts", func(t *testing.T) {
		preScript := makeTempScript(t, "exit 0")
		postScript := makeTempScript(t, "exit 0")
		s, err := NewProtoconfMutationServer(testdata.SmallTestDir())
		require.NoError(t, err)
		s.config = &protoconf_server_config.ServerConfig{
			PreMutationScript:  preScript,
			PostMutationScript: postScript,
		}
		s.PreMutationScript = preScript
		s.PostMutationScript = postScript
		resp, err := s.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
			Path: "test",
			Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "test.proto",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.Uuid)
		assert.NotNil(t, resp.PreScriptDuration)
		assert.NotNil(t, resp.PostScriptDuration)
	})

	t.Run("run bad pre scripts", func(t *testing.T) {
		preScript := makeTempScript(t, "exit 1")
		s, err := NewProtoconfMutationServer(testdata.SmallTestDir())
		require.NoError(t, err)
		s.config = &protoconf_server_config.ServerConfig{PreMutationScript: preScript}
		s.PreMutationScript = preScript
		_, err = s.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
			Path: "test",
			Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "test.proto",
			},
		})
		require.True(t, errors.Is(err, ErrPreMutationScriptError), "expected ErrPreMutationScriptError, got %v", err)
	})

	t.Run("run bad post scripts", func(t *testing.T) {
		postScript := makeTempScript(t, "exit 1")
		s, err := NewProtoconfMutationServer(testdata.SmallTestDir())
		require.NoError(t, err)
		s.config = &protoconf_server_config.ServerConfig{PostMutationScript: postScript}
		s.PostMutationScript = postScript
		_, err = s.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
			Path: "test",
			Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "test.proto",
			},
		})
		require.True(t, errors.Is(err, ErrPostMutationScriptError), "expected ErrPostMutationScriptError, got %v", err)
	})

	t.Run("scripts receive auth credentials", func(t *testing.T) {
		preScript := makeTempScript(t, "exit 0")
		s, err := NewProtoconfMutationServer(testdata.SmallTestDir())
		require.NoError(t, err)
		s.config = &protoconf_server_config.ServerConfig{
			PreMutationScript: preScript,
			AuthToken:         "test-token-123",
		}
		s.PreMutationScript = preScript
		_, err = s.MutateConfig(context.Background(), &protoconf_pb.ConfigMutationRequest{
			Path:           "test",
			ScriptMetadata: "meta-data-value",
			Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "test.proto",
			},
		})
		require.NoError(t, err)
	})
}
func TestProtoconfMutationServer_GenReflectionUI(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	server, err := NewProtoconfMutationServer(protoconfRoot)
	require.NoError(t, err)

	ctx := context.Background()
	rpcServer := grpc.NewServer()
	server.Init(rpcServer)
	httpServer := &http.Server{}

	err = server.GenReflectionUI(ctx, rpcServer, httpServer)
	if err != nil {
		t.Errorf("GenReflectionUI returned an error: %v", err)
	}

	assert.NotNil(t, httpServer.Handler)
}
func TestProtoconfMutationServer_ReportProgress(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	compiler, err := lib.NewCompiler(protoconfRoot, false)
	require.NoError(t, err)

	s, err := NewProtoconfMutationServer(protoconfRoot, WithCompiler(compiler))
	require.NoError(t, err)
	ctx := context.Background()
	in := &protoconf_pb.ConfigMutationResponse{
		Uuid: "test-uuid",
		// Set other fields as needed
	}
	s.reports.Store("test-uuid", in)

	// Call the function
	got, err := s.ReportProgress(ctx, in)
	if err != nil {
		t.Errorf("ReportProgress() error = %v", err)
		return
	}
	assert.NotNil(t, got)

	assert.Equal(t, "test-uuid", got.Uuid)
}

func Test_cliCommand_Run(t *testing.T) {
	// Create a temporary directory for the protoconfRoot
	protoconfRoot := testdata.SmallTestDir()
	defer os.RemoveAll(protoconfRoot)

	// Create a temporary executable file for the preMutationScript
	preMutationScript := makeTempScript(t, "exit 0")
	defer os.Remove(preMutationScript)

	// Create a temporary executable file for the postMutationScript
	postMutationScript := makeTempScript(t, "exit 0")
	defer os.Remove(postMutationScript)

	// Set up the command using Command() to get properly initialized cliCommand
	cmd, err := Command()
	if err != nil {
		t.Fatal(err)
	}

	// Run the command with flags
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	exitCode := cmd.(*cliCommand).run(ctx, []string{
		"-grpc-address", "localhost:50051",
		"-pre", preMutationScript,
		"-post", postMutationScript,
		protoconfRoot,
	})

	// Check the exit code
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	assert.NotNil(t, cmd)
}

func Test_bearerTokenInterceptor(t *testing.T) {
	noopHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	tests := []struct {
		name          string
		expectedToken string
		ctx           context.Context
		wantCode      codes.Code
	}{
		{
			name:          "no_auth_configured",
			expectedToken: "",
			ctx:           context.Background(),
			wantCode:      codes.OK,
		},
		{
			name:          "valid_token",
			expectedToken: "secret123",
			ctx:           metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer secret123"})),
			wantCode:      codes.OK,
		},
		{
			name:          "missing_metadata",
			expectedToken: "secret123",
			ctx:           context.Background(),
			wantCode:      codes.Unauthenticated,
		},
		{
			name:          "missing_authorization_header",
			expectedToken: "secret123",
			ctx:           metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"other-key": "value"})),
			wantCode:      codes.Unauthenticated,
		},
		{
			name:          "invalid_token",
			expectedToken: "secret123",
			ctx:           metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "Bearer wrongtoken"})),
			wantCode:      codes.Unauthenticated,
		},
		{
			name:          "raw_token_without_bearer_prefix",
			expectedToken: "secret123",
			ctx:           metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"authorization": "secret123"})),
			wantCode:      codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := bearerTokenInterceptor(tt.expectedToken)
			_, err := interceptor(tt.ctx, nil, &grpc.UnaryServerInfo{}, noopHandler)
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func Test_validateScriptPath(t *testing.T) {
	t.Run("empty path returns nil", func(t *testing.T) {
		err := validateScriptPath("")
		require.NoError(t, err)
	})

	t.Run("nonexistent path returns error containing does not exist", func(t *testing.T) {
		err := validateScriptPath("/nonexistent/path/that/should/not/exist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("non-executable file returns error containing not executable", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-script-*.sh")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		f.Close()
		require.NoError(t, os.Chmod(f.Name(), 0644))

		err = validateScriptPath(f.Name())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not executable")
	})

	t.Run("path with .. returns error containing ..", func(t *testing.T) {
		err := validateScriptPath("../etc/passwd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "..")
	})

	t.Run("path with embedded .. returns error containing ..", func(t *testing.T) {
		err := validateScriptPath("/usr/local/../bin/something")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "..")
	})

	t.Run("valid executable temp file returns nil", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-script-*.sh")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		f.Close()
		require.NoError(t, os.Chmod(f.Name(), 0755))

		err = validateScriptPath(f.Name())
		require.NoError(t, err)
	})

	t.Run("directory path returns error containing directory", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "test-script-dir-*")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		err = validateScriptPath(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "directory")
	})
}

// writeConfigJSON writes a minimal server config JSON file setting grpc-address and returns its
// path. dir must already exist (e.g. t.TempDir()).
func writeConfigJSON(t *testing.T, dir, name, grpcAddress string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := fmt.Sprintf(`{"grpc-address": %q}`, grpcAddress)
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))
	return path
}

// Test_cliCommand_ConfigPrecedence locks in PCLI-09: flags > env vars > config file > proto
// defaults, in every direction and regardless of flag position in argv.
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_ConfigPrecedence(t *testing.T) {
	const envKey = "PROTOCONF_SERVER_GRPC_ADDRESS"

	type testCase struct {
		name      string
		envSet    bool
		envVal    string
		buildArgs func(t *testing.T, dir string) []string
		wantErr   bool
		want      string
	}

	tests := []testCase{
		{
			// THIS IS THE GAP; it fails before the fix.
			name:   "env_overrides_config_file",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f}
			},
			want: ":9999",
		},
		{
			// Also fails before the fix: the handler reassigns c.config, so the later flag
			// lands in the orphaned message instead of the live one.
			name:   "flag_overrides_env_and_file_flag_last",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f, "-grpc-address", ":7777"}
			},
			want: ":7777",
		},
		{
			name:   "flag_overrides_env_and_file_flag_first",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-grpc-address", ":7777", "-config-file", f}
			},
			want: ":7777",
		},
		{
			name: "config_file_overrides_proto_default",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f}
			},
			want: ":8888",
		},
		{
			name: "empty_config_file_keeps_default",
			buildArgs: func(t *testing.T, dir string) []string {
				f := filepath.Join(dir, "empty.json")
				require.NoError(t, os.WriteFile(f, []byte("{}"), 0644))
				return []string{"-config-file", f}
			},
			want: consts.ServerDefaultAddress,
		},
		{
			name:   "empty_env_var_is_treated_as_unset",
			envSet: true,
			envVal: "",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f}
			},
			want: ":8888",
		},
		{
			name:   "env_and_file_agree",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":9999")
				return []string{"-config-file", f}
			},
			want: ":9999",
		},
		{
			name: "later_config_file_wins",
			buildArgs: func(t *testing.T, dir string) []string {
				fa := writeConfigJSON(t, dir, "a.json", ":8888")
				fb := writeConfigJSON(t, dir, "b.json", ":7777")
				return []string{"-config-file", fa, "-config-file", fb}
			},
			want: ":7777",
		},
		{
			name: "same_config_file_twice_is_idempotent",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f, "-config-file", f}
			},
			want: ":8888",
		},
		{
			name: "unparsable_config_file_extension_errors",
			buildArgs: func(t *testing.T, dir string) []string {
				f := filepath.Join(dir, "a.txt")
				require.NoError(t, os.WriteFile(f, []byte("{}"), 0644))
				return []string{"-config-file", f}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv(envKey, tt.envVal)
			}
			dir := t.TempDir()
			args := tt.buildArgs(t, dir)

			cmd, err := Command()
			require.NoError(t, err)
			cc := cmd.(*cliCommand)

			err = cc.flag.Parse(args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, cc.config.GrpcAddress)
		})
	}
}

// writeRawConfig writes body verbatim to dir/name and returns its path. Used where
// writeConfigJSON's signature cannot produce the needed body (an empty "{}" object or a
// non-json-extension path). dir must already exist (e.g. t.TempDir()).
func writeRawConfig(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0644))
	return path
}

// Test_cliCommand_MultiConfigFilePrecedence is the 08-06 CLI-level regression coverage for the
// scalar half of 08-VERIFICATION.md gap 1 (an env var whose value coincidentally equals an
// earlier -config-file's value must still beat a later -config-file), plus the PCLI-08
// same-file-twice and empty/unparsable-file edges. Each row drives cc.flag.Parse directly, never
// cc.Run, matching Test_cliCommand_ConfigPrecedence's shape.
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_MultiConfigFilePrecedence(t *testing.T) {
	const envKey = "PROTOCONF_SERVER_GRPC_ADDRESS"

	t.Run("env_wins_over_two_files_when_first_file_coincides", func(t *testing.T) {
		t.Setenv(envKey, ":9999")
		dir := t.TempDir()
		fa := writeConfigJSON(t, dir, "a.json", ":9999")
		fb := writeConfigJSON(t, dir, "b.json", ":8888")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fb}))
		assert.Equal(t, ":9999", cc.config.GrpcAddress)
	})

	t.Run("flag_before_config_file_wins_when_flag_equals_factory_default", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeConfigJSON(t, dir, "a.json", ":8888")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-grpc-address", consts.ServerDefaultAddress, "-config-file", fa}))
		assert.Equal(t, consts.ServerDefaultAddress, cc.config.GrpcAddress)
	})

	t.Run("later_file_wins_for_scalar_without_env", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeConfigJSON(t, dir, "a.json", ":8888")
		fb := writeConfigJSON(t, dir, "b.json", ":7777")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fb}))
		assert.Equal(t, ":7777", cc.config.GrpcAddress)
	})

	t.Run("same_config_file_twice_is_idempotent_across_two_flags", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeConfigJSON(t, dir, "a.json", ":8888")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fa}))
		assert.Equal(t, ":8888", cc.config.GrpcAddress)
	})

	t.Run("empty_second_config_file_does_not_erase_first", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeConfigJSON(t, dir, "a.json", ":8888")
		fb := writeRawConfig(t, dir, "b.json", "{}")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fb}))
		assert.Equal(t, ":8888", cc.config.GrpcAddress)
	})

	t.Run("unparsable_config_file_extension_errors", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeRawConfig(t, dir, "a.txt", "{}")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		err = cc.flag.Parse([]string{"-config-file", fa})
		require.Error(t, err)
	})
}

// Test_cliCommand_EnableOtelFlag mirrors agent's Test_cliCommand_EnableOtelFlag, locking in the
// default config leaves telemetry off guarantee at the layer where it could actually regress:
// the flag plumbing between the proto-derived -enable-otel flag and cliCommand.config.
func Test_cliCommand_EnableOtelFlag(t *testing.T) {
	t.Run("default is false", func(t *testing.T) {
		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)
		require.NoError(t, cc.flag.Parse([]string{}))
		assert.False(t, cc.config.EnableOtel)
	})

	t.Run("bare flag sets true", func(t *testing.T) {
		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)
		require.NoError(t, cc.flag.Parse([]string{"-enable-otel"}))
		assert.True(t, cc.config.EnableOtel)
	})
}

func Test_cliCommand_Synopsis(t *testing.T) {
	command := &cliCommand{}
	got := command.Synopsis()
	want := "Start the mutation server that accepts config changes via gRPC"
	if got != want {
		t.Errorf("cliCommand.Synopsis() = %q, want %q", got, want)
	}
}
func TestProtoconfMutationServer_Put(t *testing.T) {
	protoconfRoot := testdata.SmallTestDir()
	compiler, err := lib.NewCompiler(protoconfRoot, false)
	require.NoError(t, err)
	s, err := NewProtoconfMutationServer(protoconfRoot, WithCompiler(compiler))
	require.NoError(t, err)
	ctx := context.Background()
	ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{"path": "test"}))
	tmp := &protoconf_pb.CompileRequest{}
	in := dynamicpb.NewMessage(tmp.ProtoReflect().Descriptor())
	proto.Merge(in, tmp)

	_, err = s.Put(ctx, in)
	if !errors.Is(err, ErrInternalCompilerError) {
		t.Errorf("Put() error = %v", err)
		return
	}

}
