package main

import (
	"log"
	"os"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	pb "protoconf.com/examples/agent_client/myconfig"
	pc "protoconf.com/types/proto/v1/protoconfvalue"
)

const (
	defaultPath = "example/consul/path"
)

func main() {
	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		path = defaultPath
	}

	var value string
	if len(os.Args) > 2 {
		value = os.Args[2]
	} else {
		value = time.Now().String()
	}

	myConfig := &pb.MyConfig{Value: value}
	any, err := ptypes.MarshalAny(myConfig)
	if err != nil {
		log.Fatalf("Error marshaling MyConfig to Any, config=%s", myConfig)
	}

	protoconfValue := &pc.ProtoconfValue{Value: any}

	m := &jsonpb.Marshaler{}
	str, err := m.MarshalToString(protoconfValue)
	if err != nil {
		log.Fatalf("Error marshaling ProtoconfValue to JSON, value=%v", protoconfValue)
	}

	log.Printf("Writing to consul, path=%s json-value=%s", path, str)

	consul.Register()
	kv, err := libkv.NewStore(
		store.CONSUL,
		[]string{""},
		nil,
	)

	data, err := proto.Marshal(protoconfValue)
	if err != nil {
		log.Fatalf("Error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}

	if err := kv.Put(path, data, nil); err != nil {
		log.Fatalf("Error writing to consul, path=%s json-value=%s", path, str)
	}
}
