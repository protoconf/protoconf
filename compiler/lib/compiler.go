package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pelletier/go-toml"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/compiler/starproto"
	"github.com/protoconf/protoconf/consts"
	pc "github.com/protoconf/protoconf/datatypes/proto/v1"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	ConfigRollout = "ConfigRollout"
	RolloutStage  = "RolloutStage"
)

func NewCompiler(protoconfRoot string, verboseLogging bool) *Compiler {
	resolve.AllowNestedDef = true      // allow def statements within function bodies
	resolve.AllowLambda = true         // allow lambda expressions
	resolve.AllowFloat = true          // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true            // allow the 'set' built-in
	resolve.AllowGlobalReassign = true // allow reassignment to top-level names; also, allow if/for/while at top-level
	resolve.AllowRecursion = true      // allow while statements and recursive functions

	t := time.Now()
	ms := NewModuleService(protoconfRoot)
	slog.Info("module service loaded", "took", time.Since(t))
	err := ms.LoadFromLockFile()
	if err != nil {
		slog.Error("error loading from lock file", "err", err)
	}

	return &Compiler{
		protoconfRoot:   protoconfRoot,
		verboseLogging:  verboseLogging,
		MaterializedDir: filepath.Join(protoconfRoot, consts.CompiledConfigPath),
		parser:          parser.NewParser(ms.GetProtoFilesRegistry()),
		ModuleService:   ms,
	}
}

type Compiler struct {
	protoconfRoot   string
	verboseLogging  bool
	MaterializedDir string
	parser          *parser.Parser
	ModuleService   *ModuleService
}

var (
	ErrBadConfigExtension = errors.New("bad config extension")
	ErrNotADictionary     = errors.New("`main' returned something that's not a dict")
	ErrNotStringKey       = errors.New("`main' returned a dict with non-string key")
	ErrNoProtobufValue    = errors.New("`main' returned a dict with non-protobuf value")
)

func (c *Compiler) CompileFile(filename string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancelCause(ctx)
	ch, errCh := c.CompileFileAsync(ctx, cancel, filename)
	var err error
	for {
		select {
		case resp := <-ch:
			slog.Default().Debug("Compiled", slog.String("file", resp.Path))
		case e := <-errCh:
			err = errors.Join(err, e)
		case <-ctx.Done():
			e := context.Cause(ctx)
			if errors.Is(e, context.Canceled) {
				e = nil
			}
			return errors.Join(err, e)
		}
	}
}

