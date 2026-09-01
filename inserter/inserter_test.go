package inserter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kvtools/valkeyrie/store"
	"github.com/protoconf/protoconf/agent/dummykv"
	protoconf_inserter_config "github.com/protoconf/protoconf/inserter/config/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoconfInserter_InsertConfig(t *testing.T) {
	kvStore, _ := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	// kvStore, err := configmaps.New(context.Background(), []string{}, &configmaps.Config{Namespace: "default"})
	// require.NoError(t, err)
	testDir := testdata.SmallTestDir()

	i := NewProtoconfInserter(testDir, kvStore)
	type args struct {
		configFile string
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		want    map[string]string
	}{
		{
			name:    "test",
			args:    args{configFile: "test.materialized_JSON"},
			wantErr: nil,
			want: map[string]string{
				"test/config.data":   `Cgp0ZXN0LnByb3RvEjQKJ3R5cGUuZ29vZ2xlYXBpcy5jb20vdGVzdC52MS5UZXN0TWVzc2FnZRIJCgdJbSBoZXJlIm`,
				"test/config.json":   "{",
				"test/metadata.json": "{",
			},
		},
		{
			name:    "with_rollout_config",
			args:    args{configFile: "with_config_rollout.materialized_JSON"},
			wantErr: nil,
			want: map[string]string{
				"with_config_rollout/config.data": "",
				"with_config_rollout/config.json": "{",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := i.InsertConfigFile(tt.args.configFile); !errors.Is(tt.wantErr, err) {
				t.Errorf("ProtoconfInserter.InsertConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			for key, value := range tt.want {
				v, err := kvStore.Get(context.Background(), key, &store.ReadOptions{})
				assert.NoError(t, err)
				if !strings.HasPrefix(string(v.Value), value) {
					t.Errorf("expected key %s to be to start with:\n%v, got:\n%s", key, value, v.Value)
				}
			}
		})
	}
}

func Test_cliCommand_Run(t *testing.T) {
	testDir := testdata.SmallTestDir()
	type args struct {
		args []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test",
			args: args{args: []string{testDir, "test.materialized_JSON"}},
			want: 0,
		},
		{
			name: "test multi",
			args: args{args: []string{testDir, "test.materialized_JSON", "with_config_rollout.materialized_JSON"}},
			want: 0,
		},
		{
			name: "no args",
			args: args{args: []string{}},
			want: 1,
		},
		{
			name: "etcd",
			args: args{args: []string{"-store", "etcd", testDir, "test.materialized_JSON"}},
			want: 0,
		},
		{
			name: "zookeeper",
			args: args{args: []string{"-store", "zookeeper", testDir, "test.materialized_JSON"}},
			want: 0,
		},
		{
			name: "delete",
			args: args{args: []string{"-d", testDir, "test.materialized_JSON"}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Command()
			if err != nil {
				t.Fatalf("Command() error = %v", err)
			}
			if got := cmd.Run(tt.args.args); got != tt.want {
				t.Errorf("cliCommand.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cliCommand_Help(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "test",
			want: []string{
				`Insert materialized configs into a key-value store (Consul, etcd, ZooKeeper, or ConfigMaps)`,
				`-d`,
				`-prefix`,
				`-store`,
				`consul`,
				`etcd`,
				`zookeeper`,
				`configmaps`,
				`-store-address`,
				`-config-file`,
			},
		},
		{
			name: "synopsis contains inserter description",
			want: []string{
				"Insert materialized configs",
				"-store-address",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Command()
			if err != nil {
				t.Fatalf("Command() error = %v", err)
			}
			got := cmd.Help()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("cliCommand.Help() = %v, want %v", got, w)
				}
			}

		})
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Command()
			if (err != nil) != tt.wantErr {
				t.Errorf("Command() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Error("Command() returned nil")
			}
		})
	}
}

// writeConfigJSON writes a minimal inserter config JSON file setting prefix and returns its
// path. dir must already exist (e.g. t.TempDir()).
func writeConfigJSON(t *testing.T, dir, name, prefix string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data := fmt.Sprintf(`{"prefix": %q}`, prefix)
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))
	return path
}

