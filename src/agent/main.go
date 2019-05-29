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
	log.Printf("path=%s", request.GetPath())
	time.Sleep(4 * time.Second)
	resp := pc.ConfigUpdate{Value: libprotoconf.Get(request.GetPath())}
	if err := srv.Send(&resp); err != nil {
		log.Printf("send error err=%v", err) // FIXME: better error
		return err
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
