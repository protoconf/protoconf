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
	// FIXME: support more than 1 config file
	protoconfRoot := strings.TrimSpace(os.Args[1])
	configToInsert := strings.TrimSpace(os.Args[2])

	if !strings.HasSuffix(configToInsert, compiledConfigExtension) {
		log.Fatalf("config must be a %s file, given: %s", compiledConfigExtension, configToInsert)
	}
	kvPath := strings.TrimSuffix(configToInsert, compiledConfigExtension)

	configFile := filepath.Join(protoconfRoot, compiledConfigPath, configToInsert)
	configReader, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("error opening config file, file=%s", configFile)
	}
	defer configReader.Close()

	type configJSONType struct {
		ProtoFile string
	}
	var configJSON configJSONType
	if err := json.NewDecoder(configReader).Decode(&configJSON); err != nil {
		log.Fatal(err) // FIXME
	}

	parser := &protoparse.Parser{ImportPaths: []string{filepath.Join(protoconfRoot, configPath)}}
	descriptors, err := parser.ParseFiles(configJSON.ProtoFile)
	if err != nil {
		log.Fatalf("error parsing proto file, file=%s err=%v", configJSON.ProtoFile, err)
	}

	registry := msgregistry.NewMessageRegistryWithDefaults()
	registry.AddFile("", descriptors[0])

	if _, err := configReader.Seek(0, 0); err != nil {
		log.Fatal(err) // FIXME
	}

	um := jsonpb.Unmarshaler{AnyResolver: registry}
	protoconfValue := pc.ProtoconfValue{}
	if err := um.Unmarshal(configReader, &protoconfValue); err != nil {
		log.Fatalf("error unmarshaling, err=%s", err)
	}

	consul.Register()
	kv, err := libkv.NewStore(
		store.CONSUL,
		[]string{""},
		nil,
	)

	data, err := proto.Marshal(&protoconfValue)
	if err != nil {
		log.Fatalf("error marshaling ProtoconfValue to bytes, value=%v", protoconfValue)
	}

	if err := kv.Put(kvPath, data, nil); err != nil {
		log.Fatalf("error writing to consul, path=%s", kvPath)
	}

	fmt.Printf("Path %s written successfully\n", kvPath)
}
