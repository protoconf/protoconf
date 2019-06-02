package main

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/msgregistry"

	"go.starlark.net/starlark"
)

func newMessageType(registry *msgregistry.MessageRegistry, name string) (starlark.Value, error) {
	desc, err := registry.FindMessageTypeByUrl(name)
	if err != nil {
		return nil, fmt.Errorf("Protobuf message type %q not found", name)
	}

	mt := &starProtoMessageType{
		registry: registry,
		desc:     desc,
	}
	return mt, nil
}

// A Starlark built-in type representing a Protobuf message type. This is the
// message type itself rather than any particular message value.
type starProtoMessageType struct {
	registry *msgregistry.MessageRegistry
	desc     *desc.MessageDescriptor
}

var _ starlark.HasAttrs = (*starProtoMessageType)(nil)
var _ starlark.Callable = (*starProtoMessageType)(nil)

func (mt *starProtoMessageType) String() string {
	return fmt.Sprintf("<proto.MessageType %q>", mt.Name())
}
func (mt *starProtoMessageType) Type() string         { return "proto.MessageType" }
func (mt *starProtoMessageType) Freeze()              {}
func (mt *starProtoMessageType) Truth() starlark.Bool { return starlark.True }
func (mt *starProtoMessageType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", mt.Type())
}

func (mt *starProtoMessageType) Name() string {
	return mt.desc.GetName()
}

func (mt *starProtoMessageType) Attr(attrName string) (starlark.Value, error) {
	msgName := fmt.Sprintf("%s.%s", mt.desc.GetName(), attrName)
	if pkg := mt.desc.GetFile().GetPackage(); pkg != "" {
		msgName = fmt.Sprintf("%s.%s", pkg, msgName)
	}

	enum, err := mt.registry.FindEnumTypeByUrl(msgName)
	if err != nil {
		return &starProtoEnumType{desc: enum}, nil
	}

	return newMessageType(mt.registry, msgName)
}

func (mt *starProtoMessageType) AttrNames() []string {
	// FIXME
	return nil
}

func (mt *starProtoMessageType) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// This is semantically the constructor of a protobuf message, and we
	// want it to accept only kwargs (where keys are protobuf field names).
	// Inject a useful error message if a user tries to pass positional args.
	if err := starlark.UnpackPositionalArgs(mt.Name(), args, nil, 0); err != nil {
		return nil, err
	}

	wrapper := newStarProtoMessage(dynamic.NewMessage(mt.desc))

	// Parse the kwarg set into a map[string]starlark.Value, containing one
	// entry for each provided kwarg. Keys are the original protobuf field names.
	// This lets the starlark kwarg parser handle most of the error reporting,
	// except type errors which are deferred until later.
	var parserPairs []interface{}
	parsedKwargs := make(map[string]*starlark.Value, len(kwargs))

	for _, field := range mt.desc.GetFields() {
		v := new(starlark.Value)
		parsedKwargs[field.GetName()] = v
		parserPairs = append(parserPairs, field.GetName()+"?", v)
	}

	if err := starlark.UnpackArgs(mt.Name(), nil, kwargs, parserPairs...); err != nil {
		return nil, err
	}

	for fieldName, starlarkValue := range parsedKwargs {
		if *starlarkValue == nil {
			continue
		}
		if err := wrapper.SetField(fieldName, *starlarkValue); err != nil {
			return nil, err
		}
	}
	return wrapper, nil
}
