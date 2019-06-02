package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/golang/protobuf/jsonpb"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// A Starlark built-in type representing a Protobuf message. Provides attributes
// for accessing message fields using their original protobuf names.
type starProtoMessage struct {
	msg    *dynamic.Message
	desc   *desc.MessageDescriptor
	frozen bool

	// lets the message wrapper keep track of per-field wrappers, for freezing.
	attrCache map[string]starlark.Value
}

var _ starlark.HasAttrs = (*starProtoMessage)(nil)
var _ starlark.HasSetField = (*starProtoMessage)(nil)
var _ starlark.Comparable = (*starProtoMessage)(nil)

func (msg *starProtoMessage) String() string {
	return fmt.Sprintf("<%s %s>", msg.Type(), msg.msg)
}
func (msg *starProtoMessage) Type() string         { return msg.msg.XXX_MessageName() }
func (msg *starProtoMessage) Truth() starlark.Bool { return starlark.True }

func (msg *starProtoMessage) Freeze() {
	if !msg.frozen {
		msg.frozen = true
		for _, attr := range msg.attrCache {
			attr.Freeze()
		}
	}
}

func (msg *starProtoMessage) CompareSameType(op syntax.Token, y starlark.Value, depth int) (bool, error) {
	other, ok := y.(*starProtoMessage)
	if !ok {
		return false, nil
	}

	switch op {
	case syntax.EQL:
		eql := dynamic.Equal(msg.msg, other.msg)
		return eql, nil
	case syntax.NEQ:
		eql := dynamic.Equal(msg.msg, other.msg)
		return !eql, nil
	default:
		return false, fmt.Errorf("Only == and != operations are supported on protobufs, found %s", op.String())
	}
}

func (msg *starProtoMessage) Hash() (uint32, error) {
	return 0, fmt.Errorf("starProtoMessage.Hash: TODO")
}

func (msg *starProtoMessage) MarshalJSON() ([]byte, error) {
	var jsonMarshaler = &jsonpb.Marshaler{OrigName: true}
	jsonData, err := msg.msg.MarshalJSONPB(jsonMarshaler)
	if err != nil {
		return nil, err
	}
	return []byte(jsonData), nil
}

func newStarProtoMessage(msg *dynamic.Message) *starProtoMessage {
	wrapper := &starProtoMessage{
		msg:       msg,
		desc:      msg.GetMessageDescriptor(),
		attrCache: make(map[string]starlark.Value),
	}

	return wrapper
}

func toProtoMessage(val starlark.Value) (*dynamic.Message, bool) {
	if msg, ok := val.(*starProtoMessage); ok {
		return msg.msg, true
	}
	return nil, false
}

func (msg *starProtoMessage) checkMutable(verb string) error {
	if msg.frozen {
		return fmt.Errorf("cannot %s frozen message", verb)
	}
	return nil
}

func (msg *starProtoMessage) Attr(name string) (starlark.Value, error) {
	if attr, ok := msg.attrCache[name]; ok {
		return attr, nil
	}
	field := msg.desc.FindFieldByName(name)
	if field == nil {
		return nil, fmt.Errorf("InternalError: field %s not found in message %s", name, msg.desc)
	}

	val := &fieldValue{
		desc: field,
		msg:  msg.msg,
	}
	out := valueToStarlark(val)
	if msg.frozen {
		out.Freeze()
	}
	msg.attrCache[name] = out
	return out, nil
}

func (msg *starProtoMessage) AttrNames() []string {
	var names []string
	for _, field := range msg.desc.GetFields() {
		names = append(names, field.GetName())
	}
	sort.Strings(names)
	return names
}

func (msg *starProtoMessage) SetField(name string, star starlark.Value) error {
	field := msg.desc.FindFieldByName(name)
	if field == nil {
		return fmt.Errorf("InternalError: field %s not found in message %s", name, msg.desc)
	}

	val, err := valueFromStarlark(field, star)
	if err != nil {
		return err
	}

	if err := msg.checkMutable("set field of"); err != nil {
		return err
	}

	if oneof := field.GetOneOf(); oneof != nil {
		for _, choice := range oneof.GetChoices() {
			delete(msg.attrCache, choice.GetName()) // FIXME: is GetName the right name?
		}
	} else {
		delete(msg.attrCache, name)
	}
	msg.msg.SetField(field, val)
	return nil
}

type fieldValue struct {
	desc *desc.FieldDescriptor
	msg  *dynamic.Message
}

