package test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kvtools/consul"
	"github.com/kvtools/etcdv3"
	"github.com/kvtools/valkeyrie/store"
	"github.com/kvtools/zookeeper"
	"github.com/protoconf/protoconf/agent"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/inserter"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/server"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// BackendType represents the type of KV backend to test
type BackendType string

const (
	BackendDummy     BackendType = "dummy"
	BackendEtcd      BackendType = "etcd"
	BackendConsul    BackendType = "consul"
	BackendZookeeper BackendType = "zookeeper"
)

// BackendConfig holds configuration for a specific backend
type BackendConfig struct {
	Type          BackendType
	Servers       []string
	RequiresInfra bool // Whether this backend requires external infrastructure
}

// getBackendConfigs returns the list of backends to test based on environment
func getBackendConfigs() []BackendConfig {
	configs := []BackendConfig{
		{
			Type:          BackendDummy,
			Servers:       []string{},
			RequiresInfra: false,
		},
	}

	// Add real backends if running in CI or if E2E_BACKENDS is set
	if os.Getenv("CI") != "" || os.Getenv("E2E_BACKENDS") != "" {
		configs = append(configs,
			BackendConfig{
				Type:          BackendEtcd,
				Servers:       []string{getEnvOrDefault("ETCD_ADDRESS", consts.EtcdDefaultAddress)},
				RequiresInfra: true,
			},
			BackendConfig{
				Type:          BackendConsul,
				Servers:       []string{getEnvOrDefault("CONSUL_ADDRESS", "127.0.0.1:8500")},
				RequiresInfra: true,
			},
			BackendConfig{
				Type:          BackendZookeeper,
				Servers:       []string{getEnvOrDefault("ZOOKEEPER_ADDRESS", consts.ZookeeperDefaultAddress)},
				RequiresInfra: true,
			},
		)
	}

	return configs
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createStore creates a KV store based on the backend configuration
func createStore(ctx context.Context, config BackendConfig) (store.Store, error) {
	switch config.Type {
	case BackendDummy:
		return dummykv.New(ctx, []string{}, &dummykv.Config{})
	case BackendEtcd:
		return etcdv3.New(ctx, config.Servers, &etcdv3.Config{})
	case BackendConsul:
		return consul.New(ctx, config.Servers, &consul.Config{})
	case BackendZookeeper:
		return zookeeper.New(ctx, config.Servers, &zookeeper.Config{})
	default:
		return nil, fmt.Errorf("unknown backend type: %s", config.Type)
	}
}

// waitForBackend checks if the backend is available
func waitForBackend(ctx context.Context, config BackendConfig, timeout time.Duration) error {
	if !config.RequiresInfra {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s backend at %v", config.Type, config.Servers)
		case <-ticker.C:
			store, err := createStore(ctx, config)
			if err == nil && store != nil {
				// Try to perform a basic operation
				_, err = store.List(ctx, "test", nil)
				if err == nil || err == store.ErrKeyNotFound {
					return nil
				}
			}
		}
	}
}

// TestE2EMultiBackend runs the e2e test suite against multiple backends
func TestE2EMultiBackend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode.")
	}

	backendConfigs := getBackendConfigs()

	for _, backendConfig := range backendConfigs {
		backendConfig := backendConfig // capture range variable
		t.Run(string(backendConfig.Type), func(t *testing.T) {
			// Wait for backend to be ready
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if backendConfig.RequiresInfra {
				err := waitForBackend(ctx, backendConfig, 30*time.Second)
				if err != nil {
					t.Skipf("Skipping %s backend: %v", backendConfig.Type, err)
					return
				}
			}

			runE2ETestWithBackend(t, ctx, backendConfig)
		})
	}
}

