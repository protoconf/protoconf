package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic/msgregistry"
	pc "protoconf.com/types/proto/v1/protoconfvalue"
)

const (
	// FIXME: put these somewhere accesible to all
	compiledConfigPath      = "materialized_config/"
	compiledConfigExtension = ".materialized_JSON"
	configPath              = "src/"
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
	if !strings.HasSuffix(configName, compiledConfigExtension) {
		return fmt.Errorf("config must be a %s file, file=%s", compiledConfigExtension, configName)
	}
	kvPath := strings.TrimSuffix(configName, compiledConfigExtension)

	configFilename := filepath.Join(protoconfRoot, compiledConfigPath, configName)
	configReader, err := os.Open(configFilename)
	if err != nil {
		return fmt.Errorf("error opening config file, file=%s", configFilename)
	}
	defer configReader.Close()

	type configJSONType struct {
		ProtoFile string
	}
	var configJSON configJSONType
	if err := json.NewDecoder(configReader).Decode(&configJSON); err != nil {
		return err
	}

	parser := &protoparse.Parser{ImportPaths: []string{filepath.Join(protoconfRoot, configPath)}}
	descriptors, err := parser.ParseFiles(configJSON.ProtoFile)
	if err != nil {
		return fmt.Errorf("error parsing proto file, file=%s err=%v", configJSON.ProtoFile, err)
	}

	registry := msgregistry.NewMessageRegistryWithDefaults()
	registry.AddFile("", descriptors[0])

	if _, err := configReader.Seek(0, 0); err != nil {
		return err
	}

	um := jsonpb.Unmarshaler{AnyResolver: registry}
	protoconfValue := pc.ProtoconfValue{}
	if err := um.Unmarshal(configReader, &protoconfValue); err != nil {
		return fmt.Errorf("error unmarshaling, err=%s", err)
	}

	data, err := proto.Marshal(&protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}

	if err := kvStore.Put(kvPath, data, nil); err != nil {
		return fmt.Errorf("error writing to consul, path=%s", kvPath)
	}

	fmt.Printf("Path %s inserted successfully\n", kvPath)
	return nil
}
