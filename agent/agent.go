package agent

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/mitchellh/cli"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1/protoconfservice"
	"github.com/protoconf/protoconf/command"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/libprotoconf"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type cliCommand struct{}

type cliConfig struct {
	devProtoconfRoot  string
	grpcAddress       string
	prometheusAddress string
}

func newFlagSet() (*flag.FlagSet, *cliConfig, *command.KVStoreConfig) {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: [OPTION]...")
		flags.PrintDefaults()
	}

	kVConfig := &command.KVStoreConfig{}
	command.AddKVStoreFlags(flags, kVConfig)

	config := &cliConfig{}
	flags.StringVar(&config.devProtoconfRoot, "dev", "", "Development mode - watch a local Protoconf directory for file changes")
	flags.StringVar(&config.grpcAddress, "grpc-address", consts.AgentDefaultAddress, "Agent gRPC address")
	flags.StringVar(&config.prometheusAddress, "http-address", ":9143", "Prometheus http address")

	return flags, config, kVConfig
}

func (c *cliCommand) Run(args []string) int {
	flags, config, kVConfig := newFlagSet()
	flags.Parse(args)

	log.Printf("Starting Protoconf agent at \"%s\", version %s", config.grpcAddress, consts.Version)

	agentServer := &server{}
	var err error
	if config.devProtoconfRoot != "" {
		log.Printf("Using dev mode, watching directory protoconf_root=\"%s\"", config.devProtoconfRoot)
		agentServer.watcher, err = libprotoconf.NewFileWatcher(config.devProtoconfRoot)
	} else {
		log.Printf("Connecting to %s at \"%s\", config path prefix=\"%s\"", kVConfig.Store, kVConfig.Address, kVConfig.Prefix)
		if kVConfig.Store == command.KVStoreConsul {
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Consul, kVConfig.Address, kVConfig.Prefix)
		} else if kVConfig.Store == command.KVStoreZookeeper {
			var address string
			if kVConfig.Address != "" {
				address = kVConfig.Address
			} else {
				address = consts.ZookeeperDefaultAddress
			}
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Zookeeper, address, kVConfig.Prefix)
		} else if kVConfig.Store == command.KVStoreEtcd {
			var address string
			if kVConfig.Address != "" {
				address = kVConfig.Address
			} else {
				address = consts.EtcdDefaultAddress
			}
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Etcd, address, kVConfig.Prefix)
		} else {
			log.Fatalf("Unknown key-value store %s", kVConfig.Store)
		}
	}

	if err != nil {
		log.Printf("Error setting up Protoconf err=%s", err)
		return 1
	}

	defer agentServer.watcher.Close()

	listener, err := net.Listen("tcp", config.grpcAddress)
	if err != nil {
		log.Printf("Error listening on address=\"%s\" err=%s", config.grpcAddress, err)
		return 1
	}

	rpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	protoconfservice.RegisterProtoconfServiceServer(rpcServer, agentServer)
	grpc_prometheus.Register(rpcServer)
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Protoconf agent running")
	g, _ := errgroup.WithContext(context.TODO())
	g.Go(func() error { return rpcServer.Serve(listener) })
	g.Go(func() error { return http.ListenAndServe(config.prometheusAddress, nil) })
	err = g.Wait()
	if err != nil {
		log.Printf("Error serving gRPC, err=%s", err)
		return 1
	}

	return 0
}

func (c *cliCommand) Help() string {
	var b bytes.Buffer
	b.WriteString(c.Synopsis())
	b.WriteString("\n")
	flags, _, _ := newFlagSet()
	flags.SetOutput(&b)
	flags.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Runs a Protoconf agent"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	return &cliCommand{}, nil
}

type server struct {
	watcher libprotoconf.Watcher
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
					log.Printf("Update sent successfuly path=%s", path)
				}
			}()
		}
	}
}
