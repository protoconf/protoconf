package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
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
	protoExtension           = ".proto"
	validatorExtensionSuffix = "-validator"
)

func main() {
	// FIXME: support more than 1 config file
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s protoconf_root config_to_compile", os.Args[0])
	}
	protoconfRoot := strings.TrimSpace(os.Args[1])
	configToCompile := strings.TrimSpace(os.Args[2])

	if !strings.HasSuffix(configToCompile, configExtension) {
		log.Fatalf("Config must be a %s file, given: %s", configExtension, configToCompile)
	}

	registry := msgregistry.NewMessageRegistryWithDefaults()
	mainProto, err := compileConfig(configToCompile, protoconfRoot, registry)
	if err != nil {
		log.Fatalf("Error compiling config, err: %s", err)
	}

	any, err := registry.MarshalAny(mainProto)
	if err != nil {
		log.Fatalf("Error marshaling MyConfig to Any, config=%s", mainProto)
	}

	protoconfValue, err := dynamic.AsDynamicMessage(
		&pc.ProtoconfValue{
			ProtoFile: mainProto.GetMessageDescriptor().GetFile().GetName(),
			Value: any,
		})

	if err != nil {
		log.Fatalf("Error converting ProtoconfValue to dynamic, err: %s", err)
	}

	m := &jsonpb.Marshaler{AnyResolver: registry, Indent: "  "}
	jsonData, err := m.MarshalToString(protoconfValue)
	if err != nil {
		log.Fatalf("Error marshaling ProtoconfValue to JSON, value=%v", protoconfValue)
	}
	jsonData += "\n"

	outputFile := filepath.Join(protoconfRoot, compiledConfigPath, strings.TrimSuffix(configToCompile, configExtension)+compiledConfigExtension)
	fmt.Printf("Config compiled successfully, writing to %s: %v", outputFile, string(jsonData))

	if err := os.MkdirAll(filepath.Dir(outputFile), 0644); err != nil {
		log.Fatalf("Error creating output directory %s, err: %s", filepath.Dir(outputFile), err)
	}

	if err := ioutil.WriteFile(outputFile, []byte(jsonData), 0644); err != nil {
		log.Fatalf("Error writing to file %s, err: %s", outputFile, err)
	}
}

func compileConfig(filename string, protoconfRoot string, registry *msgregistry.MessageRegistry) (*dynamic.Message, error) {
	configfile, err := load(filename, protoconfRoot, registry)

	if err != nil {
		return nil, fmt.Errorf("error loading %s: %v", filename, err)
	}

	mainOutput, err := configfile.main()
	if err != nil {
		return nil, fmt.Errorf("error evaluating %s: %v", configfile.filename, err)
	}

	proto, ok := toProtoMessage(mainOutput)
	if !ok {
		return nil, fmt.Errorf("`main' returned something that's not a protobuf (a %s)", mainOutput.Type())
	}

	if err := configfile.validate(proto); err != nil {
		return nil, err
	}

	return proto, nil
}

type config struct {
	filename   string
	globals    starlark.StringDict
	locals     starlark.StringDict
	validators map[*desc.MessageDescriptor]*starlark.Function
}

func load(filename string, protoconfRoot string, registry *msgregistry.MessageRegistry) (*config, error) {
	configDir := filepath.Join(protoconfRoot, configPath)
	absFilename := filepath.Join(configDir, filename)
	reader := LocalFileReader(filepath.Dir(absFilename))
	modules := getModules()

	type cacheEntry struct {
		globals starlark.StringDict
		err     error
	}
	cache := make(map[string]*cacheEntry)
	protoFilesLoaded := &[]string{}

	accessor := func(name string) (io.ReadCloser, error) {
		*protoFilesLoaded = append(*protoFilesLoaded, name)
		return os.Open(name)
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
			if !strings.HasPrefix(modulePath, configDir) {
				log.Fatalf("Error, proto file must be under dir=%s, file=%s", configDir, modulePath)
			}
			protoFilename := strings.TrimPrefix(modulePath, configDir)
			descriptors, err := parser.ParseFiles(protoFilename)
			if err != nil {
				log.Fatalf("Error parsing proto file, file=%s err=%v", modulePath, err)
			}
			registry.AddFile("", descriptors[0])
			globals = starlark.StringDict{}
			for _, message := range descriptors[0].GetMessageTypes() {
				messageType, err := newMessageType(registry, message.GetName())
				if err != nil {
					return nil, err
				}
				globals[message.GetName()] = messageType
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
	}, absFilename)

	if err != nil {
		return nil, err
	}

	validators := make(map[*desc.MessageDescriptor]*starlark.Function)
	modules["add_validator"] = starlark.NewBuiltin("add_validator", getAddValidator(&validators))
	for _, proto := range *protoFilesLoaded {
		validatorFile := proto + validatorExtensionSuffix
		if stat, err := os.Stat(validatorFile); err == nil {
			if stat.IsDir() {
				return nil, fmt.Errorf("expected validator file and not a directory, file=%s", validatorFile)
			}
		} else if os.IsNotExist(err) {
			continue
		} else {
			return nil, err
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
		filename:   absFilename,
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

func (c *config) validate(proto *dynamic.Message) error {
	if validator, ok := c.validators[proto.GetMessageDescriptor()]; ok {
		thread := &starlark.Thread{
			Print: starPrint,
		}
		args := starlark.Tuple([]starlark.Value{
			newStarProtoMessage(proto),
		})
		if _, err := starlark.Call(thread, validator, args, nil); err != nil {
			return err
		}
	}

	for _, field := range proto.GetMessageDescriptor().GetFields() {
		if field.GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
			if protoField, ok := proto.GetField(field).(*dynamic.Message); !ok {
				if err := c.validate(protoField); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("expecting a message field, got=%v", field)
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

func getAddValidator(mp *map[*desc.MessageDescriptor]*starlark.Function) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
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

		(*mp)[messageType.desc] = validator
		return starlark.None, nil
	}
	return addValidator
}
