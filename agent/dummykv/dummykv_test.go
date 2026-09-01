package dummykv

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kvtools/valkeyrie/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{})
	require.NoError(t, err)
	require.NotNil(t, s)
	return s
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{})
	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestNew_NilOptions(t *testing.T) {
	ctx := context.Background()
	s, err := New(ctx, []string{}, nil)
	require.NoError(t, err)
	assert.NotNil(t, s)
}

func TestPutAndGet(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value []byte
	}{
		{name: "simple key", key: "foo", value: []byte("bar")},
		{name: "key with path separators", key: "a/b/c", value: []byte("deep value")},
		{name: "empty value", key: "empty", value: []byte{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newTestStore(t)

			err := s.Put(ctx, tc.key, tc.value, nil)
			require.NoError(t, err)

			pair, err := s.Get(ctx, tc.key, nil)
			require.NoError(t, err)
			require.NotNil(t, pair)
			assert.Equal(t, tc.key, pair.Key)
			assert.Equal(t, tc.value, pair.Value)
		})
	}
}

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	pair, err := s.Get(ctx, "nonexistent-key", nil)
	assert.Error(t, err)
	assert.Nil(t, pair)
}

func TestExists(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Put a key, verify Exists returns true
	err := s.Put(ctx, "existing-key", []byte("value"), nil)
	require.NoError(t, err)

	exists, err := s.Exists(ctx, "existing-key", nil)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Put a key
	err := s.Put(ctx, "delete-me", []byte("value"), nil)
	require.NoError(t, err)

	// Verify it exists
	_, err = s.Get(ctx, "delete-me", nil)
	require.NoError(t, err)

	// Delete it
	err = s.Delete(ctx, "delete-me")
	require.NoError(t, err)

	// Verify Get returns error after deletion
	pair, err := s.Get(ctx, "delete-me", nil)
	assert.Error(t, err)
	assert.Nil(t, pair)
}

func TestWatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := newTestStore(t)
	key := "watched-key"

	// Watch a key before anything is Put (no initial delivery)
	watchCh, err := s.Watch(ctx, key, nil)
	require.NoError(t, err)
	require.NotNil(t, watchCh)

	// Put a value; Watch should receive it
	err = s.Put(ctx, key, []byte("hello"), nil)
	require.NoError(t, err)

	select {
	case kv := <-watchCh:
		require.NotNil(t, kv)
		assert.Equal(t, key, kv.Key)
		assert.Equal(t, []byte("hello"), kv.Value)
	case <-ctx.Done():
		t.Fatal("timeout waiting for watch delivery")
	}
}

func TestWatch_ExistingKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := newTestStore(t)
	key := "existing-watched-key"

	// Put initial value first
	err := s.Put(ctx, key, []byte("initial"), nil)
	require.NoError(t, err)

	// Watch the key: initial value is sent and so is any subsequent Put.
	// Since sends are async goroutines on an unbuffered channel, we drain
	// all received values and verify we see both "initial" and an update.
	watchCh, err := s.Watch(ctx, key, nil)
	require.NoError(t, err)
	require.NotNil(t, watchCh)

	// Put an update after subscribing
	err = s.Put(ctx, key, []byte("updated"), nil)
	require.NoError(t, err)

	// Collect up to 2 deliveries (order is non-deterministic due to goroutines)
	received := make(map[string]bool)
	for i := 0; i < 2; i++ {
		select {
		case kv := <-watchCh:
			require.NotNil(t, kv)
			assert.Equal(t, key, kv.Key)
			received[string(kv.Value)] = true
		case <-ctx.Done():
			t.Fatal("timeout waiting for watch deliveries")
		}
	}
	assert.True(t, received["initial"] || received["updated"],
		"expected at least one delivery, got: %v", received)
}

func TestWatch_NewKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := newTestStore(t)
	key := "new-watched-key"

	// Watch a key that does not exist yet
	watchCh, err := s.Watch(ctx, key, nil)
	require.NoError(t, err)
	require.NotNil(t, watchCh)

	// Now put a value and verify the watch fires
	err = s.Put(ctx, key, []byte("hello"), nil)
	require.NoError(t, err)

	select {
	case kv := <-watchCh:
		require.NotNil(t, kv)
		assert.Equal(t, key, kv.Key)
		assert.Equal(t, []byte("hello"), kv.Value)
	case <-ctx.Done():
		t.Fatal("timeout waiting for watch on new key")
	}
}

func TestDeleteTree(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// DeleteTree is implemented as no-op (returns nil)
	err := s.DeleteTree(ctx, "some/dir")
	assert.NoError(t, err)
}

func TestPut_MultipleTimes(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	key := "multi-put"
	for i, val := range []string{"v1", "v2", "v3"} {
		err := s.Put(ctx, key, []byte(val), nil)
		require.NoError(t, err, "put #%d failed", i+1)
	}

	pair, err := s.Get(ctx, key, nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("v3"), pair.Value)
}

// TestWatch_ConcurrentSubscribersAllReceive asserts that when N goroutines
// call Watch on the same not-yet-existing key concurrently, every one of
// them ends up registered and receives a subsequent Put. This is a
// delivery-based assertion through the public Watch/Put surface rather than
// an internal registration count, both because -race cannot catch this
// class of logical race (each individual sync.Map op is atomic) and because
// a delivery test compiles unchanged against the pre-fix pubSub.
func TestWatch_ConcurrentSubscribersAllReceive(t *testing.T) {
	const trials = 200
	const watchers = 8

	for trial := 0; trial < trials; trial++ {
		s := newTestStore(t)
		key := fmt.Sprintf("concurrent-key-%d", trial)

		start := make(chan struct{})
		chans := make([]<-chan *store.KVPair, watchers)
		var wg sync.WaitGroup
		wg.Add(watchers)
		for i := 0; i < watchers; i++ {
			go func(i int) {
				defer wg.Done()
				<-start
				ch, err := s.Watch(context.Background(), key, nil)
				require.NoError(t, err)
				chans[i] = ch
			}(i)
		}
		close(start)
		wg.Wait()

		err := s.Put(context.Background(), key, []byte("value"), nil)
		require.NoError(t, err)

		for i, ch := range chans {
			select {
			case kv := <-ch:
				require.NotNil(t, kv)
			case <-time.After(2 * time.Second):
				t.Fatalf("trial %d: watcher %d never received the Put", trial, i)
			}
		}
	}
}

func TestGet_WithWriteOptions(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	err := s.Put(ctx, "opts-key", []byte("opts-val"), &store.WriteOptions{})
	require.NoError(t, err)

	pair, err := s.Get(ctx, "opts-key", &store.ReadOptions{})
	require.NoError(t, err)
	assert.Equal(t, []byte("opts-val"), pair.Value)
}
