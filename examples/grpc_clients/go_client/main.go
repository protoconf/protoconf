package main

import (
	"io"
	"log"
	"os"

	pc "github.com/protoconf/protoconf/agent/api/proto/v1"
	"github.com/protoconf/protoconf/consts"
	pb "github.com/protoconf/protoconf/examples/protoconf/src/crawler"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultPath = "crawler/text_crawler"
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
	address := consts.AgentDefaultAddress
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	config := &pb.CrawlerService{}
	for {
		update, err := stream.Recv()

		if err == io.EOF {
			log.Fatalf("Connection closed while streaming config path=%s", path)
		}
		if err != nil {
			log.Fatalf("Error while streaming config path=%s err=%s", path, err)
		}

		if err = anypb.UnmarshalTo(update.GetValue(), config, proto.UnmarshalOptions{}); err != nil {
			log.Fatalf("Error unmarshaling config path=%s value=%s err=%s", path, update.Value, err)
		}

		if firstRead {
			firstRead = false
			log.Printf("Config %s initial value: %s", path, config)
		} else {
			log.Printf("Config %s changed, new value: %s", path, config)
		}
	}
}
