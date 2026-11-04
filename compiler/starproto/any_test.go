package starproto

import (
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	_ "google.golang.org/protobuf/types/descriptorpb"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/typepb"
)

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

// anyChainPrelude is the Starlark shared by the memo tests and the benchmark.
// chain mirrors a hook chain: pack, unpack, mutate, repeated.
const anyChainPrelude = `
def build():
    msgs = []
    for i in range(20):
        fields = [FieldDescriptorProto(name="f%d_%d" % (i, j), number=j + 1, json_name="f%d" % j) for j in range(10)]
        nested = DescriptorProto(name="N%d" % i, field=[FieldDescriptorProto(name="n", number=1, json_name="n")])
        msgs.append(DescriptorProto(name="M%d" % i, field=fields, nested_type=[nested]))
    return FileDescriptorProto(name="bench.proto", package="bench", dependency=["a.proto", "b.proto"], message_type=msgs)

def chain(cfg, steps):
    for i in range(steps):
        a = any.new(cfg)
        cfg = any.unpack(a, FileDescriptorProto())
        cfg.package = "bench%d" % i
    return cfg
`

func anyTestPredeclared(t testing.TB, extra starlark.StringDict) starlark.StringDict {
	t.Helper()
	d := starlark.StringDict{"any": AnyModule}
	for name, full := range map[string]string{
		"FileDescriptorProto":  "google.protobuf.FileDescriptorProto",
		"DescriptorProto":      "google.protobuf.DescriptorProto",
		"FieldDescriptorProto": "google.protobuf.FieldDescriptorProto",
		"Struct":               "google.protobuf.Struct",
		"Value":                "google.protobuf.Value",
		"Duration":             "google.protobuf.Duration",
		"Option":               "google.protobuf.Option",
	} {
		md, err := desc.LoadMessageDescriptor(full)
		require.NoError(t, err)
		d[name] = NewMessageType(md)
	}
	for k, v := range extra {
		d[k] = v
	}
	return d
}

// execStar runs src on the given thread and returns the globals.
func execStar(t testing.TB, thread *starlark.Thread, src string, extra starlark.StringDict) starlark.StringDict {
	t.Helper()
	g, err := starlark.ExecFile(thread, "test.star", src, anyTestPredeclared(t, extra))
	require.NoError(t, err)
	return g
}

func anyValueBytes(t testing.TB, v starlark.Value) []byte {
	t.Helper()
	m, ok := v.(*starProtoMessage)
	require.True(t, ok, "want *starProtoMessage, got %T", v)
	b, err := m.msg.TryGetFieldByName("value")
	require.NoError(t, err)
	return b.([]byte)
}

// refDecode UnmarshalMerges the Any's bytes into base, the pre-memo behaviour.
func refDecode(t testing.TB, anyVal starlark.Value, base starlark.Value) (*dynamic.Message, error) {
	t.Helper()
	m := base.(*starProtoMessage).msg
	return m, m.UnmarshalMerge(anyValueBytes(t, anyVal))
}

func starStr(t testing.TB, v starlark.Value, field string) string {
	t.Helper()
	a, err := v.(*starProtoMessage).Attr(field)
	require.NoError(t, err)
	return string(a.(starlark.String))
}

func mustInt(t testing.TB, v starlark.Value) int64 {
	t.Helper()
	n, ok := v.(starlark.Int).Int64()
	require.True(t, ok)
	return n
}

func TestAnyUnpackMemo_HitMatchesDecode(t *testing.T) {
	th := &starlark.Thread{}
	g1 := execStar(t, th, anyChainPrelude+`a = any.new(build())`, nil)
	g := execStar(t, th, `
out = any.unpack(a, FileDescriptorProto())
ref = FileDescriptorProto()
`, starlark.StringDict{"a": g1["a"]})
	want, err := refDecode(t, g1["a"], g["ref"])
	require.NoError(t, err)
	assert.True(t, dynamic.Equal(want, g["out"].(*starProtoMessage).msg))
}

func TestAnyUnpackMemo_DoubleUnpack(t *testing.T) {
	th := &starlark.Thread{}
	g1 := execStar(t, th, anyChainPrelude+`a = any.new(build())`, nil)
	a := starlark.StringDict{"a": g1["a"]}
	o1 := execStar(t, th, `out = any.unpack(a, FileDescriptorProto())`, a)["out"]
	o2 := execStar(t, th, `out = any.unpack(a, FileDescriptorProto())`, a)["out"]
	assert.True(t, dynamic.Equal(o1.(*starProtoMessage).msg, o2.(*starProtoMessage).msg))
}

func TestAnyUnpackMemo_SourceMutatedAfterNew(t *testing.T) {
	th := &starlark.Thread{}
	g := execStar(t, th, anyChainPrelude+`
src = build()
a = any.new(src)
src.package = "mutated"
src.message_type[0].name = "X"
out = any.unpack(a, FileDescriptorProto())
ref = FileDescriptorProto()
first = out.message_type[0].name
`, nil)
	out := g["out"]
	assert.Equal(t, "bench", starStr(t, out, "package"))
	assert.Equal(t, "M0", string(g["first"].(starlark.String)))
	want, err := refDecode(t, g["a"], g["ref"])
	require.NoError(t, err)
	assert.True(t, dynamic.Equal(want, out.(*starProtoMessage).msg))
}

