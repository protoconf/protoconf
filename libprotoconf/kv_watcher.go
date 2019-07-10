package libprotoconf

import (
	"fmt"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/golang/protobuf/proto"
	protoconfvalue "protoconf.com/types/proto/v1/protoconfvalue"
)

// NewConsulWatcher creates a new consul-backed Protoconf watcher
func NewConsulWatcher(address string) (Watcher, error) {
	consul.Register()
	store, err := libkv.NewStore(store.CONSUL, []string{address}, nil)
	if err != nil {
		return nil, err
	}
	return &libkvWatcher{store: store}, nil
}

type libkvWatcher struct {
	store store.Store
}

// Watch a value given its path
func (w *libkvWatcher) Watch(path string, stopCh <-chan struct{}) (<-chan Result, error) {
	watchCh := make(chan Result)
	kVStopCh := make(chan struct{})

	go func() {
		defer func() {
			kVStopCh <- struct{}{}
			close(kVStopCh)
			close(watchCh)
		}()

		// FIXME: reimplement Watch with Get to deal with missing keys
		kVWatchCh, err := w.store.Watch(path, kVStopCh)
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

				if err = proto.Unmarshal(kVPair.Value, protoconfValue); err != nil {
					watchCh <- Result{nil, fmt.Errorf("error unmarshaling config path=%s value=%v err=%v", path, kVPair.Value, err)}
					return
				}

				watchCh <- Result{protoconfValue.GetValue(), nil}
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
