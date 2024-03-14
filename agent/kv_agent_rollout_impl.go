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
	"sync/atomic"
	"time"

	"github.com/kvtools/valkeyrie/store"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/consts"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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
	tracer      trace.Tracer

	protoconfservice.ProtoconfServiceServer
}

var (
	meter = otel.Meter("protoconf.dev/agent",
		metric.WithInstrumentationVersion(consts.Version),
	)
	lastUpdate    = time.Now().UnixMilli()
	lastInsertion = time.Now()
)

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
	tracer := otel.Tracer("protoconf.dev/agent")
	return &ProtoconfKVAgentRollout{
		store:       store,
		config:      config,
		Logger:      logger,
		sendMutex:   &sync.Mutex{},
		channelName: config.ChannelName,
		agentId:     config.AgentId,
		tracer:      tracer,
	}
}

type configVersionReporter struct {
	configVersionReporter metric.Int64UpDownCounter
	configReloads         metric.Int64Counter
	pathValue             attribute.KeyValue
	commitValue           attribute.KeyValue
	authorValue           attribute.KeyValue
	stageValue            attribute.KeyValue
	isLockedMetric        metric.Int64UpDownCounter
	attrs                 metric.MeasurementOption
	mutex                 *sync.Mutex
}

func newConfigVersionReporter(path string) *configVersionReporter {
	pathValue := attribute.String("config_path", path)
	commitValue := attribute.String("commit", "unknown")
	authorValue := attribute.String("author", "unknown")
	stageValue := attribute.String("stage", "default")
	reporter, _ := meter.Int64UpDownCounter(
		"protoconf_agent_config_version",
		metric.WithDescription("Config version"),
	)
	attrs := metric.WithAttributeSet(attribute.NewSet(pathValue, stageValue, commitValue, authorValue))
	reporter.Add(context.Background(), 1, attrs)
	_, _ = meter.Int64ObservableGauge("protoconf_agent_time_since_last_update",
		metric.WithUnit("ms"),
		metric.WithDescription("Time since last config update"),
		metric.WithInt64Callback(func(ctx context.Context, io metric.Int64Observer) error {
			lu := atomic.LoadInt64(&lastUpdate)
			io.Observe(time.Since(time.UnixMilli(lu)).Milliseconds(), attrs)
			return nil
		}),
	)
	_, _ = meter.Int64ObservableGauge("protoconf_agent_apply_timestamp",
		metric.WithDescription("Timestamp of the last config update"),
		metric.WithInt64Callback(func(ctx context.Context, io metric.Int64Observer) error {
			lu := atomic.LoadInt64(&lastUpdate)
			io.Observe(lu, attrs)
			return nil
		}),
	)
	_, _ = meter.Int64ObservableGauge("protoconf_agent_time_since_last_insertion",
		metric.WithUnit("ms"),
		metric.WithDescription("Time since last config insertion"),
		metric.WithInt64Callback(func(ctx context.Context, io metric.Int64Observer) error {
			io.Observe(time.Since(lastInsertion).Milliseconds(), attrs)
			return nil
		}),
	)
	isLockedMetric, _ := meter.Int64UpDownCounter("protoconf_agent_version_stage_inflight",
		metric.WithDescription("Whether the agent is locked to a specific stage"),
	)
	configReloads, _ := meter.Int64Counter("protoconf_agent_config_reloads", metric.WithDescription("Number of config reloads"))
	return &configVersionReporter{
		configVersionReporter: reporter,
		pathValue:             pathValue,
		commitValue:           commitValue,
		authorValue:           authorValue,
		stageValue:            stageValue,
		configReloads:         configReloads,
		isLockedMetric:        isLockedMetric,
		attrs:                 attrs,
		mutex:                 &sync.Mutex{},
	}
}

func (c *configVersionReporter) Report(ctx context.Context, version *protoconfvalue.Metadata, stage string, callbacks ...func() error) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if version != nil {
		c.configVersionReporter.Add(ctx, -1, c.attrs)
		c.commitValue = attribute.String("commit", version.Commit[:8])
		c.authorValue = attribute.String("author", version.AuthorEmail)
		c.stageValue = attribute.String("stage", stage)
		c.attrs = metric.WithAttributeSet(attribute.NewSet(c.pathValue, c.stageValue, c.commitValue, c.authorValue))
		c.configVersionReporter.Add(ctx, 1, c.attrs)
		c.configReloads.Add(ctx, 1, c.attrs)
		c.isLockedMetric.Add(ctx, 1, c.attrs)
		defer c.isLockedMetric.Add(ctx, -1, c.attrs)
	}
	g := &errgroup.Group{}
	for _, cb := range callbacks {
		g.Go(cb)
	}
	return g.Wait()
}

