package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	pbproto "github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/msgregistry"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	pc "protoconf.com/types/proto/v1/protoconfvalue"
)

const (
	compiledConfigExtension  = ".materialized_JSON"
	compiledConfigPath       = "materialized_config/"
	configExtension          = ".pconf"
	configPath               = "src/"
	multiConfigExtension     = ".mpconf"
	protoExtension           = ".proto"
	validatorExtensionSuffix = "-validator"
)

func compileFile(filename string, protoconfRoot string) error {
	multiConfig := false
	if strings.HasSuffix(filename, configExtension) {
	} else if strings.HasSuffix(filename, multiConfigExtension) {
		multiConfig = true
	} else {
		return fmt.Errorf("config file must end with either %s or %s, got: %s", configExtension, multiConfigExtension, filename)
	}

	registry := msgregistry.NewMessageRegistryWithDefaults()
	mainOutput, configFile, err := runConfig(filename, protoconfRoot, registry)
	if err != nil {
		return err
	}

	configs := make(map[string]*dynamic.Message)

	if multiConfig {
		starDict, ok := mainOutput.(*starlark.Dict)
		if !ok {
			return fmt.Errorf("`main' returned something that's not a dict, got: %s", mainOutput.Type())
		}

		outputDir := filepath.Join(protoconfRoot, compiledConfigPath, strings.TrimSuffix(filename, multiConfigExtension))
		for _, item := range starDict.Items() {
			key, ok := item[0].(starlark.String)
			if !ok {
				return fmt.Errorf("`main' returned a dict with non-string key, got: %s", item[0].Type())
			}
			value, ok := toProtoMessage(item[1])
			if !ok {
				return fmt.Errorf("`main' returned a dict with non-protobuf value, got: %s", item[1].Type())
			}
			configs[filepath.Join(outputDir, string(key))+compiledConfigExtension] = value
		}
	} else {
		proto, ok := toProtoMessage(mainOutput)
		if !ok {
			return fmt.Errorf("`main' returned something that's not a protobuf, got: %s", mainOutput.Type())
		}
		outputFile := filepath.Join(protoconfRoot, compiledConfigPath, strings.TrimSuffix(filename, configExtension)+compiledConfigExtension)
		configs[outputFile] = proto
	}

	for outputFile, proto := range configs {
		if err := configFile.validate(proto); err != nil {
			return err
		}
		if err := writeConfig(proto, outputFile, registry); err != nil {
			return err
		}
	}

	return nil
}

func writeConfig(proto *dynamic.Message, filename string, registry *msgregistry.MessageRegistry) error {
	any, err := registry.MarshalAny(proto)
	if err != nil {
		return fmt.Errorf("error marshaling proto to Any, proto=%s", proto)
	}

	protoconfValue, err := dynamic.AsDynamicMessage(
		&pc.ProtoconfValue{
			ProtoFile: proto.GetMessageDescriptor().GetFile().GetName(),
			Value:     any,
		})

	if err != nil {
		return fmt.Errorf("error converting ProtoconfValue to dynamic, err: %s", err)
	}

	m := &jsonpb.Marshaler{AnyResolver: registry, Indent: "  "}
	jsonData, err := m.MarshalToString(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%v", protoconfValue)
	}
	jsonData += "\n"

	if err := mkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("error creating output directory %s, err: %s", filepath.Dir(filename), err)
	}

	if err := writeFile(filename, []byte(jsonData)); err != nil {
		return fmt.Errorf("error writing to file %s, err: %s", filename, err)
	}

	return nil
}

func runConfig(filename string, protoconfRoot string, registry *msgregistry.MessageRegistry) (starlark.Value, *config, error) {
	configFile, err := load(filename, protoconfRoot, registry)

	if err != nil {
		return nil, nil, fmt.Errorf("error loading %s: %v", filename, err)
	}

	mainOutput, err := configFile.main()
	if err != nil {
		return nil, nil, fmt.Errorf("error evaluating %s: %v", configFile.filename, err)
	}

	return mainOutput, configFile, nil
}

type config struct {
	filename   string
	globals    starlark.StringDict
	locals     starlark.StringDict
	validators map[string]*starlark.Function
}

