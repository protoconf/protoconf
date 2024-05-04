package test

import (
	"context"
	"testing"
	"time"

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

func Test(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init
	protoconfRoot := testdata.SmallTestDir()
	t.Run("mod tidy", func(t *testing.T) {
		ms := lib.NewModuleService(protoconfRoot)
		require.NoError(t, ms.Init(ctx, "CONFIGSPACE"))
		require.NoError(t, ms.Sync(ctx))
	})

	c := lib.NewCompiler(protoconfRoot, false)
	// Create dev agent
	devConfig := &protoconf_agent_config.AgentConfig{
		Prefix: consts.CompiledConfigPath,
		Store:  protoconf_agent_config.AgentConfig_file,
	}
	devStore, err := filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: protoconfRoot})
	assert.NoError(t, err)

	devAgentServer, err := agent.NewProtoconfKVAgent(devStore, devConfig)
	var devAgentClient protoconf_pb.ProtoconfServiceClient
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	// Create production agent
	prodStore, err := dummykv.New(ctx, []string{}, &dummykv.Config{})
	assert.NoError(t, err)
	inserter := inserter.NewProtoconfInserter(protoconfRoot, prodStore)
	prodAgentServer, err := agent.NewProtoconfKVAgentRollout(prodStore, &protoconf_agent_config.AgentConfig{})
	assert.NoError(t, err)
	var prodAgentClient protoconf_pb.ProtoconfServiceClient
	prodCloser := TestServer(ctx, func(s *grpc.Server) {
		protoconf_pb.RegisterProtoconfServiceServer(s, prodAgentServer)
	}, func(conn *grpc.ClientConn) {
		prodAgentClient = protoconf_pb.NewProtoconfServiceClient(conn)
	})
	defer prodCloser()

	// Get first message from materialized_configs
	devWatcher, err := devAgentClient.SubscribeForConfig(ctx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	expected, _ := anypb.New(structpb.NewStringValue("hello"))
	t.Run("get first message on devClient", func(t *testing.T) {
		assert.NoError(t, err)

		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		if !proto.Equal(devConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, devConfigValue.Value)
		}
	})

	tCtx, _ := context.WithTimeout(ctx, 60*time.Second)
	prodWatcher, err := prodAgentClient.SubscribeForConfig(tCtx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	assert.NoError(t, err)
	// Get first message from prodStore
	t.Run("get first message on prodClient", func(t *testing.T) {
		t.Run("insert to prodStore", func(t *testing.T) {
			err = inserter.InsertConfigFile("load_mutable_test" + consts.CompiledConfigExtension)
			assert.NoError(t, err)
		})
		prodConfigValue, err := prodWatcher.Recv()
		assert.NoError(t, err)
		if err != nil {
			return
		}

		if !proto.Equal(prodConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, prodConfigValue.Value)
		}
	})
	// Change config via mutation rpc

	mutationValue, _ := anypb.New(structpb.NewStringValue("hello mutation"))
	t.Run("change config via mutation rpc", func(t *testing.T) {
		_, err = devMutationClient.MutateConfig(ctx, &protoconf_pb.ConfigMutationRequest{
			Path: "mutation_test", Value: &protoconf_pb.ProtoconfValue{
				ProtoFile: "google/protobuf/struct.proto",
				Value:     mutationValue,
			},
		})
		assert.NoError(t, err)
	})

	// Compile after mutation
	t.Run("compile after mutation", func(t *testing.T) {
		err = c.CompileFile("load_mutable_test.pconf")
		require.NoError(t, err)
	})

	// fetch update from watcher and validate the value after mutation
	t.Run("fetch update on devClient", func(t *testing.T) {
		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		if !proto.Equal(devConfigValue.Value, mutationValue) {
			t.Errorf("expected \n%s, got \n%s", mutationValue, devConfigValue.Value)
		}

	})

	devWatcher.CloseSend()

	t.Run("get update on prodClient", func(t *testing.T) {
		t.Run("insert to prodStore", func(t *testing.T) {
			err = inserter.InsertConfigFile("load_mutable_test" + consts.CompiledConfigExtension)
			assert.NoError(t, err)
		})
		prodConfigValue, err := prodWatcher.Recv()
		assert.NoError(t, err)
		if err != nil {
			return
		}

		require.Truef(t, proto.Equal(prodConfigValue.Value, mutationValue),
			"expected \n%s, got \n%s", mutationValue, prodConfigValue.Value)

	})
	prodWatcher.CloseSend()
	err = c.CompileFile("load_remote_with_load_local.pconf")
	assert.NoError(t, err)
	err = c.CompileFile("load_remote.pconf")
	assert.NoError(t, err)
	// devWatcher, err = devAgentClient.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_remote_with_load_local"})
	t.Run("load_remote on dev", func(t *testing.T) {
		devWatcher, err = devAgentClient.SubscribeForConfig(ctx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_remote"})
		assert.NoError(t, err)
		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		expected = &anypb.Any{TypeUrl: "type.googleapis.com/terraform.v1.Terraform"}
		if !proto.Equal(devConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, devConfigValue.Value)
		}
		devWatcher.CloseSend()
	})
	t.Run("load_remote prod", func(t *testing.T) {
		newCtx, _ := context.WithTimeout(ctx, 10*time.Second)
		watcher, err := prodAgentClient.SubscribeForConfig(newCtx, &protoconf_pb.ConfigSubscriptionRequest{Path: "load_remote"})
		require.NoError(t, err)
		t.Run("insert load_remote to prod", func(t *testing.T) {
			require.NoError(t,
				inserter.InsertConfigFile("load_remote"+consts.CompiledConfigExtension),
			)
		})
		value, err := watcher.Recv()
		require.NoError(t, err)
		expected = &anypb.Any{TypeUrl: "type.googleapis.com/terraform.v1.Terraform"}
		if !proto.Equal(value.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, value.Value)
		}
		watcher.CloseSend()
	})
}
