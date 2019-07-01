package main

import (
	"errors"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	protoconfservice "protoconf.com/agent/api/proto/v1/protoconfservice"
	"protoconf.com/consts"
	"protoconf.com/libprotoconf"
)

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

func main() {
	agentServer := &server{}
	var err error
	if len(os.Args) > 1 {
		agentServer.watcher, err = libprotoconf.NewFileWatcher(os.Args[1])
	} else {
		agentServer.watcher, err = libprotoconf.NewConsulWatcher()
	}

	if err != nil {
		log.Fatalf("Error setting up protoconf err=%s", err)
	}

	defer agentServer.watcher.Close()

	address := consts.DefaultAgentAddress
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error listening on address=%s err=%s", address, err)
	}

	rpcServer := grpc.NewServer()
	protoconfservice.RegisterProtoconfServiceServer(rpcServer, agentServer)

	err = rpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Error serving gRPC, err=%s", err)
	}
}
