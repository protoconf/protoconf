package command

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	protoconf_inserter_config "github.com/protoconf/protoconf/inserter/config/v1"
	protoconf_server_config "github.com/protoconf/protoconf/server/config/v1"
)

// TestLayerConfigFile pins the layering rules ConfigLayerer implements, independent of any
// CLI: a config file beats the factory defaults, an env var (or already-parsed flag) beats the
// config file, an empty file changes nothing, a later config file beats an earlier one, repeat
// loads are idempotent, and the live message pointer is never replaced. Repeated fields obey
// the same rules as scalars: no entry is ever appended or duplicated.
//
// base is now a pristine defaults snapshot handed to NewConfigLayerer, rather than a mutated
// accumulator: the accumulated file layer and the env/flag provenance set now live inside the
// ConfigLayerer value instead of inside base. Rows that layer two files construct ONE
// ConfigLayerer and reuse it for both calls.
//
// Rows use *protoconf_server_config.ServerConfig for scalar fields; the repeated-field rows
// use *protoconf_inserter_config.InserterConfig (ServerConfig has no repeated field); the
// message-typed-field rows use *protoconf_agent_config.AgentConfig (tls_config/store_tls).
// All rows run directly under this test (no intermediate t.Run groups) so each is its own
// top-level subtest.
func TestLayerConfigFile(t *testing.T) {
	t.Run("file_overrides_factory_default", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

		assert.Equal(t, ":8888", live.GrpcAddress)
	})

	t.Run("env_overrides_file", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":9999"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

		assert.Equal(t, ":9999", live.GrpcAddress)
	})

	t.Run("empty_file_keeps_default_and_env", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env"}
		live := &protoconf_server_config.ServerConfig{}

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

		want := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env"}
		assert.True(t, proto.Equal(want, live), "want %v, got %v", want, live)
	})

	t.Run("unrelated_fields_are_untouched", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301", AuthToken: "from-env"}
		live := &protoconf_server_config.ServerConfig{TlsCert: "cert.pem"}

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

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

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

		assert.Equal(t, ":8888", live.GrpcAddress)
	})

	t.Run("second_file_overrides_first", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_server_config.ServerConfig)
		live1 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_server_config.ServerConfig)
		live2 := &protoconf_server_config.ServerConfig{GrpcAddress: ":7777"}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, ":7777", live2.GrpcAddress)
	})

	t.Run("same_file_twice_is_idempotent", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_server_config.ServerConfig)
		live1 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		l.LayerConfigFile(live1, preFile1)
		singleCallResult := proto.Clone(live1)

		preFile2 := proto.Clone(live1).(*protoconf_server_config.ServerConfig)
		live2 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		l.LayerConfigFile(live2, preFile2)

		assert.True(t, proto.Equal(singleCallResult, live2), "want %v, got %v", singleCallResult, live2)
	})

	t.Run("live_pointer_identity_is_preserved", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		before := live

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

		assert.Same(t, before, live)
	})

	// The remaining ported rows pin the proto.Merge append hazard from the plan's
	// why_provenance_and_not_a_comparison_patch fact 4: without setFieldReplacing, every
	// one of them fails. ServerConfig has no repeated field, so InserterConfig's
	// store_address (repeated string) is used instead.

	t.Run("second_file_replaces_first_list", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_inserter_config.InserterConfig)
		live1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_inserter_config.InserterConfig)
		live2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file2:2"}}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, []string{"file2:2"}, live2.StoreAddress)
	})

	t.Run("same_file_twice_is_idempotent_for_list", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_inserter_config.InserterConfig)
		live1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_inserter_config.InserterConfig)
		live2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, []string{"file1:1"}, live2.StoreAddress)
	})

	t.Run("env_list_replaces_file_list", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}
		preFile := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"env1:1", "env2:2"}}
		live := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}

		l := NewConfigLayerer(base, nil)
		l.LayerConfigFile(live, preFile)

		assert.Equal(t, []string{"env1:1", "env2:2"}, live.StoreAddress)
	})

	t.Run("env_list_and_two_files", func(t *testing.T) {
		base := &protoconf_inserter_config.InserterConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"env1:1", "env2:2"}}
		live1 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file1:1"}}
		l.LayerConfigFile(live1, preFile1)
		require.Equal(t, []string{"env1:1", "env2:2"}, live1.StoreAddress)

		preFile2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"env1:1", "env2:2"}}
		live2 := &protoconf_inserter_config.InserterConfig{StoreAddress: []string{"file2:2"}}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, []string{"env1:1", "env2:2"}, live2.StoreAddress)
	})

	// The rows below are new for this plan. The first two close VERIFICATION.md's two
	// failed truths at the unit level: #7 (env_wins_over_two_files_on_value_coincidence)
	// and #8 (later_file_wins_for_message_field).

	t.Run("env_wins_over_two_files_on_value_coincidence", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		l := NewConfigLayerer(base, nil)

		// File 1 coincidentally sets the same value the env var supplied.
		preFile1 := &protoconf_server_config.ServerConfig{GrpcAddress: ":9999"}
		live1 := &protoconf_server_config.ServerConfig{GrpcAddress: ":9999"}
		l.LayerConfigFile(live1, preFile1)
		require.Equal(t, ":9999", live1.GrpcAddress)

		preFile2 := proto.Clone(live1).(*protoconf_server_config.ServerConfig)
		live2 := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, ":9999", live2.GrpcAddress)
	})

	t.Run("later_file_wins_for_message_field", func(t *testing.T) {
		base := &protoconf_agent_config.AgentConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_agent_config.AgentConfig)
		live1 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "a.pem"},
			},
		}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_agent_config.AgentConfig)
		live2 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "b.pem"},
			},
		}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, "b.pem", live2.GetTlsConfig().GetCertFile())
	})

	t.Run("env_wins_over_two_files_for_message_field", func(t *testing.T) {
		base := &protoconf_agent_config.AgentConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "env.pem"},
			},
		}
		live1 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "a.pem"},
			},
		}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_agent_config.AgentConfig)
		live2 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "b.pem"},
			},
		}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, "env.pem", live2.GetTlsConfig().GetCertFile())
	})

	t.Run("later_file_wins_for_store_tls_message_field", func(t *testing.T) {
		base := &protoconf_agent_config.AgentConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_agent_config.AgentConfig)
		live1 := &protoconf_agent_config.AgentConfig{
			StoreTls: &protoconf_agent_config.AgentConfig_TLSConfig{
				Ca: &protoconf_agent_config.AgentConfig_TLSConfig_CaFile{CaFile: "ca1.pem"},
			},
		}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_agent_config.AgentConfig)
		live2 := &protoconf_agent_config.AgentConfig{
			StoreTls: &protoconf_agent_config.AgentConfig_TLSConfig{
				Ca: &protoconf_agent_config.AgentConfig_TLSConfig_CaFile{CaFile: "ca2.pem"},
			},
		}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, "ca2.pem", live2.GetStoreTls().GetCaFile())
	})

	t.Run("message_field_recursive_merge_keeps_untouched_submessage_fields", func(t *testing.T) {
		base := &protoconf_agent_config.AgentConfig{}
		l := NewConfigLayerer(base, nil)

		preFile1 := proto.Clone(base).(*protoconf_agent_config.AgentConfig)
		live1 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "a.pem"},
			},
		}
		l.LayerConfigFile(live1, preFile1)

		preFile2 := proto.Clone(live1).(*protoconf_agent_config.AgentConfig)
		live2 := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Key: &protoconf_agent_config.AgentConfig_TLSConfig_KeyFile{KeyFile: "k.pem"},
			},
		}
		l.LayerConfigFile(live2, preFile2)

		assert.Equal(t, "a.pem", live2.GetTlsConfig().GetCertFile())
		assert.Equal(t, "k.pem", live2.GetTlsConfig().GetKeyFile())
	})

	t.Run("flag_provenance_beats_file_when_flag_equals_factory_default", func(t *testing.T) {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.String("grpc-address", ":4301", "")
		require.NoError(t, fs.Parse([]string{"-grpc-address", ":4301"}))

		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		l := NewConfigLayerer(base, fs)

		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		l.LayerConfigFile(live, preFile)

		// Without flag provenance this would yield ":8888": the flag's value equals the
		// factory default and is invisible to value comparison alone.
		assert.Equal(t, ":4301", live.GrpcAddress)
	})

	t.Run("unknown_flag_name_is_ignored", func(t *testing.T) {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.String("config-file", "", "")
		require.NoError(t, fs.Parse([]string{"-config-file", "whatever"}))

		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		l := NewConfigLayerer(base, fs)

		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}
		l.LayerConfigFile(live, preFile)

		// "config-file" matches no ServerConfig field, so it must not be mistaken for
		// grpc-address or mark any field explicit.
		assert.Equal(t, ":8888", live.GrpcAddress)
	})

	t.Run("nil_flagset_is_accepted", func(t *testing.T) {
		base := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		l := NewConfigLayerer(base, nil)

		preFile := &protoconf_server_config.ServerConfig{GrpcAddress: ":4301"}
		live := &protoconf_server_config.ServerConfig{GrpcAddress: ":8888"}

		assert.NotPanics(t, func() {
			l.LayerConfigFile(live, preFile)
		})
		assert.Equal(t, ":8888", live.GrpcAddress)
	})

	t.Run("set_field_replacing_deep_copies_message_values", func(t *testing.T) {
		// IN-01 regression test: fails if the message arm installs the submessage by
		// reference instead of deep-copying it.
		src := &protoconf_agent_config.AgentConfig{
			TlsConfig: &protoconf_agent_config.AgentConfig_TLSConfig{
				Cert: &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "a.pem"},
			},
		}
		dst := &protoconf_agent_config.AgentConfig{}

		fd := src.ProtoReflect().Descriptor().Fields().ByName("tls_config")
		require.NotNil(t, fd)
		setFieldReplacing(dst.ProtoReflect(), fd, src.ProtoReflect().Get(fd))

		src.TlsConfig.Cert = &protoconf_agent_config.AgentConfig_TLSConfig_CertFile{CertFile: "mutated.pem"}

		assert.Equal(t, "a.pem", dst.GetTlsConfig().GetCertFile())
	})
}
