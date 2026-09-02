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

// Test_cliCommand_EnableOtelFlag locks in the default config leaves telemetry off
// guarantee at the layer where it could actually regress: the flag plumbing between the
// proto-derived -enable-otel flag and cliCommand.config.
func Test_cliCommand_EnableOtelFlag(t *testing.T) {
	t.Run("default is false", func(t *testing.T) {
		cmd, err := Command()
		if err != nil {
			t.Fatalf("Command() error = %v", err)
		}
		cc := cmd.(*cliCommand)
		if err := cc.flag.Parse([]string{}); err != nil {
			t.Fatalf("flag.Parse() error = %v", err)
		}
		if cc.config.EnableOtel {
			t.Errorf("EnableOtel = true, want false with no flags")
		}
	})

	t.Run("bare flag sets true", func(t *testing.T) {
		cmd, err := Command()
		if err != nil {
			t.Fatalf("Command() error = %v", err)
		}
		cc := cmd.(*cliCommand)
		if err := cc.flag.Parse([]string{"-enable-otel"}); err != nil {
			t.Fatalf("flag.Parse() error = %v", err)
		}
		if !cc.config.EnableOtel {
			t.Errorf("EnableOtel = false, want true with -enable-otel")
		}
	})
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

// writeAgentConfigRaw writes an arbitrary JSON body into dir (which must already exist, e.g.
// t.TempDir()) and returns its path. Used by rows that need nested TLS keys, where
// writeAgentConfigJSON's single grpc-address shape is insufficient.
func writeAgentConfigRaw(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}

// Test_cliCommand_MultiConfigFilePrecedence closes VERIFICATION.md failed truths #7 and #8:
// flags > env vars > config file > proto defaults must hold across repeated -config-file
// flags, including for message-typed fields (tls-config, store-tls), and an env var must not
// be silently lost when its value coincides with an earlier config file's value. It never
// calls cc.Run, which would start the agent; it only drives cc.flag.Parse.
//
// No t.Parallel() anywhere: t.Setenv forbids it.
func Test_cliCommand_MultiConfigFilePrecedence(t *testing.T) {
	type testCase struct {
		name      string
		envKey    string
		envSet    bool
		envVal    string
		buildArgs func(t *testing.T, dir string) []string
		get       func(cc *cliCommand) string
		want      string
	}

	tests := []testCase{
		{
			// GAP 2 (VERIFICATION.md #8): currently yields "a.pem" because matchesBase
			// short-circuits message-typed fields to false, which inverts later-file-wins.
			name: "later_file_wins_for_tls_cert_file",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigRaw(t, dir, "a.json", `{"tls-config":{"cert-file":"a.pem"}}`)
				b := writeAgentConfigRaw(t, dir, "b.json", `{"tls-config":{"cert-file":"b.pem"}}`)
				return []string{"-config-file", a, "-config-file", b}
			},
			get:  func(cc *cliCommand) string { return cc.config.GetTlsConfig().GetCertFile() },
			want: "b.pem",
		},
		{
			// Same defect, second live message-typed field.
			name: "later_file_wins_for_store_tls_ca_file",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigRaw(t, dir, "a.json", `{"store-tls":{"ca-file":"ca1.pem"}}`)
				b := writeAgentConfigRaw(t, dir, "b.json", `{"store-tls":{"ca-file":"ca2.pem"}}`)
				return []string{"-config-file", a, "-config-file", b}
			},
			get:  func(cc *cliCommand) string { return cc.config.GetStoreTls().GetCaFile() },
			want: "ca2.pem",
		},
		{
			// GAP 1 (VERIFICATION.md #7): currently yields ":8888" because the first
			// config file's value coincides with the env value, making the env var
			// indistinguishable from "unset" under value-comparison provenance.
			name:   "env_wins_over_two_files_when_first_file_coincides",
			envKey: "PROTOCONF_AGENT_GRPC_ADDRESS",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigJSON(t, dir, "a.json", ":9999")
				b := writeAgentConfigJSON(t, dir, "b.json", ":8888")
				return []string{"-config-file", a, "-config-file", b}
			},
			get:  func(cc *cliCommand) string { return cc.config.GrpcAddress },
			want: ":9999",
		},
		{
			// A flag value equal to the factory default is invisible to value comparison;
			// flag provenance (flag.FlagSet.Visit) makes it visible.
			name: "flag_before_config_file_wins_when_flag_equals_factory_default",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				return []string{"-grpc-address", ":4300", "-config-file", a}
			},
			get:  func(cc *cliCommand) string { return cc.config.GrpcAddress },
			want: ":4300",
		},
		{
			// Regression guard: passes today, must keep passing.
			name: "later_file_wins_for_scalar_without_env",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				b := writeAgentConfigJSON(t, dir, "b.json", ":7777")
				return []string{"-config-file", a, "-config-file", b}
			},
			get:  func(cc *cliCommand) string { return cc.config.GrpcAddress },
			want: ":7777",
		},
		{
			// Regression guard: passes today, must keep passing.
			name:   "env_wins_over_two_files_when_neither_coincides",
			envKey: "PROTOCONF_AGENT_GRPC_ADDRESS",
			envSet: true,
			envVal: ":9999",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				b := writeAgentConfigJSON(t, dir, "b.json", ":7777")
				return []string{"-config-file", a, "-config-file", b}
			},
			get:  func(cc *cliCommand) string { return cc.config.GrpcAddress },
			want: ":9999",
		},
		{
			// Proves the accumulator still accumulates past two files.
			name: "three_files_last_one_wins",
			buildArgs: func(t *testing.T, dir string) []string {
				a := writeAgentConfigJSON(t, dir, "a.json", ":8888")
				b := writeAgentConfigJSON(t, dir, "b.json", ":7777")
				c := writeAgentConfigJSON(t, dir, "c.json", ":6666")
				return []string{"-config-file", a, "-config-file", b, "-config-file", c}
			},
			get:  func(cc *cliCommand) string { return cc.config.GrpcAddress },
			want: ":6666",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv(tt.envKey, tt.envVal)
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
			if got := tt.get(cc); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
