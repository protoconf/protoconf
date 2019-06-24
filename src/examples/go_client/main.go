package main

import (
	"io"
	"log"
	"os"

	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pc "protoconf.com/agent/api/proto/v1/protoconfservice"
	pb "protoconf.com/examples/go_client/addressbook"
)

const (
	address     = ":4300"
	defaultPath = "address_book/my_address_book"
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
	stream, err := c.SubscribeForConfig(context.Background(), &pc.ConfigSubscriptionRequest{Path: path})
	if err != nil {
		log.Fatalf("Error subscribing for config path=%s err=%v", path, err)
	}

	firstRead := true
	config := &pb.AddressBook{}
	for {
		update, err := stream.Recv()

		if err == io.EOF {
			log.Fatalf("Connection closed while streaming config path=%s", path)
		}
		if err != nil {
			log.Fatalf("Error while streaming config path=%s err=%v", path, err)
		}

		if err = ptypes.UnmarshalAny(update.GetValue(), config); err != nil {
			log.Fatalf("Error unmarshaling config path=%s value=%v err=%v", path, update.Value, err)
		}

		if firstRead {
			firstRead = false
			log.Printf("Config %s initial value: %s", path, config)
		} else {
			log.Printf("Config %s changed, new value: %s", path, config)
		}
	}
}