func (s *ProtoconfKVAgentRollout) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	ctx := srv.Context()
	versionReporter := newConfigVersionReporter(request.Path)

	logger := s.Logger.With(slog.String("key", request.Path))
	if peer, ok := peer.FromContext(ctx); ok {
		logger = logger.With(slog.Any("peer_addr", peer.Addr))
	}
	logger.InfoContext(ctx, "start watching for config changes")
	wg, ctx := errgroup.WithContext(ctx)
	kvStableConfigPath := path.Join(s.config.Prefix, request.Path, "config.data")
	wg.Go(func() error {
		ctx, span := s.tracer.Start(ctx, "protoconf.agent.WatchConfigChanges")
		defer span.End()
		return s.watchKey(ctx, kvStableConfigPath, func(ctx context.Context, kvPair *store.KVPair) error {
			ctx, span := s.tracer.Start(ctx, "protoconf.agent.ConfigUpdate")
			defer span.End()
			l := logger.WithGroup("config_update")
			l.Debug("config update received")
			result, err := parseProtoconfValue(kvPair)
			attrs := []attribute.KeyValue{}
			if result.Metadata != nil {
				l = l.With(slog.String("commit", result.Metadata.Commit[:8]))
				attrs = append(attrs,
					attribute.String("commit", result.Metadata.Commit),
					attribute.String("author", result.Metadata.AuthorEmail),
					attribute.String("insertion_time", result.Metadata.InsertedAt.String()),
				)
				lastInsertion = result.Metadata.InsertedAt.AsTime()
			}
			span.AddEvent("config update received", trace.WithAttributes(attrs...))
			if err != nil {
				s.sendWithLock(ctx, srv, &protoconfservice.ConfigUpdate{
					Error: err.Error(),
				}, time.Now())
				l.ErrorContext(ctx, err.Error())
				return nil
			}

			err = versionReporter.Report(ctx, result.Metadata, "default", func() error {
				return s.sendWithLock(ctx, srv, &protoconfservice.ConfigUpdate{
					Value: result.Value,
				}, time.Now())
			})
			if err != nil {
				logger.ErrorContext(ctx, err.Error())
				return err
			}
			l.InfoContext(ctx, "config update sent")
			return nil

		})
	})

	kvRolloutConfigPath := path.Join(s.config.Prefix, request.Path, "rollout.json")
	wg.Go(func() error {
		ctx, span := s.tracer.Start(ctx, "protoconf.agent.WatchRolloutConfig")
		defer span.End()
		return s.watchKey(ctx, kvRolloutConfigPath, func(ctx context.Context, kvPair *store.KVPair) error {
			ctx, span := s.tracer.Start(ctx, "protoconf.agent.RolloutConfigUpdate")
			defer span.End()
			if kvPair == nil {
				return nil
			}

			rolloutConfig := &protoconfvalue.ProtoconfValue_ConfigRollout{}
			err := protojson.Unmarshal(kvPair.Value, rolloutConfig)
			if err != nil {
				return errors.Join(errors.New("failed to unmarshal data received from config store"), err)
			}
			logger.With(slog.Any("rollout_config", rolloutConfig)).DebugContext(ctx, "rollout config update received")
			for _, stage := range rolloutConfig.Stages {
				if s.matchStage(request, stage) {
					logger.With(
						slog.String("stage", stage.Channel),
						slog.String("commit", strings.Split(stage.Version, ".")[1][:8]),
					).InfoContext(ctx, "found matching stage")
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
					return versionReporter.Report(ctx, result.Metadata, stage.Channel, func() error {
						return s.sendWithLock(ctx, srv, &protoconfservice.ConfigUpdate{Value: result.Value}, stage.ExpiresAt.AsTime())
					})
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

func (s *ProtoconfKVAgentRollout) watchKey(ctx context.Context, key string, callback func(context.Context, *store.KVPair) error) error {
	kvPairCh, err := s.store.Watch(ctx, key, &store.ReadOptions{})
	if err != nil {
		return err
	}
	for {
		select {
		case kvPair := <-kvPairCh:
			if kvPair != nil {
				err = callback(ctx, kvPair)
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
	ctx, span := s.tracer.Start(ctx, "protoconf.agent.SendConfigUpdate")

	s.sendMutex.Lock()
	defer s.sendMutex.Unlock()
	err := srv.Send(value)
	if err != nil {
		return err
	}
	span.End()
	atomic.StoreInt64(&lastUpdate, time.Now().UnixMilli())
	ctx, span = s.tracer.Start(ctx, "protoconf.agent.LockUntilDeadline")
	defer span.End()
	sleep, cancel := context.WithDeadline(ctx, lockUntil)
	defer cancel()
	<-sleep.Done()

	return nil
}