func valueToStarlark(val *fieldValue) starlark.Value {
	if val.desc.IsRepeated() {
		var items []starlark.Value
		length := len(val.msg.GetField(val.desc).([]interface{}))
		for i := 0; i < length; i++ {
			items = append(items, scalarToStarlark(val.desc, val.msg.GetRepeatedField(val.desc, i)))
		}
		return &protoRepeated{
			field: val,
			list:  starlark.NewList(items),
		}
	}

	if val.desc.IsMap() {
		dict := &starlark.Dict{}
		keyType := val.desc.GetMapKeyType()
		valueType := val.desc.GetMapValueType()
		mp := val.msg.GetField(val.desc).(map[interface{}]interface{})
		for key, value := range mp {
			key := scalarToStarlark(keyType, key)
			elem := scalarToStarlark(valueType, value)
			if err := dict.SetKey(key, elem); err != nil {
				panic(fmt.Sprintf("dict.SetKey(%s, %s): %v", key, elem, err))
			}
		}
		return &protoMap{
			field: val,
			dict:  dict,
		}
	}
	return scalarToStarlark(val.desc, val.msg.GetField(val.desc))
}

func scalarToStarlark(t *desc.FieldDescriptor, val interface{}) starlark.Value {
	switch t.GetType() {
	case dpb.FieldDescriptorProto_TYPE_INT32, dpb.FieldDescriptorProto_TYPE_SINT32, dpb.FieldDescriptorProto_TYPE_SFIXED32:
		return starlark.MakeInt64(int64(val.(int32)))
	case dpb.FieldDescriptorProto_TYPE_INT64, dpb.FieldDescriptorProto_TYPE_SINT64, dpb.FieldDescriptorProto_TYPE_SFIXED64:
		return starlark.MakeInt64(val.(int64))
	case dpb.FieldDescriptorProto_TYPE_UINT32, dpb.FieldDescriptorProto_TYPE_FIXED32:
		return starlark.MakeUint64(uint64(val.(uint32)))
	case dpb.FieldDescriptorProto_TYPE_UINT64, dpb.FieldDescriptorProto_TYPE_FIXED64:
		return starlark.MakeUint64(val.(uint64))
	case dpb.FieldDescriptorProto_TYPE_FLOAT:
		return starlark.Float(val.(float32))
	case dpb.FieldDescriptorProto_TYPE_DOUBLE:
		return starlark.Float(val.(float64))
	case dpb.FieldDescriptorProto_TYPE_STRING:
		return starlark.String(val.(string))
	case dpb.FieldDescriptorProto_TYPE_BYTES:
		return starlark.String(string(val.([]byte)))
	case dpb.FieldDescriptorProto_TYPE_BOOL:
		return starlark.Bool(val.(bool))
	case dpb.FieldDescriptorProto_TYPE_ENUM:
		return &starProtoEnumValue{desc: t.GetEnumType().FindValueByNumber(val.(int32))}
	case dpb.FieldDescriptorProto_TYPE_MESSAGE:
		return newStarProtoMessage(val.(*dynamic.Message))
	}

	// This should be impossible, because the set of types present
	// in a generated protobuf struct is small and limited.
	panic(fmt.Errorf("valueToStarlark: unknown type %v", t))
}

