package utils

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"

	_ "github.com/protoconf/proto-validate-reflect/validate"
	"github.com/protoconf/protoconf/consts"
	protoconfvalue "github.com/protoconf/protoconf/datatypes/proto/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	localResolver *protoregistry.Types
)

// ReadConfig reads a materialized config
func ReadConfig(protoconfRoot string, configName string) (*protoconfvalue.ProtoconfValue, error) {
	filename := filepath.Join(protoconfRoot, consts.CompiledConfigPath, configName+consts.CompiledConfigExtension)

	configReader, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file, file=%s", filename)
	}

	if localResolver == nil {
		localResolver = LocalResolver(protoconfRoot)
	}

	val := &protoconfvalue.ProtoconfValue{}
	err = protojson.UnmarshalOptions{Resolver: localResolver}.Unmarshal(configReader, val)
	if err != nil {
		return nil, err
	}
	return val, nil

}

func LocalResolver(protoconfRoot string) *protoregistry.Types {
	return localLinkedResolver(protoconfRoot, false)
}

func LocalLinkedResolver(protoconfRoot string) *protoregistry.Types {
	return localLinkedResolver(protoconfRoot, true)
}

func localLinkedResolver(protoconfRoot string, link bool) *protoregistry.Types {

	localTypes := new(protoregistry.Types)
	localFiles, err := LoadLocalProtoFiles(protoconfRoot, link)
	if err != nil {
		log.Fatal("LocalResolver:", err)
	}
	localFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		_, err := protoregistry.GlobalFiles.FindFileByPath(file.Path())
		if err != nil {
			protoregistry.GlobalFiles.RegisterFile(file)
		}
		return true
	})
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if fd.Messages().Len() > 0 {
			for i := 0; i < fd.Messages().Len(); i++ {
				fdm := fd.Messages().Get(i)
				msg := dynamicpb.NewMessageType(fdm)
				localTypes.RegisterMessage(msg)
			}
		}
		if fd.Enums().Len() > 0 {
			for i := 0; i < fd.Enums().Len(); i++ {
				enum := fd.Enums().Get(i)
				localTypes.RegisterEnum(dynamicpb.NewEnumType(enum))
			}
		}
		return true
	})
	return localTypes
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

func find(root, ext string) []string {
	var a []string
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ext {
			a = append(a, strings.TrimPrefix(s, root+"/"))
		}
		return nil
	})
	return a
}

func LoadLocalProtoFiles(root string, link bool) (*protoregistry.Files, error) {
	rootPath := filepath.Join(root, consts.SrcPath)
	files := find(rootPath, ".proto")
	parser := &protoparse.Parser{
		ImportPaths:                     []string{rootPath},
		InterpretOptionsInUnlinkedFiles: true,
		LookupImport:                    desc.LoadFileDescriptor,
	}
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{}}

	if link {
		descriptors, err := parser.ParseFiles(files...)
		if err != nil {
			return nil, fmt.Errorf("parser: %v", err)
		}
		for _, fd := range descriptors {
			fds.File = append(fds.File, fd.AsFileDescriptorProto())
		}

	} else {
		descriptors, err := parser.ParseFilesButDoNotLink(files...)
		if err != nil {
			return nil, fmt.Errorf("parser: %v", err)
		}
		fds.File = descriptors

	}

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if strings.HasPrefix(fd.Path(), "google") {
			fds.File = append(fds.File,
				protodesc.ToFileDescriptorProto(fd),
			)
		}
		return true
	})
	fileoptions := &protodesc.FileOptions{AllowUnresolvable: true}
	protoFiles, err := fileoptions.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("new files: %v", err)
	}
	return protoFiles, err

}
