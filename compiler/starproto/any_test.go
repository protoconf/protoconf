package starproto

import (
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

// newAnyDescriptor loads the google.protobuf.Any message descriptor.
func newAnyDescriptor(t *testing.T) *desc.MessageDescriptor {
	t.Helper()
	fd, err := desc.LoadFileDescriptor("google/protobuf/any.proto")
	require.NoError(t, err, "failed to load any.proto")
	sym := fd.FindSymbol("google.protobuf.Any")
	require.NotNil(t, sym)
	msgDesc, ok := sym.(*desc.MessageDescriptor)
	require.True(t, ok)
	return msgDesc
}

// callNewAny invokes the any.new Starlark builtin.
func callNewAny(t *testing.T, msg *starProtoMessage) (starlark.Value, error) {
	t.Helper()
	thread := &starlark.Thread{Name: "test"}
	args := starlark.Tuple{msg}
	return newAny(thread, nil, args, nil)
}

// callUnpackAny invokes the any.unpack Starlark builtin.
func callUnpackAny(t *testing.T, anySrc *starProtoMessage, target *starProtoMessage) (starlark.Value, error) {
	t.Helper()
	thread := &starlark.Thread{Name: "test"}
	args := starlark.Tuple{anySrc, target}
	return unpackAny(thread, nil, args, nil)
}

func TestAnyWrap_Duration(t *testing.T) {
	// Create a Duration message with a value
	msgDesc := loadDurationDescriptor(t)
	msg := dynamic.NewMessage(msgDesc)
	require.NoError(t, msg.TrySetField(msgDesc.FindFieldByName("seconds"), int64(42)))

	wrapped := NewStarProtoMessage(msg)

	// Wrap in Any
	result, err := callNewAny(t, wrapped)
	require.NoError(t, err, "any.new should not error on valid message")
	require.NotNil(t, result)

	// The result should be a *starProtoMessage wrapping a google.protobuf.Any
	anyMsg, ok := result.(*starProtoMessage)
	require.True(t, ok, "any.new should return *starProtoMessage, got %T", result)

	// Any has type_url and value fields
	typeURLAttr, err := anyMsg.Attr("type_url")
	require.NoError(t, err)
	typeURLStr, ok := typeURLAttr.(starlark.String)
	require.True(t, ok, "type_url should be a string, got %T", typeURLAttr)
	assert.Contains(t, string(typeURLStr), "google.protobuf.Duration",
		"type_url should contain the message type name")
}

func TestAnyWrap_TypeURLFormat(t *testing.T) {
	msgDesc := loadDurationDescriptor(t)
	msg := dynamic.NewMessage(msgDesc)
	wrapped := NewStarProtoMessage(msg)

	result, err := callNewAny(t, wrapped)
	require.NoError(t, err)

	anyMsg := result.(*starProtoMessage)
	typeURLAttr, err := anyMsg.Attr("type_url")
	require.NoError(t, err)
	typeURLStr := string(typeURLAttr.(starlark.String))

	// Standard type URL format is "type.googleapis.com/fully.qualified.TypeName"
	assert.Contains(t, typeURLStr, "type.googleapis.com/",
		"type_url should use type.googleapis.com prefix")
}

func TestAnyUnwrap_Duration(t *testing.T) {
	// Create and wrap a Duration
	msgDesc := loadDurationDescriptor(t)
	msg := dynamic.NewMessage(msgDesc)
	secondsFD := msgDesc.FindFieldByName("seconds")
	require.NoError(t, msg.TrySetField(secondsFD, int64(77)))

	wrapped := NewStarProtoMessage(msg)

	// Wrap into Any
	anyResult, err := callNewAny(t, wrapped)
	require.NoError(t, err)
	anyMsg := anyResult.(*starProtoMessage)

	// Create an empty Duration target for unpacking
	emptyMsg := dynamic.NewMessage(msgDesc)
	target := NewStarProtoMessage(emptyMsg)

	// Unpack
	unpacked, err := callUnpackAny(t, anyMsg, target)
	require.NoError(t, err, "any.unpack should not error")
	require.NotNil(t, unpacked)

	// Verify unpacked message has the original value
	unpackedMsg, ok := unpacked.(*starProtoMessage)
	require.True(t, ok)

	secondsAttr, err := unpackedMsg.Attr("seconds")
	require.NoError(t, err)
	secondsInt, ok := secondsAttr.(starlark.Int)
	require.True(t, ok)
	n, ok := secondsInt.Int64()
	require.True(t, ok)
	assert.Equal(t, int64(77), n, "unpacked Duration should have original seconds value")
}

func TestAnyRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (*starProtoMessage, *desc.MessageDescriptor)
		validate func(t *testing.T, unpacked *starProtoMessage)
	}{
		{
			name: "Duration with seconds=100",
			setup: func(t *testing.T) (*starProtoMessage, *desc.MessageDescriptor) {
				msgDesc := loadDurationDescriptor(t)
				msg := dynamic.NewMessage(msgDesc)
				require.NoError(t, msg.TrySetField(msgDesc.FindFieldByName("seconds"), int64(100)))
				require.NoError(t, msg.TrySetField(msgDesc.FindFieldByName("nanos"), int32(500)))
				return NewStarProtoMessage(msg), msgDesc
			},
			validate: func(t *testing.T, unpacked *starProtoMessage) {
				secAttr, err := unpacked.Attr("seconds")
				require.NoError(t, err)
				n, ok := secAttr.(starlark.Int).Int64()
				require.True(t, ok)
				assert.Equal(t, int64(100), n)

				nanoAttr, err := unpacked.Attr("nanos")
				require.NoError(t, err)
				nn, ok := nanoAttr.(starlark.Int).Int64()
				require.True(t, ok)
				assert.Equal(t, int64(500), nn)
			},
		},
		{
			name: "Duration with zero values",
			setup: func(t *testing.T) (*starProtoMessage, *desc.MessageDescriptor) {
				msgDesc := loadDurationDescriptor(t)
				msg := dynamic.NewMessage(msgDesc)
				// Do not set any fields — zero values
				return NewStarProtoMessage(msg), msgDesc
			},
			validate: func(t *testing.T, unpacked *starProtoMessage) {
				secAttr, err := unpacked.Attr("seconds")
				require.NoError(t, err)
				n, ok := secAttr.(starlark.Int).Int64()
				require.True(t, ok)
				assert.Equal(t, int64(0), n)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			original, msgDesc := tc.setup(t)

			// Wrap in Any
			anyResult, err := callNewAny(t, original)
			require.NoError(t, err)
			anyMsg := anyResult.(*starProtoMessage)

			// Create empty target
			emptyTarget := NewStarProtoMessage(dynamic.NewMessage(msgDesc))

			// Unpack from Any
			unpacked, err := callUnpackAny(t, anyMsg, emptyTarget)
			require.NoError(t, err)

			unpackedMsg, ok := unpacked.(*starProtoMessage)
			require.True(t, ok)

			tc.validate(t, unpackedMsg)
		})
	}
}