func valueFromStarlark(t *desc.FieldDescriptor, star starlark.Value) (interface{}, error) {
	switch star := star.(type) {
	case starlark.Int:
		switch t.GetType() {
		case dpb.FieldDescriptorProto_TYPE_INT64, dpb.FieldDescriptorProto_TYPE_SINT64:
			if val, ok := star.Int64(); ok {
				return val, nil
			}
			return nil, fmt.Errorf("ValueError: value %v overflows type `int64'", star)
		case dpb.FieldDescriptorProto_TYPE_UINT64, dpb.FieldDescriptorProto_TYPE_SFIXED64:
			if val, ok := star.Uint64(); ok {
				return val, nil
			}
			return nil, fmt.Errorf("ValueError: value %v overflows type `uint64'", star)
		case dpb.FieldDescriptorProto_TYPE_INT32, dpb.FieldDescriptorProto_TYPE_SINT32:
			if val, ok := star.Int64(); ok && val >= math.MinInt32 && val <= math.MaxInt32 {
				return int32(val), nil
			}
			return nil, fmt.Errorf("ValueError: value %v overflows type `int32'", star)
		case dpb.FieldDescriptorProto_TYPE_UINT32, dpb.FieldDescriptorProto_TYPE_SFIXED32:
			if val, ok := star.Uint64(); ok && val <= math.MaxUint32 {
				return uint32(val), nil
			}
			return nil, fmt.Errorf("ValueError: value %v overflows type `int32'", star)
		}
	case starlark.Float:
		switch t.GetType() {
		case dpb.FieldDescriptorProto_TYPE_DOUBLE:
			if val, ok := starlark.AsFloat(star); ok {
				return val, nil
			}
		case dpb.FieldDescriptorProto_TYPE_FLOAT:
			if val, ok := starlark.AsFloat(star); ok {
				return float32(val), nil
			}
		}
	case starlark.String:
		switch t.GetType() {
		case dpb.FieldDescriptorProto_TYPE_STRING:
			return string(star), nil
		case dpb.FieldDescriptorProto_TYPE_BYTES:
			return []byte(string(star)), nil
		}
	case starlark.Bool:
		if t.GetType() == dpb.FieldDescriptorProto_TYPE_BOOL {
			return bool(star), nil
		}
	case starlark.NoneType:
	case *starProtoEnumValue:
		return enumFromStarlark(t.GetEnumType(), star)
	case *starProtoMessage:
		if t.GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
			return star.msg, nil
		}
	case *protoRepeated:
		return valueFromStarlark(t, star.list)
	case *starlark.List:
		if t.IsRepeated() {
			sl := make([]interface{}, star.Len())
			for i := 0; i < star.Len(); i++ {
				elem, err := valueFromStarlark(t, star.Index(i))
				if err != nil {
					return nil, err
				}
				sl[i] = elem
			}
			return sl, nil
		}
	case *protoMap:
		return valueFromStarlark(t, star.dict)
	case *starlark.Dict:
		if t.IsMap() {
			keyType := t.GetMapKeyType()
			valueType := t.GetMapValueType()
			mp := map[interface{}]interface{}{}
			for _, item := range star.Items() {
				key, err := valueFromStarlark(keyType, item[0])
				if err != nil {
					return nil, err
				}
				value, err := valueFromStarlark(valueType, item[1])
				if err != nil {
					return nil, err
				}
				mp[key] = value
			}
			return mp, nil
		}
	}

	return nil, typeError(t, star)
}

func typeName(t *desc.FieldDescriptor) string {
	// FIXME: Print scalar type for scalar, message name for message/enum, think about lists/maps etc.
	return "FIX_ME typeName"

	// Special-case protobuf types to get more useful error messages when
	// the wrong protobuf type is assigned.
	// messageType := reflect.TypeOf((*proto.Message)(nil)).Elem()
	// if t.Implements(messageType) {
	// 	return reflect.Zero(t).Interface().(proto.Message).XXX_MessageName()
	// }
	// enumType := reflect.TypeOf((*protoEnum)(nil)).Elem()
	// if t.Implements(enumType) {
	// 	return enumTypeName(reflect.Zero(t).Interface().(protoEnum))
	// }
	// if t.PkgPath() == "" {
	// 	return t.String()
	// }
	// return fmt.Sprintf("%q.%s", t.PkgPath(), t.Name())
}

func typeError(t *desc.FieldDescriptor, star starlark.Value) error {
	return fmt.Errorf("TypeError: value %s (type `%s') can't be assigned to type `%s'",
		star.String(), star.Type(), typeName(t))
}

type protoRepeated struct {
	field *fieldValue
	list  *starlark.List
}

var _ starlark.Value = (*protoRepeated)(nil)
var _ starlark.Iterable = (*protoRepeated)(nil)
var _ starlark.Sequence = (*protoRepeated)(nil)
var _ starlark.Indexable = (*protoRepeated)(nil)
var _ starlark.HasAttrs = (*protoRepeated)(nil)
var _ starlark.HasSetIndex = (*protoRepeated)(nil)
var _ starlark.HasBinary = (*protoRepeated)(nil)
var _ starlark.Comparable = (*protoRepeated)(nil)

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

var listMethods = map[string]func(*protoRepeated) starlark.Value{
	"clear":  (*protoRepeated).wrapClear,
	"append": (*protoRepeated).wrapAppend,
	"extend": (*protoRepeated).wrapExtend,
	// "insert": (*protoRepeated).wrapInsert,
	// "pop":    (*protoRepeated).wrapPop,
	// "remove": (*protoRepeated).wrapRemove,
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

type protoMap struct {
	field *fieldValue
	dict  *starlark.Dict
}

var _ starlark.Value = (*protoMap)(nil)
var _ starlark.Iterable = (*protoMap)(nil)
var _ starlark.Sequence = (*protoMap)(nil)
var _ starlark.HasAttrs = (*protoMap)(nil)
var _ starlark.HasSetKey = (*protoMap)(nil)
var _ starlark.Comparable = (*protoMap)(nil)

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
