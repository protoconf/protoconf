package zookeeper

import (
	"testing"
	"time"

	"github.com/protoconf/protoconf/libkv"
	"github.com/protoconf/protoconf/libkv/store"
	"github.com/protoconf/protoconf/libkv/testutils"
	"github.com/stretchr/testify/assert"
)

var (
	client = "zookeeper:2181"
)

func makeZkClient(t *testing.T) store.Store {
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

	kv, err := libkv.NewStore(store.ZK, []string{client}, nil)
	assert.NoError(t, err)
	assert.NotNil(t, kv)

	if _, ok := kv.(*Zookeeper); !ok {
		t.Fatal("Error registering and initializing zookeeper")
	}
}

func TestZkStore(t *testing.T) {
	kv := makeZkClient(t)
	ttlKV := makeZkClient(t)

	defer testutils.RunCleanup(t, kv)

	testutils.RunTestCommon(t, kv)
	testutils.RunTestAtomic(t, kv)
	testutils.RunTestWatch(t, kv)
	testutils.RunTestLock(t, kv)
	testutils.RunTestTTL(t, kv, ttlKV)
}
