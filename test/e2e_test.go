package test

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"github.com/protoconf/protoconf/agent"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/consts"
	v1 "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/protoconf/protoconf/inserter"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/server"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
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
	var devAgentClient protoconfservice.ProtoconfServiceClient
	assert.NoError(t, err)

	devMutationServer := server.NewProtoconfMutationServer(protoconfRoot)
	var devMutationClient protoconfmutation.ProtoconfMutationServiceClient
	devCloser := testServer(ctx, func(s *grpc.Server) {
		protoconfservice.RegisterProtoconfServiceServer(s, devAgentServer)
		protoconf_pb.RegisterProtoconfMutationServiceServer(s, devMutationServer)
	}, func(conn *grpc.ClientConn) {
		devAgentClient = protoconfservice.NewProtoconfServiceClient(conn)
		devMutationClient = protoconfmutation.NewProtoconfMutationServiceClient(conn)
	})
	defer devCloser()
	assert.NoError(t, err)

	// Create production agent
	prodStore, err := dummykv.New(ctx, []string{}, &dummykv.Config{})
	assert.NoError(t, err)
	inserter := inserter.NewProtoconfInserter(protoconfRoot, prodStore)
	prodAgentServer, err := agent.NewProtoconfKVAgentRollout(prodStore, &protoconf_agent_config.AgentConfig{})
	assert.NoError(t, err)
	var prodAgentClient protoconfservice.ProtoconfServiceClient
	prodCloser := testServer(ctx, func(s *grpc.Server) {
		protoconfservice.RegisterProtoconfServiceServer(s, prodAgentServer)
	}, func(conn *grpc.ClientConn) {
		prodAgentClient = protoconfservice.NewProtoconfServiceClient(conn)
	})
	defer prodCloser()

	// Get first message from materialized_configs
	devWatcher, err := devAgentClient.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	expected := &anypb.Any{TypeUrl: "type.googleapis.com/TestMessage", Value: []byte("\n\x05hello")}
	t.Run("get first message on devClient", func(t *testing.T) {
		assert.NoError(t, err)

		devConfigValue, err := devWatcher.Recv()
		assert.NoError(t, err)
		if !proto.Equal(devConfigValue.Value, expected) {
			t.Errorf("expected \n%s, got \n%s", expected, devConfigValue.Value)
		}
	})

	tCtx, _ := context.WithTimeout(ctx, 60*time.Second)
	prodWatcher, err := prodAgentClient.SubscribeForConfig(tCtx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_mutable_test"})
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
		_, err = devMutationClient.MutateConfig(ctx, &protoconfmutation.ConfigMutationRequest{
			Path: "mutation_test", Value: &v1.ProtoconfValue{
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
		devWatcher, err = devAgentClient.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_remote"})
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
		watcher, err := prodAgentClient.SubscribeForConfig(newCtx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_remote"})
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

type regServer func(s *grpc.Server)
type addConnectionToClient func(conn *grpc.ClientConn)

func testServer(ctx context.Context, regServer regServer, addConnectionToClient addConnectionToClient) func() {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)
	baseServer := grpc.NewServer()
	regServer(baseServer)
	go func() {
		context.AfterFunc(ctx, func() { baseServer.GracefulStop() })
		if err := baseServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	closer := func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	}

	addConnectionToClient(conn)

	return closer
}
