package protostdlib

import (
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/internal"
)

var ProtoStdLib map[string]*dpb.FileDescriptorProto

func init() {
	filenames := []string{}

	ProtoStdLib = map[string]*dpb.FileDescriptorProto{}
	for _, fn := range filenames {
		fd, err := internal.LoadFileDescriptor(fn)
		if err != nil {
			panic(err.Error())
		}
		ProtoStdLib[fn] = fd
	}
}
