package mutate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// testutil imported for NewAny helper
	_ "github.com/protoconf/protoconf/testutil"
)

// loadDurationDescriptor loads google.protobuf.Duration — has int64 "seconds" and int32 "nanos" fields.
func loadDurationDescriptor(t *testing.T) *desc.MessageDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/duration.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.Duration")
	require.NotNil(t, sym)
	md, ok := sym.(*desc.MessageDescriptor)
	require.True(t, ok)
	return md
}

// newDurationMsg returns a fresh dynamic Duration message.
func newDurationMsg(t *testing.T) *dynamic.Message {
	t.Helper()
	return dynamic.NewMessage(loadDurationDescriptor(t))
}

// identityTyper passes its value through unchanged.
var identityTyper typerFunc = func(s interface{}) interface{} { return s }

// int32Typer converts int64 to int32, matching the INT32 branch in Run.
var int32Typer typerFunc = func(s interface{}) interface{} { return int32(s.(int64)) }

// ---------------------------------------------------------------------------
// TestCommand
// ---------------------------------------------------------------------------

func TestCommand(t *testing.T) {
	cmd, err := Command()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Synopsis())
	assert.NotEmpty(t, cmd.Help())
}

// ---------------------------------------------------------------------------
// TestSetNumeric
// ---------------------------------------------------------------------------

