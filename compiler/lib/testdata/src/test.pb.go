// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.9.0
// source: compiler/lib/testdata/src/test.proto

package test

import (
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type AnotherMessageWithEnum_AnotherTestEnum int32

const (
	AnotherMessageWithEnum_ZERO AnotherMessageWithEnum_AnotherTestEnum = 0
	AnotherMessageWithEnum_ONE  AnotherMessageWithEnum_AnotherTestEnum = 1
	AnotherMessageWithEnum_TWO  AnotherMessageWithEnum_AnotherTestEnum = 2
)

// Enum value maps for AnotherMessageWithEnum_AnotherTestEnum.
var (
	AnotherMessageWithEnum_AnotherTestEnum_name = map[int32]string{
		0: "ZERO",
		1: "ONE",
		2: "TWO",
	}
	AnotherMessageWithEnum_AnotherTestEnum_value = map[string]int32{
		"ZERO": 0,
		"ONE":  1,
		"TWO":  2,
	}
)

func (x AnotherMessageWithEnum_AnotherTestEnum) Enum() *AnotherMessageWithEnum_AnotherTestEnum {
	p := new(AnotherMessageWithEnum_AnotherTestEnum)
	*p = x
	return p
}

func (x AnotherMessageWithEnum_AnotherTestEnum) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AnotherMessageWithEnum_AnotherTestEnum) Descriptor() protoreflect.EnumDescriptor {
	return file_compiler_lib_testdata_src_test_proto_enumTypes[0].Descriptor()
}

func (AnotherMessageWithEnum_AnotherTestEnum) Type() protoreflect.EnumType {
	return &file_compiler_lib_testdata_src_test_proto_enumTypes[0]
}

func (x AnotherMessageWithEnum_AnotherTestEnum) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AnotherMessageWithEnum_AnotherTestEnum.Descriptor instead.
func (AnotherMessageWithEnum_AnotherTestEnum) EnumDescriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{1, 0}
}

type MessageWithEnum_TestEnum int32

const (
	MessageWithEnum_ZERO MessageWithEnum_TestEnum = 0
	MessageWithEnum_ONE  MessageWithEnum_TestEnum = 1
	MessageWithEnum_TWO  MessageWithEnum_TestEnum = 2
)

// Enum value maps for MessageWithEnum_TestEnum.
var (
	MessageWithEnum_TestEnum_name = map[int32]string{
		0: "ZERO",
		1: "ONE",
		2: "TWO",
	}
	MessageWithEnum_TestEnum_value = map[string]int32{
		"ZERO": 0,
		"ONE":  1,
		"TWO":  2,
	}
)

func (x MessageWithEnum_TestEnum) Enum() *MessageWithEnum_TestEnum {
	p := new(MessageWithEnum_TestEnum)
	*p = x
	return p
}

func (x MessageWithEnum_TestEnum) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (MessageWithEnum_TestEnum) Descriptor() protoreflect.EnumDescriptor {
	return file_compiler_lib_testdata_src_test_proto_enumTypes[1].Descriptor()
}

func (MessageWithEnum_TestEnum) Type() protoreflect.EnumType {
	return &file_compiler_lib_testdata_src_test_proto_enumTypes[1]
}

func (x MessageWithEnum_TestEnum) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use MessageWithEnum_TestEnum.Descriptor instead.
func (MessageWithEnum_TestEnum) EnumDescriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{2, 0}
}

type TestMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StringValue string     `protobuf:"bytes,1,opt,name=stringValue,proto3" json:"stringValue,omitempty"`
	AnyField    *any.Any   `protobuf:"bytes,3,opt,name=any_field,json=anyField,proto3" json:"any_field,omitempty"`
	AnyRepeated []*any.Any `protobuf:"bytes,4,rep,name=any_repeated,json=anyRepeated,proto3" json:"any_repeated,omitempty"`
}

func (x *TestMessage) Reset() {
	*x = TestMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMessage) ProtoMessage() {}

func (x *TestMessage) ProtoReflect() protoreflect.Message {
	mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMessage.ProtoReflect.Descriptor instead.
func (*TestMessage) Descriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{0}
}

func (x *TestMessage) GetStringValue() string {
	if x != nil {
		return x.StringValue
	}
	return ""
}

