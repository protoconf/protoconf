package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/compiler/starproto"
	"github.com/protoconf/protoconf/consts"
	pc "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/qri-io/starlib"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type cacheEntry struct {
	globals starlark.StringDict
	err     error
}

type starlarkLoader struct {
	cache      map[string]*cacheEntry
	Modules    starlark.StringDict
	mutableDir string
	srcDir     string
	parser     *parser.Parser
}

func (l *starlarkLoader) loadConfig(moduleName string) (starlark.StringDict, map[string]*starlark.Function, error) {
	thread := &starlark.Thread{
		Print: starPrint,
		Load:  l.Load,
	}

	locals, err := l.Load(thread, moduleName)
	if err != nil {
		return nil, nil, err
	}

	validators, err := l.loadValidators()
	if err != nil {
		return nil, nil, err
	}

	return locals, validators, nil
}

func (l *starlarkLoader) Load(thread *starlark.Thread, moduleName string) (starlark.StringDict, error) {
	if moduleName == "any.star" {
		return starlark.StringDict{"any": starproto.AnyModule}, nil
	}
	starlibResult, err := starlib.Loader(thread, moduleName)
	if err == nil {
		return starlibResult, nil
	} else {
		err = nil
	}
	var fromPath string
	if thread.CallStackDepth() > 0 {
		fromPath = thread.CallFrame(0).Pos.Filename()
	}
	modulePath, err := toCanonicalPath(moduleName, fromPath)
	if err != nil {
		return nil, err
	}

	entry, ok := l.cache[modulePath]
	if entry != nil {
		return entry.globals, entry.err
	}
	if ok {
		return nil, fmt.Errorf("cycle in load graph")
	}

	// Init to nil while parsing to detect cycles
	l.cache[modulePath] = nil
	globals, err := l.loadInner(thread, modulePath)
	l.cache[modulePath] = &cacheEntry{globals, err}

	return globals, err
}

func (l *starlarkLoader) loadValidators() (map[string]*starlark.Function, error) {
	validators := make(map[string]*starlark.Function)

	l.Modules["add_validator"] = starlark.NewBuiltin("add_validator", starAddValidator(&validators))
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		protoFileDesc, _ := desc.WrapFile(fd)
		protoFile := protoFileDesc.GetName()
		validatorFile := protoFile + consts.ValidatorExtensionSuffix
		validatorAbsPath := filepath.Join(l.srcDir, validatorFile)
		if exists, isDir, err := stat(validatorAbsPath); err != nil {
			log.Fatalf("error getting file stat for validator: %v", err)
			return true
		} else if isDir {
			log.Fatalf("expected validator file and not a directory, file=%s", validatorAbsPath)
		} else if !exists {
			return true
		}
		thread := &starlark.Thread{
			Print: starPrint,
			Load:  l.Load,
		}

		if _, err := l.Load(thread, filepath.ToSlash(validatorFile)); err != nil {
			log.Fatalf("error loading validator: %v", err)
			return true
		}
		return true
	})

	return validators, nil
}

func (l *starlarkLoader) loadInner(thread *starlark.Thread, modulePath string) (starlark.StringDict, error) {
	if strings.HasPrefix(modulePath, consts.MutableConfigPrefix) {
		return l.loadMutable(modulePath)
	}

	if strings.HasSuffix(modulePath, consts.ProtoExtension) {
		return l.loadProto(modulePath)
	}

	return l.loadStarlark(thread, modulePath)
}

func (l *starlarkLoader) loadMutable(modulePath string) (starlark.StringDict, error) {
	filename := filepath.Join(
		l.mutableDir,
		strings.TrimPrefix(modulePath, consts.MutableConfigPrefix)+consts.CompiledConfigExtension,
	)

	configReader, err := openFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening mutable config file, file=%s, err=%s", filename, err)
	}
	defer configReader.Close()

	jsonData, err := io.ReadAll(configReader)
	if err != nil {
		return nil, fmt.Errorf("error reading from mutable config file, file=%s, err=%s", filename, err)
	}

	type configJSONType struct {
		ProtoFile string
	}
	var configJSON configJSONType
	if err = json.Unmarshal(jsonData, &configJSON); err != nil {
		return nil, err
	}

	// parser := &protoparse.Parser{ImportPaths: []string{l.srcDir}, Accessor: l.protoAccessor}
	// l.parser.Accessor = l.protoAccessor
	descriptors, err := l.parser.ParseFilesX(configJSON.ProtoFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%s", configJSON.ProtoFile, err)
	}
	// l.parser.Accessor = nil
	fileDescriptor := descriptors[0]
	anyResolver := dynamic.AnyResolver(nil, fileDescriptor)

	protoconfValue := &pc.ProtoconfValue{}
	um := jsonpb.Unmarshaler{AnyResolver: anyResolver}
	if err = um.Unmarshal(io.NopCloser(bytes.NewReader(jsonData)), protoconfValue); err != nil {
		return nil, fmt.Errorf("error unmarshaling, err=%s", err)
	}

	name, err := ptypes.AnyMessageName(protoconfValue.Value)
	if err != nil {
		return nil, err
	}

	value, err := anyResolver.Resolve(name)
	if err != nil {
		return nil, err
	}

	if err = ptypes.UnmarshalAny(protoconfValue.Value, value); err != nil {
		return nil, err
	}

	message, err := dynamic.AsDynamicMessage(value)
	if err != nil {
		return nil, err
	}

	globals := starlark.StringDict{}
	globals["value"] = starproto.NewStarProtoMessage(message)
	return globals, nil
}

func (l *starlarkLoader) loadProto(modulePath string) (starlark.StringDict, error) {
	//parser := &protoparse.Parser{ImportPaths: []string{l.srcDir}, Accessor: l.protoAccessor}
	descriptors, err := l.parser.ParseFilesX(modulePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%v", modulePath, err)
	}
	fileDescriptor := descriptors[0]
	globals := starlark.StringDict{}
	for _, message := range fileDescriptor.GetMessageTypes() {
		globals[message.GetName()] = starproto.NewMessageType(message)
	}
	for _, enum := range fileDescriptor.GetEnumTypes() {
		globals[enum.GetName()] = starproto.NewEnumType(enum)
	}
	return globals, nil
}

func (l *starlarkLoader) loadStarlark(thread *starlark.Thread, modulePath string) (starlark.StringDict, error) {
	reader, err := openFile(filepath.Join(l.srcDir, modulePath))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	moduleSource, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return starlark.ExecFile(thread, modulePath, moduleSource, l.Modules)
}

func toCanonicalPath(name string, fromPath string) (string, error) {
	isMutableConfig := false
	if strings.HasPrefix(name, consts.MutableConfigPrefix) {
		isMutableConfig = true
		name = strings.TrimPrefix(name, consts.MutableConfigPrefix)
	}

	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return "", fmt.Errorf("load(%s): invalid character in module name", name)
	}
	canonicalPath := filepath.FromSlash(path.Clean(name))
	if strings.HasPrefix(canonicalPath, string(filepath.Separator)) {
		canonicalPath = strings.TrimPrefix(canonicalPath, string(filepath.Separator))
	} else if !isMutableConfig {
		canonicalPath = filepath.Join(filepath.Dir(fromPath), canonicalPath)
	}

	if isMutableConfig {
		canonicalPath = filepath.Join(consts.MutableConfigPrefix, canonicalPath)
	}

	return canonicalPath, nil
}