func TestAnyUnpackMemo_ValueBytesReplaced(t *testing.T) {
	other := execStar(t, &starlark.Thread{}, anyChainPrelude+`
o = build()
o.package = "other"
oa = any.new(o)
`, nil)["oa"]
	th := &starlark.Thread{}
	g := execStar(t, th, anyChainPrelude+`
a = any.new(build())
a.value = other.value
out = any.unpack(a, FileDescriptorProto())
ref = FileDescriptorProto()
`, starlark.StringDict{"other": other})
	assert.Equal(t, "other", starStr(t, g["out"], "package"))
	want, err := refDecode(t, g["a"], g["ref"])
	require.NoError(t, err)
	assert.True(t, dynamic.Equal(want, g["out"].(*starProtoMessage).msg))
}

func TestAnyUnpackMemo_MergeIntoPopulatedTarget(t *testing.T) {
	t.Run("repeated append and scalar overwrite", func(t *testing.T) {
		th := &starlark.Thread{}
		g := execStar(t, th, anyChainPrelude+`
a = any.new(build())
def pre():
    return FileDescriptorProto(name="keep.proto", dependency=["pre.proto"], message_type=[DescriptorProto(name="Pre")])
out = any.unpack(a, pre())
ref = pre()
n = len(out.message_type)
first = out.message_type[0].name
deps = list(out.dependency)
name = out.name
`, nil)
		assert.Equal(t, "bench.proto", string(g["name"].(starlark.String)))
		assert.Equal(t, "Pre", string(g["first"].(starlark.String)))
		assert.Equal(t, int64(21), mustInt(t, g["n"]))
		assert.Equal(t, `["pre.proto", "a.proto", "b.proto"]`, g["deps"].String())
		want, err := refDecode(t, g["a"], g["ref"])
		require.NoError(t, err)
		assert.True(t, dynamic.Equal(want, g["out"].(*starProtoMessage).msg))
	})
	t.Run("singular nested message merge", func(t *testing.T) {
		th := &starlark.Thread{}
		g := execStar(t, th, `
def tgt():
    return Value(struct_value=Struct(fields={"pre": Value(string_value="x")}))
a = any.new(Value(struct_value=Struct(fields={"a": Value(number_value=1.0)})))
out = any.unpack(a, tgt())
ref = tgt()
`, nil)
		want, err := refDecode(t, g["a"], g["ref"])
		require.NoError(t, err)
		assert.True(t, dynamic.Equal(want, g["out"].(*starProtoMessage).msg))
		sv, err := g["out"].(*starProtoMessage).msg.TryGetFieldByName("struct_value")
		require.NoError(t, err)
		fields, err := sv.(*dynamic.Message).TryGetFieldByName("fields")
		require.NoError(t, err)
		assert.Len(t, fields, 2)
	})
}

func TestAnyUnpackMemo_TypeMismatch(t *testing.T) {
	thread := &starlark.Thread{}
	pre := anyTestPredeclared(t, nil)
	a := execStar(t, thread, anyChainPrelude+`a = any.new(build())`, nil)["a"]
	dur, err := starlark.Call(thread, pre["Duration"], nil, nil)
	require.NoError(t, err)
	refDur, err := starlark.Call(thread, pre["Duration"], nil, nil)
	require.NoError(t, err)
	got, gotErr := starlark.Call(thread, AnyModule.Members["unpack"], starlark.Tuple{a, dur}, nil)
	want, wantErr := refDecode(t, a, refDur)
	assert.Equal(t, wantErr == nil, gotErr == nil)
	if gotErr == nil && wantErr == nil {
		assert.True(t, dynamic.Equal(want, got.(*starProtoMessage).msg))
	}
}

func TestAnyUnpackMemo_AnyInParentField(t *testing.T) {
	t.Run("stored in parent field and read back", func(t *testing.T) {
		th := &starlark.Thread{}
		g := execStar(t, th, anyChainPrelude+`
h = Option(name="o", value=any.new(build()))
out = any.unpack(h.value, FileDescriptorProto())
ref = FileDescriptorProto()
`, nil)
		hv, err := g["h"].(*starProtoMessage).Attr("value")
		require.NoError(t, err)
		want, err := refDecode(t, hv, g["ref"])
		require.NoError(t, err)
		assert.True(t, dynamic.Equal(want, g["out"].(*starProtoMessage).msg))
	})
	t.Run("carried inside another Any", func(t *testing.T) {
		th := &starlark.Thread{}
		g := execStar(t, th, anyChainPrelude+`
h = Option(name="o", value=any.new(build()))
h2 = any.unpack(any.new(h), Option())
out2 = any.unpack(h2.value, FileDescriptorProto())
ref2 = FileDescriptorProto()
`, nil)
		h2v, err := g["h2"].(*starProtoMessage).Attr("value")
		require.NoError(t, err)
		want2, err := refDecode(t, h2v, g["ref2"])
		require.NoError(t, err)
		assert.True(t, dynamic.Equal(want2, g["out2"].(*starProtoMessage).msg))
	})
}

func BenchmarkAnyRoundTrip(b *testing.B) {
	g := execStar(b, &starlark.Thread{}, anyChainPrelude, nil)
	cfg, err := starlark.Call(&starlark.Thread{}, g["build"], nil, nil)
	require.NoError(b, err)
	chain := g["chain"]
	b.ReportAllocs()
	for b.Loop() {
		if _, err := starlark.Call(&starlark.Thread{}, chain, starlark.Tuple{cfg, starlark.MakeInt(5)}, nil); err != nil {
			b.Fatal(err)
		}
	}
}
