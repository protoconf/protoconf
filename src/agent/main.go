package main

import (
	"log"
	"net"
	"time"

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
	log.Printf("path=%s", path)
	for {
		config, err := libprotoconf.Get(path)
		if err != nil {
			log.Printf("Error reading config, key=%s err=%v", path, err)
			return err
		}
		resp := pc.ConfigUpdate{Value: config}
		if err := srv.Send(&resp); err != nil {
			log.Printf("Error sending config update, path=%s srv=%v err=%v", path, srv, err)
			return err
		}
		time.Sleep(4 * time.Second)
	}
	return nil
}

func main() {
	libprotoconf.Setup()

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
