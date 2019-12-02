package lib

import (
	"fmt"
	"log"

	"github.com/protoconf/protoconf/compiler/proto"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

func getModules() starlark.StringDict {
	return starlark.StringDict{
		"fail":   starlark.NewBuiltin("fail", starFail),
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
	}
}

func starPrint(t *starlark.Thread, msg string) {
	log.Printf("[%v] %s", t.CallFrame(1).Pos, msg)
}

func starFail(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var msg string
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &msg); err != nil {
		return nil, err
	}
	callStack := t.CallStack()
	callStack.Pop()
	return nil, fmt.Errorf("[%s] %s\n%s", callStack.At(0).Pos, msg, callStack.String())
}

func starAddValidator(mp *map[string]*starlark.Function) func(*starlark.Thread, *starlark.Builtin, starlark.Tuple, []starlark.Tuple) (starlark.Value, error) {
	addValidator := func(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var arg1 starlark.Value
		var arg2 starlark.Value
		if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 2, &arg1, &arg2); err != nil {
			return nil, err
		}

		messageName, ok := proto.MessageTypeName(arg1)
		if !ok {
			return nil, fmt.Errorf("expected a proto message type, got=%v", arg1)
		}

		validator, ok := arg2.(*starlark.Function)
		if ok {
			if numParams := validator.NumParams(); numParams != 1 {
				return nil, fmt.Errorf("expected a function that get 1 param, got=%d", numParams)
			}
		} else {
			return nil, fmt.Errorf("expected a function, got=%v", validator)
		}

		(*mp)[messageName] = validator

		return starlark.None, nil
	}
	return addValidator
}
