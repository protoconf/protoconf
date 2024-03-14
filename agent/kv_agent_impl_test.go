package agent

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/kvtools/valkeyrie/store"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newAny(msg proto.Message) *anypb.Any {
	a, _ := anypb.New(msg)
	return a
}

func TestProtoconfKVAgent_SubscribeForConfig(t *testing.T) {
	request := &protoconfservice.ConfigSubscriptionRequest{
		Path: "test",
	}
	expects := &protoconfservice.ConfigUpdate{
		Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	storeClient, _ := dummykv.New(ctx, []string{}, &dummykv.Config{})
	server, err := NewProtoconfKVAgent(storeClient, &protoconf_agent_config.AgentConfig{})
	require.NoError(t, err)
	stub, closer := testServer(ctx, server)
	defer closer()
	watcher, err := stub.SubscribeForConfig(ctx, request)
	require.NoError(t, err)
	b, _ := proto.Marshal(&protoconfvalue.ProtoconfValue{Value: expects.Value})
	t.Run("put and recv", func(t *testing.T) {
		storeClient.Put(
			ctx, request.Path,
			[]byte(base64.StdEncoding.EncodeToString(b)),
			&store.WriteOptions{})
		item, err := watcher.Recv()
		require.NoError(t, err)
		if !proto.Equal(item.Value, expects.Value) {
			t.Errorf("bad config, expected %v, got %v", expects.Value, item.Value)
		}

	})
	t.Run("not base 64", func(t *testing.T) {
		storeClient.Put(ctx, request.GetPath(), []byte("Fail"), &store.WriteOptions{})
		_, err := watcher.Recv()
		assert.NoError(t, err)
		// assert.Equal(t, "failed to unmarshal data received from config store", v.Error)

	})
	t.Run("not a proto message", func(t *testing.T) {
		storeClient.Put(ctx, request.GetPath(), []byte(base64.StdEncoding.EncodeToString([]byte("Fail"))), &store.WriteOptions{})
		_, err := watcher.Recv()
		assert.NoError(t, err)
	})
}

func protoB64Bytes(msg proto.Message) []byte {
	b, _ := proto.Marshal(msg)
	return []byte(base64.StdEncoding.EncodeToString(b))
}

func BenchmarkProtoconfAgent(b *testing.B) {
	storeClient, _ := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	agent, _ := NewProtoconfKVAgent(storeClient, &protoconf_agent_config.AgentConfig{})
	agent.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	client, closer := testServer(context.Background(), agent)
	defer closer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := "root"
	for i := 0; i < b.N; i++ {
		localPath := filepath.Join(path, fmt.Sprintf("%d", i))
		rootWatcher, _ := client.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: path})
		watcher, _ := client.SubscribeForConfig(ctx, &protoconfservice.ConfigSubscriptionRequest{Path: localPath})
		go func() {
			counter := 0
			for {
				update, err := rootWatcher.Recv()
				counter++
				b.Log(counter, update, err)
				if err != nil {
					break
				}
			}
		}()
		go func() {
			update, err := watcher.Recv()
			b.Log(localPath, update, err)
		}()
		err := storeClient.Put(
			context.Background(),
			path,
			protoB64Bytes(&protoconfvalue.ProtoconfValue{Value: newAny(timestamppb.Now())}),
			&store.WriteOptions{},
		)
		if err != nil {
			b.Fail()
		}
		err = storeClient.Put(
			context.Background(),
			localPath,
			protoB64Bytes(&protoconfvalue.ProtoconfValue{Value: newAny(timestamppb.Now())}),
			&store.WriteOptions{},
		)
		if err != nil {
			b.Fail()
		}
	}
}

func testServer(ctx context.Context, srv protoconfservice.ProtoconfServiceServer) (protoconfservice.ProtoconfServiceClient, func()) {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)
	baseServer := grpc.NewServer()
	protoconfservice.RegisterProtoconfServiceServer(baseServer, srv)
	go func() {
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

	client := protoconfservice.NewProtoconfServiceClient(conn)

	return client, closer
}
