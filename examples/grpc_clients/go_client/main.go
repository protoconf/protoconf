package main

import (
	"io"
	"log/slog"
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
		slog.Error("Error connecting to server ", "address", address, "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	c := pc.NewProtoconfServiceClient(conn)
	stream, err := c.SubscribeForConfig(context.Background(), &pc.ConfigSubscriptionRequest{Path: path})
	if err != nil {
		slog.Error("Error subscribing for config", "path", path, "error", err)
		os.Exit(1)
	}

	firstRead := true
	config := &pb.CrawlerService{}
	for {
		update, err := stream.Recv()

		if err == io.EOF {
			slog.Error("Connection closed while streaming config", "path", path)
			os.Exit(1)
		}
		if err != nil {
			slog.Error("Error while streaming config", "path", path, "error", err)
			os.Exit(1)
		}

		if err = anypb.UnmarshalTo(update.GetValue(), config, proto.UnmarshalOptions{}); err != nil {
			slog.Error("Error unmarshaling config", "path", path, "value", update.Value, "error", err)
			os.Exit(1)
		}

		if firstRead {
			firstRead = false
			slog.Info("Config initial value", "path", path, "value", config)
		} else {
			slog.Info("Config changed, new value:", "path", path, "value", config)
		}
	}
}
