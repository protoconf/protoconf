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
	defaultKey = "example/consul/path"
)

func main() {
	var key string
	if len(os.Args) > 1 {
		key = os.Args[1]
	} else {
		key = defaultKey
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

	log.Printf("Writing to consul, key=%s json-value=%s", key, str)

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

	if err := kv.Put(key, data, nil); err != nil {
		log.Fatalf("Error writing to consul, key=%s json-value=%s", key, str)
	}
}
