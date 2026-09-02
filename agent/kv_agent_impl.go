package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"path"

	"github.com/kvtools/valkeyrie/store"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type ProtoconfKVAgent struct {
	store  store.Store
	config *protoconf_agent_config.AgentConfig
	Logger *slog.Logger
	protoconf_pb.ProtoconfServiceServer
}

var tracer = otel.Tracer("protoconf-agent")

// storeProbeKey is the sentinel key used by checkStoreAvailable to verify the
// store is reachable at startup. It is dot-prefixed so it will not collide
// with a real config path.
const storeProbeKey = ".protoconf-agent-healthcheck"

// checkStoreAvailable does a read-only reachability check against the store
// at construction time, so an unreachable store fails fast once at boot
// rather than on every client subscription.
//
// The probe key must stay non-empty after the backend's leading-slash
// normalization: etcd rejects an empty key with a transport error that
// Exists does not swallow, so path.Join(config.GetPrefix(), storeProbeKey)
// is used instead of a bare "/" to guarantee the joined key is never empty,
// for any value of config.Prefix including empty. An absent key is
// deliberately treated as success — Exists maps store.ErrKeyNotFound to
// (false, nil), so only a transport error reaches the caller here.
func checkStoreAvailable(ctx context.Context, s store.Store, config *protoconf_agent_config.AgentConfig) error {
	key := path.Join(config.GetPrefix(), storeProbeKey)
	_, err := s.Exists(ctx, key, nil)
	if err != nil {
		return errors.Join(errors.New("store is not available"), err)
	}
	return nil
}

func NewProtoconfKVAgent(store store.Store, config *protoconf_agent_config.AgentConfig) (*ProtoconfKVAgent, error) {
	if err := checkStoreAvailable(context.Background(), store, config); err != nil {
		return nil, err
	}
	logger := slog.Default()
	return &ProtoconfKVAgent{store: store, config: config, Logger: logger}, nil
}

func (s *ProtoconfKVAgent) SubscribeForConfig(request *protoconf_pb.ConfigSubscriptionRequest, srv protoconf_pb.ProtoconfService_SubscribeForConfigServer) error {
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
			result := &protoconf_pb.ProtoconfValue{}
			if kvPair == nil {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(string(kvPair.Value))
			if err != nil {
				srv.Send(&protoconf_pb.ConfigUpdate{
					Error: "failed to decode data from config store, expected base64 encoded value",
				})
				logger.Error(err.Error())
				continue
				// return err
			}
			err = proto.Unmarshal(data, result)
			if err != nil {
				srv.Send(&protoconf_pb.ConfigUpdate{
					Error: "failed to unmarshal data received from config store",
				})
				logger.Error(err.Error())
				continue
				// return err
			}
			err = srv.Send(&protoconf_pb.ConfigUpdate{
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

// GetConfig reads a single config value without opening a watch stream.
func (s *ProtoconfKVAgent) GetConfig(ctx context.Context, request *protoconf_pb.ConfigRequest) (*protoconf_pb.ConfigUpdate, error) {
	ctx, span := tracer.Start(ctx, "GetConfig")
	defer span.End()

	logger := s.Logger.With(slog.String("key", request.Path))
	if peer, ok := peer.FromContext(ctx); ok {
		logger = logger.With(slog.Any("peer_addr", peer.Addr))
	}
	logger.Info("got get request")

	key := path.Join(s.config.Prefix, request.Path)
	kvPair, err := s.store.Get(ctx, key, &store.ReadOptions{})
	if err != nil {
		logger.Error(err.Error())
		if errors.Is(err, store.ErrKeyNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	if kvPair == nil {
		return nil, status.Error(codes.NotFound, "key not found in store")
	}

	result, err := parseProtoconfValue(kvPair)
	if err != nil {
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &protoconf_pb.ConfigUpdate{Value: result.Value}, nil
}
