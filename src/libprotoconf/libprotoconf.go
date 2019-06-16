package libprotoconf

import (
	"fmt"
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

// Result of the Watch operation or erro
type Result struct {
	Value *any.Any
	Error error
}

// Watch a value given its path
func Watch(path string) <-chan Result {
	watchCh := make(chan Result)

	go func() {
		defer close(watchCh)
		// FIXME: reimplement Watch with Get to deal with missing keys
		kVWatchCh, err := kv.Watch(path, nil)
		if err != nil {
			watchCh <- Result{nil, err}
			return
		}

		config := &pb.ProtoconfValue{}
		for {
			kVPair := <-kVWatchCh
			if kVPair == nil {
				watchCh <- Result{nil, fmt.Errorf("error reading path %s", path)}
				return
			}

			if err = proto.Unmarshal(kVPair.Value, config); err != nil {
				watchCh <- Result{nil, fmt.Errorf("error unmarshaling config path=%s value=%v err=%v", path, kVPair.Value, err)}
				return
			}

			watchCh <- Result{config.GetValue(), nil}
			log.Println(config)
		}
	}()

	return watchCh
}

// Setup the kv backend connection
func Setup() error {
	consul.Register()
	var err error
	kv, err = libkv.NewStore(
		store.CONSUL,
		[]string{""},
		nil,
	)
	return err
}
