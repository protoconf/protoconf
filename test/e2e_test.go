package test

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/protoconf/protoconf/agent"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/consts"
	v1 "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/protoconf/protoconf/server"
	protoconfmutation "github.com/protoconf/protoconf/server/api/proto/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init
	protoconfRoot := testdata.SmallTestDir()
	c := lib.NewCompiler(protoconfRoot, false)
	assert.NoError(t, c.ModuleService.Init(ctx, "init.pinc"))
	assert.NoError(t, c.SyncModules(ctx))

	// Load dev agent
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
		protoconfmutation.RegisterProtoconfMutationServiceServer(s, devMutationServer)
	}, func(conn *grpc.ClientConn) {
		devAgentClient = protoconfservice.NewProtoconfServiceClient(conn)
		devMutationClient = protoconfmutation.NewProtoconfMutationServiceClient(conn)
	})
	defer devCloser()
	devWatcher, err := devAgentClient.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: "load_mutable_test"})
	assert.NoError(t, err)

	devConfigValue, err := devWatcher.Recv()
	assert.NoError(t, err)
	expected := &anypb.Any{TypeUrl: "type.googleapis.com/TestMessage", Value: []byte("\n\x05hello")}
	if !proto.Equal(devConfigValue.Value, expected) {
		t.Errorf("expected \n%s, got \n%s", expected, devConfigValue.Value)
	}
	mutationValue := &anypb.Any{TypeUrl: "type.googleapis.com/TestMessage", Value: []byte("\n\x05world")}
	_, err = devMutationClient.MutateConfig(ctx, &protoconfmutation.ConfigMutationRequest{
		Path: "mutation_test", Value: &v1.ProtoconfValue{
			ProtoFile: "test.proto",
			Value:     mutationValue,
		},
	})
	assert.NoError(t, err)
	err = c.CompileFile("load_mutable_test.pconf")
	assert.NoError(t, err)

	devConfigValue, err = devWatcher.Recv()
	assert.NoError(t, err)
	if !proto.Equal(devConfigValue.Value, mutationValue) {
		t.Errorf("expected \n%s, got \n%s", mutationValue, devConfigValue.Value)
	}
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
