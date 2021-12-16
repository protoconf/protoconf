package lib

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/compiler/proto"
	"github.com/protoconf/protoconf/consts"
	pc "github.com/protoconf/protoconf/datatypes/proto/v1"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func NewCompiler(protoconfRoot string, verboseLogging bool) *Compiler {
	resolve.AllowNestedDef = true      // allow def statements within function bodies
	resolve.AllowLambda = true         // allow lambda expressions
	resolve.AllowFloat = true          // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true            // allow the 'set' built-in
	resolve.AllowGlobalReassign = true // allow reassignment to top-level names; also, allow if/for/while at top-level
	resolve.AllowRecursion = true      // allow while statements and recursive functions

	return &Compiler{
		protoconfRoot:    protoconfRoot,
		verboseLogging:   verboseLogging,
		disableWriting:   false,
		protoFilesLoaded: make(map[string]interface{}),
		MaterializedDir:  filepath.Join(protoconfRoot, consts.CompiledConfigPath),
	}
}

type Compiler struct {
	protoconfRoot    string
	verboseLogging   bool
	disableWriting   bool
	protoFilesLoaded map[string]interface{}
	MaterializedDir  string
}

func (c *Compiler) DisableWriting() error {
	c.disableWriting = true
	return nil
}

func (c *Compiler) CompileFile(filename string) error {
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

		outputDir := filepath.Join(c.MaterializedDir, strings.TrimSuffix(filename, consts.MultiConfigExtension))
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
		outputFile := filepath.Join(c.MaterializedDir, strings.TrimSuffix(filename, consts.ConfigExtension)+consts.CompiledConfigExtension)
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

func (c *Compiler) writeConfig(message *dynamic.Message, filename string) error {
	if c.disableWriting {
		return nil
	}
	any, err := ptypes.MarshalAny(message)
	if err != nil {
		return fmt.Errorf("error marshaling proto to Any, message=%s", message)
	}

	protoconfValue := &pc.ProtoconfValue{
		ProtoFile: filepath.ToSlash(message.GetMessageDescriptor().GetFile().GetName()),
		Value:     any,
	}

	m := &protojson.MarshalOptions{Resolver: protoregistry.GlobalTypes, Indent: "  "}
	jsonBytes, err := m.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%v, error: %v", protoconfValue, err)
	}
	jsonData := string(jsonBytes)
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

func (c *Compiler) runConfig(filename string) (starlark.Value, *config, error) {
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

func (c *Compiler) load(filename string) (*config, error) {

	loader := c.GetLoader()
	locals, validators, err := loader.loadConfig(filepath.ToSlash(filename))
	for _, f := range *loader.protoFilesLoaded {
		c.protoFilesLoaded[f] = true
	}
	if err != nil {
		return nil, err
	}

	return &config{
		filename:   filename,
		locals:     locals,
		validators: validators,
	}, nil
}

func (c *Compiler) GetLoader() *starlarkLoader {
	return &starlarkLoader{
		cache:            make(map[string]*cacheEntry),
		Modules:          getModules(),
		mutableDir:       filepath.Join(c.protoconfRoot, consts.MutableConfigPath),
		protoFilesLoaded: &[]string{},
		srcDir:           filepath.Join(c.protoconfRoot, consts.SrcPath),
	}
}
