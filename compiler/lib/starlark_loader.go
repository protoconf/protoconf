package lib

import (
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/compiler/lib/parser"
	"github.com/protoconf/protoconf/compiler/starproto"
	"github.com/protoconf/protoconf/consts"
	protoconf_pb "github.com/protoconf/protoconf/pb/protoconf/v1"
	"github.com/qri-io/starlib"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type cacheEntry struct {
	globals starlark.StringDict
	err     error
}

type starlarkLoader struct {
	cache         map[string]*cacheEntry
	Modules       starlark.StringDict
	mutableDir    string
	srcDir        string
	parser        *parser.Parser
	moduleService *ModuleService
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
	metadata := ParseModulePath(moduleName)
	if metadata != nil && metadata.Repo != "" {
		return l.moduleService.Load(thread, moduleName)
	}

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
	l.parser.FilesResolver.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		protoFile := fd.Path()
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

var (
	ErrLoadMutable = errors.New("error opening mutable config file")
	ErrReadMutable = errors.New("error reading from mutable config file")
)

func (l *starlarkLoader) loadMutable(modulePath string) (starlark.StringDict, error) {
	filename := filepath.Join(
		l.mutableDir,
		strings.TrimPrefix(modulePath, consts.MutableConfigPrefix)+consts.CompiledConfigExtension,
	)
	protoconfValue := &protoconf_pb.ProtoconfValue{}
	err := l.parser.ReadConfig(filename, protoconfValue)
	if err != nil {
		return nil, errors.Join(ErrLoadMutable, fmt.Errorf("file=%s", filename), err)
	}

	mt, err := l.parser.LocalResolver.FindMessageByURL(protoconfValue.Value.TypeUrl)
	if err != nil {
		return nil, err
	}
	new := dynamicpb.NewMessage(mt.Descriptor())
	err = protoconfValue.Value.UnmarshalTo(new)
	if err != nil {
		return nil, errors.Join(ErrReadMutable, fmt.Errorf("file=%s", filename), err)
	}

	d, err := l.moduleService.GetProtoRegistry().MessageRegistry.FindMessageTypeByUrl(protoconfValue.Value.TypeUrl)
	if err != nil {
		return nil, err
	}
	message := dynamic.NewMessage(d)

	if err != nil {
		return nil, err
	}
	b, err := proto.Marshal(new)
	if err != nil {
		return nil, err
	}
	err = message.Unmarshal(b)
	if err != nil {
		return nil, err
	}

	starProtoMsg := starproto.NewStarProtoMessage(message)
	globals := starlark.StringDict{"value": starProtoMsg}
	return globals, nil
}

func (l *starlarkLoader) loadProto(modulePath string) (starlark.StringDict, error) {
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
		return nil, errors.Join(errors.New("cannot load starlark file"), err)
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
