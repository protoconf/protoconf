package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	protoconf_inserter_config "github.com/protoconf/protoconf/inserter/config/v1"
	protoconf_server_config "github.com/protoconf/protoconf/server/config/v1"
)

// TestLayerConfigFile pins the layering rules LayerConfigFile implements,
// independent of any CLI: a config file beats the factory defaults, an env
// var (or already-parsed flag) beats the config file, an empty file changes
// nothing, a later config file beats an earlier one, repeat loads are
// idempotent, and the live message pointer is never replaced. Repeated
// fields obey the same rules as scalars: no entry is ever appended or
// duplicated.
//
// Rows use *protoconf_server_config.ServerConfig for scalar fields; the
// repeated-field rows use *protoconf_inserter_config.InserterConfig
// (ServerConfig has no repeated field). All twelve rows run directly under
// this test (no intermediate t.Run groups) so each is its own top-level
// subtest.
func TestLayerConfigFile(t *testing.T) {
	t.Run("file_overrides_factory_default", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}

		LayerConfigFile(live, base, preFile)

		assert.Equal(t, ":8888", live.GrpcAddress)
	})

	t.Run("env_overrides_file", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":9999"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}

		LayerConfigFile(live, base, preFile)

		assert.Equal(t, ":9999", live.GrpcAddress)
	})

	t.Run("empty_file_keeps_default_and_env", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env"}
		live := &protoconf_server_config.ServerConfig{}

		LayerConfigFile(live, base, preFile)

		want := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env"}
		assert.True(t, proto.Equal(want, live), "want %v, got %v", want, live)
	})

	t.Run("unrelated_fields_are_untouched", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env"}
		live := &protoconf_server_config.ServerConfig{TlsCert: "cert.pem"}

		LayerConfigFile(live, base, preFile)

		want := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env", TlsCert: "cert.pem"}
		assert.True(t, proto.Equal(want, live), "want %v, got %v", want, live)
	})

	t.Run("env_value_equal_to_default_is_indistinguishable", func(t *testing.T) {
		// Documented limitation: an env var whose value equals the factory default is
		// indistinguishable from unset, so the config file's value wins here even
		// though the caller believes the env var explicitly supplied that value.
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}

		LayerConfigFile(live, base, preFile)

		assert.Equal(t, ":8888", live.GrpcAddress)
	})

	t.Run("second_file_overrides_first", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}

		preFile1 := proto.Clone(base).(*protoconf_server_config.ServerConfig)
		live1 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		LayerConfigFile(live1, base, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_server_config.ServerConfig)
		live2 := &protoconf_server_config.ServerConfig{GrpcAddress: ":7777"}
		LayerConfigFile(live2, base, preFile2)

		assert.Equal(t, ":7777", live2.GrpcAddress)
	})

	t.Run("same_file_twice_is_idempotent", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}

		preFile1 := proto.Clone(base).(*protoconf_server_config.ServerConfig)
		live1 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		LayerConfigFile(live1, base, preFile1)
		singleCallResult := proto.Clone(live1)

		preFile2 := proto.Clone(live1).(*protoconf_server_config.ServerConfig)
		live2 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		LayerConfigFile(live2, base, preFile2)

		assert.True(t, proto.Equal(singleCallResult, live2), "want %v, got %v", singleCallResult, live2)
	})

	t.Run("live_pointer_identity_is_preserved", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		before := live

		LayerConfigFile(live, base, preFile)

		assert.Same(t, before, live)
	})

	// The remaining rows pin the proto.Merge append hazard from fact 4 of the plan's
	// why_not_the_one_line_reversal: without setFieldReplacing and the list arm of
	// matchesBase, every one of them fails. ServerConfig has no repeated field, so
	// InserterConfig's store_address (repeated string) is used instead.

	t.Run("second_file_replaces_first_list", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}

		preFile1 := proto.Clone(base).(*protoconf_inserter_config.InserterConfig)
		live1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		LayerConfigFile(live1, base, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_inserter_config.InserterConfig)
		live2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file2:2"}}
		LayerConfigFile(live2, base, preFile2)

		assert.Equal(t, []string{"file2:2"}, live2.StoreAddress)
	})

	t.Run("same_file_twice_is_idempotent_for_list", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}

		preFile1 := proto.Clone(base).(*protoconf_inserter_config.InserterConfig)
		live1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		LayerConfigFile(live1, base, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_inserter_config.InserterConfig)
		live2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		LayerConfigFile(live2, base, preFile2)

		assert.Equal(t, []string{"file1:1"}, live2.StoreAddress)
	})

	t.Run("env_list_replaces_file_list", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}
		preFile := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"env1:1", "env2:2"}}
		live := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}

		LayerConfigFile(live, base, preFile)

		assert.Equal(t, []string{"env1:1", "env2:2"}, live.StoreAddress)
	})

	t.Run("env_list_and_two_files", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}

		preFile1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"env1:1", "env2:2"}}
		live1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		LayerConfigFile(live1, base, preFile1)
		require.Equal(t, []string{"env1:1", "env2:2"}, live1.StoreAddress)

		preFile2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"env1:1", "env2:2"}}
		live2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file2:2"}}
		LayerConfigFile(live2, base, preFile2)

		assert.Equal(t, []string{"env1:1", "env2:2"}, live2.StoreAddress)
	})
}
