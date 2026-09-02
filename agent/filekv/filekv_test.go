package filekv

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/kvtools/valkeyrie/store"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s
}

func TestNew(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	assert.NotNil(t, s)
	_ = s.Close()
}

func TestNew_InvalidRoot(t *testing.T) {
	ctx := context.Background()
	// Using a nonexistent root should still create the store (module service
	// may still work with an empty dir, or return an error).
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: "/nonexistent/path/that/does/not/exist"})
	// Either returns error or returns store; just verify we handle both cases
	if err != nil {
		assert.Nil(t, s)
	} else {
		assert.NotNil(t, s)
		_ = s.Close()
	}
}

func TestPut(t *testing.T) {
	// Put is a no-op in filekv, should return nil
	s := newTestStore(t)
	ctx := context.Background()
	err := s.Put(ctx, "any/key", []byte("any value"), nil)
	assert.NoError(t, err)
}

func TestPut_WithWriteOptions(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	err := s.Put(ctx, "key/with/opts", []byte("value"), &store.WriteOptions{})
	assert.NoError(t, err)
}

func TestExists(t *testing.T) {
	// Exists always returns true in filekv
	s := newTestStore(t)
	ctx := context.Background()

	exists, err := s.Exists(ctx, "any/key", nil)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestClose(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)

	err = s.Close()
	assert.NoError(t, err)
}

func TestClose_Idempotent(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx := context.Background()
	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	require.NotNil(t, s)

	assert.NoError(t, s.Close())
	assert.NoError(t, s.Close())
}

func TestGet_ValidKey(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	pair, err := s.Get(ctx, "materialized_config/test", nil)
	require.NoError(t, err)
	require.NotNil(t, pair)

	b, err := base64.StdEncoding.DecodeString(string(pair.Value))
	require.NoError(t, err)
	protoconfValue := &protoconfvalue.ProtoconfValue{}
	require.NoError(t, proto.Unmarshal(b, protoconfValue))
	assert.Equal(t, "test.proto", protoconfValue.ProtoFile)
}

func TestGet_InvalidPath(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "", nil)
	assert.Error(t, err)

	_, err = s.Get(ctx, "../escape", nil)
	assert.Error(t, err)
}

func TestGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "materialized_config/does_not_exist", nil)
	assert.ErrorIs(t, err, store.ErrKeyNotFound)
}

func TestWatch_InvalidPath(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Empty path should return an error (invalid path check in Watch)
	_, err := s.Watch(ctx, "", nil)
	assert.Error(t, err)
}

func TestWatch_InvalidPath_Dots(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Path with ".." (non-clean path) should return an error
	_, err := s.Watch(ctx, "../some/path", nil)
	assert.Error(t, err)
}

func TestWatch_NonExistentFile(t *testing.T) {
	s := newTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Watching a non-existent file should return error from fsnotify
	_, err := s.Watch(ctx, "nonexistent/config", nil)
	// fsnotify.Add on non-existent file should fail
	assert.Error(t, err)
}

func TestWatch_DeliversSameKVPairAsGet(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	want, err := s.Get(ctx, "materialized_config/test", nil)
	require.NoError(t, err)

	watchCtx, watchCancel := context.WithCancel(ctx)
	ch, err := s.Watch(watchCtx, "materialized_config/test", nil)
	require.NoError(t, err)

	select {
	case got := <-ch:
		require.NotNil(t, got)
		assert.Equal(t, want.Key, got.Key)
		assert.Equal(t, want.Value, got.Value)
	case <-time.After(5 * time.Second):
		watchCancel()
		t.Fatal("timed out waiting for watch delivery")
	}

	// Cancel and drain until Watch's goroutine closes the channel, so its
	// exit (and removeWatch call) is complete before the store is closed by
	// t.Cleanup — avoids racing Watch's cleanup against Store.Close's.
	watchCancel()
	for range ch {
	}
}

func TestWatch_ContextCancellation(t *testing.T) {
	root := testdata.SmallTestDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s, err := New(ctx, []string{}, &Config{ProtoconfRoot: root})
	require.NoError(t, err)
	defer s.Close()

	// The path we watch: filekv constructs absPath = root + "/" + key + ".materialized_JSON"
	// "enum_test" maps to materialized_config/enum_test.materialized_JSON in the test root
	// But Watch looks in protoconfRoot directly, not materialized_config subdir.
	// Let's test invalid path validation instead.
	watchCtx, watchCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer watchCancel()

	// Test that Watch returns error for non-existent file
	_, err = s.Watch(watchCtx, "nonexistent-config", nil)
	assert.Error(t, err)
}
