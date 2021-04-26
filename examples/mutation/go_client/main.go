package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/protoconf/protoconf/consts"
	pv "github.com/protoconf/protoconf/datatypes/proto/v1"
	pb "github.com/protoconf/protoconf/examples/protoconf/src/crawler"
	pc "github.com/protoconf/protoconf/server/api/proto/v1"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
		log.Fatalf("%s", err)
	}

	log.Printf("Mutated %s successfully", path)
}

func mutate(path string, value descriptor.Message, scriptMetadata string) error {
	fileDesc, _ := descriptor.ForMessage(value)
	any, err := ptypes.MarshalAny(value)
	if err != nil {
		return fmt.Errorf("error marshalling message to any message=%s err=%s", value, err)
	}
	config := &pv.ProtoconfValue{ProtoFile: fileDesc.GetName(), Value: any}
	request := &pc.ConfigMutationRequest{Path: path, Value: config, ScriptMetadata: scriptMetadata}

	address := consts.ServerDefaultAddress
	conn, err := grpc.Dial(address, grpc.WithInsecure())
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
