package agent

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"log/slog"
	"math/big"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/kvtools/valkeyrie/store"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type ProtoconfKVAgentRollout struct {
	store       store.Store
	config      *protoconf_agent_config.AgentConfig
	Logger      *slog.Logger
	sendMutex   *sync.Mutex
	channelName string
	agentId     string

	protoconfservice.ProtoconfServiceServer
}

func NewProtoconfKVAgentRollout(store store.Store, config *protoconf_agent_config.AgentConfig) (*ProtoconfKVAgentRollout, error) {
	_, err := store.Exists(context.Background(), "/", nil)
	if err != nil {
		return nil, errors.Join(errors.New("store is not available"), err)
	}
	return newProtoconfKVAgentRollout(store, config), nil
}

func newProtoconfKVAgentRollout(store store.Store, config *protoconf_agent_config.AgentConfig) *ProtoconfKVAgentRollout {
	logger := slog.Default()
	if config.ChannelName != "" {
		logger = logger.With(slog.String("channel_name", config.ChannelName))
	}
	if config.AgentId != "" {
		logger = logger.With(slog.String("agent_id", config.AgentId))
	}
	slog.SetDefault(logger)
	return &ProtoconfKVAgentRollout{
		store:       store,
		config:      config,
		Logger:      logger,
		sendMutex:   &sync.Mutex{},
		channelName: config.ChannelName,
		agentId:     config.AgentId,
	}
}

func (s *ProtoconfKVAgentRollout) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	ctx := srv.Context()

	logger := s.Logger.With(slog.String("key", request.Path))
	if peer, ok := peer.FromContext(ctx); ok {
		logger = logger.With(slog.Any("peer_addr", peer.Addr))
	}
	logger.Info("start watching for config changes")
	wg, ctx := errgroup.WithContext(ctx)
	kvStableConfigPath := path.Join(s.config.Prefix, request.Path, "config.data")
	wg.Go(func() error {
		return s.watchKey(ctx, kvStableConfigPath, func(kvPair *store.KVPair) error {
			l := logger.WithGroup("config_update")
			l.Debug("config update received")
			result, err := parseProtoconfValue(kvPair)
			if result.Metadata != nil {
				l = l.With(slog.String("commit", result.Metadata.Commit[:8]))
			}
			if err != nil {
				s.sendWithLock(ctx, srv, &protoconfservice.ConfigUpdate{
					Error: err.Error(),
				}, time.Now())
				l.Error(err.Error())
				return nil
			}
			err = s.sendWithLock(ctx, srv, &protoconfservice.ConfigUpdate{
				Value: result.Value,
			}, time.Now())
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			l.Info("config update sent")
			return nil

		})
	})

	kvStableMetadataPath := path.Join(s.config.Prefix, request.Path, "metadata.json")
	wg.Go(func() error {
		return s.watchKey(ctx, kvStableMetadataPath, func(kvPair *store.KVPair) error {
			metadata := &protoconfvalue.Metadata{}
			err := protojson.Unmarshal(kvPair.Value, metadata)
			if err != nil {
				return errors.Join(errors.New("failed to unmarshal data received from config store"), err)
			}
			logger.With(slog.Any("metadata", metadata)).Debug("metadata update received")
			return nil
		})
	})

	kvRolloutConfigPath := path.Join(s.config.Prefix, request.Path, "rollout.json")
	wg.Go(func() error {
		return s.watchKey(ctx, kvRolloutConfigPath, func(kvPair *store.KVPair) error {
			if kvPair == nil {
				return nil
			}

			rolloutConfig := &protoconfvalue.ProtoconfValue_ConfigRollout{}
			err := protojson.Unmarshal(kvPair.Value, rolloutConfig)
			if err != nil {
				return errors.Join(errors.New("failed to unmarshal data received from config store"), err)
			}
			logger.With(slog.Any("rollout_config", rolloutConfig)).Debug("rollout config update received")
			for _, stage := range rolloutConfig.Stages {
				if s.matchStage(request, stage) {
					logger.With(slog.String("stage", stage.Channel), slog.String("commit", strings.Split(stage.Version, ".")[1][:8])).Info("found matching stage")
					kvStageConfigPath := path.Join(s.config.Prefix, request.Path, stage.Version, "config.data")
					kvPair, err := s.store.Get(ctx, kvStageConfigPath, &store.ReadOptions{})
					if err != nil {
						logger.Error(err.Error())
						return err
					}
					result, err := parseProtoconfValue(kvPair)
					if err != nil {
						logger.Error(err.Error())
						return err
					}
					return s.sendWithLock(ctx, srv, &protoconfservice.ConfigUpdate{Value: result.Value}, stage.ExpiresAt.AsTime())
				}
			}
			return nil
		})
	})
	err := wg.Wait()
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}

func (s *ProtoconfKVAgentRollout) matchStage(request *protoconfservice.ConfigSubscriptionRequest, stage *protoconfvalue.ProtoconfValue_ConfigRollout_Stage) bool {
	if chName := s.getChannelName(request); chName != "" && chName == stage.Channel {
		return true
	}
	// get hash of the hostname + request path
	hasher := md5.New()
	hasher.Write([]byte(request.Path))
	hasher.Write([]byte(s.agentId))
	h := hasher.Sum(nil)
	// get int value of the h[0:8] % 100
	// if the value is less than the percentage, return true
	// otherwise return false
	// this is to ensure that the same host always gets the same stage
	// and that the stage is consistent across all hosts
	// but that the stage is different for different hosts
	n := big.NewInt(0).SetBytes(h[0:8])
	percentage := n.Mod(n, big.NewInt(100))

	return percentage.Cmp(big.NewInt(int64(stage.Percentile))) < 0

}

func (s *ProtoconfKVAgentRollout) getChannelName(request *protoconfservice.ConfigSubscriptionRequest) string {
	if request.Channel != "" {
		return request.Channel
	}
	return s.channelName
}

func parseProtoconfValue(kvPair *store.KVPair) (*protoconfvalue.ProtoconfValue, error) {
	result := &protoconfvalue.ProtoconfValue{}
	data, err := base64.StdEncoding.DecodeString(string(kvPair.Value))
	if err != nil {
		return nil, errors.Join(errors.New("failed to decode data from config store, expected base64 encoded value"), err)
	}
	err = proto.Unmarshal(data, result)
	if err != nil {
		return nil, errors.Join(errors.New("failed to unmarshal data received from config store"), err)
	}
	return result, nil
}

func (s *ProtoconfKVAgentRollout) watchKey(ctx context.Context, key string, callback func(*store.KVPair) error) error {
	kvPairCh, err := s.store.Watch(ctx, key, &store.ReadOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case kvPair := <-kvPairCh:
			if kvPair != nil {
				err = callback(kvPair)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *ProtoconfKVAgentRollout) sendWithLock(ctx context.Context, srv protoconfservice.ProtoconfService_SubscribeForConfigServer, value *protoconfservice.ConfigUpdate, lockUntil time.Time) error {
	s.sendMutex.Lock()
	defer s.sendMutex.Unlock()
	err := srv.Send(value)
	if err != nil {
		return err
	}
	sleep, cancel := context.WithDeadline(ctx, lockUntil)
	defer cancel()
	<-sleep.Done()

	return nil
}
