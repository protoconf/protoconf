package main

import (
	"io"
	"log"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pc "protoconf.com/agent/api/proto/v1/protoconfservice"
)

const (
	address     = ":4300" // FIXME: can we get this from somewhere?
	defaultPath = "example/consul/path"
)

func main() {
	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		path = defaultPath
	}

	listenToChanges(path)
}

func listenToChanges(path string) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error connecting to server address=%s err=%v", address, err)
	}
	defer conn.Close()

	c := pc.NewProtoconfServiceClient(conn)

	for {
		stream, err := c.SubscribeForConfig(context.Background(), &pc.ConfigSubscriptionRequest{Path: path})
		if err != nil {
			log.Fatalf("Error subscribing for config path=%s err=%v", path, err)
		}
		update, err := stream.Recv()
		if err == io.EOF {
			log.Fatalf("Connection closed while streaming config path=%s", path)
		}
		if err != nil {
			log.Fatalf("Error while streaming config path=%s err=%v", path, err)
		}
		log.Printf("Config %s changed, new value: %s", path, update.Value)
	}
}
