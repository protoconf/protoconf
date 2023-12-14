// Package template contains the Example store implementation.
// This package is a template only.
package filekv

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/protoconf/protoconf/compiler/lib"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/consts"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/proto"
)

// StoreName the name of the store.
// TODO implement me.
const StoreName = "filekv"

// registers Example to Valkeyrie.
func init() {
	valkeyrie.Register(StoreName, newStore)
}

// Config the Example configuration.
// TODO implement me.
type Config struct {
	ProtoconfRoot string
}

func newStore(ctx context.Context, endpoints []string, options valkeyrie.Config) (store.Store, error) {
	cfg, ok := options.(*Config)
	if !ok && options != nil {
		return nil, &store.InvalidConfigurationError{Store: StoreName, Config: options}
	}

	return New(ctx, endpoints, cfg)
}

// Store implements the store.Store interface.
// TODO implement me.
type Store struct {
	fsnotifyWatcher *fsnotify.Watcher
	protoconfRoot   string
	watches         map[string]([]chan struct{})
	lock            sync.Mutex
	parser          *parser.Parser
}

// New creates a new Example client.
// TODO implement me.
func New(ctx context.Context, endpoints []string, options *Config) (*Store, error) {
	fsnotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(options.ProtoconfRoot)
	if err != nil {
		return nil, err
	}
	ms := lib.NewModuleService(absRoot)
	ms.LoadFromLockFile()

	watcher := &Store{
		fsnotifyWatcher: fsnotifyWatcher,
		protoconfRoot:   absRoot,
		watches:         make(map[string]([]chan struct{})),
		parser:          parser.NewParser(ms.GetProtoFilesRegistry()),
	}

	go watcher.readEvents()

	return watcher, nil
}

// Put a value at the specified key.
func (s Store) Put(ctx context.Context, key string, value []byte, opts *store.WriteOptions) error {
	return nil

}

// Get a value given its key.
func (s Store) Get(ctx context.Context, key string, opts *store.ReadOptions) (*store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// Delete the value at the specified key.
func (s Store) Delete(ctx context.Context, key string) error {
	// TODO implement me
	panic("implement me")
}

// Exists Verify if a Key exists in the store.
func (s Store) Exists(ctx context.Context, key string, opts *store.ReadOptions) (bool, error) {
	return true, nil
}

// Watch for changes on a key.
func (s *Store) Watch(ctx context.Context, key string, opts *store.ReadOptions) (<-chan *store.KVPair, error) {
	// TODO implement me
	if key != filepath.ToSlash(filepath.Clean(key)) || key == "" {
		return nil, fmt.Errorf("invalid path to watch, path=%s", key)
	}

	absPath := filepath.Join(s.protoconfRoot, key+consts.CompiledConfigExtension)
	fsCh := make(chan struct{})
	if err := s.addWatch(absPath, fsCh); err != nil {
		return nil, err
	}

	watchCh := make(chan *store.KVPair)
	go func() {
		defer func() {
			close(watchCh)
			_ = s.removeWatch(absPath, fsCh)
		}()

		for {
			protoconfValue := &protoconfvalue.ProtoconfValue{}
			err := s.parser.ReadConfig(absPath, protoconfValue)
			if err != nil {
				log.Println(err)
				return
			}
			b, err := proto.Marshal(protoconfValue)
			if err != nil {
				log.Println(err)
				return
			}
			watchCh <- &store.KVPair{
				Key:   key,
				Value: []byte(base64.StdEncoding.EncodeToString(b)),
			}
			if err != nil {
				return
			}

			select {
			case _, ok := <-fsCh:
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return watchCh, nil
}

// WatchTree watches for changes on child nodes under a given directory.
func (s Store) WatchTree(ctx context.Context, directory string, opts *store.ReadOptions) (<-chan []*store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// NewLock creates a lock for a given key.
// The returned Locker is not held and must be acquired with `.Lock`.
// The Value is optional.
func (s Store) NewLock(ctx context.Context, key string, opts *store.LockOptions) (store.Locker, error) {
	// TODO implement me
	panic("implement me")
}

// List the content of a given prefix.
func (s Store) List(ctx context.Context, directory string, opts *store.ReadOptions) ([]*store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// DeleteTree deletes a range of keys under a given directory.
func (s Store) DeleteTree(ctx context.Context, directory string) error {
	// TODO implement me
	panic("implement me")
}

// AtomicPut Atomic CAS operation on a single value.
// Pass previous = nil to create a new key.
func (s Store) AtomicPut(ctx context.Context, key string, value []byte, previous *store.KVPair, opts *store.WriteOptions) (bool, *store.KVPair, error) {
	// TODO implement me
	panic("implement me")
}

// AtomicDelete Atomic delete of a single value.
func (s Store) AtomicDelete(ctx context.Context, key string, previous *store.KVPair) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Store) addWatch(path string, ch chan struct{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.watches[path] = append(w.watches[path], ch)
	if err := w.fsnotifyWatcher.Add(path); err != nil {
		w.watches[path] = removeChannel(ch, w.watches[path])
		return fmt.Errorf("error fs watching path %s, err=%s", path, err)
	}

	return nil
}

func (w *Store) removeWatch(path string, ch chan struct{}) error {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.watches[path] = removeChannel(ch, w.watches[path])
	if len(w.watches[path]) == 0 {
		return w.fsnotifyWatcher.Remove(path)
	}

	return nil
}

func removeChannel(ch chan struct{}, chs []chan struct{}) []chan struct{} {
	for idx, val := range chs {
		if val == ch {
			chs[idx] = chs[len(chs)-1]
			chs[len(chs)-1] = nil
			return chs[:len(chs)-1]
		}
	}
	return chs
}

func (w *Store) readEvents() {
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

func (w *Store) closeWatchers() {
	for _, pathWatches := range w.watches {
		for _, watch := range pathWatches {
			close(watch)
		}
	}
	w.watches = nil
}

func (w *Store) Close() error {
	w.closeWatchers()
	return w.fsnotifyWatcher.Close()
}