// writeStoreAddressJSON writes a config file setting store-address (and, when storeEnum is
// non-empty, store) and returns its path. dir must already exist (e.g. t.TempDir()).
func writeStoreAddressJSON(t *testing.T, dir, name string, storeAddress []string, storeEnum string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	addrJSON, err := json.Marshal(storeAddress)
	require.NoError(t, err)
	data := fmt.Sprintf(`{"store-address": %s`, addrJSON)
	if storeEnum != "" {
		data += fmt.Sprintf(`, "store": %q`, storeEnum)
	}
	data += "}"
	require.NoError(t, os.WriteFile(path, []byte(data), 0644))
	return path
}

// Test_cliCommand_ConfigPrecedence locks in PCLI-09: flags > env vars > config file > proto
// defaults, in every direction and regardless of flag position in argv, for all three field
// kinds this config uses: a plain string (prefix), a repeated string (store-address), and an
// enum (store).
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_ConfigPrecedence(t *testing.T) {
	const prefixEnvKey = "PROTOCONF_INSERTER_PREFIX"

	type testCase struct {
		name      string
		envSet    bool
		envVal    string
		buildArgs func(t *testing.T, dir string) []string
		want      string
	}

	prefixTests := []testCase{
		{
			name:   "env_overrides_config_file",
			envSet: true,
			envVal: "env-prefix",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file-prefix")
				return []string{"-config-file", f}
			},
			want: "env-prefix",
		},
		{
			name:   "flag_overrides_env_and_file_flag_last",
			envSet: true,
			envVal: "env-prefix",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file-prefix")
				return []string{"-config-file", f, "-prefix", "flag-prefix"}
			},
			want: "flag-prefix",
		},
		{
			name:   "flag_overrides_env_and_file_flag_first",
			envSet: true,
			envVal: "env-prefix",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file-prefix")
				return []string{"-prefix", "flag-prefix", "-config-file", f}
			},
			want: "flag-prefix",
		},
		{
			name: "config_file_value_is_applied",
			buildArgs: func(t *testing.T, dir string) []string {
				f := writeConfigJSON(t, dir, "a.json", "file-prefix")
				return []string{"-config-file", f}
			},
			want: "file-prefix",
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
				f := writeConfigJSON(t, dir, "a.json", "file-prefix")
				return []string{"-config-file", f}
			},
			want: "file-prefix",
		},
	}

	for _, tt := range prefixTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv(prefixEnvKey, tt.envVal)
			}
			dir := t.TempDir()
			args := tt.buildArgs(t, dir)

			cmd, err := Command()
			require.NoError(t, err)
			cc := cmd.(*cliCommand)

			require.NoError(t, cc.flag.Parse(args))
			assert.Equal(t, tt.want, cc.config.Prefix)
		})
	}

	t.Run("env_list_replaces_config_file_list", func(t *testing.T) {
		t.Setenv("PROTOCONF_INSERTER_STORE_ADDRESS", "env1:1,env2:2")
		dir := t.TempDir()
		f := writeStoreAddressJSON(t, dir, "a.json", []string{"file1:1"}, "")
		args := []string{"-config-file", f}

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse(args))
		assert.Equal(t, []string{"env1:1", "env2:2"}, cc.config.StoreAddress)
	})

	t.Run("two_config_files_later_list_replaces_earlier", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeStoreAddressJSON(t, dir, "a.json", []string{"file1:1"}, "")
		fb := writeStoreAddressJSON(t, dir, "b.json", []string{"file2:2"}, "")
		args := []string{"-config-file", fa, "-config-file", fb}

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse(args))
		assert.Equal(t, []string{"file2:2"}, cc.config.StoreAddress)
	})

	t.Run("env_overrides_config_file_store_enum", func(t *testing.T) {
		t.Setenv("PROTOCONF_INSERTER_STORE", "etcd")
		dir := t.TempDir()
		f := writeStoreAddressJSON(t, dir, "a.json", nil, "zookeeper")
		args := []string{"-config-file", f}

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse(args))
		assert.Equal(t, protoconf_inserter_config.InserterConfig_etcd, cc.config.Store)
	})

	t.Run("flag_overrides_env_and_file_store_enum", func(t *testing.T) {
		t.Setenv("PROTOCONF_INSERTER_STORE", "etcd")
		dir := t.TempDir()
		f := writeStoreAddressJSON(t, dir, "a.json", nil, "zookeeper")
		args := []string{"-config-file", f, "-store", "configmaps"}

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse(args))
		assert.Equal(t, protoconf_inserter_config.InserterConfig_configmaps, cc.config.Store)
	})

	// Documented limitation (see command/configfile.go's matchesBase): consul is enum number
	// 0, so an env var setting it is indistinguishable from the field being unset, and the
	// config file's value wins instead. Not a bug — see 08-03's matchesBase note.
	t.Run("zero_value_enum_from_env_is_indistinguishable_from_unset", func(t *testing.T) {
		t.Setenv("PROTOCONF_INSERTER_STORE", "consul")
		dir := t.TempDir()
		f := writeStoreAddressJSON(t, dir, "a.json", nil, "zookeeper")
		args := []string{"-config-file", f}

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse(args))
		assert.Equal(t, protoconf_inserter_config.InserterConfig_zookeeper, cc.config.Store)
	})

	t.Run("flag_can_still_select_the_zero_value_enum", func(t *testing.T) {
		dir := t.TempDir()
		f := writeStoreAddressJSON(t, dir, "a.json", nil, "zookeeper")
		args := []string{"-config-file", f, "-store", "consul"}

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse(args))
		assert.Equal(t, protoconf_inserter_config.InserterConfig_consul, cc.config.Store)
	})
}

