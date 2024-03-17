package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"syscall"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/kvtools/consul"
	"github.com/kvtools/etcdv3"
	"github.com/kvtools/valkeyrie/store"
	"github.com/kvtools/zookeeper"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/agent/configmaps"
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/agent/otelkv"
	"github.com/protoconf/protoconf/consts"
	slogotel "github.com/remychantenay/slog-otel"
	"github.com/stephenafamo/orchestra"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

func defaultServers(config *protoconf_agent_config.AgentConfig) []string {
	if len(config.Servers) > 0 {
		return config.Servers
	}
	switch config.Store {
	case protoconf_agent_config.AgentConfig_consul:
		return []string{"127.0.0.1:8500"}
	case protoconf_agent_config.AgentConfig_etcd:
		return []string{consts.EtcdDefaultAddress}
	case protoconf_agent_config.AgentConfig_zookeeper:
		return []string{consts.ZookeeperDefaultAddress}
	}
	return []string{}
}

func RunAgent(ctx context.Context, config *protoconf_agent_config.AgentConfig) error {
	hostname, _ := os.Hostname()
	var err error
	var store store.Store

	expTracer, err := otlptracegrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	resources, _ := resource.New(ctx,
		resource.WithFromEnv(),   // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcess(),   // This option configures a set of Detectors that discover process information
		resource.WithOS(),        // This option configures a set of Detectors that discover OS information
		resource.WithContainer(), // This option configures a set of Detectors that discover container information
		resource.WithHost(),      // This option configures a set of Detectors that discover host information
		resource.WithAttributes(
			semconv.ServiceNameKey.String("protoconf"),
			semconv.ServiceVersionKey.String(consts.Version),
		),
	)
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(expTracer),
		trace.WithResource(resources),
	)
	defer func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			slog.Default().Error("error shutting down tracer provider", slog.String("error", err.Error()))
		}
	}()
	otel.SetTracerProvider(tracerProvider)

	// From here, the tracerProvider can be used by instrumentation to collect
	// telemetry.
	expMeter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		panic(err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(expMeter)),
		metric.WithResource(resources),
	)
	defer func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Default().Error("error shutting down meter provider", slog.String("error", err.Error()))
		}
	}()
	otel.SetMeterProvider(meterProvider)

	var loggerHandler slog.Handler
	loggerHandlerOptions := &slog.HandlerOptions{
		Level:     slog.Level(config.LogLevel),
		AddSource: config.LogSource,
	}
	if config.LogAsJson {
		loggerHandler = slog.NewJSONHandler(os.Stderr, loggerHandlerOptions)
	} else {
		loggerHandler = slog.NewTextHandler(os.Stderr, loggerHandlerOptions)
	}

	logger := slog.New(
		slogotel.OtelHandler{
			Next: loggerHandler,
		},
	)
	slog.SetDefault(logger)
	logger.Info("Starting Protoconf agent", slog.String("address", config.GrpcAddress), slog.String("version", consts.Version), slog.String("http-address", config.HttpAddress), slog.Int("pid", os.Getpid()))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if config.DevRoot != "" {
		logger.Info("Using dev mode", slog.String("root", config.DevRoot))
		store, err = filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: config.DevRoot})
		config.Prefix = consts.CompiledConfigPath
		config.Store = protoconf_agent_config.AgentConfig_file
	} else {
		logger.Info("Connecting to store", slog.Any("type", config.Store), slog.Any("servers", defaultServers(config)), slog.String("prefix", config.Prefix))
		switch config.Store {
		case protoconf_agent_config.AgentConfig_consul:
			store, err = consul.New(ctx, defaultServers(config), &consul.Config{})
		case protoconf_agent_config.AgentConfig_zookeeper:
			store, err = zookeeper.New(ctx, defaultServers(config), &zookeeper.Config{})
		case protoconf_agent_config.AgentConfig_etcd:
			store, err = etcdv3.New(ctx, defaultServers(config), &etcdv3.Config{})
		case protoconf_agent_config.AgentConfig_configmaps:
			store, err = configmaps.New(ctx, defaultServers(config), &configmaps.Config{Namespace: config.Namespace})
		default:
			err = fmt.Errorf("unknown key-value store %s", config.Store)
		}
	}
	if err != nil {
		return errors.Join(errors.New("error setting config store"), err)
	}

	var agent protoconfservice.ProtoconfServiceServer
	if config.EnableRollout {
		if err == nil {
			config.AgentId = hostname
		}
		agent, err = NewProtoconfKVAgentRollout(otelkv.New(ctx, store), config)
	} else {
		agent, err = NewProtoconfKVAgent(otelkv.New(ctx, store), config)
	}
	if err != nil {
		return errors.Join(errors.New("error setting up protoconf agent"), err)
	}

	listener, err := net.Listen("tcp", config.GrpcAddress)
	if err != nil {
		return errors.Join(errors.New("error creating listener"), err)
	}

	rpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	protoconfservice.RegisterProtoconfServiceServer(rpcServer, agent)
	grpc_prometheus.Register(rpcServer)
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof", pprof.Profile)
	mux.Handle("/metrics", promhttp.Handler())

	err = orchestra.PlayUntilSignal(ctx, &orchestra.Conductor{Players: map[string]orchestra.Player{
		"grpc": orchestra.PlayerFunc(func(ctx context.Context) error {
			context.AfterFunc(ctx, func() {
				logger.Info("stopping grpc server")
				rpcServer.GracefulStop()
				logger.Info("protoconf grpc agent stopped")
			})
			logger.Info("starting protoconf agent")
			return rpcServer.Serve(listener)
		}),
		"http": orchestra.NewServerPlayer(
			&http.Server{Addr: config.HttpAddress, Handler: mux},
		),
	},
		Logger: orchestra.LoggerFromSlog(slog.LevelInfo, logger),
	}, os.Interrupt, syscall.SIGTERM)
	if err != nil {
		return errors.Join(errors.New("error serving protoconf agent"), err)
	}

	return nil
}
