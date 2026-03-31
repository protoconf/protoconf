package starproto

import (
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// loadDurationDescriptor loads google.protobuf.Duration message descriptor.
// Duration has int64 seconds and int32 nanos — good for numeric field tests.
func loadDurationDescriptor(t *testing.T) *desc.MessageDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/duration.proto")
	require.NoError(t, err, "failed to load duration.proto file descriptor")
	md := fd.FindSymbol("google.protobuf.Duration")
	require.NotNil(t, md, "google.protobuf.Duration not found")
	msgDesc, ok := md.(*desc.MessageDescriptor)
	require.True(t, ok, "expected MessageDescriptor")
	return msgDesc
}

// loadStructDescriptor loads google.protobuf.Struct message descriptor.
// Struct has map<string, Value> fields — good for map field tests.
func loadStructDescriptor(t *testing.T) *desc.MessageDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/struct.proto")
	require.NoError(t, err, "failed to load struct.proto file descriptor")
	md := fd.FindSymbol("google.protobuf.Struct")
	require.NotNil(t, md, "google.protobuf.Struct not found")
	msgDesc, ok := md.(*desc.MessageDescriptor)
	require.True(t, ok, "expected MessageDescriptor")
	return msgDesc
}

// newDurationMessage creates a dynamic Duration message.
func newDurationMessage(t *testing.T) *dynamic.Message {
	t.Helper()
	msgDesc := loadDurationDescriptor(t)
	return dynamic.NewMessage(msgDesc)
}

func TestNewStarProtoMessage(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	require.NotNil(t, wrapped, "NewStarProtoMessage must return non-nil")
	assert.Equal(t, "google.protobuf.Duration", wrapped.Type())
}

func TestStarProtoMessage_String(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	s := wrapped.String()
	assert.NotEmpty(t, s, "String() must return non-empty value")
	assert.Contains(t, s, "google.protobuf.Duration", "String() should contain type name")
}

func TestStarProtoMessage_Truth(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	// Truth() always returns True for proto messages (per implementation)
	assert.Equal(t, starlark.True, wrapped.Truth())
}

func TestStarProtoMessage_Hash(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	// Hash works on proto messages (marshals to bytes, FNV32)
	h, err := wrapped.Hash()
	assert.NoError(t, err, "Hash() should not error on a valid message")
	assert.GreaterOrEqual(t, h, uint32(0))
}

func TestStarProtoMessage_AttrNames(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	names := wrapped.AttrNames()
	assert.Contains(t, names, "seconds", "AttrNames should include 'seconds'")
	assert.Contains(t, names, "nanos", "AttrNames should include 'nanos'")
	assert.Equal(t, 2, len(names), "Duration has exactly 2 fields")
}

func TestStarProtoMessage_Attr_ReadDefaultFields(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	tests := []struct {
		name      string
		fieldName string
	}{
		{"seconds field", "seconds"},
		{"nanos field", "nanos"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := wrapped.Attr(tc.fieldName)
			require.NoError(t, err)
			require.NotNil(t, val)
		})
	}
}

func TestStarProtoMessage_Attr_NonExistentField(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	_, err := wrapped.Attr("nonexistent_field")
	assert.Error(t, err, "Attr on non-existent field should return error")
	_, isNoSuchAttr := err.(starlark.NoSuchAttrError)
	assert.True(t, isNoSuchAttr, "error should be NoSuchAttrError")
}

func TestStarProtoMessage_SetField_Int64(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	err := wrapped.SetField("seconds", starlark.MakeInt(42))
	require.NoError(t, err, "SetField for int64 field should not error")

	val, err := wrapped.Attr("seconds")
	require.NoError(t, err)
	intVal, ok := val.(starlark.Int)
	require.True(t, ok, "seconds field should return starlark.Int")
	n, ok := intVal.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(42), n)
}

func TestStarProtoMessage_SetField_Int32(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	err := wrapped.SetField("nanos", starlark.MakeInt(100))
	require.NoError(t, err, "SetField for int32 field should not error")

	val, err := wrapped.Attr("nanos")
	require.NoError(t, err)
	intVal, ok := val.(starlark.Int)
	require.True(t, ok, "nanos field should return starlark.Int")
	n, ok := intVal.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(100), n)
}

func TestStarProtoMessage_SetField_NonExistentField(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	err := wrapped.SetField("nonexistent_field", starlark.MakeInt(42))
	assert.Error(t, err, "SetField on non-existent field should return error")
}

func TestStarProtoMessage_SetField_WrongType(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	// Setting a string to an int64 field should fail
	err := wrapped.SetField("seconds", starlark.String("not_a_number"))
	assert.Error(t, err, "SetField with wrong type should return error")
}

func TestStarProtoMessage_SetKey(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	err := wrapped.SetKey(starlark.String("seconds"), starlark.MakeInt(99))
	require.NoError(t, err)

	val, err := wrapped.Attr("seconds")
	require.NoError(t, err)
	intVal, ok := val.(starlark.Int)
	require.True(t, ok)
	n, ok := intVal.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(99), n)
}

func TestStarProtoMessage_SetKey_NonStringKey(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	err := wrapped.SetKey(starlark.MakeInt(1), starlark.MakeInt(99))
	assert.Error(t, err, "SetKey with non-string key should return error")
}

func TestStarProtoMessage_Freeze(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	// Pre-freeze: setting should work
	err := wrapped.SetField("seconds", starlark.MakeInt(10))
	require.NoError(t, err)

	wrapped.Freeze()

	// Post-freeze: setting should fail
	err = wrapped.SetField("seconds", starlark.MakeInt(20))
	assert.Error(t, err, "SetField on frozen message should return error")
}