func TestSetNumeric(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		val     string
		typer   typerFunc
		wantErr bool
	}{
		{
			name:    "valid int64 seconds",
			key:     "seconds",
			val:     "42",
			typer:   identityTyper,
			wantErr: false,
		},
		{
			name:    "valid large int64",
			key:     "seconds",
			val:     "9999999999",
			typer:   identityTyper,
			wantErr: false,
		},
		{
			name:    "valid int32 nanos via int32Typer",
			key:     "nanos",
			val:     "500",
			typer:   int32Typer,
			wantErr: false,
		},
		{
			name:    "invalid non-numeric",
			key:     "seconds",
			val:     "not_a_number",
			typer:   identityTyper,
			wantErr: true,
		},
		{
			name:    "invalid empty string",
			key:     "seconds",
			val:     "",
			typer:   identityTyper,
			wantErr: true,
		},
		{
			name:    "invalid float string",
			key:     "seconds",
			val:     "3.14",
			typer:   identityTyper,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := newDurationMsg(t)
			err := setNumeric(msg, tc.key, tc.val, tc.typer)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSetFloat
// ---------------------------------------------------------------------------

func TestSetFloat(t *testing.T) {
	// google.protobuf.Duration has no float fields, so we use an arbitrary
	// dynamic message built from google.protobuf.DescriptorProto which has no
	// float either.  Instead, exercise the parsing contract only — setFloat
	// calls strconv.ParseFloat then delegates to setField. We verify error
	// behaviour rather than field-setting because the Duration fields are
	// int64/int32, not float.  We use a nil message to prove the parse
	// happens before field access.
	tests := []struct {
		name    string
		val     string
		wantErr bool
	}{
		{name: "valid float 3.14", val: "3.14", wantErr: false},
		{name: "valid negative -2.5", val: "-2.5", wantErr: false},
		{name: "valid integer string", val: "0", wantErr: false},
		{name: "invalid alpha", val: "abc", wantErr: true},
		{name: "invalid empty", val: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// We pass a nil msg here intentionally. setFloat only calls
			// strconv.ParseFloat and, on success, delegates to setField which
			// calls msg.TrySetFieldByName. A nil msg will panic on a successful
			// parse, so use a real Duration message for the success cases.
			if tc.wantErr {
				// For error cases msg is never reached — nil is safe.
				err := setFloat(nil, "seconds", tc.val, identityTyper)
				assert.Error(t, err)
			} else {
				// For success cases, use a real message that ignores set errors
				// (TrySetFieldByName logs but doesn't crash on wrong type).
				msg := newDurationMsg(t)
				err := setFloat(msg, "seconds", tc.val, identityTyper)
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestSetField
// ---------------------------------------------------------------------------

func TestSetField(t *testing.T) {
	t.Run("set int64 seconds", func(t *testing.T) {
		msg := newDurationMsg(t)
		setField(msg, "seconds", int64(123), identityTyper)
		val := msg.GetFieldByName("seconds")
		assert.Equal(t, int64(123), val)
	})

	t.Run("set int32 nanos", func(t *testing.T) {
		msg := newDurationMsg(t)
		setField(msg, "nanos", int32(999), identityTyper)
		val := msg.GetFieldByName("nanos")
		assert.Equal(t, int32(999), val)
	})

	t.Run("unknown field name logs no panic", func(t *testing.T) {
		// setField calls msg.TrySetFieldByName which returns an error that is
		// only logged; the function itself must not panic.
		msg := newDurationMsg(t)
		assert.NotPanics(t, func() {
			setField(msg, "nonexistent_field", "value", identityTyper)
		})
	})
}

// ---------------------------------------------------------------------------
// TestRun_MissingArgs
// ---------------------------------------------------------------------------

func TestRun_MissingArgs(t *testing.T) {
	cmd, err := Command()
	require.NoError(t, err)
	// No required flags provided — Run must return 0 and print help (per code:
	// "if ... len < 1 { c.ui.Output(c.Help()); return 0 }").
	code := cmd.Run([]string{})
	assert.Equal(t, 0, code, "empty args should print help and return 0")
}

// ---------------------------------------------------------------------------
// TestRun_InvalidServer
// ---------------------------------------------------------------------------

func TestRun_InvalidServer(t *testing.T) {
	cmd, err := Command()
	require.NoError(t, err)

	// Provide all required fields but point at a guaranteed-unreachable address.
	// The server address is resolved only on the gRPC dial; NewClient succeeds
	// lazily, so the error surfaces at MutateConfig RPC time.
	// We exercise the error path through invalid root which fails earlier.
	code := cmd.Run([]string{
		"-proto=test.proto",
		"-path=test/path",
		"-msg=test.Message",
		"-field=field=value",
		"-root=/nonexistent/path/that/does/not/exist",
		"-addr=localhost:19999",
	})
	assert.NotEqual(t, 0, code, "invalid protoconf root should return non-zero exit code")
}

// writeConfigJSON writes a minimal mutate config JSON file setting addr and returns its path.
// dir must already exist (e.g. t.TempDir()).
func writeConfigJSON(t *testing.T, dir, name, addr string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := fmt.Sprintf(`{"addr": %q}`, addr)
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))
	return path
}

// Test_cliCommand_ConfigPrecedence locks in PCLI-09: flags > env vars > config file > proto
// defaults, in every direction and regardless of flag position in argv.
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_ConfigPrecedence(t *testing.T) {
	const envKey = "PROTOCONF_MUTATE_SERVER_ADDRESS"
	const factoryDefault = "localhost:4301"

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
				return []string{"-config-file", f, "-addr", "flag:7777"}
			},
			want: "flag:7777",
		},
		{
			name:   "flag_overrides_env_and_file_flag_first",
			envSet: true,
			envVal: "env:9999",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-addr", "flag:7777", "-config-file", f}
			},
			want: "flag:7777",
		},
		{
			name: "config_file_overrides_proto_default",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file:8888")
				return []string{"-config-file", f}
			},
			want: "file:8888",
		},
		{
			name: "empty_config_file_keeps_default",
			buildArgs: func(t *testing.T, dir string) []string {
				f := filepath.Join(dir, "empty.json")
				require.NoError(t, os.WriteFile(f, []byte("{}"), 0644))
				return []string{"-config-file", f}
			},
			want: factoryDefault,
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
			assert.Equal(t, tt.want, cc.config.ServerAddress)
		})
	}
}