func TestAnyModule_Members(t *testing.T) {
	// Verify AnyModule exposes "new" and "unpack"
	require.NotNil(t, AnyModule)
	assert.Equal(t, "any", AnyModule.Name)

	newBuiltin, ok := AnyModule.Members["new"]
	require.True(t, ok, "AnyModule should have 'new' member")
	assert.NotNil(t, newBuiltin)

	unpackBuiltin, ok := AnyModule.Members["unpack"]
	require.True(t, ok, "AnyModule should have 'unpack' member")
	assert.NotNil(t, unpackBuiltin)
}

func TestAnyWrap_ErrorOnWrongArgs(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}

	// newAny requires exactly 1 positional arg of *starProtoMessage type
	// Passing wrong type (string instead of message)
	args := starlark.Tuple{starlark.String("not_a_message")}
	_, err := newAny(thread, nil, args, nil)
	assert.Error(t, err, "newAny with wrong arg type should error")
}

func TestAnyWrap_ErrorOnNoArgs(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}

	// Passing no args
	_, err := newAny(thread, nil, starlark.Tuple{}, nil)
	assert.Error(t, err, "newAny with no args should error")
}

func TestAnyUnwrap_ErrorOnNoArgs(t *testing.T) {
	thread := &starlark.Thread{Name: "test"}

	// unpackAny requires 2 positional args
	_, err := unpackAny(thread, nil, starlark.Tuple{}, nil)
	assert.Error(t, err, "unpackAny with no args should error")
}

func TestAnyWrap_ValueField(t *testing.T) {
	// Verify the "value" bytes field is set in the Any
	msgDesc := loadDurationDescriptor(t)
	msg := dynamic.NewMessage(msgDesc)
	require.NoError(t, msg.TrySetField(msgDesc.FindFieldByName("seconds"), int64(55)))

	wrapped := NewStarProtoMessage(msg)
	result, err := callNewAny(t, wrapped)
	require.NoError(t, err)

	anyMsg := result.(*starProtoMessage)

	// Any has a "value" field containing the marshaled bytes
	valueAttr, err := anyMsg.Attr("value")
	require.NoError(t, err)
	require.NotNil(t, valueAttr)

	// The value should be non-None (it contains actual bytes)
	assert.NotEqual(t, starlark.None, valueAttr)
}