func (x *TestMessage) GetAnyField() *any.Any {
	if x != nil {
		return x.AnyField
	}
	return nil
}

func (x *TestMessage) GetAnyRepeated() []*any.Any {
	if x != nil {
		return x.AnyRepeated
	}
	return nil
}

type AnotherMessageWithEnum struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	E AnotherMessageWithEnum_AnotherTestEnum `protobuf:"varint,1,opt,name=e,proto3,enum=AnotherMessageWithEnum_AnotherTestEnum" json:"e,omitempty"`
}

func (x *AnotherMessageWithEnum) Reset() {
	*x = AnotherMessageWithEnum{}
	if protoimpl.UnsafeEnabled {
		mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AnotherMessageWithEnum) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AnotherMessageWithEnum) ProtoMessage() {}

func (x *AnotherMessageWithEnum) ProtoReflect() protoreflect.Message {
	mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AnotherMessageWithEnum.ProtoReflect.Descriptor instead.
func (*AnotherMessageWithEnum) Descriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{1}
}

func (x *AnotherMessageWithEnum) GetE() AnotherMessageWithEnum_AnotherTestEnum {
	if x != nil {
		return x.E
	}
	return AnotherMessageWithEnum_ZERO
}

type MessageWithEnum struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	E MessageWithEnum_TestEnum `protobuf:"varint,1,opt,name=e,proto3,enum=MessageWithEnum_TestEnum" json:"e,omitempty"`
}

func (x *MessageWithEnum) Reset() {
	*x = MessageWithEnum{}
	if protoimpl.UnsafeEnabled {
		mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MessageWithEnum) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MessageWithEnum) ProtoMessage() {}

func (x *MessageWithEnum) ProtoReflect() protoreflect.Message {
	mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MessageWithEnum.ProtoReflect.Descriptor instead.
func (*MessageWithEnum) Descriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{2}
}

func (x *MessageWithEnum) GetE() MessageWithEnum_TestEnum {
	if x != nil {
		return x.E
	}
	return MessageWithEnum_ZERO
}

