package agent

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/avast/retry-go"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/libprotoconf"
	"github.com/stephenafamo/orchestra"
	"google.golang.org/grpc"
)

type server struct {
	watcher libprotoconf.Watcher
	protoconfservice.ProtoconfServiceServer
}

type logger struct{}

func (logger) Log(items ...interface{}) error {
	return nil
}

func defaultServers(config *protoconf_agent_config.AgentConfig) string {
	if len(config.Servers) > 0 {
		return strings.Join(config.Servers, ",")
	}
	switch config.Store {
	case protoconf_agent_config.AgentConfig_consul:
		return "127.0.0.1:8500"
	case protoconf_agent_config.AgentConfig_etcd:
		return consts.EtcdDefaultAddress
	case protoconf_agent_config.AgentConfig_zookeeper:
		return consts.ZookeeperDefaultAddress
	}
	return ""
}

func RunAgent(config *protoconf_agent_config.AgentConfig) int {
	log.Printf("Starting Protoconf agent at \"%s\", version %s", config.GrpcAddress, consts.Version)

	agentServer := &server{}
	var err error
	if config.DevRoot != "" {
		log.Printf("Using dev mode, watching directory protoconf_root=\"%s\"", config.DevRoot)
		agentServer.watcher, err = libprotoconf.NewFileWatcher(config.DevRoot)
	} else {
		log.Printf("Connecting to %s at \"%s\", config path prefix=\"%s\"", config.Store, defaultServers(config), config.Prefix)
		if config.Store == protoconf_agent_config.AgentConfig_consul {
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Consul, defaultServers(config), config.Prefix)
		} else if config.Store == protoconf_agent_config.AgentConfig_zookeeper {
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Zookeeper, defaultServers(config), config.Prefix)
		} else if config.Store == protoconf_agent_config.AgentConfig_etcd {
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Etcd, defaultServers(config), config.Prefix)
		} else {
			log.Fatalf("Unknown key-value store %s", config.Store)
		}
	}
	if err != nil {
		log.Printf("Error setting up Protoconf err=%s", err)
		return 1
	}

	log.Printf("checking connection to store (type: %s, servers: %s)", config.Store, defaultServers(config))
	err = retry.Do(func() error {
		return agentServer.watcher.Ping()
	})
	if err != nil {
		log.Println(err)
		return 1
	}

	defer agentServer.watcher.Close()

	listener, err := net.Listen("tcp", config.GrpcAddress)
	if err != nil {
		log.Printf("Error listening on address=\"%s\" err=%s", config.GrpcAddress, err)
		return 1
	}

	rpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	protoconfservice.RegisterProtoconfServiceServer(rpcServer, agentServer)
	grpc_prometheus.Register(rpcServer)
	http.Handle("/metrics", promhttp.Handler())

	err = orchestra.PlayUntilSignal(&orchestra.Conductor{Players: map[string]orchestra.Player{
		"grpc": orchestra.PlayerFunc(func(ctx context.Context) error {
			go func() {
				<-ctx.Done()
				rpcServer.Stop()
			}()
			log.Println("starting protoconf agent")
			err = rpcServer.Serve(listener)
			log.Println("protoconf agent stopped")
			return err
		}),
		"http": &orchestra.ServerPlayer{
			&http.Server{Addr: config.HttpAddress},
		},
	},
	// Logger: &logger{},
	})
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
