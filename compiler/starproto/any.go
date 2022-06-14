package starproto

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/jhump/protoreflect/dynamic"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var AnyModule = &starlarkstruct.Module{
	Name: "any",
	Members: starlark.StringDict{
		"new":    starlark.NewBuiltin("new", newAny),
		"unpack": starlark.NewBuiltin("unpack", unpackAny),
	},
}

func newAny(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg *starProtoMessage
	err := starlark.UnpackPositionalArgs("new", args, kwargs, 1, &msg)
	if err != nil {
		return nil, err
	}
	any, err := ptypes.MarshalAny(msg.msg)
	if err != nil {
		return nil, err
	}
	d, err := dynamic.AsDynamicMessage(any)
	if err != nil {
		return nil, err
	}
	return NewStarProtoMessage(d), nil
}

func unpackAny(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var anysrc *starProtoMessage
	var target *starProtoMessage
	err := starlark.UnpackPositionalArgs("new", args, kwargs, 2, &anysrc, &target)
	if err != nil {
		return nil, err
	}
	v, err := anysrc.msg.TryGetFieldByName("value")
	if err != nil {
		return nil, err
	}
	s, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("cannot read message from any")
	}
	err = target.msg.UnmarshalMerge(s)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal any: %v", err)
	}
	return target, nil
}