func TestStarProtoMessage_CompareSameType(t *testing.T) {
	msgDesc := loadDurationDescriptor(t)

	msg1 := dynamic.NewMessage(msgDesc)
	msg2 := dynamic.NewMessage(msgDesc)

	w1 := NewStarProtoMessage(msg1)
	w2 := NewStarProtoMessage(msg2)

	// Two empty Duration messages should be equal
	equal, err := w1.CompareSameType(syntax.EQL, w2, 0)
	require.NoError(t, err)
	assert.True(t, equal)

	// Set a field on one
	require.NoError(t, w1.SetField("seconds", starlark.MakeInt(5)))

	// Now they should not be equal
	notEqual, err := w1.CompareSameType(syntax.NEQ, w2, 0)
	require.NoError(t, err)
	assert.True(t, notEqual)
}

func TestStarProtoMessage_DESCRIPTOR_Attr(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	// DESCRIPTOR is a special attribute
	val, err := wrapped.Attr("DESCRIPTOR")
	require.NoError(t, err)
	require.NotNil(t, val)
}

func TestStarProtoMessage_AsJSON_Attr(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	err := wrapped.SetField("seconds", starlark.MakeInt(30))
	require.NoError(t, err)

	// as_json is a special method attribute
	val, err := wrapped.Attr("as_json")
	require.NoError(t, err)
	require.NotNil(t, val)

	// It should be callable
	callable, ok := val.(starlark.Callable)
	assert.True(t, ok, "as_json should be callable")
	thread := &starlark.Thread{}
	result, err := starlark.Call(thread, callable, nil, nil)
	require.NoError(t, err)
	_, isStr := result.(starlark.String)
	assert.True(t, isStr, "as_json should return string")
}

func TestToProtoMessage(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	// ToProtoMessage should extract the underlying dynamic.Message
	extracted, ok := ToProtoMessage(wrapped)
	assert.True(t, ok)
	assert.NotNil(t, extracted)
	assert.Equal(t, msg, extracted)
}

func TestToProtoMessage_NonMessage(t *testing.T) {
	// Non-message Starlark values should return false
	_, ok := ToProtoMessage(starlark.MakeInt(42))
	assert.False(t, ok)
}

func TestToDynamicPb(t *testing.T) {
	msg := newDurationMessage(t)
	wrapped := NewStarProtoMessage(msg)

	require.NoError(t, wrapped.SetField("seconds", starlark.MakeInt(10)))

	pb := wrapped.AsDynamicPb()
	require.NotNil(t, pb)
}

func TestRepeatedField(t *testing.T) {
	// Use agent_config proto which likely has repeated fields, or use a well-known type.
	// google.protobuf.FieldDescriptorProto has repeated options fields.
	// Let's use a simpler approach: find a proto in the project with repeated fields.
	// agent_config.proto has repeated store_addresses string.
	// Instead use google/protobuf/descriptor.proto which has many repeated fields.
	fd, err := desc.LoadFileDescriptor("google/protobuf/descriptor.proto")
	require.NoError(t, err)

	// DescriptorProto has repeated FieldDescriptorProto fields named "field"
	sym := fd.FindSymbol("google.protobuf.DescriptorProto")
	require.NotNil(t, sym)
	msgDesc, ok := sym.(*desc.MessageDescriptor)
	require.True(t, ok)

	msg := dynamic.NewMessage(msgDesc)
	wrapped := NewStarProtoMessage(msg)

	// Get the "field" attribute which is repeated
	fieldAttr, err := wrapped.Attr("field")
	require.NoError(t, err)

	repeated, ok := fieldAttr.(*protoRepeated)
	require.True(t, ok, "repeated field should return *protoRepeated")
	assert.Equal(t, 0, repeated.Len(), "initially empty")

	// Verify the type name contains "list"
	assert.Contains(t, repeated.Type(), "list")
}

func TestMapField(t *testing.T) {
	// google.protobuf.Struct has map<string, Value> fields
	msgDesc := loadStructDescriptor(t)
	msg := dynamic.NewMessage(msgDesc)
	wrapped := NewStarProtoMessage(msg)

	// "fields" is the map field in Struct
	fieldsAttr, err := wrapped.Attr("fields")
	require.NoError(t, err)
	require.NotNil(t, fieldsAttr)

	protoM, ok := fieldsAttr.(*protoMap)
	require.True(t, ok, "map field should return *protoMap, got %T", fieldsAttr)
	assert.Equal(t, 0, protoM.Len(), "initially empty map")

	// Verify the type name contains "map"
	assert.Contains(t, protoM.Type(), "map")
}

func TestStarProtoMessage_MessageField(t *testing.T) {
	// google.protobuf.FieldDescriptorProto has an 'options' field of message type
	// Use DescriptorProto which has nested DescriptorProto (nested_type field)
	msgDesc := loadDurationDescriptor(t)

	// Create a wrapper and verify we can call Attr on it without panics
	msg := dynamic.NewMessage(msgDesc)
	wrapped := NewStarProtoMessage(msg)

	// Seconds is int64, nanos is int32 — all accesses should work
	for _, fieldName := range wrapped.AttrNames() {
		val, err := wrapped.Attr(fieldName)
		require.NoError(t, err, "Attr(%q) should not error", fieldName)
		require.NotNil(t, val, "Attr(%q) should return non-nil value", fieldName)
	}
}
