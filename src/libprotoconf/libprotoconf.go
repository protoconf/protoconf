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

// Watch a value given its path
func Watch(path string) (<-chan *any.Any, error) {
	watchCh := make(chan *any.Any)

	go func() {
		defer close(watchCh)
		kVWatchCh, err := kv.Watch(path, nil)
		if err != nil {
			log.Printf("Error getting path from the store, path=%s", path)
			return
		}

		config := &pb.ProtoconfValue{}
		for {
			kVPair := <-kVWatchCh
			if err = proto.Unmarshal(kVPair.Value, config); err != nil {
				log.Printf("Error unmarshaling config path=%s value=%v err=%v", path, kVPair.Value, err)
				return
			}

			watchCh <- config.GetValue()
		}
	}()

	return watchCh, nil
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
