package wktbuilders

import (
	"log/slog"
	"os"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
)

var (
	DurationBuilder = fromFile("google/protobuf/duration.proto")
	StructBuilder   = fromFile("google/protobuf/struct.proto")
	AnyBuilder      = fromFile("google/protobuf/any.proto")
)

func fromFile(fileName string) *builder.FileBuilder {
	fileDesc, err := desc.LoadFileDescriptor(fileName)
	if err != nil {
		slog.Error("error", err)
		os.Exit(1)
	}
	builder, err := builder.FromFile(fileDesc)
	if err != nil {
		slog.Error("error", err)
		os.Exit(1)
	}
	return builder
}
