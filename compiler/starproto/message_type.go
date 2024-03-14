package starproto

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/runtime/protoiface"

	"go.starlark.net/starlark"
)

func MessageTypeName(val starlark.Value) (string, bool) {
	if msg, ok := val.(*starProtoMessageType); ok {
		return msg.desc.GetFullyQualifiedName(), true
	}
	return "", false
}

func NewMessageType(desc *desc.MessageDescriptor) starlark.Value {
	mt := &starProtoMessageType{
		desc: desc,
	}
	return mt
}

type NewBuiltinCallback func(*dynamic.Message) error

func NewBuiltin(msg protoiface.MessageV1, callbacks ...NewBuiltinCallback) *starlark.Builtin {
	d, _ := desc.LoadMessageDescriptorForMessage(msg)
	starMsg, _ := NewMessageType(d).(starlark.Callable)

	f := func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		v, err := starMsg.CallInternal(thread, args, kwargs)
		if err != nil {
			return nil, err
		}
		msg, ok := ToProtoMessage(v)
		if !ok {
			return nil, errors.New("failed to create proto message")
		}
		for _, c := range callbacks {
			err := c(msg)
			if err != nil {
				return nil, err
			}
		}

		return NewStarProtoMessage(msg), nil
	}

	return starlark.NewBuiltin(starMsg.Name(), f)
}

// A Starlark built-in type representing a Protobuf message type. This is the
// message type itself rather than any particular message value.
type starProtoMessageType struct {
	desc *desc.MessageDescriptor
}

func (mt *starProtoMessageType) String() string {
	return fmt.Sprintf("<proto.MessageType %s>", mt.Name())
}
func (mt *starProtoMessageType) Type() string         { return mt.desc.GetFullyQualifiedName() }
func (mt *starProtoMessageType) Freeze()              {}
func (mt *starProtoMessageType) Truth() starlark.Bool { return starlark.True }
func (mt *starProtoMessageType) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable type: %s", mt.Type())
}

func (mt *starProtoMessageType) Name() string {
	return mt.desc.GetName()
}

func (mt *starProtoMessageType) Attr(attrName string) (starlark.Value, error) {
	for _, enum := range mt.desc.GetNestedEnumTypes() {
		if attrName == enum.GetName() {
			return NewEnumType(enum), nil
		}
	}

	for _, message := range mt.desc.GetNestedMessageTypes() {
		if attrName == message.GetName() {
			return NewMessageType(message), nil
		}
	}

	// FIXME: iterate nested extensions as well?
	return nil, nil
}

func (mt *starProtoMessageType) AttrNames() []string {
	// FIXME: fields, nested enum/message types, nested extensions?
	return nil
}

func (mt *starProtoMessageType) CallInternal(thread *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// This is semantically the constructor of a protobuf message, and we
	// want it to accept only kwargs (where keys are protobuf field names).
	// Inject a useful error message if a user tries to pass positional args.
	if err := starlark.UnpackPositionalArgs(mt.Name(), args, nil, 0); err != nil {
		return nil, err
	}

	wrapper := NewStarProtoMessage(dynamic.NewMessage(mt.desc))

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

var (
	_ starlark.HasAttrs = (*starProtoMessageType)(nil)
	_ starlark.Callable = (*starProtoMessageType)(nil)
)
