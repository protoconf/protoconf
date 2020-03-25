package golangimporter

import (
	"log"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
)

var wkt = make(map[string]*builder.FileBuilder)

func init() {
	durationFile, err := desc.LoadFileDescriptor("google/protobuf/duration.proto")
	if err != nil {
		log.Fatal(err)
	}
	durationBuilder, err := builder.FromFile(durationFile)
	if err != nil {
		log.Fatal(err)
	}
	wkt["duration"] = durationBuilder

	structFile, err := desc.LoadFileDescriptor("google/protobuf/struct.proto")
	if err != nil {
		log.Fatal(err)
	}
	structBuilder, err := builder.FromFile(structFile)
	if err != nil {
		log.Fatal(err)
	}
	wkt["struct"] = structBuilder
}

func goFieldTypeToProtoFieldType(x string) *builder.FieldType {
	x = strings.Replace(x, "*", "", -1)
	switch x {
	case "bool":
		return builder.FieldTypeBool()
	case "byte":
		return builder.FieldTypeBytes()
	case "float":
		return builder.FieldTypeFloat()
	case "float64":
		return builder.FieldTypeFloat()
	case "int":
		return builder.FieldTypeInt32()
	case "int8":
		return builder.FieldTypeInt32()
	case "int32":
		return builder.FieldTypeInt32()
	case "int64":
		return builder.FieldTypeInt64()
	case "uint":
		return builder.FieldTypeUInt32()
	case "uint8":
		return builder.FieldTypeUInt32()
	case "uint16":
		return builder.FieldTypeUInt32()
	case "uint32":
		return builder.FieldTypeUInt32()
	case "uint64":
		return builder.FieldTypeUInt64()
	case "string":
		return builder.FieldTypeString()
	case "time.Duration":
		return builder.FieldTypeMessage(wkt["duration"].GetMessage("Duration"))
	case "interface{}":
		return builder.FieldTypeMessage(wkt["struct"].GetMessage("Value"))
	case "error":
		return nil
	}
	return nil
}
