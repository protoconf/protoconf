package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/consts"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
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

	anyResolver, err := LoadAnyResolver(filepath.Join(protoconfRoot, consts.SrcPath), configJSON.ProtoFile)
	if err != nil {
		return nil, err
	}

	if _, err = configReader.Seek(0, 0); err != nil {
		return nil, err
	}

	protoconfValue := &protoconfvalue.ProtoconfValue{}
	um := jsonpb.Unmarshaler{AnyResolver: anyResolver}
	if err = um.Unmarshal(configReader, protoconfValue); err != nil {
		return nil, fmt.Errorf("error marshaling, err=%s", err)
	}

	return protoconfValue, nil
}

// LoadAnyResolver is a util that helps resolve `Any` messages
func LoadAnyResolver(rootPath string, parseFiles ...string) (jsonpb.AnyResolver, error) {
	parser := &protoparse.Parser{ImportPaths: []string{rootPath}}
	descriptors, err := parser.ParseFiles(parseFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%v", parseFiles, err)
	}
	return dynamic.AnyResolver(nil, descriptors...), nil
}

// ReplaceProtoBytes replaces the information inside a proto serialized byte array
func ReplaceProtoBytes(protoBytes []byte, pos int, length int, replacement []byte) ([]byte, error) {
	cb := newCodedBuffer(protoBytes)
	ret := &codedBuffer{}

	for {
		start := cb.index
		_, wireType, err := cb.decodeTagAndWireType()
		if err != nil {
			return nil, err
		}

		dataStart := cb.index
		if err = unmarshalUnknownField(wireType, cb); err != nil {
			return nil, err
		}
		end := cb.index

		if (wireType != proto.WireBytes && wireType != proto.WireStartGroup) || end < pos {
			ret.buf = append(ret.buf, cb.buf[start:end]...)
			ret.index = len(ret.buf)
			continue
		}

		cb.index = dataStart
		oldLength, err := cb.decodeVarint()
		if err != nil {
			return nil, err
		}

		var newBytes []byte
		if cb.index == pos {
			if wireType != proto.WireBytes {
				return nil, fmt.Errorf("expecting wire type bytes got=%d", wireType)
			}
			if int(oldLength) != length {
				return nil, fmt.Errorf("expecting length=%d got=%d", length, oldLength)
			}
			newBytes = replacement
		} else {
			newBytes, err = ReplaceProtoBytes(cb.buf[cb.index:cb.index+int(oldLength)], pos-cb.index, length, replacement)
			if err != nil {
				return nil, err
			}
		}
		ret.buf = append(ret.buf, cb.buf[start:dataStart]...)
		ret.index = len(ret.buf)
		if err = ret.encodeRawBytes(newBytes); err != nil {
			return nil, err
		}
		ret.buf = append(ret.buf, cb.buf[end:]...)
		ret.index = len(ret.buf)
		return ret.buf, nil
	}
}
