package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/msgregistry"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	pc "protoconf.com/types/proto/v1/protoconfvalue"
)

const (
	compiledConfigDir       = "materialized_config/"
	compiledConfigExtension = ".materialized_JSON"
	configDir               = "src/"
	configExtension         = ".pconf"
	protoExtension          = ".proto"
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

	if !strings.HasPrefix(configToCompile, configDir) {
		log.Fatalf("Config must be under %s directory, given: %s", configDir, configToCompile)
	}

	registry := msgregistry.NewMessageRegistryWithDefaults()
	mainProto, err := compileConfig(filepath.Join(protoconfRoot, configToCompile), registry)
	if err != nil {
		log.Fatalf("Error compiling config, err: %s", err)
	}

	any, err := registry.MarshalAny(mainProto)
	if err != nil {
		log.Fatalf("Error marshaling MyConfig to Any, config=%s", mainProto)
	}

	protoconfValue, err := dynamic.AsDynamicMessage(&pc.ProtoconfValue{Value: any})
	if err != nil {
		log.Fatalf("Error converting ProtoconfValue to dynamic, err: %s", err)
	}

	m := &jsonpb.Marshaler{AnyResolver: registry}
	jsonData, err := m.MarshalToString(protoconfValue)
	if err != nil {
		log.Fatalf("Error marshaling ProtoconfValue to JSON, value=%v", protoconfValue)
	}

	outputFile := filepath.Join(protoconfRoot, compiledConfigDir, strings.TrimPrefix(strings.TrimSuffix(configToCompile, configExtension), configDir)+compiledConfigExtension)
	log.Printf("Config compiled successfully, writing to %s: %v", outputFile, string(jsonData))

	if err := os.MkdirAll(filepath.Dir(outputFile), 0644); err != nil {
		log.Fatalf("Error creating output directory %s, err: %s", filepath.Dir(outputFile), err)
	}

	if err := ioutil.WriteFile(outputFile, []byte(jsonData), 0644); err != nil {
		log.Fatalf("Error writing to file %s, err: %s", outputFile, err)
	}
}

func compileConfig(filename string, registry *msgregistry.MessageRegistry) (*dynamic.Message, error) {
	configfile, err := load(filename, registry)

	if err != nil {
		return nil, fmt.Errorf("Error loading %s: %v", filename, err)
	}

	mainOutput, err := configfile.main()
	if err != nil {
		return nil, fmt.Errorf("Error evaluating %s: %v\n", configfile.filename, err)
	}

	proto, ok := toProtoMessage(mainOutput)
	if !ok {
		return nil, fmt.Errorf("`main' returned something that's not a protobuf (a %s)", mainOutput.Type())
	}

	// FIXME: run validator(s - recursively) on proto
	return proto, nil
}

type config struct {
	filename string
	globals  starlark.StringDict
	locals   starlark.StringDict
}

func load(filename string, registry *msgregistry.MessageRegistry) (*config, error) {
	reader := LocalFileReader(filepath.Dir(filename))
	modules := getModules()

	type cacheEntry struct {
		globals starlark.StringDict
		err     error
	}
	cache := make(map[string]*cacheEntry)

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
			parser := &protoparse.Parser{}
			descriptors, err := parser.ParseFiles(modulePath)
			if err != nil {
				return nil, err
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
			moduleSource, err := reader.ReadFile(context.Background(), modulePath)
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

	return &config{
		filename: filename,
		globals:  starlark.StringDict{},
		locals:   locals,
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
