package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/inserter"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtoconfKVAgentRollout_GetConfig(t *testing.T) {
	ctx := context.Background()
	testDir := testdata.SmallTestDir()
	kvStore, _ := dummykv.New(ctx, []string{}, &dummykv.Config{})
	ins := inserter.NewProtoconfInserter(testDir, kvStore)
	client, closeFn := testServer(ctx, newProtoconfKVAgentRollout(kvStore, &protoconf_agent_config.AgentConfig{}))
	defer closeFn()

	value := &protoconf_pb.ProtoconfValue{Value: newAny(structpb.NewStringValue("hello getconfig"))}
	metadata := &protoconf_pb.Metadata{Commit: "commit_getconfig", CommittedAt: timestamppb.Now()}
	require.NoError(t, ins.InsertConfig("getconfig_rollout_test", value, metadata))

	t.Run("returns inserted value", func(t *testing.T) {
		got, err := client.GetConfig(ctx, &protoconf_pb.ConfigRequest{Path: "getconfig_rollout_test"})
		require.NoError(t, err)
		assert.True(t, proto.Equal(got.Value, value.Value))

		// A real inserter.InsertConfig writes both config.data and config.json;
		// this proves getRawConfigJSON's key shape matches the production
		// layout, not just the hand-seeded dummykv fixtures in http_test.go.
		require.NotNil(t, got.Raw)
		assert.Equal(t, "application/json", got.Raw.ContentType)
		var decoded any
		assert.NoError(t, json.Unmarshal(got.Raw.Data, &decoded), "raw.data must parse as JSON")
	})

	t.Run("uninserted path returns NotFound", func(t *testing.T) {
		_, err := client.GetConfig(ctx, &protoconf_pb.ConfigRequest{Path: "never_inserted"})
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

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
		{
			name: "no_rollout",
			args: args{
				updates: []*update{
					{
						configName:     "simple_key",
						protoconfValue: &protoconf_pb.ProtoconfValue{Value: newAny(structpb.NewStringValue("plain value"))},
						metadata:       &protoconf_pb.Metadata{Commit: "abcdef1234567890abcdef1234567890abcdef12", CommittedAt: timestamppb.Now()},
					},
				},
				want: []*want{
					{
						agentChannel: "alpha",
						agentClient:  alphaClient,
						request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "simple_key"},
						expects: []*result{
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("plain value"))}, within: time.Second * 5},
						},
					},
					{
						agentChannel: "beta",
						agentClient:  betaClient,
						request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "simple_key"},
						expects: []*result{
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("plain value"))}, within: time.Second * 5},
						},
					},
					{
						agentChannel: "prod",
						agentClient:  prodClient,
						request:      &protoconf_pb.ConfigSubscriptionRequest{Path: "simple_key"},
						expects: []*result{
							{update: &protoconf_pb.ConfigUpdate{Value: newAny(structpb.NewStringValue("plain value"))}, within: time.Second * 5},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		kvStore.DeleteTree(ctx, "/")
		t.Run(tt.name, func(t *testing.T) {
			var ready sync.WaitGroup
			var done sync.WaitGroup
			ready.Add(len(tt.args.want))
			done.Add(len(tt.args.want))
			for _, wantResults := range tt.args.want {
				go func(want *want) {
					defer done.Done()
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					watcher, err := want.agentClient.SubscribeForConfig(ctx, want.request)
					if !assert.NoErrorf(t, err, "%s: subscribe failed", want.agentChannel) {
						ready.Done()
						return
					}
					configCh := recvCh(ctx, watcher)
					ready.Done()
					for i, expect := range want.expects {
						sleep, cancelSleep := context.WithTimeout(ctx, expect.within)
						select {
						case <-sleep.Done():
							err := context.Cause(ctx)
							assert.NoErrorf(t, err, "%s: expectation %d", want.agentChannel, i)
							t.Errorf("%s: expectation %d: timeout waiting for update", want.agentChannel, i)
							cancelSleep()
							cancel()
							return
						case item := <-configCh:
							assert.Truef(t, proto.Equal(expect.update, item), "%s: expectation %d: expected \n%s, got \n%s", want.agentChannel, i, expect.update, item)
							cancelSleep()
						}
					}
				}(wantResults)
			}
			// ready.Wait() proves the client-side stream exists before the first
			// insert, not that the server handler has already reached
			// store.Watch. The residual window is closed by the initial-value
			// path in dummykv.Store.Watch — if a Put beats the server's Watch,
			// the key now exists and Get delivers the current value to the new
			// watcher.
			// ponytail: this barrier cannot survive a server handler stalling
			// for the full 2s between inserts (a scheduling pathology, not a
			// race) — upgrade to a server-side ack if that ever proves flaky.
			ready.Wait()
			for _, update := range tt.args.updates {
				assert.NoError(t, inserter.InsertConfig(update.configName, update.protoconfValue, update.metadata))
				time.Sleep(time.Second * 2)
			}
			done.Wait()
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
