// Package template contains the Example store implementation.
// This package is a template only.
package dummykv

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
)

// StoreName the name of the store.
// TODO implement me.
const StoreName = "dummykv"

// registers Example to Valkeyrie.
func init() {
	valkeyrie.Register(StoreName, newStore)
}

// Config the Example configuration.
// TODO implement me.
type Config struct {
}

func newStore(ctx context.Context, endpoints []string, options valkeyrie.Config) (store.Store, error) {
	cfg, ok := options.(*Config)
	if !ok && options != nil {
		return nil, &store.InvalidConfigurationError{Store: StoreName, Config: options}
	}

	return New(ctx, endpoints, cfg)
}

type pubSub struct {
	topics *sync.Map
}

func newPubSub() *pubSub {
	return &pubSub{
		topics: &sync.Map{},
	}
}

func (p *pubSub) Publish(kv *store.KVPair) {
	result, ok := p.topics.Load(kv.Key)
	if ok {
		switch x := result.(type) {
		case []chan *store.KVPair:
			for _, ch := range x {
				go func(ch chan *store.KVPair) {
					ch <- kv
				}(ch)
			}
		}
	}
}

func (p *pubSub) Channel(key string) chan *store.KVPair {
	ch := make(chan *store.KVPair)
	arr := []chan *store.KVPair{ch}
	result, ok := p.topics.Load(key)
	if ok {
		switch x := result.(type) {
		case []chan *store.KVPair:
			arr = append(arr, x...)
		}
	}
	p.topics.Store(key, arr)
	return ch
}

// Store implements the store.Store interface.
// TODO implement me.
type Store struct {
	channels *pubSub
	store    *sync.Map
	mux      *sync.RWMutex
}

// New creates a new Example client.
// TODO implement me.
func New(ctx context.Context, endpoints []string, options *Config) (*Store, error) {
	return &Store{
		channels: newPubSub(),
		store:    &sync.Map{},
		mux:      &sync.RWMutex{},
	}, nil
}

// Put a value at the specified key.
func (s *Store) Put(ctx context.Context, key string, value []byte, opts *store.WriteOptions) error {
	kv := &store.KVPair{Key: key, Value: value}
	s.store.Store(key, kv)
	go s.channels.Publish(kv)
	return nil
}

// Get a value given its key.
func (s Store) Get(ctx context.Context, key string, opts *store.ReadOptions) (*store.KVPair, error) {
	if val, ok := s.store.Load(key); ok {
		switch x := val.(type) {
		case *store.KVPair:
			return x, nil
		}
	}
	return nil, fmt.Errorf("could not find key %s", key)
}

// Delete the value at the specified key.
func (s Store) Delete(ctx context.Context, key string) error {
	s.store.Delete(key)
	return nil
}

// Exists Verify if a Key exists in the store.
func (s Store) Exists(ctx context.Context, key string, opts *store.ReadOptions) (bool, error) {
	// TODO implement me
	return true, nil
}

// Watch for changes on a key.
func (s Store) Watch(ctx context.Context, key string, opts *store.ReadOptions) (<-chan *store.KVPair, error) {
	slog.Default().Info("dummykv watch", "key", key, "ctx", ctx)
	ch := s.channels.Channel(key)

	if current, err := s.Get(ctx, key, opts); err == nil {
		go func(ch chan *store.KVPair) {
			ch <- current
		}(ch)
	}

	return ch, nil
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

// Close the store connection.
func (s Store) Close() error {
	// TODO implement me
	panic("implement me")
}
