package libprotoconf

import (
	"log"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
)

var (
	kv store.Store
)

// Get a value given its key
func Get(key string) []byte {
	value, err := kv.Get(key)
	if err != nil {
		log.Printf("Error getting key from the store, key=%s", key)
		return nil
	}
	return value.Value
}

// Setup the kv backend connection
func Setup() {
	consul.Register()
	var err error
	kv, err = libkv.NewStore(
		store.CONSUL,
		[]string{""},
		nil,
	)
	log.Println("hello from libprotoconf setup", kv, err)
}
