package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var jsonConfig string

func init() {
	dir := os.TempDir()
	jsonConfig = filepath.Join(dir, "config.json")
	os.WriteFile(jsonConfig, []byte(`{}`), 0644)
}
func Test_cliCommand_Run(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "cannot listen port 10000000000",
			args: args{
				args: []string{
					"-dev", "/some/fake/path",
					"-grpc-address", ":10000000000",
					"-http-address", ":0",
				},
			},
			want: 1,
		},
		{
			name: "run consul server",
			args: args{
				args: []string{
					"-grpc-address", ":0",
					"-http-address", ":0",
					"-store", "consul",
				},
			},
			want: 1,
		},
		{
			name: "run etcd server",
			args: args{
				args: []string{
					"-log-as-json",
					"-grpc-address", ":0",
					"-http-address", ":0",
					"-store", "etcd",
				},
			},
			want: 1,
		},
		{
			name: "run zookeeper server",
			args: args{
				args: []string{
					"-grpc-address", ":0",
					"-http-address", ":0",
					"-store", "zookeeper",
				},
			},
			want: 1,
		},
		{
			name: "run unknown server",
			args: args{
				args: []string{
					"-store", "file",
				},
			},
			want: 1,
		},
		{
			name: "help",
			args: args{
				args: []string{
					"-h",
				},
			},
			want: 2,
		},
		{
			name: "config-file not exists",
			args: args{
				args: []string{
					"-config-file", "config",
				},
			},
			want: 2,
		},
		{
			name: "config-file empty",
			args: args{
				args: []string{
					"-config-file", os.DevNull,
				},
			},
			want: 2,
		},
		{
			name: "config-file non empty",
			args: args{
				args: []string{
					"-config-file", jsonConfig,
				},
			},
			want: 1,
		},
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
	c, _ := Command()
	c.Help()
}

// writeAgentConfigJSON writes a minimal agent config JSON file setting grpc-address, into dir
// (which must already exist, e.g. t.TempDir()), and returns its path. Named distinctly from a
// generic writeConfigJSON to avoid colliding with this file's shared os.TempDir() jsonConfig
// fixture convention used by Test_cliCommand_Run above.
func writeAgentConfigJSON(t *testing.T, dir, name, grpcAddress string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := fmt.Sprintf(`{"grpc-address": %q}`, grpcAddress)
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}

// Test_cliCommand_ConfigPrecedence locks in PCLI-09: flags > env vars > config file > proto
// defaults, in every direction and regardless of flag position in argv. It never calls
// cc.Run, which would start the agent; it only drives cc.flag.Parse.
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_ConfigPrecedence(t *testing.T) {
	const envKey = "PROTOCONF_AGENT_GRPC_ADDRESS"
	const factoryDefault = ":4300"

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
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f}
			},
			want: ":9999",
		},
		{
			name:   "flag_overrides_env_and_file_flag_last",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f, "-grpc-address", ":7777"}
			},
			want: ":7777",
		},
		{
			name:   "flag_overrides_env_and_file_flag_first",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-grpc-address", ":7777", "-config-file", f}
			},
			want: ":7777",
		},
		{
			name: "config_file_overrides_proto_default",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f}
			},
			want: ":8888",
		},
		{
			name: "empty_config_file_keeps_default",
			buildArgs: func(t *testing.T, dir string) []string {
				f := filepath.Join(dir, "empty.json")
				if err := os.WriteFile(f, []byte("{}"), 0644); err != nil {
					t.Fatalf("failed to write config file: %v", err)
				}
				return []string{"-config-file", f}
			},
			want: factoryDefault,
		},
		{
			name:   "empty_env_var_is_treated_as_unset",
			envSet: true,
			envVal: "",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-config-file", f}
			},
			want: ":8888",
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
			if err != nil {
				t.Fatalf("Command() error = %v", err)
			}
			cc := cmd.(*cliCommand)

			if err := cc.flag.Parse(args); err != nil {
				t.Fatalf("flag.Parse() error = %v", err)
			}
			if got := cc.config.GrpcAddress; got != tt.want {
				t.Errorf("cc.config.GrpcAddress = %q, want %q", got, tt.want)
			}
		})
	}
}
