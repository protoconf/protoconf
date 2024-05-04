package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/inserter"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtoconfKVAgentRollout_SubscribeForConfig(t *testing.T) {
	ctx := context.Background()
	testDir := testdata.SmallTestDir()
	kvStore, _ := dummykv.New(ctx, []string{}, &dummykv.Config{})
	// kvStore, _ := consul.New(ctx, []string{
	// 	// "host.docker.internal:8500",
	// 	"127.0.0.1:8500",
	// }, &consul.Config{})
	inserter := inserter.NewProtoconfInserter(testDir, kvStore)
	alphaClient, close := testServer(ctx, newProtoconfKVAgentRollout(kvStore, &protoconf_agent_config.AgentConfig{ChannelName: "alpha"}))
	defer close()
	betaClient, close := testServer(ctx, newProtoconfKVAgentRollout(kvStore, &protoconf_agent_config.AgentConfig{ChannelName: "beta"}))
	defer close()
	prodClient, close := testServer(ctx, newProtoconfKVAgentRollout(kvStore, &protoconf_agent_config.AgentConfig{ChannelName: "prod"}))
	defer close()

	type update struct {
		configName     string
		protoconfValue *protoconf_pb.ProtoconfValue
		metadata       *protoconf_pb.Metadata
	}
	type result struct {
		update *protoconf_pb.ConfigUpdate
		within time.Duration
	}
	type want struct {
		agentChannel string
		agentClient  protoconf_pb.ProtoconfServiceClient
		request      *protoconf_pb.ConfigSubscriptionRequest
		expects      []*result
	}
	type args struct {
		updates []*update
		want    []*want
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name: "test1",
			args: args{
				updates: []*update{
					{
						configName:     "test",
						protoconfValue: &protoconf_pb.ProtoconfValue{Value: newAny(structpb.NewStringValue("hello world!"))},
						metadata:       &protoconf_pb.Metadata{Commit: "commit_1", CommittedAt: timestamppb.Now()},
					},
					{
						configName: "test",
						protoconfValue: &protoconf_pb.ProtoconfValue{
							Value: newAny(structpb.NewStringValue("hello protoconf!")),
							RolloutConfig: &protoconf_pb.ProtoconfValue_ConfigRollout{
								DefaultCooldownTime: durationpb.New(time.Second * 5),
								Stages: []*protoconf_pb.ProtoconfValue_ConfigRollout_Stage{
									{Channel: "alpha", Percentile: 10},
									{Channel: "beta", Percentile: 50},
								},
							},
						},
						metadata: &protoconf_pb.Metadata{Commit: "commit_2", CommittedAt: timestamppb.New(time.Now().Add(time.Second * 5))},
					},
				},
				want: []*want{
					{
						agentChannel: "alpha",
						agentClient:  alphaClient,
						request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "test"},
						expects: []*result{
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("hello world!"))}, within: time.Second},
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("hello protoconf!"))}, within: time.Second * 6},
						},
					},
					{
						agentChannel: "beta",
						agentClient:  betaClient,
						request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "test"},
						expects: []*result{
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("hello world!"))}, within: time.Second},
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("hello protoconf!"))}, within: time.Second * 11},
						},
					},
					{
						agentChannel: "prod",
						agentClient:  prodClient,
						request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "test"},
						expects: []*result{
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("hello world!"))}, within: time.Second},
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("hello protoconf!"))}, within: time.Second * 16},
						},
					},
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		kvStore.DeleteTree(ctx, "/")
		t.Run(tt.name, func(t *testing.T) {
			for _, wantResults := range tt.args.want {
				go func(want *want) {
					t.Run(want.agentChannel, func(t *testing.T) {
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						watcher, err := want.agentClient.SubscribeForConfig(ctx, want.request)
						configCh := recvCh(ctx, watcher)
						assert.NoError(t, err)
						for i, expect := range want.expects {
							t.Run(fmt.Sprint(i), func(t *testing.T) {
								sleep, cancelSleep := context.WithTimeout(ctx, expect.within)
								defer cancelSleep()
								select {
								case <-sleep.Done():
									err := context.Cause(ctx)
									assert.NoError(t, err)
									t.Errorf("timeout waiting for update")
									cancel()
									return
								case item := <-configCh:
									assert.Truef(t, proto.Equal(expect.update, item), "expected \n%s, got \n%s", expect.update, item)
									cancelSleep()
								}
							})
						}
					})
				}(wantResults)
			}
			for _, update := range tt.args.updates {
				assert.NoError(t, inserter.InsertConfig(update.configName, update.protoconfValue, update.metadata))
				time.Sleep(time.Second * 2)
			}
		})
	}
}

func recvCh(ctx context.Context, watcher protoconf_pb.ProtoconfService_SubscribeForConfigClient) chan *protoconf_pb.ConfigUpdate {
	ch := make(chan *protoconf_pb.ConfigUpdate)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				item, err := watcher.Recv()
				if err != nil {
					return
				}
				ch <- item
			}
		}
	}()
	return ch
}
