package libprotoconf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/fsnotify/fsnotify"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic/msgregistry"
	"protoconf.com/consts"
	protoconfvalue "protoconf.com/types/proto/v1/protoconfvalue"
)

// Watcher enables getting updates on protoconf paths
type Watcher interface {
	Watch(path string, stopCh <-chan struct{}) (<-chan Result, error)
	Close()
}

// Result of the Watch operation or error
type Result struct {
	Value *any.Any
	Error error
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

// NewConsulWatcher creates a new consul-backed protoconf watcher
func NewConsulWatcher() (Watcher, error) {
	consul.Register()
	store, err := libkv.NewStore(store.CONSUL, []string{""}, nil)
	if err != nil {
		return nil, err
	}
	return &libkvWatcher{store: store}, nil
}

type fileWatcher struct {
	fsnotifyWatcher *fsnotify.Watcher
	protoconfRoot   string
	watches         map[string]([]chan struct{})
}

// Watch a value given its path
func (w *fileWatcher) Watch(path string, stopCh <-chan struct{}) (<-chan Result, error) {
	if path != filepath.Clean(path) || path == "" {
		return nil, fmt.Errorf("invalid path to watch, path=%s", path)
	}

	absPath := filepath.Join(w.protoconfRoot, consts.CompiledConfigPath, path+consts.CompiledConfigExtension)
	fsCh := make(chan struct{})
	w.watches[absPath] = append(w.watches[absPath], fsCh)

	if err := w.fsnotifyWatcher.Add(absPath); err != nil {
		return nil, fmt.Errorf("error fs watching path %s, err=%s", path, err)
	}

	watchCh := make(chan Result)
	go func() {
		defer close(watchCh)
		for {
			protoconfValue, err := ReadConfig(w.protoconfRoot, path)
			if err != nil {
				watchCh <- Result{nil, err}
				return
			}
			watchCh <- Result{protoconfValue.GetValue(), nil}
			select {
			case _, ok := <-fsCh:
				if !ok {
					return
				}
			case <-stopCh:
				w.watches[absPath] = removeWatch(fsCh, w.watches[absPath])
				if len(w.watches[absPath]) == 0 {
					w.fsnotifyWatcher.Remove(absPath)
				}
				return
			}
		}
	}()

	return watchCh, nil
}

func removeWatch(watch chan struct{}, watches []chan struct{}) []chan struct{} {
	for idx, val := range watches {
		if val == watch {
			watches[idx] = watches[len(watches)-1]
			watches[len(watches)-1] = nil
			return watches[:len(watches)-1]
		}
	}
	return watches
}

func (w *fileWatcher) readEvents() {
	for {
		select {
		case event, ok := <-w.fsnotifyWatcher.Events:
			if !ok {
				w.closeWatchers()
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				for _, channel := range w.watches[event.Name] {
					channel <- struct{}{}
				}
			}
		case <-w.fsnotifyWatcher.Errors:
			w.closeWatchers()
		}
	}
}

func (w *fileWatcher) closeWatchers() {
	for _, pathWatches := range w.watches {
		for _, watch := range pathWatches {
			close(watch)
		}
	}
	w.watches = nil
}

func (w *fileWatcher) Close() {
	w.closeWatchers()
	w.fsnotifyWatcher.Close()
}

// NewFileWatcher creates a new file-backed protoconf watcher
func NewFileWatcher(protoconfRoot string) (Watcher, error) {
	fsnotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(protoconfRoot)
	if err != nil {
		return nil, err
	}

	watcher := &fileWatcher{
		fsnotifyWatcher: fsnotifyWatcher,
		protoconfRoot:   absRoot,
		watches:         make(map[string]([]chan struct{})),
	}

	go watcher.readEvents()

	return watcher, nil
}

// ReadConfig reads a materialized config
func ReadConfig(protoconfRoot string, configName string) (*protoconfvalue.ProtoconfValue, error) {
	filename := filepath.Join(protoconfRoot, consts.CompiledConfigPath, configName+consts.CompiledConfigExtension)

	configReader, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file, file=%s", filename)
	}
	defer configReader.Close()

	type configJSONType struct {
		ProtoFile string
	}
	var configJSON configJSONType
	if err := json.NewDecoder(configReader).Decode(&configJSON); err != nil {
		return nil, err
	}

	parser := &protoparse.Parser{ImportPaths: []string{filepath.Join(protoconfRoot, consts.ConfigPath)}}
	descriptors, err := parser.ParseFiles(configJSON.ProtoFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%v", configJSON.ProtoFile, err)
	}

	registry := msgregistry.NewMessageRegistryWithDefaults()
	registry.AddFile("", descriptors[0])

	if _, err := configReader.Seek(0, 0); err != nil {
		return nil, err
	}

	protoconfValue := &protoconfvalue.ProtoconfValue{}
	um := jsonpb.Unmarshaler{AnyResolver: registry}
	if err := um.Unmarshal(configReader, protoconfValue); err != nil {
		return protoconfValue, fmt.Errorf("error unmarshaling, err=%s", err)
	}
	return protoconfValue, nil
}