func (c *Compiler) CompileFileAsync(ctx context.Context, cancel context.CancelCauseFunc, filename string) (chan *protoconf_pb.CompileResponse, chan error) {
	multiConfig := false
	if strings.HasSuffix(filename, consts.ConfigExtension) {
	} else if strings.HasSuffix(filename, consts.MultiConfigExtension) {
		multiConfig = true
	} else {
		cancel(errors.Join(ErrBadConfigExtension, fmt.Errorf("config file must end with either %s or %s, got: %s", consts.ConfigExtension, consts.MultiConfigExtension, filename)))
		return nil, nil
	}

	mainOutput, configFile, err := c.runConfig(filename)
	if err != nil {
		cancel(err)
		return nil, nil
	}

	ch := make(chan *protoconf_pb.CompileResponse)
	errCh := make(chan error)
	configs := make(map[string]*dynamic.Message)
	outputs := make(map[string]*dynamic.Message)

	sendErr := func(err error) error {
		ch <- &protoconf_pb.CompileResponse{
			Path:   filename,
			Errors: []string{err.Error()},
		}
		errCh <- err
		cancel(err)
		return err
	}
	go func() {
		if multiConfig {
			starDict, ok := mainOutput.(*starlark.Dict)
			if !ok {
				err := errors.Join(ErrNotADictionary, fmt.Errorf("got: %s", mainOutput.Type()))
				sendErr(err)
				return
			}

			outputDir := filepath.Join(c.MaterializedDir, strings.TrimSuffix(filename, consts.MultiConfigExtension))
			for _, item := range starDict.Items() {
				key, ok := item[0].(starlark.String)
				if !ok {
					err = errors.Join(ErrNotStringKey, fmt.Errorf("got: %s", item[0].Type()))
					sendErr(err)
					return
				}
				value, ok := starproto.ToProtoMessage(item[1])
				if !ok {
					err = errors.Join(ErrNoProtobufValue, fmt.Errorf("got: %s", item[1].Type()))
					sendErr(err)
					return
				}
				configs[filepath.Join(outputDir, string(key))+consts.CompiledConfigExtension] = value
				if hasAnySuffix(string(key), ".json", ".yaml", ".yml", ".toml") {
					outputs[filepath.Join(strings.TrimSuffix(filename, consts.MultiConfigExtension), string(key))] = value
				}
			}
		} else {
			message, ok := starproto.ToProtoMessage(mainOutput)
			if !ok {
				err = errors.Join(ErrNoProtobufValue, fmt.Errorf("got: %s", mainOutput.Type()))
				sendErr(err)
				return
			}
			outputFile := filepath.Join(c.MaterializedDir, strings.TrimSuffix(filename, consts.ConfigExtension)+consts.CompiledConfigExtension)
			configs[outputFile] = message
			if hasAnySuffix(strings.TrimSuffix(filename, consts.ConfigExtension), ".json", ".yaml", ".yml", ".toml") {
				outputs[filepath.Join(strings.TrimSuffix(filename, consts.ConfigExtension))] = message
			}
		}

		for outputFile, message := range configs {
			protoconfValue, err := toProtoconfValue(message)
			result := &protoconf_pb.CompileResponse{
				Path:   outputFile,
				File:   filename,
				Result: protoconfValue,
				Errors: []string{},
			}
			if err != nil {
				sendErr(err)
			}
			if err := configFile.validate(message); err != nil {
				sendErr(err)
			}
			if err := c.writeConfig(message, outputFile); err != nil {
				sendErr(err)
			}
			ch <- result

		}

		for outputFile, message := range outputs {
			if err := c.writeOutput(message, outputFile); err != nil {
				sendErr(err)
			}
		}
		slog.Default().Info("Compiled", slog.String("file", filename))
		cancel(nil)
	}()
	return ch, errCh
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func (c *Compiler) writeOutput(message *dynamic.Message, filename string) error {
	dirname := filepath.Dir(filename)
	data, err := message.MarshalJSONIndent()
	if err != nil {
		return err
	}
	if hasAnySuffix(filename, ".yaml", ".yml") {
		data, err = yaml.JSONToYAML(data)
		if err != nil {
			return err
		}
	}
	if hasAnySuffix(filename, ".toml") {
		var m map[string]interface{}
		if err = json.Unmarshal(data, &m); err != nil {
			return err
		}
		tree, tomlErr := toml.TreeFromMap(m)
		if tomlErr != nil {
			return tomlErr
		}
		tomlString, strErr := tree.ToTomlString()
		if strErr != nil {
			return strErr
		}
		data = []byte(tomlString)
	}
	if err = mkdirAll(filepath.Join(consts.OutputsDir, dirname), 0755); err != nil {
		return err
	}
	if err = writeFile(filepath.Join(consts.OutputsDir, filename), data); err != nil {
		return err
	}

	return nil
}

func toProtoconfValue(message *dynamic.Message) (*protoconf_pb.ProtoconfValue, error) {
	protoconfValue := &protoconf_pb.ProtoconfValue{}
	err := message.MergeInto(protoconfValue)
	if err != nil {
		any, err := anypb.New(starproto.ToDynamicPb(message))
		if err != nil {
			return nil, fmt.Errorf("error marshaling proto to Any, message=%s", message)
		}

		return &protoconf_pb.ProtoconfValue{
			ProtoFile: filepath.ToSlash(message.GetMessageDescriptor().GetFile().GetName()),
			Value:     any,
		}, nil
	}
	return protoconfValue, nil
}

func (c *Compiler) writeConfig(message *dynamic.Message, filename string) error {
	protoconfValue, err := toProtoconfValue(message)
	if err != nil {
		return err
	}

	jsonData, err := protojson.MarshalOptions{
		Resolver: c.parser.LocalResolver,
	}.Marshal(protoconfValue)
	if err != nil {
		return fmt.Errorf("error marshaling ProtoconfValue to JSON, value=%v, err: %v", protoconfValue, err)
	}

	if err := mkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("error creating output directory %s, err: %s", filepath.Dir(filename), err)
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, jsonData, "", "  "); err != nil {
		return fmt.Errorf("failed to prettify json: %v", err)
	}
	if err := writeFile(filename, prettyJSON.Bytes()); err != nil {
		return fmt.Errorf("error writing to file %s, err: %s", filename, err)
	}

	if c.verboseLogging {
		log.Printf("Writing to %s:\n%s", filename, jsonData)
	}

	return nil
}

