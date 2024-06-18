package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/protoconf/protoconf/consts"
	pv "github.com/protoconf/protoconf/datatypes/proto/v1"
	pb "github.com/protoconf/protoconf/examples/protoconf/src/crawler"
	pc "github.com/protoconf/protoconf/server/api/proto/v1"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultPath      = "crawler/extra_crawler"
	defaultUserAgent = "Linux/ time=%s"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s script_metadata [path] [user_agent]", os.Args[0])
	}

	scriptMetadata := os.Args[1]

	var path string
	if len(os.Args) > 2 {
		path = os.Args[2]
	} else {
		path = defaultPath
	}

	var userAgent string
	if len(os.Args) > 3 {
		userAgent = os.Args[3]
	} else {
		userAgent = fmt.Sprintf(defaultUserAgent, strconv.FormatInt(time.Now().Unix(), 10))
	}

	if err := mutate(path, &pb.Crawler{HttpTimeout: 30, UserAgent: userAgent}, scriptMetadata); err != nil {
		slog.Error("error", err)
		os.Exit(1)
	}

	slog.Info("Mutated successfully", "path", path)
}

func mutate(path string, value proto.Message, scriptMetadata string) error {
	any, err := anypb.New(value)
	if err != nil {
		return fmt.Errorf("error marshalling message to any message=%s err=%s", value, err)
	}
	config := &pv.ProtoconfValue{Value: any}
	request := &pc.ConfigMutationRequest{Path: path, Value: config, ScriptMetadata: scriptMetadata}

	address := consts.ServerDefaultAddress
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("error connecting to server address=%s err=%s", address, err)
	}
	defer conn.Close()

	c := pc.NewProtoconfMutationServiceClient(conn)

	// Wait until the server finishes long git operations
	timeout := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err := c.MutateConfig(ctx, request); err != nil {
		return fmt.Errorf("error mutating path=%s err=%s", path, err)
	}

	return nil
}
