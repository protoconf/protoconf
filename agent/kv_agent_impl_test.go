package agent

import (
	"context"
	"encoding/base64"
	"log"
	"net"
	"testing"

	"github.com/kvtools/valkeyrie/store"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func newAny(msg proto.Message) *anypb.Any {
	a, _ := anypb.New(msg)
	return a
}

func TestProtoconfKVAgent_SubscribeForConfig(t *testing.T) {
	type args struct {
		request  *protoconfservice.ConfigSubscriptionRequest
		expects  []*protoconfservice.ConfigUpdate
		limit    int
		limitB64 int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				limit:    10,
				limitB64: 10,
				request: &protoconfservice.ConfigSubscriptionRequest{
					Path: "test",
				},
				expects: []*protoconfservice.ConfigUpdate{
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
				},
			},
		},
		{
			name: "test multi",
			args: args{
				limit:    10,
				limitB64: 10,
				request: &protoconfservice.ConfigSubscriptionRequest{
					Path: "test/multi",
				},
				expects: []*protoconfservice.ConfigUpdate{
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
				},
			},
		},
		{
			name: "test not base64",
			args: args{
				limit:    10,
				limitB64: 2,
				request: &protoconfservice.ConfigSubscriptionRequest{
					Path: "test/multi",
				},
				expects: []*protoconfservice.ConfigUpdate{
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
					{
						Value: newAny(&structpb.Value{Kind: &structpb.Value_StringValue{StringValue: "hello world!"}}),
					},
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			storeClient, _ := dummykv.New(ctx, []string{}, &dummykv.Config{})
			server, _ := NewProtoconfKVAgent(storeClient, &protoconf_agent_config.AgentConfig{})
			stub, closer := testServer(ctx, server)
			defer closer()
			watcher, _ := stub.SubscribeForConfig(ctx, tt.args.request)
			outs := []*protoconfservice.ConfigUpdate{}
			for i, item := range tt.args.expects {
				if i > tt.args.limit {
					storeClient.Put(ctx, tt.args.request.GetPath(), []byte("Fail"), &store.WriteOptions{})
				}
				if i > tt.args.limitB64 {
					storeClient.Put(ctx, tt.args.request.GetPath(), []byte(base64.StdEncoding.EncodeToString([]byte("Fail"))), &store.WriteOptions{})
				}
				b, _ := proto.Marshal(&protoconfvalue.ProtoconfValue{Value: item.Value})
				storeClient.Put(
					ctx, tt.args.request.Path,
					[]byte(base64.StdEncoding.EncodeToString(b)),
					&store.WriteOptions{})
				item, err := watcher.Recv()
				outs = append(outs, item)
				if err != nil {
					break
				}
				if item.Error != "" {
					break
				}
			}
			// storeClient.Put(ctx, tt.args.request.Path, []byte("Fail"), &store.WriteOptions{})
			for i, item := range tt.args.expects {
				if (i > tt.args.limit) || i > tt.args.limitB64 {
					break
				}
				if !proto.Equal(item, outs[i]) {
					t.Errorf("These items does not match(%d): got: %s, expected: %s", i, item, outs[i])
				}
			}
			// if err := s.SubscribeForConfig(tt.args.request, tt.args.srv); (err != nil) != tt.wantErr {
			// 	t.Errorf("ProtoconfKVAgent.SubscribeForConfig() error = %v, wantErr %v", err, tt.wantErr)
			// }
		})
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
