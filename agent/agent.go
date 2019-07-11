package agent

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/mitchellh/cli"
	"google.golang.org/grpc"
	protoconfservice "protoconf.com/agent/api/proto/v1/protoconfservice"
	"protoconf.com/command"
	"protoconf.com/consts"
	"protoconf.com/libprotoconf"
)

type cliCommand struct{}

type cliConfig struct {
	devProtoconfRoot string
	grpcAddress string
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

	return flags, config, kVConfig
}

func (c *cliCommand) Run(args []string) int {
	flags, config, kVConfig := newFlagSet()
	flags.Parse(args)

	agentServer := &server{prefix: kVConfig.Prefix}
	var err error
	if config.devProtoconfRoot != "" {
		agentServer.watcher, err = libprotoconf.NewFileWatcher(config.devProtoconfRoot)
	} else {
		agentServer.watcher, err = libprotoconf.NewConsulWatcher(kVConfig.Address)
	}

	if err != nil {
		log.Printf("Error setting up protoconf err=%s", err)
		return 1
	}

	defer agentServer.watcher.Close()

	listener, err := net.Listen("tcp", config.grpcAddress)
	if err != nil {
		log.Printf("Error listening on address=%s err=%s", config.grpcAddress, err)
		return 1
	}

	rpcServer := grpc.NewServer()
	protoconfservice.RegisterProtoconfServiceServer(rpcServer, agentServer)

	err = rpcServer.Serve(listener)
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
	prefix string
}

func (s server) SubscribeForConfig(request *protoconfservice.ConfigSubscriptionRequest, srv protoconfservice.ProtoconfService_SubscribeForConfigServer) error {
	path := fmt.Sprintf("%s%s", s.prefix, request.GetPath())
	log.Printf("Watching path=%s", path)

	stopCh := make(chan struct{})
	watchCh, err := s.watcher.Watch(path, stopCh)
	if err != nil {
		return err
	}

	defer func() {
		stopCh <- struct{}{}
		close(stopCh)
	}()

	ctx := srv.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case config, ok := <-watchCh:
			if !ok {
				log.Printf("Watch channel closed for path=%s", path)
				return errors.New("watch channel closed")
			}

			if config.Error != nil {
				log.Printf("Error watching config, path=%s err=%v", path, config.Error)
				return config.Error
			}

			resp := protoconfservice.ConfigUpdate{Value: config.Value}
			if err := srv.Send(&resp); err != nil {
				log.Printf("Error sending config update, path=%s srv=%v err=%v", path, srv, err)
				return err
			}
		}
	}

	return nil
}
