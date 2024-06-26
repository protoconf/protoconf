// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.25.3
// source: datatypes/proto/v1/protoconf_value.proto

package datatypes

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	anypb "google.golang.org/protobuf/types/known/anypb"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ProtoconfValue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProtoFile     string                        `protobuf:"bytes,1,opt,name=proto_file,json=protoFile,proto3" json:"proto_file,omitempty"`
	Value         *anypb.Any                    `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	RolloutConfig *ProtoconfValue_ConfigRollout `protobuf:"bytes,3,opt,name=rollout_config,json=rolloutConfig,proto3" json:"rollout_config,omitempty"`
	Metadata      *Metadata                     `protobuf:"bytes,4,opt,name=metadata,proto3" json:"metadata,omitempty"`
}

func (x *ProtoconfValue) Reset() {
	*x = ProtoconfValue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoconfValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoconfValue) ProtoMessage() {}

func (x *ProtoconfValue) ProtoReflect() protoreflect.Message {
	mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoconfValue.ProtoReflect.Descriptor instead.
func (*ProtoconfValue) Descriptor() ([]byte, []int) {
	return file_datatypes_proto_v1_protoconf_value_proto_rawDescGZIP(), []int{0}
}

func (x *ProtoconfValue) GetProtoFile() string {
	if x != nil {
		return x.ProtoFile
	}
	return ""
}

func (x *ProtoconfValue) GetValue() *anypb.Any {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *ProtoconfValue) GetRolloutConfig() *ProtoconfValue_ConfigRollout {
	if x != nil {
		return x.RolloutConfig
	}
	return nil
}

func (x *ProtoconfValue) GetMetadata() *Metadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type Metadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Commit         string                 `protobuf:"bytes,1,opt,name=commit,proto3" json:"commit,omitempty"`
	CommitterEmail string                 `protobuf:"bytes,2,opt,name=committer_email,json=committerEmail,proto3" json:"committer_email,omitempty"`
	AuthorEmail    string                 `protobuf:"bytes,3,opt,name=author_email,json=authorEmail,proto3" json:"author_email,omitempty"`
	CommittedAt    *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=committed_at,json=committedAt,proto3" json:"committed_at,omitempty"`
	AuthoredAt     *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=authored_at,json=authoredAt,proto3" json:"authored_at,omitempty"`
	InsertedAt     *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=inserted_at,json=insertedAt,proto3" json:"inserted_at,omitempty"`
	Channel        string                 `protobuf:"bytes,7,opt,name=channel,proto3" json:"channel,omitempty"`
}

func (x *Metadata) Reset() {
	*x = Metadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metadata) ProtoMessage() {}

func (x *Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metadata.ProtoReflect.Descriptor instead.
func (*Metadata) Descriptor() ([]byte, []int) {
	return file_datatypes_proto_v1_protoconf_value_proto_rawDescGZIP(), []int{1}
}

func (x *Metadata) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *Metadata) GetCommitterEmail() string {
	if x != nil {
		return x.CommitterEmail
	}
	return ""
}

func (x *Metadata) GetAuthorEmail() string {
	if x != nil {
		return x.AuthorEmail
	}
	return ""
}

func (x *Metadata) GetCommittedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CommittedAt
	}
	return nil
}

func (x *Metadata) GetAuthoredAt() *timestamppb.Timestamp {
	if x != nil {
		return x.AuthoredAt
	}
	return nil
}

func (x *Metadata) GetInsertedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.InsertedAt
	}
	return nil
}

func (x *Metadata) GetChannel() string {
	if x != nil {
		return x.Channel
	}
	return ""
}

type ProtoconfValue_ConfigRollout struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// cooldown is the duration of time the inserter waits before applying
	// the next stage, Uses protobuf google.protobuf.Duration well-known-type
	DefaultCooldownTime *durationpb.Duration `protobuf:"bytes,1,opt,name=default_cooldown_time,json=defaultCooldownTime,proto3" json:"default_cooldown_time,omitempty"`
	// expiration will be added to the time of insertion to be set as
	// `expires_at`. When `expires_at` is due, the config should revert
	// to default.
	DefaultExpirationTime *durationpb.Duration                  `protobuf:"bytes,2,opt,name=default_expiration_time,json=defaultExpirationTime,proto3" json:"default_expiration_time,omitempty"`
	Stages                []*ProtoconfValue_ConfigRollout_Stage `protobuf:"bytes,3,rep,name=stages,proto3" json:"stages,omitempty"`
	Namespace             string                                `protobuf:"bytes,4,opt,name=namespace,proto3" json:"namespace,omitempty"`
}

