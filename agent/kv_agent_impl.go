package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"path"

	"github.com/kvtools/valkeyrie/store"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

type ProtoconfKVAgent struct {
	store  store.Store
	config *protoconf_agent_config.AgentConfig
	Logger *slog.Logger
	protoconfservice.ProtoconfServiceServer
}

var tracer = otel.Tracer("protoconf-agent")

func NewProtoconfKVAgent(store store.Store, config *protoconf_agent_config.AgentConfig) (*ProtoconfKVAgent, error) {
	_, err := store.Exists(context.Background(), "/", nil)
	if err != nil {
		return nil, errors.Join(errors.New("store is not available"), err)
	}
	logger := slog.Default()
	return &ProtoconfKVAgent{store: store, config: config, Logger: logger}, nil
}

func (s *ProtoconfKVAgent) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	ctx := srv.Context()
	ctx, span := tracer.Start(ctx, "SubscribeForConfig")
	defer span.End()

	logger := s.Logger.With(slog.String("key", request.Path))
	if peer, ok := peer.FromContext(ctx); ok {
		logger = logger.With(slog.Any("peer_addr", peer.Addr))
	}
	logger.Info("got watch request")
	kvPairCh, err := s.store.Watch(ctx, path.Join(s.config.Prefix, request.Path), &store.ReadOptions{})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	for {
		select {
		case kvPair := <-kvPairCh:
			span.AddEvent("received config update")
			result := &protoconfvalue.ProtoconfValue{}
			if kvPair == nil {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(string(kvPair.Value))
			if err != nil {
				srv.Send(&protoconfservice.ConfigUpdate{
					Error: "failed to decode data from config store, expected base64 encoded value",
				})
				logger.Error(err.Error())
				continue
				// return err
			}
			err = proto.Unmarshal(data, result)
			if err != nil {
				srv.Send(&protoconfservice.ConfigUpdate{
					Error: "failed to unmarshal data received from config store",
				})
				logger.Error(err.Error())
				continue
				// return err
			}
			err = srv.Send(&protoconfservice.ConfigUpdate{
				Value: result.Value,
			})
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			logger.Info("config update sent")
		case <-ctx.Done():
			logger.Info("client stopped watching", slog.Any("error", ctx.Err()))
			return nil
		}
	}

}