// runE2ETestWithBackend runs the complete e2e test suite with a specific backend
func runE2ETestWithBackend(t *testing.T, ctx context.Context, backendConfig BackendConfig) {
	// Init
	protoconfRoot := testdata.SmallTestDir()
	t.Run("mod_tidy", func(t *testing.T) {
		ms := lib.NewModuleService(protoconfRoot)
		require.NoError(t, ms.Init(ctx, "CONFIGSPACE"))
		require.NoError(t, ms.Sync(ctx))
	})

	c := lib.NewCompiler(protoconfRoot, false)

	// Create dev agent (always uses file backend for local development)
	devConfig := &protoconf_agent_config.AgentConfig{
		Prefix: consts.CompiledConfigPath,
		Store:  protoconf_agent_config.AgentConfig_file,
	}
	devStore, err := filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: protoconfRoot})
	require.NoError(t, err)

	devAgentServer, err := agent.NewProtoconfKVAgent(devStore, devConfig)
	require.NoError(t, err)

	var devAgentClient protoconf_pb.ProtoconfServiceClient
	devMutationServer := server.NewProtoconfMutationServer(protoconfRoot)
	var devMutationClient protoconf_pb.ProtoconfMutationServiceClient
	devCloser := TestServer(ctx, func(s *grpc.Server) {
		protoconf_pb.RegisterProtoconfServiceServer(s, devAgentServer)
		protoconf_pb.RegisterProtoconfMutationServiceServer(s, devMutationServer)
	}, func(conn *grpc.ClientConn) {
		devAgentClient = protoconf_pb.NewProtoconfServiceClient(conn)
		devMutationClient = protoconf_pb.NewProtoconfMutationServiceClient(conn)
	})
	defer devCloser()

	// Create production agent with the specified backend
	prodStore, err := createStore(ctx, backendConfig)
	require.NoError(t, err, "failed to create %s store", backendConfig.Type)

	// Clean up any existing test data
	testPrefix := "test_" + string(backendConfig.Type) + "/"
	if backendConfig.RequiresInfra {
		// Try to clean up, but don't fail if it doesn't exist
		_ = prodStore.DeleteTree(ctx, testPrefix)
	}

	inserter := inserter.NewProtoconfInserter(protoconfRoot, prodStore)
	prodAgentConfig := &protoconf_agent_config.AgentConfig{
		Prefix: testPrefix,
	}
	prodAgentServer, err := agent.NewProtoconfKVAgentRollout(prodStore, prodAgentConfig)
	require.NoError(t, err)

	var prodAgentClient protoconf_pb.ProtoconfServiceClient
	prodCloser := TestServer(ctx, func(s *grpc.Server) {
		protoconf_pb.RegisterProtoconfServiceServer(s, prodAgentServer)
	}, func(conn *grpc.ClientConn) {
		prodAgentClient = protoconf_pb.NewProtoconfServiceClient(conn)
	})
	defer prodCloser()

	// Test 1: Get first message from materialized_configs on dev client
	devWatcher, err := devAgentClient.SubscribeForConfig(ctx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	require.NoError(t, err)

	expected, _ := anypb.New(structpb.NewStringValue("hello"))
	t.Run("get_first_message_on_devClient", func(t *testing.T) {
		devConfigValue, err := devWatcher.Recv()
		require.NoError(t, err)
		assert.True(t, proto.Equal(devConfigValue.Value, expected),
			"expected %s, got %s", expected, devConfigValue.Value)
	})

	// Test 2: Get first message from prod store
	tCtx, tCancel := context.WithTimeout(ctx, 60*time.Second)
	defer tCancel()

	prodWatcher, err := prodAgentClient.SubscribeForConfig(tCtx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	require.NoError(t, err)

	t.Run("get_first_message_on_prodClient", func(t *testing.T) {
		t.Run("insert_to_prodStore", func(t *testing.T) {
			err = inserter.InsertConfigFile("load_mutable_test" + consts.CompiledConfigExtension)
			require.NoError(t, err)
		})

		prodConfigValue, err := prodWatcher.Recv()
		require.NoError(t, err)
		assert.True(t, proto.Equal(prodConfigValue.Value, expected),
			"expected %s, got %s", expected, prodConfigValue.Value)
	})

	// Test 3: Change config via mutation rpc
	mutationValue, _ := anypb.New(structpb.NewStringValue("hello mutation"))
	t.Run("change_config_via_mutation_rpc", func(t *testing.T) {
		_, err = devMutationClient.MutateConfig(ctx, &protoconf_pb.ConfigMutationRequest{
			Path: "mutation_test",
			Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "google/protobuf/struct.proto",
				Value:     mutationValue,
			},
		})
		require.NoError(t, err)
	})

	// Test 4: Compile after mutation
	t.Run("compile_after_mutation", func(t *testing.T) {
		err = c.CompileFile("load_mutable_test.pconf")
		require.NoError(t, err)
	})

	// Test 5: Fetch update from watcher and validate the value after mutation
	t.Run("fetch_update_on_devClient", func(t *testing.T) {
		devConfigValue, err := devWatcher.Recv()
		require.NoError(t, err)
		assert.True(t, proto.Equal(devConfigValue.Value, mutationValue),
			"expected %s, got %s", mutationValue, devConfigValue.Value)
	})

	devWatcher.CloseSend()

	// Test 6: Get update on prod client
	t.Run("get_update_on_prodClient", func(t *testing.T) {
		t.Run("insert_to_prodStore", func(t *testing.T) {
			err = inserter.InsertConfigFile("load_mutable_test" + consts.CompiledConfigExtension)
			require.NoError(t, err)
		})

		prodConfigValue, err := prodWatcher.Recv()
		require.NoError(t, err)
		assert.True(t, proto.Equal(prodConfigValue.Value, mutationValue),
			"expected %s, got %s", mutationValue, prodConfigValue.Value)
	})

	prodWatcher.CloseSend()

	// Test 7: Test load_remote
	err = c.CompileFile("load_remote_with_load_local.pconf")
	require.NoError(t, err)
	err = c.CompileFile("load_remote.pconf")
	require.NoError(t, err)

	t.Run("load_remote_on_dev", func(t *testing.T) {
		devWatcher, err := devAgentClient.SubscribeForConfig(ctx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_remote"})
		require.NoError(t, err)
		defer devWatcher.CloseSend()

		devConfigValue, err := devWatcher.Recv()
		require.NoError(t, err)

		expected := &anypb.Any{TypeUrl: "type.googleapis.com/terraform.v1.Terraform"}
		assert.True(t, proto.Equal(devConfigValue.Value, expected),
			"expected %s, got %s", expected, devConfigValue.Value)
	})

	t.Run("load_remote_prod", func(t *testing.T) {
		newCtx, newCancel := context.WithTimeout(ctx, 10*time.Second)
		defer newCancel()

		watcher, err := prodAgentClient.SubscribeForConfig(newCtx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_remote"})
		require.NoError(t, err)
		defer watcher.CloseSend()

		t.Run("insert_load_remote_to_prod", func(t *testing.T) {
			require.NoError(t,
				inserter.InsertConfigFile("load_remote"+consts.CompiledConfigExtension),
			)
		})

		value, err := watcher.Recv()
		require.NoError(t, err)

		expected := &anypb.Any{TypeUrl: "type.googleapis.com/terraform.v1.Terraform"}
		assert.True(t, proto.Equal(value.Value, expected),
			"expected %s, got %s", expected, value.Value)
	})

	// Cleanup
	if backendConfig.RequiresInfra {
		_ = prodStore.DeleteTree(ctx, testPrefix)
	}
}