func (x *ProtoconfValue_ConfigRollout) Reset() {
	*x = ProtoconfValue_ConfigRollout{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoconfValue_ConfigRollout) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoconfValue_ConfigRollout) ProtoMessage() {}

func (x *ProtoconfValue_ConfigRollout) ProtoReflect() protoreflect.Message {
	mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoconfValue_ConfigRollout.ProtoReflect.Descriptor instead.
func (*ProtoconfValue_ConfigRollout) Descriptor() ([]byte, []int) {
	return file_datatypes_proto_v1_protoconf_value_proto_rawDescGZIP(), []int{0, 0}
}

func (x *ProtoconfValue_ConfigRollout) GetDefaultCooldownTime() *durationpb.Duration {
	if x != nil {
		return x.DefaultCooldownTime
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout) GetDefaultExpirationTime() *durationpb.Duration {
	if x != nil {
		return x.DefaultExpirationTime
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout) GetStages() []*ProtoconfValue_ConfigRollout_Stage {
	if x != nil {
		return x.Stages
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

type ProtoconfValue_ConfigRollout_Stage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Channel    string                 `protobuf:"bytes,1,opt,name=channel,proto3" json:"channel,omitempty"`
	Percentile uint32                 `protobuf:"varint,2,opt,name=percentile,proto3" json:"percentile,omitempty"`
	Labels     map[string]string      `protobuf:"bytes,3,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Manual     bool                   `protobuf:"varint,4,opt,name=manual,proto3" json:"manual,omitempty"`
	Cooldown   *durationpb.Duration   `protobuf:"bytes,5,opt,name=cooldown,proto3" json:"cooldown,omitempty"`
	Expiration *durationpb.Duration   `protobuf:"bytes,6,opt,name=expiration,proto3" json:"expiration,omitempty"`
	ExpiresAt  *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=expires_at,json=expiresAt,proto3" json:"expires_at,omitempty"`
	Version    string                 `protobuf:"bytes,8,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *ProtoconfValue_ConfigRollout_Stage) Reset() {
	*x = ProtoconfValue_ConfigRollout_Stage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoconfValue_ConfigRollout_Stage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoconfValue_ConfigRollout_Stage) ProtoMessage() {}

func (x *ProtoconfValue_ConfigRollout_Stage) ProtoReflect() protoreflect.Message {
	mi := &file_datatypes_proto_v1_protoconf_value_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoconfValue_ConfigRollout_Stage.ProtoReflect.Descriptor instead.
func (*ProtoconfValue_ConfigRollout_Stage) Descriptor() ([]byte, []int) {
	return file_datatypes_proto_v1_protoconf_value_proto_rawDescGZIP(), []int{0, 0, 0}
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetChannel() string {
	if x != nil {
		return x.Channel
	}
	return ""
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetPercentile() uint32 {
	if x != nil {
		return x.Percentile
	}
	return 0
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetManual() bool {
	if x != nil {
		return x.Manual
	}
	return false
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetCooldown() *durationpb.Duration {
	if x != nil {
		return x.Cooldown
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetExpiration() *durationpb.Duration {
	if x != nil {
		return x.Expiration
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetExpiresAt() *timestamppb.Timestamp {
	if x != nil {
		return x.ExpiresAt
	}
	return nil
}

func (x *ProtoconfValue_ConfigRollout_Stage) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

var File_datatypes_proto_v1_protoconf_value_proto protoreflect.FileDescriptor

var file_datatypes_proto_v1_protoconf_value_proto_rawDesc = []byte{
	0x0a, 0x28, 0x64, 0x61, 0x74, 0x61, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x5f, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31, 0x1a, 0x19,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f,
	0x61, 0x6e, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8a, 0x07, 0x0a, 0x0e, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x1d, 0x0a,
	0x0a, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x2a, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x14, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x41, 0x6e,
	0x79, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x47, 0x0a, 0x0e, 0x72, 0x6f, 0x6c, 0x6c,
	0x6f, 0x75, 0x74, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x20, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x6f, 0x6c, 0x6c, 0x6f,
	0x75, 0x74, 0x52, 0x0d, 0x72, 0x6f, 0x6c, 0x6c, 0x6f, 0x75, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x28, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x1a, 0xb9, 0x05, 0x0a, 0x0d,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x6f, 0x6c, 0x6c, 0x6f, 0x75, 0x74, 0x12, 0x4d, 0x0a,
	0x15, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x63, 0x6f, 0x6f, 0x6c, 0x64, 0x6f, 0x77,
	0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x13, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74,
	0x43, 0x6f, 0x6f, 0x6c, 0x64, 0x6f, 0x77, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x51, 0x0a, 0x17,
	0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x5f, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x15, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c,
	0x74, 0x45, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x3e, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x67, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x26, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x6f, 0x6c, 0x6c, 0x6f, 0x75,
	0x74, 0x2e, 0x53, 0x74, 0x61, 0x67, 0x65, 0x52, 0x06, 0x73, 0x74, 0x61, 0x67, 0x65, 0x73, 0x12,
	0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x1a, 0xa7, 0x03,
	0x0a, 0x05, 0x53, 0x74, 0x61, 0x67, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65,
	0x6c, 0x12, 0x1e, 0x0a, 0x0a, 0x70, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x69, 0x6c, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x70, 0x65, 0x72, 0x63, 0x65, 0x6e, 0x74, 0x69, 0x6c,
	0x65, 0x12, 0x4a, 0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x32, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x6f, 0x6c, 0x6c,
	0x6f, 0x75, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x67, 0x65, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x12, 0x16, 0x0a,
	0x06, 0x6d, 0x61, 0x6e, 0x75, 0x61, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x6d,
	0x61, 0x6e, 0x75, 0x61, 0x6c, 0x12, 0x35, 0x0a, 0x08, 0x63, 0x6f, 0x6f, 0x6c, 0x64, 0x6f, 0x77,
	0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x52, 0x08, 0x63, 0x6f, 0x6f, 0x6c, 0x64, 0x6f, 0x77, 0x6e, 0x12, 0x39, 0x0a, 0x0a,
	0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x65, 0x78, 0x70,
	0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x39, 0x0a, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72,
	0x65, 0x73, 0x5f, 0x61, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x73,
	0x41, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x08, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x1a, 0x39, 0x0a, 0x0b,
	0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xc1, 0x02, 0x0a, 0x08, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x27, 0x0a, 0x0f,
	0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x72, 0x5f, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x72,
	0x45, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x21, 0x0a, 0x0c, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x5f,
	0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x61, 0x75, 0x74,
	0x68, 0x6f, 0x72, 0x45, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x3d, 0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0b, 0x63, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x3b, 0x0a, 0x0b, 0x61, 0x75, 0x74, 0x68, 0x6f,
	0x72, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72,
	0x65, 0x64, 0x41, 0x74, 0x12, 0x3b, 0x0a, 0x0b, 0x69, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x65, 0x64,
	0x5f, 0x61, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x65, 0x72, 0x74, 0x65, 0x64, 0x41,
	0x74, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x42, 0x59, 0x0a, 0x1a, 0x63,
	0x6f, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2e, 0x64, 0x61, 0x74,
	0x61, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x76, 0x31, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x74, 0x79,
	0x70, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x31, 0x3b, 0x64, 0x61, 0x74,
	0x61, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_datatypes_proto_v1_protoconf_value_proto_rawDescOnce sync.Once
	file_datatypes_proto_v1_protoconf_value_proto_rawDescData = file_datatypes_proto_v1_protoconf_value_proto_rawDesc
)

func file_datatypes_proto_v1_protoconf_value_proto_rawDescGZIP() []byte {
	file_datatypes_proto_v1_protoconf_value_proto_rawDescOnce.Do(func() {
		file_datatypes_proto_v1_protoconf_value_proto_rawDescData = protoimpl.X.CompressGZIP(file_datatypes_proto_v1_protoconf_value_proto_rawDescData)
	})
	return file_datatypes_proto_v1_protoconf_value_proto_rawDescData
}

var file_datatypes_proto_v1_protoconf_value_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_datatypes_proto_v1_protoconf_value_proto_goTypes = []interface{}{
	(*ProtoconfValue)(nil),                     // 0: v1.ProtoconfValue
	(*Metadata)(nil),                           // 1: v1.Metadata
	(*ProtoconfValue_ConfigRollout)(nil),       // 2: v1.ProtoconfValue.ConfigRollout
	(*ProtoconfValue_ConfigRollout_Stage)(nil), // 3: v1.ProtoconfValue.ConfigRollout.Stage
	nil,                           // 4: v1.ProtoconfValue.ConfigRollout.Stage.LabelsEntry
	(*anypb.Any)(nil),             // 5: google.protobuf.Any
	(*timestamppb.Timestamp)(nil), // 6: google.protobuf.Timestamp
	(*durationpb.Duration)(nil),   // 7: google.protobuf.Duration
}
var file_datatypes_proto_v1_protoconf_value_proto_depIdxs = []int32{
	5,  // 0: v1.ProtoconfValue.value:type_name -> google.protobuf.Any
	2,  // 1: v1.ProtoconfValue.rollout_config:type_name -> v1.ProtoconfValue.ConfigRollout
	1,  // 2: v1.ProtoconfValue.metadata:type_name -> v1.Metadata
	6,  // 3: v1.Metadata.committed_at:type_name -> google.protobuf.Timestamp
	6,  // 4: v1.Metadata.authored_at:type_name -> google.protobuf.Timestamp
	6,  // 5: v1.Metadata.inserted_at:type_name -> google.protobuf.Timestamp
	7,  // 6: v1.ProtoconfValue.ConfigRollout.default_cooldown_time:type_name -> google.protobuf.Duration
	7,  // 7: v1.ProtoconfValue.ConfigRollout.default_expiration_time:type_name -> google.protobuf.Duration
	3,  // 8: v1.ProtoconfValue.ConfigRollout.stages:type_name -> v1.ProtoconfValue.ConfigRollout.Stage
	4,  // 9: v1.ProtoconfValue.ConfigRollout.Stage.labels:type_name -> v1.ProtoconfValue.ConfigRollout.Stage.LabelsEntry
	7,  // 10: v1.ProtoconfValue.ConfigRollout.Stage.cooldown:type_name -> google.protobuf.Duration
	7,  // 11: v1.ProtoconfValue.ConfigRollout.Stage.expiration:type_name -> google.protobuf.Duration
	6,  // 12: v1.ProtoconfValue.ConfigRollout.Stage.expires_at:type_name -> google.protobuf.Timestamp
	13, // [13:13] is the sub-list for method output_type
	13, // [13:13] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_datatypes_proto_v1_protoconf_value_proto_init() }
func file_datatypes_proto_v1_protoconf_value_proto_init() {
	if File_datatypes_proto_v1_protoconf_value_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_datatypes_proto_v1_protoconf_value_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProtoconfValue); i {
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
		file_datatypes_proto_v1_protoconf_value_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metadata); i {
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
		file_datatypes_proto_v1_protoconf_value_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProtoconfValue_ConfigRollout); i {
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
		file_datatypes_proto_v1_protoconf_value_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProtoconfValue_ConfigRollout_Stage); i {
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
			RawDescriptor: file_datatypes_proto_v1_protoconf_value_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_datatypes_proto_v1_protoconf_value_proto_goTypes,
		DependencyIndexes: file_datatypes_proto_v1_protoconf_value_proto_depIdxs,
		MessageInfos:      file_datatypes_proto_v1_protoconf_value_proto_msgTypes,
	}.Build()
	File_datatypes_proto_v1_protoconf_value_proto = out.File
	file_datatypes_proto_v1_protoconf_value_proto_rawDesc = nil
	file_datatypes_proto_v1_protoconf_value_proto_goTypes = nil
	file_datatypes_proto_v1_protoconf_value_proto_depIdxs = nil
}
