package starproto

import (
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
	"google.golang.org/protobuf/types/descriptorpb"
)

// makeFieldDescriptor creates a FieldDescriptor of the given type from a well-known proto.
// Uses google.protobuf.Duration which has int64 (seconds) and int32 (nanos) fields.
func makeDurationFieldDescriptors(t *testing.T) (secondsFD, nanosFD *desc.FieldDescriptor) {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/duration.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.Duration")
	require.NotNil(t, sym)
	msgDesc := sym.(*desc.MessageDescriptor)
	secondsFD = msgDesc.FindFieldByName("seconds")
	require.NotNil(t, secondsFD)
	nanosFD = msgDesc.FindFieldByName("nanos")
	require.NotNil(t, nanosFD)
	return
}

// makeStringFieldDescriptor loads a message with string fields.
// google.protobuf.DescriptorProto has a "name" string field.
func makeStringFieldDescriptor(t *testing.T) *desc.FieldDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/descriptor.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.DescriptorProto")
	require.NotNil(t, sym)
	msgDesc := sym.(*desc.MessageDescriptor)
	nameFD := msgDesc.FindFieldByName("name")
	require.NotNil(t, nameFD)
	return nameFD
}

// makeEnumFieldDescriptor returns a FieldDescriptor for an enum field.
// FieldDescriptorProto has a "type" field which is an enum.
func makeEnumFieldDescriptor(t *testing.T) *desc.FieldDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/descriptor.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.FieldDescriptorProto")
	require.NotNil(t, sym)
	msgDesc := sym.(*desc.MessageDescriptor)
	typeFD := msgDesc.FindFieldByName("type")
	require.NotNil(t, typeFD)
	assert.Equal(t, descriptorpb.FieldDescriptorProto_TYPE_ENUM, typeFD.GetType())
	return typeFD
}

// makeBoolFieldDescriptor returns a FieldDescriptor for a bool field.
// FieldOptions has a "deprecated" field which is bool.
func makeBoolFieldDescriptor(t *testing.T) *desc.FieldDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/descriptor.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.FieldOptions")
	require.NotNil(t, sym)
	msgDesc := sym.(*desc.MessageDescriptor)
	deprecatedFD := msgDesc.FindFieldByName("deprecated")
	require.NotNil(t, deprecatedFD)
	return deprecatedFD
}

// makeFloatFieldDescriptor returns a FieldDescriptor for a float field.
// Use FileOptions.java_outer_classname (string) ... actually use FeatureSet
// or use a descriptor with float. Instead build one via a proto with float.
// google.protobuf.Struct.Value has a number_value field of double type.
func makeDoubleFieldDescriptor(t *testing.T) *desc.FieldDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/struct.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.Value")
	require.NotNil(t, sym)
	msgDesc := sym.(*desc.MessageDescriptor)
	numberFD := msgDesc.FindFieldByName("number_value")
	require.NotNil(t, numberFD)
	return numberFD
}

// makeMessageWithField creates a dynamic.Message with a FieldDescriptor and sets a field value.
func makeMessageWithField(t *testing.T, fieldDesc *desc.FieldDescriptor) (*dynamic.Message, *fieldValue) {
	t.Helper()
	msg := dynamic.NewMessage(fieldDesc.GetOwner())
	fv := &fieldValue{
		desc: fieldDesc,
		msg:  msg,
	}
	return msg, fv
}

func TestValueToStarlark_Int64(t *testing.T) {
	secondsFD, _ := makeDurationFieldDescriptors(t)
	msg := dynamic.NewMessage(secondsFD.GetOwner())
	require.NoError(t, msg.TrySetField(secondsFD, int64(42)))

	fv := &fieldValue{desc: secondsFD, msg: msg}
	result := valueToStarlark(fv)

	intVal, ok := result.(starlark.Int)
	require.True(t, ok, "expected starlark.Int, got %T", result)
	n, ok := intVal.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(42), n)
}

func TestValueToStarlark_Int32(t *testing.T) {
	_, nanosFD := makeDurationFieldDescriptors(t)
	msg := dynamic.NewMessage(nanosFD.GetOwner())
	require.NoError(t, msg.TrySetField(nanosFD, int32(100)))

	fv := &fieldValue{desc: nanosFD, msg: msg}
	result := valueToStarlark(fv)

	intVal, ok := result.(starlark.Int)
	require.True(t, ok, "expected starlark.Int, got %T", result)
	n, ok := intVal.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(100), n)
}