type ValidateMe struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Notempty       string            `protobuf:"bytes,1,opt,name=notempty,proto3" json:"notempty,omitempty"`
	ValidateMap    map[string]string `protobuf:"bytes,2,rep,name=validate_map,json=validateMap,proto3" json:"validate_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Nested         *TestMessage      `protobuf:"bytes,3,opt,name=nested,proto3" json:"nested,omitempty"`
	RepeatedString []string          `protobuf:"bytes,4,rep,name=repeated_string,json=repeatedString,proto3" json:"repeated_string,omitempty"`
}

func (x *ValidateMe) Reset() {
	*x = ValidateMe{}
	if protoimpl.UnsafeEnabled {
		mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ValidateMe) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ValidateMe) ProtoMessage() {}

func (x *ValidateMe) ProtoReflect() protoreflect.Message {
	mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ValidateMe.ProtoReflect.Descriptor instead.
func (*ValidateMe) Descriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{3}
}

func (x *ValidateMe) GetNotempty() string {
	if x != nil {
		return x.Notempty
	}
	return ""
}

func (x *ValidateMe) GetValidateMap() map[string]string {
	if x != nil {
		return x.ValidateMap
	}
	return nil
}

func (x *ValidateMe) GetNested() *TestMessage {
	if x != nil {
		return x.Nested
	}
	return nil
}

func (x *ValidateMe) GetRepeatedString() []string {
	if x != nil {
		return x.RepeatedString
	}
	return nil
}

type MessageWithSubMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Sub *MessageWithSubMessage_SubMessage `protobuf:"bytes,1,opt,name=sub,proto3" json:"sub,omitempty"`
}

func (x *MessageWithSubMessage) Reset() {
	*x = MessageWithSubMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MessageWithSubMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MessageWithSubMessage) ProtoMessage() {}

func (x *MessageWithSubMessage) ProtoReflect() protoreflect.Message {
	mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MessageWithSubMessage.ProtoReflect.Descriptor instead.
func (*MessageWithSubMessage) Descriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{4}
}

func (x *MessageWithSubMessage) GetSub() *MessageWithSubMessage_SubMessage {
	if x != nil {
		return x.Sub
	}
	return nil
}

type MessageWithSubMessage_SubMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *MessageWithSubMessage_SubMessage) Reset() {
	*x = MessageWithSubMessage_SubMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MessageWithSubMessage_SubMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MessageWithSubMessage_SubMessage) ProtoMessage() {}

func (x *MessageWithSubMessage_SubMessage) ProtoReflect() protoreflect.Message {
	mi := &file_compiler_lib_testdata_src_test_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MessageWithSubMessage_SubMessage.ProtoReflect.Descriptor instead.
func (*MessageWithSubMessage_SubMessage) Descriptor() ([]byte, []int) {
	return file_compiler_lib_testdata_src_test_proto_rawDescGZIP(), []int{4, 0}
}

func (x *MessageWithSubMessage_SubMessage) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

var File_compiler_lib_testdata_src_test_proto protoreflect.FileDescriptor

var file_compiler_lib_testdata_src_test_proto_rawDesc = []byte{
	0x0a, 0x24, 0x63, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x2f, 0x6c, 0x69, 0x62, 0x2f, 0x74,
	0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x73, 0x72, 0x63, 0x2f, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x19, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x9b, 0x01, 0x0a, 0x0b, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x12, 0x20, 0x0a, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x12, 0x31, 0x0a, 0x09, 0x61, 0x6e, 0x79, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e, 0x79, 0x52, 0x08, 0x61, 0x6e,
	0x79, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x37, 0x0a, 0x0c, 0x61, 0x6e, 0x79, 0x5f, 0x72, 0x65,
	0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41,
	0x6e, 0x79, 0x52, 0x0b, 0x61, 0x6e, 0x79, 0x52, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x22,
	0x7e, 0x0a, 0x16, 0x41, 0x6e, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x57, 0x69, 0x74, 0x68, 0x45, 0x6e, 0x75, 0x6d, 0x12, 0x35, 0x0a, 0x01, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x27, 0x2e, 0x41, 0x6e, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x57, 0x69, 0x74, 0x68, 0x45, 0x6e, 0x75, 0x6d, 0x2e, 0x41, 0x6e,
	0x6f, 0x74, 0x68, 0x65, 0x72, 0x54, 0x65, 0x73, 0x74, 0x45, 0x6e, 0x75, 0x6d, 0x52, 0x01, 0x65,
	0x22, 0x2d, 0x0a, 0x0f, 0x41, 0x6e, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x54, 0x65, 0x73, 0x74, 0x45,
	0x6e, 0x75, 0x6d, 0x12, 0x08, 0x0a, 0x04, 0x5a, 0x45, 0x52, 0x4f, 0x10, 0x00, 0x12, 0x07, 0x0a,
	0x03, 0x4f, 0x4e, 0x45, 0x10, 0x01, 0x12, 0x07, 0x0a, 0x03, 0x54, 0x57, 0x4f, 0x10, 0x02, 0x22,
	0x62, 0x0a, 0x0f, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x57, 0x69, 0x74, 0x68, 0x45, 0x6e,
	0x75, 0x6d, 0x12, 0x27, 0x0a, 0x01, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x19, 0x2e,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x57, 0x69, 0x74, 0x68, 0x45, 0x6e, 0x75, 0x6d, 0x2e,
	0x54, 0x65, 0x73, 0x74, 0x45, 0x6e, 0x75, 0x6d, 0x52, 0x01, 0x65, 0x22, 0x26, 0x0a, 0x08, 0x54,
	0x65, 0x73, 0x74, 0x45, 0x6e, 0x75, 0x6d, 0x12, 0x08, 0x0a, 0x04, 0x5a, 0x45, 0x52, 0x4f, 0x10,
	0x00, 0x12, 0x07, 0x0a, 0x03, 0x4f, 0x4e, 0x45, 0x10, 0x01, 0x12, 0x07, 0x0a, 0x03, 0x54, 0x57,
	0x4f, 0x10, 0x02, 0x22, 0xf8, 0x01, 0x0a, 0x0a, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x4d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x6e, 0x6f, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6e, 0x6f, 0x74, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x3f,
	0x0a, 0x0c, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4d,
	0x65, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x0b, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4d, 0x61, 0x70, 0x12,
	0x24, 0x0a, 0x06, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0c, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x06, 0x6e,
	0x65, 0x73, 0x74, 0x65, 0x64, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65,
	0x64, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e,
	0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x1a, 0x3e,
	0x0a, 0x10, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x70,
	0x0a, 0x15, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x57, 0x69, 0x74, 0x68, 0x53, 0x75, 0x62,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x33, 0x0a, 0x03, 0x73, 0x75, 0x62, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x57, 0x69,
	0x74, 0x68, 0x53, 0x75, 0x62, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x53, 0x75, 0x62,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x03, 0x73, 0x75, 0x62, 0x1a, 0x22, 0x0a, 0x0a,
	0x53, 0x75, 0x62, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_compiler_lib_testdata_src_test_proto_rawDescOnce sync.Once
	file_compiler_lib_testdata_src_test_proto_rawDescData = file_compiler_lib_testdata_src_test_proto_rawDesc
)

func file_compiler_lib_testdata_src_test_proto_rawDescGZIP() []byte {
	file_compiler_lib_testdata_src_test_proto_rawDescOnce.Do(func() {
		file_compiler_lib_testdata_src_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_compiler_lib_testdata_src_test_proto_rawDescData)
	})
	return file_compiler_lib_testdata_src_test_proto_rawDescData
}

var file_compiler_lib_testdata_src_test_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_compiler_lib_testdata_src_test_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_compiler_lib_testdata_src_test_proto_goTypes = []interface{}{
	(AnotherMessageWithEnum_AnotherTestEnum)(0), // 0: AnotherMessageWithEnum.AnotherTestEnum
	(MessageWithEnum_TestEnum)(0),               // 1: MessageWithEnum.TestEnum
	(*TestMessage)(nil),                         // 2: TestMessage
	(*AnotherMessageWithEnum)(nil),              // 3: AnotherMessageWithEnum
	(*MessageWithEnum)(nil),                     // 4: MessageWithEnum
	(*ValidateMe)(nil),                          // 5: ValidateMe
	(*MessageWithSubMessage)(nil),               // 6: MessageWithSubMessage
	nil,                                         // 7: ValidateMe.ValidateMapEntry
	(*MessageWithSubMessage_SubMessage)(nil),    // 8: MessageWithSubMessage.SubMessage
	(*any.Any)(nil),                             // 9: google.protobuf.Any
}
var file_compiler_lib_testdata_src_test_proto_depIdxs = []int32{
	9, // 0: TestMessage.any_field:type_name -> google.protobuf.Any
	9, // 1: TestMessage.any_repeated:type_name -> google.protobuf.Any
	0, // 2: AnotherMessageWithEnum.e:type_name -> AnotherMessageWithEnum.AnotherTestEnum
	1, // 3: MessageWithEnum.e:type_name -> MessageWithEnum.TestEnum
	7, // 4: ValidateMe.validate_map:type_name -> ValidateMe.ValidateMapEntry
	2, // 5: ValidateMe.nested:type_name -> TestMessage
	8, // 6: MessageWithSubMessage.sub:type_name -> MessageWithSubMessage.SubMessage
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_compiler_lib_testdata_src_test_proto_init() }
func file_compiler_lib_testdata_src_test_proto_init() {
	if File_compiler_lib_testdata_src_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_compiler_lib_testdata_src_test_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_compiler_lib_testdata_src_test_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AnotherMessageWithEnum); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_compiler_lib_testdata_src_test_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MessageWithEnum); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_compiler_lib_testdata_src_test_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ValidateMe); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_compiler_lib_testdata_src_test_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MessageWithSubMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_compiler_lib_testdata_src_test_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MessageWithSubMessage_SubMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_compiler_lib_testdata_src_test_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_compiler_lib_testdata_src_test_proto_goTypes,
		DependencyIndexes: file_compiler_lib_testdata_src_test_proto_depIdxs,
		EnumInfos:         file_compiler_lib_testdata_src_test_proto_enumTypes,
		MessageInfos:      file_compiler_lib_testdata_src_test_proto_msgTypes,
	}.Build()
	File_compiler_lib_testdata_src_test_proto = out.File
	file_compiler_lib_testdata_src_test_proto_rawDesc = nil
	file_compiler_lib_testdata_src_test_proto_goTypes = nil
	file_compiler_lib_testdata_src_test_proto_depIdxs = nil
}
