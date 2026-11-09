package compiler

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func Test_cliCommand_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := testdata.SmallTestDir()
	svc, err := NewCompilerService(dir, true)
	require.NoError(t, err)
	rpcServer := grpc.NewServer()
	protoconf_pb.RegisterProtoconfCompileServer(rpcServer, svc)
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	go func() {
		context.AfterFunc(ctx, func() { rpcServer.GracefulStop() })
		if err := rpcServer.Serve(lis); err != nil {
			t.Logf("error serving server: %v", err)
		}
	}()

	type args struct {
		args []string
	}
	tests := []struct {
		name string
		c    *cliCommand
		args args
		want int
	}{
		{name: "Test Single file", c: &cliCommand{}, args: args{args: []string{dir, "test.pconf"}}, want: 0},
		{name: "Test All files", c: &cliCommand{}, args: args{args: []string{dir}}, want: 1},
		{name: "Test Profiling", c: &cliCommand{}, args: args{args: []string{"-cpuprofile=/dev/null", "-memprofile=/dev/null", dir, "test.pconf"}}, want: 0},
		{name: "Test Remote Single file", c: &cliCommand{}, args: args{args: []string{"-compiler-address", lis.Addr().String(), dir, "test.pconf"}}, want: 0},
		{name: "Test Remote All files", c: &cliCommand{}, args: args{args: []string{"-compiler-address", lis.Addr().String(), dir}}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := Command()
			if got := c.Run(tt.args.args); got != tt.want {
				t.Errorf("cliCommand.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_cliCommand_StarlarkProfile checks -starlark-profile writes a finalized gzip pprof file.
// The Starlark profiler is process-global, and compile goroutines that earlier tests leave
// running (Test_cliCommand_Run's remote cases) race with StartProfile. So the compile runs
// in a child copy of this test binary, which has no other Starlark threads.
func Test_cliCommand_StarlarkProfile(t *testing.T) {
	if out := os.Getenv("STARLARK_PROFILE_OUT"); out != "" {
		c, err := Command()
		require.NoError(t, err)
		require.Equal(t, 0, c.Run([]string{"-starlark-profile", out, testdata.SmallTestDir(), "test.pconf"}))
		return
	}
	out := filepath.Join(t.TempDir(), "starlark.pprof")
	cmd := exec.Command(os.Args[0], "-test.run=^Test_cliCommand_StarlarkProfile$")
	cmd.Env = append(os.Environ(), "STARLARK_PROFILE_OUT="+out)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "child test failed:\n%s", output)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	// Reading to EOF verifies the gzip trailer, which only exists once StopProfile ran.
	gz, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	_, err = io.ReadAll(gz)
	require.NoError(t, err)
}

func Test_cliCommand_Help(t *testing.T) {
	c, err := Command()
	require.NoError(t, err)
	result := c.Help()
	assert.NotEmpty(t, result)
}

// writeConfigJSON writes a minimal compiler config JSON file setting compiler-address and
// returns its path. dir must already exist (e.g. t.TempDir()).
func writeConfigJSON(t *testing.T, dir, name, compilerAddress string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := fmt.Sprintf(`{"compiler-address": %q}`, compilerAddress)
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))
	return path
}

// Test_cliCommand_ConfigPrecedence locks in PCLI-09: flags > env vars > config file > proto
// defaults, in every direction and regardless of flag position in argv.
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_ConfigPrecedence(t *testing.T) {
	const envKey = "PROTOCONF_COMPILER_COMPILER_ADDRESS"

	type testCase struct {
		name      string
		envSet    bool
		envVal    string
		buildArgs func(t *testing.T, dir string) []string
		want      string
	}

	tests := []testCase{
		{
			name:   "env_overrides_config_file",
			envSet: true,
			envVal: "env:9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-config-file", f}
			},
			want: "env:9999",
		},
		{
			name:   "flag_overrides_env_and_file_flag_last",
			envSet: true,
			envVal: "env:9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-config-file", f, "-compiler-address", "flag:7777"}
			},
			want: "flag:7777",
		},
		{
			name:   "flag_overrides_env_and_file_flag_first",
			envSet: true,
			envVal: "env:9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-compiler-address", "flag:7777", "-config-file", f}
			},
			want: "flag:7777",
		},
		{
			name: "config_file_value_is_applied",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-config-file", f}
			},
			want: "file:8888",
		},
		{
			name: "empty_config_file_leaves_field_empty",
			buildArgs: func(t *testing.T, dir string) []string {
				f := filepath.Join(dir, "empty.json")
				require.NoError(t, os.WriteFile(f, []byte("{}"), 0644))
				return []string{"-config-file", f}
			},
			want: "",
		},
		{
			name:   "empty_env_var_is_treated_as_unset",
			envSet: true,
			envVal: "",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-config-file", f}
			},
			want: "file:8888",
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

			require.NoError(t, cc.flag.Parse(args))
			assert.Equal(t, tt.want, cc.config.CompilerAddress)
		})
	}
}

// Test_cliCommand_LogLevel locks in the -log-level flag / PROTOCONF_COMPILER_LOG_LEVEL env
// var path from CompilerConfig.LogLevel through to the process-global slog level.
//
// No t.Parallel() anywhere in this test: the subtests mutate process globals (slog's default
// logger level, log's output writer) and one subtest uses t.Setenv.
func Test_cliCommand_LogLevel(t *testing.T) {
	resetGlobals := func(t *testing.T) {
		t.Helper()
		old := slog.SetLogLoggerLevel(slog.LevelInfo)
		t.Cleanup(func() { slog.SetLogLoggerLevel(old) })
		oldWriter := log.Writer()
		t.Cleanup(func() { log.SetOutput(oldWriter) })
	}

	t.Run("debug_flag_enables_debug_output", func(t *testing.T) {
		resetGlobals(t)

		c, err := Command()
		require.NoError(t, err)
		got := c.Run([]string{"-log-level", "DEBUG"})
		assert.Equal(t, 1, got)

		assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))

		var buf bytes.Buffer
		log.SetOutput(&buf)
		slog.Debug("log-level-probe")
		assert.Contains(t, buf.String(), "DEBUG")
		assert.Contains(t, buf.String(), "log-level-probe")
	})

	t.Run("lowercase_accepted", func(t *testing.T) {
		resetGlobals(t)

		c, err := Command()
		require.NoError(t, err)
		got := c.Run([]string{"-log-level", "debug"})
		assert.Equal(t, 1, got)
		assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
	})

	t.Run("invalid_level_is_usage_error", func(t *testing.T) {
		resetGlobals(t)

		c, err := Command()
		require.NoError(t, err)
		got := c.Run([]string{"-log-level", "LOUD", "root"})
		assert.Equal(t, 2, got)
	})

	t.Run("env_var_does_not_panic", func(t *testing.T) {
		resetGlobals(t)
		t.Setenv("PROTOCONF_COMPILER_LOG_LEVEL", "DEBUG")

		c, err := Command()
		require.NoError(t, err)
		got := c.Run(nil)
		assert.Equal(t, 1, got)
		assert.True(t, slog.Default().Enabled(context.Background(), slog.LevelDebug))
	})
}