func TestValueToStarlark_String(t *testing.T) {
	nameFD := makeStringFieldDescriptor(t)
	msg := dynamic.NewMessage(nameFD.GetOwner())
	require.NoError(t, msg.TrySetField(nameFD, "hello"))

	fv := &fieldValue{desc: nameFD, msg: msg}
	result := valueToStarlark(fv)

	strVal, ok := result.(starlark.String)
	require.True(t, ok, "expected starlark.String, got %T", result)
	assert.Equal(t, starlark.String("hello"), strVal)
}

func TestValueToStarlark_Bool(t *testing.T) {
	boolFD := makeBoolFieldDescriptor(t)
	msg := dynamic.NewMessage(boolFD.GetOwner())
	require.NoError(t, msg.TrySetField(boolFD, true))

	fv := &fieldValue{desc: boolFD, msg: msg}
	result := valueToStarlark(fv)

	boolVal, ok := result.(starlark.Bool)
	require.True(t, ok, "expected starlark.Bool, got %T", result)
	assert.Equal(t, starlark.True, boolVal)
}

func TestValueToStarlark_Double(t *testing.T) {
	doubleFD := makeDoubleFieldDescriptor(t)
	msg := dynamic.NewMessage(doubleFD.GetOwner())
	require.NoError(t, msg.TrySetField(doubleFD, float64(3.14)))

	fv := &fieldValue{desc: doubleFD, msg: msg}
	result := valueToStarlark(fv)

	floatVal, ok := result.(starlark.Float)
	require.True(t, ok, "expected starlark.Float, got %T", result)
	assert.InDelta(t, 3.14, float64(floatVal), 0.0001)
}

func TestValueToStarlark_Enum(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	msg := dynamic.NewMessage(enumFD.GetOwner())

	// TYPE_DOUBLE = 1 from FieldDescriptorProto_Type
	require.NoError(t, msg.TrySetField(enumFD, int32(1)))

	fv := &fieldValue{desc: enumFD, msg: msg}
	result := valueToStarlark(fv)

	enumVal, ok := result.(*starProtoEnumValue)
	require.True(t, ok, "expected *starProtoEnumValue, got %T", result)
	assert.Equal(t, int32(1), enumVal.desc.GetNumber())
}

func TestValueToStarlark_DefaultValues(t *testing.T) {
	// Test reading default (zero) values from fields
	secondsFD, nanosFD := makeDurationFieldDescriptors(t)
	msg := dynamic.NewMessage(secondsFD.GetOwner())

	// Read without setting — should return zero values as Starlark
	fvSeconds := &fieldValue{desc: secondsFD, msg: msg}
	fvNanos := &fieldValue{desc: nanosFD, msg: msg}

	secResult := valueToStarlark(fvSeconds)
	nanoResult := valueToStarlark(fvNanos)

	// Default int64 = 0
	secInt, ok := secResult.(starlark.Int)
	require.True(t, ok)
	n, ok := secInt.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(0), n)

	// Default int32 = 0
	nanoInt, ok := nanoResult.(starlark.Int)
	require.True(t, ok)
	nn, ok := nanoInt.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(0), nn)
}

func TestValueFromStarlark_Int64(t *testing.T) {
	secondsFD, _ := makeDurationFieldDescriptors(t)

	result, err := valueFromStarlark(secondsFD, starlark.MakeInt(42))
	require.NoError(t, err)
	assert.Equal(t, int64(42), result)
}

func TestValueFromStarlark_Int32(t *testing.T) {
	_, nanosFD := makeDurationFieldDescriptors(t)

	result, err := valueFromStarlark(nanosFD, starlark.MakeInt(100))
	require.NoError(t, err)
	assert.Equal(t, int32(100), result)
}

