// Package template contains the Example store implementation.
// This package is a template only.
package otelkv

import (
	"context"

	"github.com/kvtools/valkeyrie/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StoreName the name of the store.
// TODO implement me.
// const StoreName = "dummykv"

// // registers Example to Valkeyrie.
// func init() {
// 	valkeyrie.Register(StoreName, newStore)
// }

// Config the Example configuration.
// TODO implement me.
type Config struct {
}

// Store implements the store.Store interface.
// TODO implement me.
type Store struct {
	tracer trace.Tracer
	next   store.Store
}

// New creates a new Example client.
// TODO implement me.
func New(ctx context.Context, next store.Store) *Store {
	tracer := otel.Tracer("valkeyrie")
	return &Store{
		tracer: tracer,
		next:   next,
	}
}

// Put a value at the specified key.
func (s *Store) Put(ctx context.Context, key string, value []byte, opts *store.WriteOptions) error {
	ctx, span := s.tracer.Start(ctx, "kvstore.Put", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.Put(ctx, key, value, opts)
}

// Get a value given its key.
func (s Store) Get(ctx context.Context, key string, opts *store.ReadOptions) (*store.KVPair, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.Get", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.Get(ctx, key, opts)
}

// Delete the value at the specified key.
func (s Store) Delete(ctx context.Context, key string) error {
	ctx, span := s.tracer.Start(ctx, "kvstore.Delete", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.Delete(ctx, key)
}

// Exists Verify if a Key exists in the store.
func (s Store) Exists(ctx context.Context, key string, opts *store.ReadOptions) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.Exists", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.Exists(ctx, key, opts)
}

// Watch for changes on a key.
func (s Store) Watch(ctx context.Context, key string, opts *store.ReadOptions) (<-chan *store.KVPair, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.Watch", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.Watch(ctx, key, opts)
}

// WatchTree watches for changes on child nodes under a given directory.
func (s Store) WatchTree(ctx context.Context, directory string, opts *store.ReadOptions) (<-chan []*store.KVPair, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.WatchTree", trace.WithAttributes(attribute.String("directory", directory)))
	defer span.End()
	return s.next.WatchTree(ctx, directory, opts)
}

// NewLock creates a lock for a given key.
// The returned Locker is not held and must be acquired with `.Lock`.
// The Value is optional.
func (s Store) NewLock(ctx context.Context, key string, opts *store.LockOptions) (store.Locker, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.NewLock", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.NewLock(ctx, key, opts)
}

// List the content of a given prefix.
func (s Store) List(ctx context.Context, directory string, opts *store.ReadOptions) ([]*store.KVPair, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.List", trace.WithAttributes(attribute.String("directory", directory)))
	defer span.End()
	return s.next.List(ctx, directory, opts)
}

// DeleteTree deletes a range of keys under a given directory.
func (s Store) DeleteTree(ctx context.Context, directory string) error {
	ctx, span := s.tracer.Start(ctx, "kvstore.DeleteTree", trace.WithAttributes(attribute.String("directory", directory)))
	defer span.End()
	return s.next.DeleteTree(ctx, directory)
}

// AtomicPut Atomic CAS operation on a single value.
// Pass previous = nil to create a new key.
func (s Store) AtomicPut(ctx context.Context, key string, value []byte, previous *store.KVPair, opts *store.WriteOptions) (bool, *store.KVPair, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.AtomicPut", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.AtomicPut(ctx, key, value, previous, opts)
}

// AtomicDelete Atomic delete of a single value.
func (s Store) AtomicDelete(ctx context.Context, key string, previous *store.KVPair) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kvstore.AtomicDelete", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()
	return s.next.AtomicDelete(ctx, key, previous)
}

// Close the store connection.
func (s Store) Close() error {
	_, span := s.tracer.Start(context.Background(), "kvstore.Close")
	defer span.End()
	return s.next.Close()
}
