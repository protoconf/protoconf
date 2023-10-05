package agent

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/mitchellh/cli"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	configtool "github.com/protoconf/libprotoconf"
	protoconfservice "github.com/protoconf/protoconf/agent/api/proto/v1"
	protoconf_agent_config "github.com/protoconf/protoconf/agent/config/v1"
	"github.com/protoconf/protoconf/consts"
	"github.com/protoconf/protoconf/libprotoconf"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type cliCommand struct {
	config *protoconf_agent_config.AgentConfig
	flag   *flag.FlagSet
}

func (c *cliCommand) Run(args []string) int {
	c.flag.Parse(args)
	config := c.config

	log.Printf("Starting Protoconf agent at \"%s\", version %s", config.GrpcAddress, consts.Version)

	agentServer := &server{}
	var err error
	if config.DevRoot != "" {
		log.Printf("Using dev mode, watching directory protoconf_root=\"%s\"", config.DevRoot)
		agentServer.watcher, err = libprotoconf.NewFileWatcher(config.DevRoot)
	} else {
		log.Printf("Connecting to %s at \"%s\", config path prefix=\"%s\"", config.Store, config.Servers, config.Prefix)
		if config.Store == protoconf_agent_config.AgentConfig_consul {
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Consul, strings.Join(config.Servers, ","), config.Prefix)
		} else if config.Store == protoconf_agent_config.AgentConfig_zookeeper {
			var address string
			if len(config.Servers) > 0 {
				address = strings.Join(config.Servers, ",")
			} else {
				address = consts.ZookeeperDefaultAddress
			}
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Zookeeper, address, config.Prefix)
		} else if config.Store == protoconf_agent_config.AgentConfig_etcd {
			var address string
			if len(config.Servers) > 0 {
				address = strings.Join(config.Servers, ",")
			} else {
				address = consts.EtcdDefaultAddress
			}
			agentServer.watcher, err = libprotoconf.NewKVWatcher(libprotoconf.Etcd, address, config.Prefix)
		} else {
			log.Fatalf("Unknown key-value store %s", config.Store)
		}
	}

	if err != nil {
		log.Printf("Error setting up Protoconf err=%s", err)
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
	log.Println("Protoconf agent running")
	g, _ := errgroup.WithContext(context.TODO())
	g.Go(func() error { return rpcServer.Serve(listener) })
	g.Go(func() error { return http.ListenAndServe(config.HttpAddress, nil) })
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
	c.flag.SetOutput(&b)
	c.flag.Usage()
	return b.String()
}

func (c *cliCommand) Synopsis() string {
	return "Runs a Protoconf agent"
}

// Command is a cli.CommandFactory
func Command() (cli.Command, error) {
	c := &cliCommand{
		config: &protoconf_agent_config.AgentConfig{
			GrpcAddress: ":4300",
			HttpAddress: ":4380",
			Servers:     []string{"127.0.0.1:8500"},
		}}
	lpc := configtool.NewConfig(c.config)
	lpc.SetEnvKeyPrefix("PROTOCONF_AGENT")
	lpc.Environment()
	c.flag = lpc.DefaultFlagSet()
	c.flag.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "dev":
			f.Usage = "Run the agent in development mode watching local protoconf directory for file changes\n[env: PROTOCONF_AGENT_DEV]"
		case "grpc-address":
			f.Usage = "Address to bind the gRPC listener\n[env: PROTOCONF_AGENT_GRPC_ADDRESS]"
		case "http-address":
			f.Usage = "Address to bind the admin HTTP listener\n[env: PROTOCONF_AGENT_HTTP_ADDRESS]"
		case "insecure":
			f.Usage = "Skip TLS gRPC TLS configuration\n[env: PROTOCONF_AGENT_INSECURE]"
		case "prefix":
			f.Usage = "Key-value store key prefix\n[env: PROTOCONF_AGENT_PREFIX]"
		case "store-address":
			f.Usage = "Key-value store addresses\n" + f.Usage
		case "store":
			f.Usage = "Key-value store type\n" + f.Usage
		}
	})
	c.flag.Func("config-file", "Agent configuration file (available formats: json, jsonnet, yaml, pb)", func(filename string) error {
		b, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		err = lpc.Unmarshal(filename, b)
		if err != nil {
			return fmt.Errorf("failed to parse config file: %v", err)
		}
		return nil
	})

	return c, nil
}
