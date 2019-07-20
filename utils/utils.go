package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"protoconf.com/consts"
	protoconfvalue "protoconf.com/datatypes/proto/v1/protoconfvalue"
	"protoconf.com/protostdlib"
)

// ReadConfig reads a materialized config
func ReadConfig(protoconfRoot string, configName string) (*protoconfvalue.ProtoconfValue, error) {
	filename := filepath.Join(protoconfRoot, consts.CompiledConfigPath, configName+consts.CompiledConfigExtension)

	configReader, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file, file=%s", filename)
	}
	defer configReader.Close()

	type configJSONType struct {
		ProtoFile string
	}
	var configJSON configJSONType
	if err = json.NewDecoder(configReader).Decode(&configJSON); err != nil {
		return nil, err
	}

	parser := &protoparse.Parser{ImportPaths: []string{filepath.Join(protoconfRoot, consts.SrcPath)}, ProtoStdLib: protostdlib.ProtoStdLib}
	descriptors, err := parser.ParseFiles(configJSON.ProtoFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%v", configJSON.ProtoFile, err)
	}
	anyResolver := dynamic.AnyResolver(nil, descriptors[0])

	if _, err := configReader.Seek(0, 0); err != nil {
		return nil, err
	}

	protoconfValue := &protoconfvalue.ProtoconfValue{}
	um := jsonpb.Unmarshaler{AnyResolver: anyResolver}
	if err := um.Unmarshal(configReader, protoconfValue); err != nil {
		return nil, fmt.Errorf("error unmarshaling, err=%s", err)
	}
	return protoconfValue, nil
}

func MessageFQN(msg descriptor.Message) string {
	fileDesc, protoDesc := descriptor.ForMessage(msg)
	fqn := protoDesc.GetName()
	if fileDesc.GetPackage() != "" {
		fqn = fileDesc.GetPackage() + "." + fqn
	}
	return fqn
}
