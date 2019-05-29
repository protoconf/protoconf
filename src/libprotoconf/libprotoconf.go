package libprotoconf

import (
	"log"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	pb "protoconf.com/types/proto/v1/protoconfvalue"
)

var (
	kv store.Store
)

// Get a value given its key
func Get(key string) (*any.Any, error) {
	value, err := kv.Get(key)
	if err != nil {
		log.Printf("Error getting key from the store, key=%s", key)
		return nil, err
	}
	config := &pb.ProtoconfValue{}
	if err = proto.Unmarshal(value.Value, config); err != nil {
		log.Printf("Error unmarshaling config key=%s value=%v err=%v", key, value.Value, err)
		return nil, err
	}
	return config.GetValue(), nil
}

// Setup the kv backend connection
func Setup() {
	consul.Register()
	var err error
	kv, err = libkv.NewStore(
		store.CONSUL,
		[]string{""},
		nil,
	)
	log.Println("hello from libprotoconf setup", kv, err)
}