func load(filename string, protoconfRoot string, registry *msgregistry.MessageRegistry) (*config, error) {
	configDir := filepath.Join(protoconfRoot, configPath)
	reader := LocalFileReader(configDir)
	modules := getModules()

	type cacheEntry struct {
		globals starlark.StringDict
		err     error
	}
	cache := make(map[string]*cacheEntry)
	protoFilesLoaded := &[]string{}

	accessor := func(name string) (io.ReadCloser, error) {
		if !strings.HasPrefix(name, configDir) {
			return nil, fmt.Errorf("proto path must be under %s, got=%s", configDir, name)
		}
		*protoFilesLoaded = append(*protoFilesLoaded, strings.TrimPrefix(name, configDir))
		return openFile(name)
	}

	var load func(thread *starlark.Thread, moduleName string) (starlark.StringDict, error)
	load = func(thread *starlark.Thread, moduleName string) (starlark.StringDict, error) {
		var fromPath string
		if thread.TopFrame() != nil {
			fromPath = thread.TopFrame().Position().Filename()
		}
		modulePath, err := reader.Resolve(context.Background(), moduleName, fromPath)
		if err != nil {
			return nil, err
		}

		e, ok := cache[modulePath]
		if e != nil {
			return e.globals, e.err
		}
		if ok {
			return nil, fmt.Errorf("cycle in load graph")
		}

		// Init to nil while parsing to detect cycles
		cache[modulePath] = nil

		var globals starlark.StringDict

		if strings.HasSuffix(modulePath, protoExtension) {
			parser := &protoparse.Parser{ImportPaths: []string{configDir}, Accessor: accessor}
			descriptors, err := parser.ParseFiles(modulePath)
			if err != nil {
				return nil, fmt.Errorf("Error parsing proto file, file=%s err=%v", modulePath, err)
			}
			fileDescriptor := descriptors[0]
			registry.AddFile("", fileDescriptor)
			globals = starlark.StringDict{}
			for _, message := range fileDescriptor.GetMessageTypes() {
				globals[message.GetName()] = newMessageTypeByDesc(registry, message)
			}
			err = nil
		} else {
			var moduleSource []byte
			moduleSource, err = reader.ReadFile(context.Background(), modulePath)
			if err != nil {
				cache[modulePath] = &cacheEntry{nil, err}
				return nil, err
			}

			globals, err = starlark.ExecFile(thread, modulePath, moduleSource, modules)
		}

		cache[modulePath] = &cacheEntry{globals, err}

		return globals, err
	}
	locals, err := load(&starlark.Thread{
		Print: starPrint,
		Load:  load,
	}, filename)

	if err != nil {
		return nil, err
	}

	validators := make(map[string]*starlark.Function)
	modules["add_validator"] = starlark.NewBuiltin("add_validator", getAddValidator(&validators))
	for _, proto := range *protoFilesLoaded {
		validatorFile := proto + validatorExtensionSuffix
		validatorAbsPath := filepath.Join(configDir, validatorFile)
		if exists, isDir, err := stat(validatorAbsPath); err != nil {
			return nil, err
		} else if isDir {
			return nil, fmt.Errorf("expected validator file and not a directory, file=%s", validatorAbsPath)
		} else if !exists {
			continue
		}
		_, err := load(&starlark.Thread{
			Print: starPrint,
			Load:  load,
		}, validatorFile)

		if err != nil {
			return nil, err
		}
	}

	return &config{
		filename:   filename,
		globals:    starlark.StringDict{},
		locals:     locals,
		validators: validators,
	}, nil
}

func (c *config) main() (starlark.Value, error) {
	mainVal, ok := c.locals["main"]
	if !ok {
		return nil, fmt.Errorf("no `main' function found in %q", c.filename)
	}
	main, ok := mainVal.(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("`main' must be a function (got a %s)", mainVal.Type())
	}

	thread := &starlark.Thread{
		Print: starPrint,
	}

	mainVal, err := starlark.Call(thread, main, nil, nil)
	if err != nil {
		return nil, err
	}
	return mainVal, nil
}

func (c *config) validate(proto interface{}) error {
	protoMessage, ok := proto.(*dynamic.Message)
	if !ok {
		if _, ok := proto.(pbproto.Message); ok {
			return nil
		}
		return fmt.Errorf("expecting a proto message to validate, got=%v", proto)
	}

	if protoMessage == nil {
		return nil
	}

	if validator, ok := c.validators[protoMessage.GetMessageDescriptor().GetFullyQualifiedName()]; ok {
		thread := &starlark.Thread{
			Print: starPrint,
		}
		args := starlark.Tuple([]starlark.Value{
			newStarProtoMessage(protoMessage),
		})
		if _, err := starlark.Call(thread, validator, args, nil); err != nil {
			return err
		}
	}

	for _, field := range protoMessage.GetMessageDescriptor().GetFields() {
		if field.GetType() != dpb.FieldDescriptorProto_TYPE_MESSAGE {
			continue
		}
		if field.IsRepeated() {
			length := len(protoMessage.GetField(field).([]interface{}))
			for i := 0; i < length; i++ {
				if err := c.validate(protoMessage.GetRepeatedField(field, i)); err != nil {
					return err
				}
			}
		} else if field.IsMap() {
			mp := protoMessage.GetField(field).(map[interface{}]interface{})
			if field.GetMapKeyType().GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
				for key := range mp {
					if err := c.validate(key); err != nil {
						return err
					}
				}
			}
			if field.GetMapValueType().GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
				for _, value := range mp {
					if err := c.validate(value); err != nil {
						return err
					}
				}
			}
		} else {
			if err := c.validate(protoMessage.GetField(field)); err != nil {
				return err
			}
		}
	}
	return nil
}

func starPrint(t *starlark.Thread, msg string) {
	log.Printf("[%v] %s", t.Caller().Position(), msg)
}

func getModules() starlark.StringDict {
	return starlark.StringDict{
		"fail":   starlark.NewBuiltin("fail", starFail),
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
	}
}

func starFail(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &msg); err != nil {
		return nil, err
	}
	buf := new(strings.Builder)
	t.Caller().WriteBacktrace(buf)
	return nil, fmt.Errorf("[%s] %s\n%s", t.Caller().Position(), msg, buf.String())
}

func getAddValidator(mp *map[string]*starlark.Function) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	addValidator := func(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var arg1 starlark.Value
		var arg2 starlark.Value
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &arg1, &arg2); err != nil {
			return nil, err
		}

		messageType, ok := arg1.(*starProtoMessageType)
		if !ok {
			return nil, fmt.Errorf("expected a proto message type, got=%v", arg1)
		}

		validator, ok := arg2.(*starlark.Function)
		if ok {
			if numParams := validator.NumParams(); numParams != 1 {
				return nil, fmt.Errorf("expected a function that get 1 param, got=%d", numParams)
			}
		} else {
			return nil, fmt.Errorf("expected a function, got=%v", validator)
		}

		(*mp)[messageType.desc.GetFullyQualifiedName()] = validator

		return starlark.None, nil
	}
	return addValidator
}
