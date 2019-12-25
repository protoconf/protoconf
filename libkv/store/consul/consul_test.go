package consul

import (
	"testing"
	"time"
	"sync"

	"github.com/protoconf/protoconf/libkv"
	"github.com/protoconf/protoconf/libkv/store"
	"github.com/protoconf/protoconf/libkv/testutils"
	"github.com/stretchr/testify/assert"
)

var (
	client = "consul:8500"
)

func makeConsulClient(t *testing.T) store.Store {

	kv, err := New(
		[]string{client},
		&store.Config{
			ConnectionTimeout: 3 * time.Second,
		},
	)

	if err != nil {
		t.Fatalf("cannot create store: %v", err)
	}

	return kv
}

func TestRegister(t *testing.T) {
	Register()

	kv, err := libkv.NewStore(store.CONSUL, []string{client}, nil)
	assert.NoError(t, err)
	assert.NotNil(t, kv)

	if _, ok := kv.(*Consul); !ok {
		t.Fatal("Error registering and initializing consul")
	}
}

func TestConsulStore(t *testing.T) {
	kv := makeConsulClient(t)
	lockKV := makeConsulClient(t)
	ttlKV := makeConsulClient(t)

	defer testutils.RunCleanup(t, kv)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestCommon(t, kv)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestAtomic(t, kv)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestWatch(t, kv)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestLock(t, kv)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestLockTTL(t, kv, lockKV)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestLockWait(t, kv, lockKV)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testutils.RunTestTTL(t, kv, ttlKV)
	}()
	wg.Wait()
}

func TestGetActiveSession(t *testing.T) {
	kv := makeConsulClient(t)

	consul := kv.(*Consul)

	key := "foo"
	value := []byte("bar")

	// Put the first key with the Ephemeral flag
	err := kv.Put(key, value, &store.WriteOptions{TTL: 20 * time.Second})
	assert.NoError(t, err)

	// Session should not be empty
	session, err := consul.getActiveSession(key)
	assert.NoError(t, err)
	assert.NotEqual(t, session, "")

	// Delete the key
	err = kv.Delete(key)
	assert.NoError(t, err)

	// Check the session again, it should return nothing
	session, err = consul.getActiveSession(key)
	assert.NoError(t, err)
	assert.Equal(t, session, "")
}