// Test_cliCommand_MultiConfigFilePrecedence is the 08-06 CLI-level regression coverage for the
// repeated-string half of 08-VERIFICATION.md gap 1 (an env var whose value coincidentally
// equals an earlier -config-file's list must still beat a later -config-file), plus the
// later-file-replaces-earlier-list guarantee across two and three files. Each row drives
// cc.flag.Parse directly, never cc.Run, matching Test_cliCommand_ConfigPrecedence's shape.
//
// No t.Parallel() anywhere in this test: t.Setenv forbids it.
func Test_cliCommand_MultiConfigFilePrecedence(t *testing.T) {
	const storeAddressEnvKey = "PROTOCONF_INSERTER_STORE_ADDRESS"

	t.Run("env_list_wins_over_two_files_when_first_file_coincides", func(t *testing.T) {
		t.Setenv(storeAddressEnvKey, "env1:1,env2:2")
		dir := t.TempDir()
		fa := writeStoreAddressJSON(t, dir, "a.json", []string{"env1:1", "env2:2"}, "")
		fb := writeStoreAddressJSON(t, dir, "b.json", []string{"file2:2"}, "")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fb}))
		assert.Equal(t, []string{"env1:1", "env2:2"}, cc.config.StoreAddress)
	})

	t.Run("later_file_replaces_earlier_list_without_env", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeStoreAddressJSON(t, dir, "a.json", []string{"file1:1"}, "")
		fb := writeStoreAddressJSON(t, dir, "b.json", []string{"file2:2"}, "")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fb}))
		require.Len(t, cc.config.StoreAddress, 1)
		assert.Equal(t, []string{"file2:2"}, cc.config.StoreAddress)
	})

	t.Run("three_files_last_list_wins", func(t *testing.T) {
		dir := t.TempDir()
		fa := writeStoreAddressJSON(t, dir, "a.json", []string{"file1:1"}, "")
		fb := writeStoreAddressJSON(t, dir, "b.json", []string{"file2:2"}, "")
		fc := writeStoreAddressJSON(t, dir, "c.json", []string{"file3:3"}, "")

		cmd, err := Command()
		require.NoError(t, err)
		cc := cmd.(*cliCommand)

		require.NoError(t, cc.flag.Parse([]string{"-config-file", fa, "-config-file", fb, "-config-file", fc}))
		require.Len(t, cc.config.StoreAddress, 1)
		assert.Equal(t, []string{"file3:3"}, cc.config.StoreAddress)
	})
}
