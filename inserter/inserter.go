package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/golang/protobuf/proto"
	"protoconf.com/consts"
	"protoconf.com/libprotoconf"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s protoconf_root config_to_insert...", os.Args[0])
	}

	protoconfRoot := strings.TrimSpace(os.Args[1])

	consul.Register()
	kvStore, err := libkv.NewStore(store.CONSUL, []string{""}, nil)
	if err != nil {
		log.Fatalf("Error connecting to key-value store, err=%s", err)
	}

	for i := 2; i < len(os.Args); i++ {
		configName := strings.TrimSpace(os.Args[i])
		if err := insertFile(configName, protoconfRoot, kvStore); err != nil {
			log.Fatalf("Error inserting file %s, err=%s", configName, err)
		}
	}
}

func insertFile(configName string, protoconfRoot string, kvStore store.Store) error {
	if !strings.HasSuffix(configName, consts.CompiledConfigExtension) {
		return fmt.Errorf("config must be a %s file, file=%s", consts.CompiledConfigExtension, configName)
	}
	kvPath := strings.TrimSuffix(configName, consts.CompiledConfigExtension)

	protoconfValue, err := libprotoconf.ReadConfig(protoconfRoot, kvPath)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}

	if err := kvStore.Put(kvPath, data, nil); err != nil {
		return fmt.Errorf("error writing to consul, path=%s", kvPath)
	}

	fmt.Printf("Path %s inserted successfully\n", kvPath)
	return nil
}
