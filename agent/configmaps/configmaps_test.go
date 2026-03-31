package configmaps

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// newTestStore creates a configmaps Store with a fake Kubernetes clientset
// injected directly, bypassing getClientset() which requires a real cluster.
func newTestStore(t *testing.T, namespace string, initialObjects ...v1.Object) *Store {
	t.Helper()
	fakeClient := fake.NewSimpleClientset()
	for _, obj := range initialObjects {
		switch o := obj.(type) {
		case *cv1.ConfigMap:
			_, err := fakeClient.CoreV1().ConfigMaps(namespace).Create(
				context.Background(), o, v1.CreateOptions{},
			)
			require.NoError(t, err)
		}
	}
	cfg := &Config{Namespace: namespace}
	return &Store{
		clientset: fakeClient,
		config:    cfg,
		mutex:     &sync.Mutex{},
		logger:    slog.Default(),
	}
}

func TestNew_WithFakeClient(t *testing.T) {
	// Verify the Store struct can be created with injected fake client
	s := newTestStore(t, "default")
	assert.NotNil(t, s)
	assert.Equal(t, "default", s.config.Namespace)
}

func TestPut_CreatesConfigMap(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "test-ns")

	// Put creates a ConfigMap under the namespace
	err := s.Put(ctx, "myapp/config", []byte("value1"), nil)
	require.NoError(t, err)

	// Verify ConfigMap was created in the fake client
	cm, err := s.clientset.CoreV1().ConfigMaps("test-ns").Get(ctx, "myapp", v1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "value1", cm.Data["config"])
}

func TestPutAndGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	key := "mydir/mykey"
	value := []byte("test-value")

	// Put value
	err := s.Put(ctx, key, value, nil)
	require.NoError(t, err)

	// Get it back
	pair, err := s.Get(ctx, key, nil)
	require.NoError(t, err)
	require.NotNil(t, pair)
	assert.Equal(t, key, pair.Key)
	assert.Equal(t, value, pair.Value)
}

func TestGet_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Get a key that was never put (ConfigMap doesn't exist)
	pair, err := s.Get(ctx, "nonexistent/key", nil)
	assert.Error(t, err)
	assert.Nil(t, pair)
}

func TestGet_KeyNotInConfigMap(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Create a ConfigMap with a different key
	err := s.Put(ctx, "dir/key1", []byte("value1"), nil)
	require.NoError(t, err)

	// Try to get a key not present in the ConfigMap data
	pair, err := s.Get(ctx, "dir/key2", nil)
	assert.Error(t, err)
	assert.Nil(t, pair)
}

func TestExists(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Exists always returns true in current implementation
	exists, err := s.Exists(ctx, "any/key", nil)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Put a key
	err := s.Put(ctx, "dir/key", []byte("to-delete"), nil)
	require.NoError(t, err)

	// Verify it exists
	pair, err := s.Get(ctx, "dir/key", nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("to-delete"), pair.Value)

	// Delete it
	err = s.Delete(ctx, "dir/key")
	require.NoError(t, err)

	// Verify it's gone from ConfigMap data
	pair, err = s.Get(ctx, "dir/key", nil)
	assert.Error(t, err)
	assert.Nil(t, pair)
}

func TestDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Delete a key whose ConfigMap doesn't exist — should return error
	err := s.Delete(ctx, "nodir/nokey")
	assert.Error(t, err)
}

func TestPut_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Put multiple keys into the same ConfigMap (same directory)
	pairs := map[string][]byte{
		"config/key1": []byte("v1"),
		"config/key2": []byte("v2"),
		"config/key3": []byte("v3"),
	}
	for k, v := range pairs {
		err := s.Put(ctx, k, v, nil)
		require.NoError(t, err, "putting key %s", k)
	}

	// Verify all keys are retrievable
	for k, expected := range pairs {
		pair, err := s.Get(ctx, k, nil)
		require.NoError(t, err, "getting key %s", k)
		assert.Equal(t, expected, pair.Value, "key %s", k)
	}
}

func TestWatch_ReturnsChannel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s := newTestStore(t, "default")

	// Put a key so Watch has an initial value
	err := s.Put(ctx, "dir/key", []byte("watch-value"), nil)
	require.NoError(t, err)

	// Watch should return a channel
	ch, err := s.Watch(ctx, "dir/key", nil)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Should receive the initial value
	select {
	case kv := <-ch:
		require.NotNil(t, kv)
		assert.Equal(t, "dir/key", kv.Key)
		assert.Equal(t, []byte("watch-value"), kv.Value)
	case <-ctx.Done():
		t.Fatal("timeout waiting for Watch initial delivery")
	}
}

func TestPut_KeyNameMapping(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t, "default")

	// Verify the key path → ConfigMap name mapping:
	// dir "/" → "---" in configmap name (lowercased)
	err := s.Put(ctx, "some/deep/path/value", []byte("mapped"), nil)
	require.NoError(t, err)

	pair, err := s.Get(ctx, "some/deep/path/value", nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("mapped"), pair.Value)
}
