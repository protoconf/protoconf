package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/protoconf/protoconf/compiler/proto"
	"github.com/protoconf/protoconf/consts"
	pc "github.com/protoconf/protoconf/datatypes/proto/v1"
	"github.com/qri-io/starlib"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	gproto "google.golang.org/protobuf/proto"
)

type cacheEntry struct {
	globals starlark.StringDict
	err     error
}

type starlarkLoader struct {
	cache            map[string]*cacheEntry
	Modules          starlark.StringDict
	mutableDir       string
	protoFilesLoaded *[]string
	srcDir           string
}

func (l *starlarkLoader) protoAccessor(name string) (io.ReadCloser, error) {
	if !strings.HasPrefix(name, l.srcDir) {
		return nil, fmt.Errorf("proto path must be under %s, got=%s", l.srcDir, name)
	}
	*l.protoFilesLoaded = append(*l.protoFilesLoaded, strings.TrimPrefix(name, l.srcDir))
	return openFile(name)
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
	for _, protoFile := range *l.protoFilesLoaded {
		validatorFile := protoFile + consts.ValidatorExtensionSuffix
		validatorAbsPath := filepath.Join(l.srcDir, validatorFile)
		if exists, isDir, err := stat(validatorAbsPath); err != nil {
			return nil, err
		} else if isDir {
			return nil, fmt.Errorf("expected validator file and not a directory, file=%s", validatorAbsPath)
		} else if !exists {
			continue
		}
		thread := &starlark.Thread{
			Print: starPrint,
			Load:  l.Load,
		}

		if _, err := l.Load(thread, filepath.ToSlash(validatorFile)); err != nil {
			return nil, err
		}
	}

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

	jsonData, err := ioutil.ReadAll(configReader)
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

	fileDescriptor, err := l.loadProtoDescriptors(configJSON.ProtoFile)
	if err != nil {
		return nil, err
	}

	protoconfValue := &pc.ProtoconfValue{}
	um := &protojson.UnmarshalOptions{Resolver: protoregistry.GlobalTypes}
	if err = um.Unmarshal(jsonData, protoconfValue); err != nil {
		return nil, fmt.Errorf("error unmarshaling, err=%s", err)
	}

	desc, err := protoregistry.GlobalTypes.FindMessageByName(protoconfValue.Value.MessageName())
	if err != nil {
		return nil, err
	}
	value := dynamicpb.NewMessage(desc.Descriptor())

	if err = anypb.UnmarshalTo(protoconfValue.Value, value, gproto.UnmarshalOptions{}); err != nil {
		return nil, err
	}

	message := dynamic.NewMessage(fileDescriptor.FindMessage(string(protoconfValue.Value.MessageName())))
	message.UnmarshalMergeText([]byte(value.String()))

	globals := starlark.StringDict{}
	globals["value"] = proto.NewStarProtoMessage(message)
	return globals, nil
}

func (l *starlarkLoader) loadProtoDescriptors(modulePath string) (*desc.FileDescriptor, error) {
	parser := &protoparse.Parser{ImportPaths: []string{l.srcDir}, Accessor: l.protoAccessor}
	descriptors, err := parser.ParseFiles(modulePath)
	if err != nil {
		return nil, fmt.Errorf("error parsing proto file, file=%s err=%v", modulePath, err)
	}
	fileDescriptor := descriptors[0]
	fileOpts := protodesc.FileOptions{AllowUnresolvable: true}
	fd, err := fileOpts.New(fileDescriptor.AsFileDescriptorProto(), protoregistry.GlobalFiles)
	if err != nil {
		return nil, err
	}
	for i := 0; i < fd.Messages().Len(); i++ {
		md := fd.Messages().Get(i)
		if _, e := protoregistry.GlobalTypes.FindMessageByName(md.FullName()); e != nil {
			d := dynamicpb.NewMessage(md)
			protoregistry.GlobalTypes.RegisterMessage(d.Type())
		}
	}
	return fileDescriptor, nil
}

func (l *starlarkLoader) loadProto(modulePath string) (starlark.StringDict, error) {
	globals := starlark.StringDict{}
	fileDescriptor, err := l.loadProtoDescriptors(modulePath)
	if err != nil {
		return nil, err
	}
	for _, message := range fileDescriptor.GetMessageTypes() {
		globals[message.GetName()] = proto.NewMessageType(message)
	}
	return globals, err
}

func (l *starlarkLoader) loadStarlark(thread *starlark.Thread, modulePath string) (starlark.StringDict, error) {
	reader, err := openFile(filepath.Join(l.srcDir, modulePath))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	moduleSource, err := ioutil.ReadAll(reader)
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