func TestValueFromStarlark_String(t *testing.T) {
	nameFD := makeStringFieldDescriptor(t)

	result, err := valueFromStarlark(nameFD, starlark.String("hello"))
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestValueFromStarlark_Bool(t *testing.T) {
	boolFD := makeBoolFieldDescriptor(t)

	result, err := valueFromStarlark(boolFD, starlark.Bool(true))
	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestValueFromStarlark_Double(t *testing.T) {
	doubleFD := makeDoubleFieldDescriptor(t)

	result, err := valueFromStarlark(doubleFD, starlark.Float(2.71))
	require.NoError(t, err)
	floatResult, ok := result.(float64)
	require.True(t, ok)
	assert.InDelta(t, 2.71, floatResult, 0.0001)
}

func TestValueFromStarlark_WrongType_StringToInt(t *testing.T) {
	secondsFD, _ := makeDurationFieldDescriptors(t)

	_, err := valueFromStarlark(secondsFD, starlark.String("not_a_number"))
	assert.Error(t, err, "string assigned to int64 field should fail")
}

func TestValueFromStarlark_WrongType_IntToString(t *testing.T) {
	nameFD := makeStringFieldDescriptor(t)

	_, err := valueFromStarlark(nameFD, starlark.MakeInt(123))
	assert.Error(t, err, "int assigned to string field should fail")
}

func TestValueFromStarlark_None(t *testing.T) {
	secondsFD, _ := makeDurationFieldDescriptors(t)

	// NoneType is a valid Starlark value but should produce a type error for int64 field
	_, err := valueFromStarlark(secondsFD, starlark.None)
	assert.Error(t, err, "None assigned to int64 field should fail")
}

func TestValueFromStarlark_Enum(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()

	// Find a valid enum value
	values := enumDesc.GetValues()
	require.NotEmpty(t, values)
	firstValue := values[0]

	starEnumVal := &starProtoEnumValue{desc: firstValue}
	result, err := valueFromStarlark(enumFD, starEnumVal)
	require.NoError(t, err)
	assert.Equal(t, firstValue.GetNumber(), result)
}

func TestValueFromStarlark_List_RepeatedField(t *testing.T) {
	// Use DescriptorProto.field which is repeated FieldDescriptorProto
	fd, err := desc.LoadFileDescriptor("google/protobuf/descriptor.proto")
	require.NoError(t, err)
	sym := fd.FindSymbol("google.protobuf.DescriptorProto")
	require.NotNil(t, sym)
	msgDesc := sym.(*desc.MessageDescriptor)
	repeatedFD := msgDesc.FindFieldByName("oneof_decl")
	require.NotNil(t, repeatedFD)
	require.True(t, repeatedFD.IsRepeated())

	// Create an empty list — empty repeated is valid
	emptyList := starlark.NewList(nil)
	result, err := valueFromStarlark(repeatedFD, emptyList)
	require.NoError(t, err)
	resultSlice, ok := result.([]interface{})
	require.True(t, ok)
	assert.Equal(t, 0, len(resultSlice))
}

func TestEnumType_AttrNames(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()

	enumType := NewEnumType(enumDesc)
	et, ok := enumType.(*starProtoEnumType)
	require.True(t, ok)

	names := et.AttrNames()
	assert.NotEmpty(t, names, "enum should have value names")
}

func TestEnumType_Attr(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()

	enumType := NewEnumType(enumDesc)
	et, ok := enumType.(*starProtoEnumType)
	require.True(t, ok)

	names := et.AttrNames()
	require.NotEmpty(t, names)

	// Attribute lookup by name should return enum value
	val, err := et.Attr(names[0])
	require.NoError(t, err)
	require.NotNil(t, val)

	enumVal, ok := val.(*starProtoEnumValue)
	require.True(t, ok)
	assert.Equal(t, names[0], enumVal.desc.GetName())
}

func TestEnumType_Attr_NonExistent(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()

	enumType := NewEnumType(enumDesc)
	et := enumType.(*starProtoEnumType)

	val, err := et.Attr("NONEXISTENT_VALUE")
	assert.NoError(t, err) // Returns nil, nil for non-existent
	assert.Nil(t, val)
}

func TestEnumValue_CompareSameType(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()
	values := enumDesc.GetValues()
	require.GreaterOrEqual(t, len(values), 2)

	v1 := &starProtoEnumValue{desc: values[0]}
	v2 := &starProtoEnumValue{desc: values[0]}
	v3 := &starProtoEnumValue{desc: values[1]}

	// Same value should be equal
	eq, err := v1.CompareSameType(syntax.EQL, v2, 0)
	require.NoError(t, err)
	assert.True(t, eq)

	// Different values should not be equal
	neq, err := v1.CompareSameType(syntax.NEQ, v3, 0)
	require.NoError(t, err)
	assert.True(t, neq)
}

func TestEnumValue_Hash(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()
	values := enumDesc.GetValues()
	require.NotEmpty(t, values)

	v := &starProtoEnumValue{desc: values[0]}
	h, err := v.Hash()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, h, uint32(0))
}

func TestEnumValue_StringAndType(t *testing.T) {
	enumFD := makeEnumFieldDescriptor(t)
	enumDesc := enumFD.GetEnumType()
	values := enumDesc.GetValues()
	require.NotEmpty(t, values)

	v := &starProtoEnumValue{desc: values[0]}
	assert.NotEmpty(t, v.String())
	assert.NotEmpty(t, v.Type())
}
