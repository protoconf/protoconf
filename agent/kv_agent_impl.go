package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"path"

	"github.com/kvtools/valkeyrie/store"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

type ProtoconfKVAgent struct {
	store  store.Store
	config *protoconf_agent_config.AgentConfig
	Logger *zap.Logger
	protoconfservice.ProtoconfServiceServer
}

func NewProtoconfKVAgent(store store.Store, config *protoconf_agent_config.AgentConfig) (*ProtoconfKVAgent, error) {
	_, err := store.Exists(context.Background(), "/", nil)
	if err != nil {
		return nil, errors.Join(errors.New("store is not available"), err)
	}
	logger := zap.New(zapcore.NewNopCore())
	return &ProtoconfKVAgent{store: store, config: config, Logger: logger}, nil
}

func (s *ProtoconfKVAgent) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	ctx := srv.Context()
	logger := s.Logger.With(zap.String("key", request.Path))
	logger.Info("got watch request")
	kvPairCh, err := s.store.Watch(ctx, path.Join(s.config.Prefix, request.Path), &store.ReadOptions{})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	for {
		select {
		case kvPair := <-kvPairCh:
			result := &protoconfvalue.ProtoconfValue{}
			data, err := base64.StdEncoding.DecodeString(string(kvPair.Value))
			if err != nil {
				srv.Send(&protoconfservice.ConfigUpdate{
					Error: err.Error(),
				})
				logger.Error(err.Error())
				// return err
			}
			err = proto.Unmarshal(data, result)
			if err != nil {
				srv.Send(&protoconfservice.ConfigUpdate{
					Error: err.Error(),
				})
				logger.Error(err.Error())
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
			logger.Info("client stopped watching", zap.Error(ctx.Err()))
			return nil
		}
	}

}
