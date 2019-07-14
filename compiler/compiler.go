package compiler

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/dynamic"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"protoconf.com/compiler/proto"
	"protoconf.com/consts"
	pc "protoconf.com/datatypes/proto/v1/protoconfvalue"
)

func NewCompiler(protoconfRoot string, verboseLogging bool) *compiler {
	resolve.AllowNestedDef = true      // allow def statements within function bodies
	resolve.AllowLambda = true         // allow lambda expressions
	resolve.AllowFloat = true          // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true            // allow the 'set' built-in
	resolve.AllowGlobalReassign = true // allow reassignment to top-level names; also, allow if/for/while at top-level
	resolve.AllowRecursion = true      // allow while statements and recursive functions

	return &compiler{
		protoconfRoot:  protoconfRoot,
		verboseLogging: verboseLogging,
	}
}

type compiler struct {
	protoconfRoot  string
	verboseLogging bool
}

func (c *compiler) CompileFile(filename string) error {
	multiConfig := false
	if strings.HasSuffix(filename, consts.ConfigExtension) {
	} else if strings.HasSuffix(filename, consts.MultiConfigExtension) {
		multiConfig = true
	} else {
		return fmt.Errorf("config file must end with either %s or %s, got: %s", consts.ConfigExtension, consts.MultiConfigExtension, filename)
	}

	mainOutput, configFile, err := c.runConfig(filename)
	if err != nil {
		return err
	}

	configs := make(map[string]*dynamic.Message)

	if multiConfig {
		starDict, ok := mainOutput.(*starlark.Dict)
		if !ok {
			return fmt.Errorf("`main' returned something that's not a dict, got: %s", mainOutput.Type())
		}

		outputDir := filepath.Join(c.protoconfRoot, consts.CompiledConfigPath, strings.TrimSuffix(filename, consts.MultiConfigExtension))
		for _, item := range starDict.Items() {
			key, ok := item[0].(starlark.String)
			if !ok {
				return fmt.Errorf("`main' returned a dict with non-string key, got: %s", item[0].Type())
			}
			value, ok := proto.ToProtoMessage(item[1])
			if !ok {
				return fmt.Errorf("`main' returned a dict with non-protobuf value, got: %s", item[1].Type())
			}
			configs[filepath.Join(outputDir, string(key))+consts.CompiledConfigExtension] = value
		}
	} else {
		message, ok := proto.ToProtoMessage(mainOutput)
		if !ok {
			return fmt.Errorf("`main' returned something that's not a protobuf, got: %s", mainOutput.Type())
		}
		outputFile := filepath.Join(c.protoconfRoot, consts.CompiledConfigPath, strings.TrimSuffix(filename, consts.ConfigExtension)+consts.CompiledConfigExtension)
		configs[outputFile] = message
	}

	for outputFile, message := range configs {
		if err := configFile.validate(message); err != nil {
			return err
		}
		if err := c.writeConfig(message, outputFile); err != nil {
			return err
		}
	}

	return nil
}

func (c *compiler) writeConfig(message *dynamic.Message, filename string) error {
	any, err := ptypes.MarshalAny(message)
	if err != nil {
		return fmt.Errorf("error marshaling proto to Any, message=%s", message)
	}

	protoconfValue := &pc.ProtoconfValue{
		ProtoFile: filepath.ToSlash(message.GetMessageDescriptor().GetFile().GetName()),
		Value:     any,
	}

	if err != nil {
		return fmt.Errorf("error converting ProtoconfValue to dynamic, err=%s", err)
	}

	// FIXME: support nested Any types by traversing FileDescriptors on starProtoMessage
	m := &jsonpb.Marshaler{AnyResolver: dynamic.AnyResolver(nil, message.GetMessageDescriptor().GetFile()), Indent: "  "}
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

	if c.verboseLogging {
		log.Printf("Writing to %s:\n%s", filename, jsonData)
	}

	return nil
}

func (c *compiler) runConfig(filename string) (starlark.Value, *config, error) {
	configFile, err := c.load(filename)

	if err != nil {
		return nil, nil, fmt.Errorf("error loading %s: %v", filename, err)
	}

	mainOutput, err := configFile.main()
	if err != nil {
		return nil, nil, fmt.Errorf("error evaluating %s: %v", configFile.filename, err)
	}

	return mainOutput, configFile, nil
}

func (c *compiler) load(filename string) (*config, error) {

	loader := c.getLoader()
	locals, validators, err := loader.loadConfig(filepath.ToSlash(filename))
	if err != nil {
		return nil, err
	}

	return &config{
		filename:   filename,
		locals:     locals,
		validators: validators,
	}, nil
}

func (c *compiler) getLoader() *starlarkLoader {
	return &starlarkLoader{
		cache:            make(map[string]*cacheEntry),
		modules:          getModules(),
		mutableDir:       filepath.Join(c.protoconfRoot, consts.MutableConfigPath),
		protoFilesLoaded: &[]string{},
		srcDir:           filepath.Join(c.protoconfRoot, consts.SrcPath),
	}
}
