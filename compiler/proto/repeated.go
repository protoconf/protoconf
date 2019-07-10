package proto

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type protoRepeated struct {
	field *fieldValue
	list  *starlark.List
}

var listMethods = map[string]func(*protoRepeated) starlark.Value{
	"clear":  (*protoRepeated).wrapClear,
	"append": (*protoRepeated).wrapAppend,
	"extend": (*protoRepeated).wrapExtend,
	// "insert": (*protoRepeated).wrapInsert,
	// "pop":    (*protoRepeated).wrapPop,
	// "remove": (*protoRepeated).wrapRemove,
}

func (r *protoRepeated) Attr(name string) (starlark.Value, error) {
	wrapper, ok := listMethods[name]
	if !ok {
		return nil, nil
	}
	if wrapper != nil {
		return wrapper(r), nil
	}
	return r.list.Attr(name)
}

func (r *protoRepeated) AttrNames() []string                 { return r.list.AttrNames() }
func (r *protoRepeated) Freeze()                             { r.list.Freeze() }
func (r *protoRepeated) Hash() (uint32, error)               { return r.list.Hash() }
func (r *protoRepeated) Index(i int) starlark.Value          { return r.list.Index(i) }
func (r *protoRepeated) Iterate() starlark.Iterator          { return r.list.Iterate() }
func (r *protoRepeated) Len() int                            { return r.list.Len() }
func (r *protoRepeated) Slice(x, y, step int) starlark.Value { return r.list.Slice(x, y, step) }
func (r *protoRepeated) String() string                      { return r.list.String() }
func (r *protoRepeated) Truth() starlark.Bool                { return r.list.Truth() }

func (r *protoRepeated) Type() string {
	return fmt.Sprintf("list<%s>", typeName(r.field.desc))
}

func (r *protoRepeated) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other, ok := y.(*protoRepeated)
	if !ok {
		return false, nil
	}

	return starlark.CompareDepth(op, r.list, other.list, depth)
}

func (r *protoRepeated) wrapClear() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := starlark.UnpackPositionalArgs("clear", args, kwargs, 0); err != nil {
			return nil, err
		}
		if err := r.Clear(); err != nil {
			return nil, err
		}
		return starlark.None, nil
	}
	return starlark.NewBuiltin("clear", impl).BindReceiver(r)
}

func (r *protoRepeated) wrapAppend() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var val starlark.Value
		if err := starlark.UnpackPositionalArgs("append", args, kwargs, 1, &val); err != nil {
			return nil, err
		}
		if err := r.Append(val); err != nil {
			return nil, err
		}
		return starlark.None, nil
	}
	return starlark.NewBuiltin("append", impl).BindReceiver(r)
}

func (r *protoRepeated) wrapExtend() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var val starlark.Iterable
		if err := starlark.UnpackPositionalArgs("extend", args, kwargs, 1, &val); err != nil {
			return nil, err
		}
		if err := r.implExtend(thread, val); err != nil {
			return nil, err
		}
		return starlark.None, nil
	}
	return starlark.NewBuiltin("extend", impl).BindReceiver(r)
}

func (r *protoRepeated) Clear() error {
	if err := r.list.Clear(); err != nil {
		return err
	}
	r.field.msg.ClearField(r.field.desc)
	return nil
}

func (r *protoRepeated) Append(v starlark.Value) error {
	if v == starlark.None {
		return typeError(r.field.desc, v)
	}
	goVal, err := valueFromStarlark(r.field.desc, v)
	if err != nil {
		return err
	}
	if err := r.list.Append(v); err != nil {
		return err
	}
	r.field.msg.AddRepeatedField(r.field.desc, goVal)
	return nil
}

func (r *protoRepeated) implExtend(t *starlark.Thread, iterable starlark.Iterable) error {
	var starValues []starlark.Value
	var goValues []interface{}
	iter := iterable.Iterate()
	defer iter.Done()
	var starVal starlark.Value
	for iter.Next(&starVal) {
		if starVal == starlark.None {
			return typeError(r.field.desc, starVal)
		}
		goVal, err := valueFromStarlark(r.field.desc, starVal)
		if err != nil {
			return err
		}
		starValues = append(starValues, starVal)
		goValues = append(goValues, goVal)
	}

	listExtend, _ := r.list.Attr("extend")
	args := starlark.Tuple([]starlark.Value{
		starlark.NewList(starValues),
	})
	if _, err := starlark.Call(t, listExtend, args, nil); err != nil {
		return err
	}
	for _, goVal := range goValues {
		r.field.msg.AddRepeatedField(r.field.desc, goVal)
	}
	return nil
}

func (r *protoRepeated) SetIndex(i int, v starlark.Value) error {
	if v == starlark.None {
		return typeError(r.field.desc, v)
	}
	goVal, err := valueFromStarlark(r.field.desc, v)
	if err != nil {
		return err
	}
	if err := r.list.SetIndex(i, v); err != nil {
		return err
	}
	r.field.msg.SetRepeatedField(r.field.desc, i, goVal)
	return nil
}

func (r *protoRepeated) Binary(op syntax.Token, y starlark.Value, side starlark.Side) (starlark.Value, error) {
	if op == syntax.PLUS {
		if side == starlark.Left {
			switch y := y.(type) {
			case *starlark.List:
				return starlark.Binary(op, r.list, y)
			case *protoRepeated:
				return starlark.Binary(op, r.list, y.list)
			}
			return nil, nil
		}
		if side == starlark.Right {
			if _, ok := y.(*starlark.List); ok {
				return starlark.Binary(op, y, r.list)
			}
			return nil, nil
		}
	}
	return nil, nil
}

var (
	_ starlark.Value       = (*protoRepeated)(nil)
	_ starlark.Iterable    = (*protoRepeated)(nil)
	_ starlark.Sequence    = (*protoRepeated)(nil)
	_ starlark.Indexable   = (*protoRepeated)(nil)
	_ starlark.HasAttrs    = (*protoRepeated)(nil)
	_ starlark.HasSetIndex = (*protoRepeated)(nil)
	_ starlark.HasBinary   = (*protoRepeated)(nil)
	_ starlark.Comparable  = (*protoRepeated)(nil)
)
