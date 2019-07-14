package proto

import (
	"fmt"
	"math"

	pbproto "github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"go.starlark.net/starlark"
)

type fieldValue struct {
	desc *desc.FieldDescriptor
	msg  *dynamic.Message
}

func valueToStarlark(val *fieldValue) starlark.Value {
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
		message, err := dynamic.AsDynamicMessage(val.(pbproto.Message))
		if err != nil {
			panic(fmt.Errorf("scalarToStarlark: error converting proto.Message to dynamic.Message %v", err))
		}
		return NewStarProtoMessage(message)
	}

	// This should be impossible, because the set of types present
	// in a generated Protobuf struct is small and limited.
	panic(fmt.Errorf("scalarToStarlark: unknown type %v", t))
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
		if t.GetEnumType().GetFullyQualifiedName() == star.desc.GetEnum().GetFullyQualifiedName() {
			return star.desc.GetNumber(), nil
		}
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
	return t.String()
}

func typeError(t *desc.FieldDescriptor, star starlark.Value) error {
	return fmt.Errorf("type error: value %s (type `%s') can't be assigned to type `%s'",
		star.String(), star.Type(), typeName(t))
}
