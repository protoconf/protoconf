package compiler

import (
	"context"
	"fmt"
	"net"
	"os"
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
