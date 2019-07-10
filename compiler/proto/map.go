package proto

import (
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type protoMap struct {
	field *fieldValue
	dict  *starlark.Dict
}

var dictMethods = map[string]func(*protoMap) starlark.Value{
	"clear": (*protoMap).wrapClear,
	"get":   nil,
	"items": nil,
	"keys":  nil,
	// "pop":        (*protoMap).wrapPop,
	// "popitem":    (*protoMap).wrapPopItem,
	"setdefault": (*protoMap).wrapSetDefault,
	"update":     (*protoMap).wrapUpdate,
	"values":     nil,
}

func (m *protoMap) Attr(name string) (starlark.Value, error) {
	wrapper, ok := dictMethods[name]
	if !ok {
		return nil, nil
	}
	if wrapper != nil {
		return wrapper(m), nil
	}
	return m.dict.Attr(name)
}

func (m *protoMap) AttrNames() []string                                { return m.dict.AttrNames() }
func (m *protoMap) Freeze()                                            { m.dict.Freeze() }
func (m *protoMap) Hash() (uint32, error)                              { return m.dict.Hash() }
func (m *protoMap) Get(k starlark.Value) (starlark.Value, bool, error) { return m.dict.Get(k) }
func (m *protoMap) Iterate() starlark.Iterator                         { return m.dict.Iterate() }
func (m *protoMap) Len() int                                           { return m.dict.Len() }
func (m *protoMap) String() string                                     { return m.dict.String() }
func (m *protoMap) Truth() starlark.Bool                               { return m.dict.Truth() }

func (m *protoMap) Type() string {
	return fmt.Sprintf("map<%s, %s>", typeName(m.field.desc.GetMapKeyType()), typeName(m.field.desc.GetMapValueType()))

}

func (m *protoMap) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other, ok := y.(*protoMap)
	if !ok {
		return false, nil
	}

	return starlark.CompareDepth(op, m.dict, other.dict, depth)
}

func (m *protoMap) wrapClear() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		if err := starlark.UnpackPositionalArgs("clear", args, kwargs, 0); err != nil {
			return nil, err
		}
		if err := m.dict.Clear(); err != nil {
			return nil, err
		}
		m.field.msg.ClearField(m.field.desc)
		return starlark.None, nil
	}
	return starlark.NewBuiltin("clear", impl).BindReceiver(m)
}

func (m *protoMap) wrapSetDefault() starlark.Value {
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var key, defaultValue starlark.Value = nil, starlark.None
		if err := starlark.UnpackPositionalArgs("setdefault", args, kwargs, 1, &key, &defaultValue); err != nil {
			return nil, err
		}
		if val, ok, err := m.dict.Get(key); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return defaultValue, m.SetKey(key, defaultValue)
	}
	return starlark.NewBuiltin("setdefault", impl).BindReceiver(m)
}

func (m *protoMap) wrapUpdate() starlark.Value {
	keyType := m.field.desc.GetMapKeyType()
	valueType := m.field.desc.GetMapValueType()
	impl := func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// Use the underlying starlark `dict.update()` to get a Dict containing
		// all the new values, so we don't have to recreate the API here. After
		// the temp dict is constructed, type check.
		tempDict := &starlark.Dict{}
		tempUpdate, _ := tempDict.Attr("update")
		if _, err := starlark.Call(thread, tempUpdate, args, kwargs); err != nil {
			return nil, err
		}
		for _, item := range tempDict.Items() {
			if item[0] == starlark.None {
				return nil, typeError(keyType, item[0])
			}
			if item[1] == starlark.None {
				return nil, typeError(valueType, item[1])
			}
		}

		// Update the Dict first to catch potential immutability.
		for _, item := range tempDict.Items() {
			if err := m.dict.SetKey(item[0], item[1]); err != nil {
				return nil, err
			}
		}

		for _, item := range tempDict.Items() {
			if err := m.SetKey(item[0], item[1]); err != nil {
				return nil, err
			}
		}
		return starlark.None, nil
	}
	return starlark.NewBuiltin("update", impl).BindReceiver(m)
}

func (m *protoMap) SetKey(k, v starlark.Value) error {
	keyType := m.field.desc.GetMapKeyType()
	if k == starlark.None {
		return typeError(keyType, k)
	}
	valueType := m.field.desc.GetMapValueType()
	if v == starlark.None {
		return typeError(valueType, v)
	}
	goKey, err := valueFromStarlark(keyType, k)
	if err != nil {
		return err
	}
	goVal, err := valueFromStarlark(valueType, v)
	if err != nil {
		return err
	}
	if err := m.dict.SetKey(k, v); err != nil {
		return err
	}
	m.field.msg.PutMapField(m.field.desc, goKey, goVal)
	return nil
}

var (
	_ starlark.Value      = (*protoMap)(nil)
	_ starlark.Iterable   = (*protoMap)(nil)
	_ starlark.Sequence   = (*protoMap)(nil)
	_ starlark.HasAttrs   = (*protoMap)(nil)
	_ starlark.HasSetKey  = (*protoMap)(nil)
	_ starlark.Comparable = (*protoMap)(nil)
)