var (
	ErrStarlarkEval = errors.New("error evaluating starlark file")
	ErrLoadStarlark = errors.New("error loading starlark file")
)

func (c *Compiler) runConfig(filename string) (starlark.Value, *config, error) {
	configFile, err := c.load(filename)

	if err != nil {
		return nil, nil, errors.Join(ErrLoadStarlark, err)
	}

	mainOutput, err := configFile.main()
	if err != nil {
		if evalError, ok := err.(*starlark.EvalError); ok {
			return nil, nil, errors.Join(ErrStarlarkEval, evalError, fmt.Errorf("\n%s", evalError.Backtrace()))
		}
		return nil, nil, errors.Join(ErrStarlarkEval, fmt.Errorf("%s", configFile.filename), err)
	}

	return mainOutput, configFile, nil
}

func (c *Compiler) load(filename string) (*config, error) {

	loader := c.GetLoader()
	locals, validators, err := loader.loadConfig(filepath.ToSlash(filename))
	if err != nil {
		return nil, err
	}

	return &config{
		filename:        filename,
		locals:          locals,
		validators:      validators,
		protoResolver:   c.parser.LocalResolver,
		messageRegistry: c.ModuleService.GetProtoRegistry().MessageRegistry,
	}, nil
}

func (c *Compiler) GetLoader() *starlarkLoader {
	modules := getModules()
	modules[ConfigRollout] = starlark.NewBuiltin(ConfigRollout, newConfigRollout)
	modules[RolloutStage] = starproto.NewBuiltin(&pc.ProtoconfValue_ConfigRollout_Stage{})
	return &starlarkLoader{
		cache:         make(map[string]*cacheEntry),
		Modules:       modules,
		mutableDir:    filepath.Join(c.protoconfRoot, consts.MutableConfigPath),
		srcDir:        filepath.Join(c.protoconfRoot, consts.SrcPath),
		parser:        c.parser,
		moduleService: c.ModuleService,
	}
}

func newConfigRollout(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var config starlark.Value

	// UnpackArgs returns an expected err and is not currently supports `errors.Is`.
	// We will check wether `config` is `nil` instead.
	err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &config)
	if config == starlark.None {
		return nil, err
	}

	msg, ok := starproto.ToProtoMessage(config)
	if !ok {
		return nil, errors.New("value is not a proto message")
	}

	any, err := anypb.New(starproto.ToDynamicPb(msg))
	if err != nil {
		return nil, err
	}

	ret := &pc.ProtoconfValue{
		Value:         any,
		RolloutConfig: &pc.ProtoconfValue_ConfigRollout{},
	}

	_, err = starproto.NewBuiltin(ret.RolloutConfig, func(m *dynamic.Message) error {
		return m.MergeInto(ret.RolloutConfig)
	}).CallInternal(thread, nil, kwargs)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.AsDynamicMessage(ret)
	if err != nil {
		return nil, err
	}

	return starproto.NewStarProtoMessage(dyn), nil

}
