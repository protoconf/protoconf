package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	pc "protoconf.com/agent/api/proto/v1/protoconfservice"
	"protoconf.com/libprotoconf"
)

const (
	address = ":4300"
)

type server struct{}

func (s server) SubscribeForConfig(request *pc.ConfigSubscriptionRequest, srv pc.ProtoconfService_SubscribeForConfigServer) error {
	path := request.GetPath()
	log.Printf("Watching path=%s", path)

	watchCh := libprotoconf.Watch(path)

	for {
		config := <-watchCh
		if config.Error != nil {
			log.Printf("Error watching config, path=%s err=%v", path, config.Error)
			return config.Error
		}

		resp := pc.ConfigUpdate{Value: config.Value}
		if err := srv.Send(&resp); err != nil {
			log.Printf("Error sending config update, path=%s srv=%v err=%v", path, srv, err)
			return err
		}
	}

	return nil
}

func main() {
	err := libprotoconf.Setup()
	if err != nil {
		log.Fatalf("Error setting up protoconf err=%v", err)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error listening on address=%s err=%v", address, err)
	}

	s := grpc.NewServer()
	pc.RegisterProtoconfServiceServer(s, server{})

	if err := s.Serve(listener); err != nil {
		log.Fatalf("Error serving, err=%v", err)
	}
}
