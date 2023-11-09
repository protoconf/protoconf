package agent

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
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
	"github.com/protoconf/protoconf/agent/filekv"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/libprotoconf"
	"github.com/stephenafamo/orchestra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type server struct {
	watcher libprotoconf.Watcher
	protoconfservice.ProtoconfServiceServer
}

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

func RunAgent(config *protoconf_agent_config.AgentConfig) int {
	var err error
	var store store.Store
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	logger.Info("Starting Protoconf agent", zap.String("address", config.GrpcAddress), zap.String("version", consts.Version))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if config.DevRoot != "" {
		logger.Info("Using dev mode", zap.String("root", config.DevRoot))
		store, err = filekv.New(ctx, []string{}, &filekv.Config{ProtoconfRoot: config.DevRoot})
		config.Prefix = consts.CompiledConfigPath
		config.Store = protoconf_agent_config.AgentConfig_file
	} else {
		logger.Info("Connecting to store", zap.Any("type", config.Store), zap.Any("servers", defaultServers(config)), zap.String("prefix", config.Prefix))
		if config.Store == protoconf_agent_config.AgentConfig_consul {
			store, err = consul.New(ctx, defaultServers(config), &consul.Config{})
		} else if config.Store == protoconf_agent_config.AgentConfig_zookeeper {
			store, err = zookeeper.New(ctx, defaultServers(config), &zookeeper.Config{})
		} else if config.Store == protoconf_agent_config.AgentConfig_etcd {
			store, err = etcdv3.New(ctx, defaultServers(config), &etcdv3.Config{})
		} else {
			log.Fatalf("Unknown key-value store %s", config.Store)
		}
	}
	if err != nil {
		logger.Error("Error setting config store", zap.Error(err))
		return 1
	}
	agent, err := NewProtoconfKVAgent(store, config)
	agent.Logger = logger

	if err != nil {
		logger.Error("Error setting up Protoconf agent", zap.Error(err))
		return 1
	}

	listener, err := net.Listen("tcp", config.GrpcAddress)
	if err != nil {
		log.Printf("Error listening on address=\"%s\" err=%s", config.GrpcAddress, err)
		return 1
	}

	rpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	protoconfservice.RegisterProtoconfServiceServer(rpcServer, agent)
	grpc_prometheus.Register(rpcServer)
	http.Handle("/metrics", promhttp.Handler())

	err = orchestra.PlayUntilSignal(&orchestra.Conductor{Players: map[string]orchestra.Player{
		"grpc": orchestra.PlayerFunc(func(ctx context.Context) error {
			go func() {
				<-ctx.Done()
				rpcServer.Stop()
			}()
			logger.Info("starting protoconf agent")
			err = rpcServer.Serve(listener)
			logger.Info("protoconf agent stopped")
			return err
		}),
		"http": orchestra.NewServerPlayer(
			orchestra.WithHTTPServer(
				&http.Server{Addr: config.HttpAddress},
			),
		),
	},
	}, os.Interrupt, syscall.SIGTERM)
	if err != nil {
		log.Printf("Error serving gRPC, err=%s", err)
		return 1
	}

	return 0
}

func (s server) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	path := request.GetPath()
	log.Printf("Watching path=%s", path)

	stopCh := make(chan struct{})
	watchCh, err := s.watcher.Watch(path, stopCh)
	if err != nil {
		return err
	}

	defer func() {
		select {
		case stopCh <- struct{}{}:
		default:
		}
		close(stopCh)
	}()

	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("Client stopped watching path=%s", path)
			return ctx.Err()
		case config, ok := <-watchCh:
			if !ok {
				log.Printf("Watch channel closed for path=%s", path)
				return errors.New("watch channel closed")
			}

			if config.Error != nil {
				log.Printf("Error watching config, path=%s err=%s", path, config.Error)
				return config.Error
			}

			log.Printf("Sending update on path=%s", path)
			resp := protoconfservice.ConfigUpdate{Value: config.Value}
			go func() {
				if err := srv.Send(&resp); err != nil {
					log.Printf("Error sending config update, path=%s srv=%s err=%s", path, srv, err)
				} else {
					log.Printf("Update sent successfully path=%s", path)
				}
			}()
		}
	}
}
