package libprotoconf

import (
	"encoding/base64"
	"fmt"

	"github.com/abronan/valkeyrie"
	"github.com/abronan/valkeyrie/store"
	"github.com/abronan/valkeyrie/store/consul"
	etcd "github.com/abronan/valkeyrie/store/etcd/v2"
	"github.com/abronan/valkeyrie/store/zookeeper"
	"github.com/golang/protobuf/proto"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
)

type KVStore int

const (
	Consul KVStore = iota
	Zookeeper
	Etcd
)

// NewWatcher creates a new kv-backed Protoconf watcher
func NewKVWatcher(kvType KVStore, address string, prefix string) (Watcher, error) {
	var backend store.Backend
	switch kvType {
	case Consul:
		consul.Register()
		backend = store.CONSUL
	case Zookeeper:
		zookeeper.Register()
		backend = store.ZK
	case Etcd:
		etcd.Register()
		backend = store.ETCD
	default:
		return nil, fmt.Errorf("unknown kvType=%d", kvType)
	}

	store, err := valkeyrie.NewStore(backend, []string{address}, nil)
	if err != nil {
		return nil, err
	}
	return &libkvWatcher{
		prefix: prefix,
		store:  store,
	}, nil
}

type libkvWatcher struct {
	prefix string
	store  store.Store
}

// Watch a value given its path
func (w *libkvWatcher) Watch(pathNoPrefix string, stopCh <-chan struct{}) (<-chan Result, error) {
	path := w.prefix + pathNoPrefix

	watchCh := make(chan Result)
	kVStopCh := make(chan struct{})

	go func() {
		defer func() {
			select {
			case kVStopCh <- struct{}{}:
			default:
			}
			close(kVStopCh)
			close(watchCh)
		}()

		kVWatchCh, err := w.store.Watch(path, kVStopCh, &store.ReadOptions{})
		if err != nil {
			watchCh <- Result{nil, err}
			return
		}

		protoconfValue := &protoconfvalue.ProtoconfValue{}
		for {
			select {
			case kVPair, ok := <-kVWatchCh:
				if !ok {
					return
				}

				if kVPair == nil {
					watchCh <- Result{nil, fmt.Errorf("error reading path %s", path)}
					return
				}

				data, err := base64.StdEncoding.DecodeString(string(kVPair.Value))
				if err != nil {
					watchCh <- Result{nil, fmt.Errorf("error decoding config path=%s value=%s err=%s", path, kVPair.Value, err)}
				}
				if err = proto.Unmarshal(data, protoconfValue); err != nil {
					watchCh <- Result{nil, fmt.Errorf("error unmarshaling config path=%s value=%s err=%s", path, kVPair.Value, err)}
					return
				}

				any, err := injectSecrets(protoconfValue)
				if err != nil {
					watchCh <- Result{nil, fmt.Errorf("error injecting secrets path=%s protoconf_value=%s err=%s", path, protoconfValue, err)}
					return
				}

				watchCh <- Result{any, nil}
			case <-stopCh:
				kVStopCh <- struct{}{}
				return
			}
		}
	}()

	return watchCh, nil
}

func (w *libkvWatcher) Close() {
	w.store.Close()
}
