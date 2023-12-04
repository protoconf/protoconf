// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.7
// source: remote_repo.proto

package module

import (
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

type RemoteRepo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Label string `protobuf:"bytes,1,opt,name=label,proto3" json:"label,omitempty"`
	Url   string `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty"`
	// Types that are assignable to Pin:
	//	*RemoteRepo_Tag
	//	*RemoteRepo_Branch
	//	*RemoteRepo_Commit
	//	*RemoteRepo_Checksum
	Pin                  isRemoteRepo_Pin       `protobuf_oneof:"pin"`
	SourcePath           string                 `protobuf:"bytes,7,opt,name=source_path,json=sourcePath,proto3" json:"source_path,omitempty"`
	AdditionalProtoDirs  []string               `protobuf:"bytes,8,rep,name=additional_proto_dirs,json=additionalProtoDirs,proto3" json:"additional_proto_dirs,omitempty"`
	Deps                 map[string]*RemoteRepo `protobuf:"bytes,9,rep,name=deps,proto3" json:"deps,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Integrity            string                 `protobuf:"bytes,10,opt,name=integrity,proto3" json:"integrity,omitempty"`
	GetterUrl            string                 `protobuf:"bytes,11,opt,name=getter_url,json=getterUrl,proto3" json:"getter_url,omitempty"`
	FileDescriptorSetSum string                 `protobuf:"bytes,12,opt,name=file_descriptor_set_sum,json=fileDescriptorSetSum,proto3" json:"file_descriptor_set_sum,omitempty"`
}

func (x *RemoteRepo) Reset() {
	*x = RemoteRepo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_remote_repo_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoteRepo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoteRepo) ProtoMessage() {}

func (x *RemoteRepo) ProtoReflect() protoreflect.Message {
	mi := &file_remote_repo_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoteRepo.ProtoReflect.Descriptor instead.
func (*RemoteRepo) Descriptor() ([]byte, []int) {
	return file_remote_repo_proto_rawDescGZIP(), []int{0}
}

func (x *RemoteRepo) GetLabel() string {
	if x != nil {
		return x.Label
	}
	return ""
}

func (x *RemoteRepo) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (m *RemoteRepo) GetPin() isRemoteRepo_Pin {
	if m != nil {
		return m.Pin
	}
	return nil
}

func (x *RemoteRepo) GetTag() string {
	if x, ok := x.GetPin().(*RemoteRepo_Tag); ok {
		return x.Tag
	}
	return ""
}

func (x *RemoteRepo) GetBranch() string {
	if x, ok := x.GetPin().(*RemoteRepo_Branch); ok {
		return x.Branch
	}
	return ""
}

func (x *RemoteRepo) GetCommit() string {
	if x, ok := x.GetPin().(*RemoteRepo_Commit); ok {
		return x.Commit
	}
	return ""
}

func (x *RemoteRepo) GetChecksum() string {
	if x, ok := x.GetPin().(*RemoteRepo_Checksum); ok {
		return x.Checksum
	}
	return ""
}

func (x *RemoteRepo) GetSourcePath() string {
	if x != nil {
		return x.SourcePath
	}
	return ""
}

func (x *RemoteRepo) GetAdditionalProtoDirs() []string {
	if x != nil {
		return x.AdditionalProtoDirs
	}
	return nil
}

func (x *RemoteRepo) GetDeps() map[string]*RemoteRepo {
	if x != nil {
		return x.Deps
	}
	return nil
}

func (x *RemoteRepo) GetIntegrity() string {
	if x != nil {
		return x.Integrity
	}
	return ""
}

func (x *RemoteRepo) GetGetterUrl() string {
	if x != nil {
		return x.GetterUrl
	}
	return ""
}

func (x *RemoteRepo) GetFileDescriptorSetSum() string {
	if x != nil {
		return x.FileDescriptorSetSum
	}
	return ""
}

type isRemoteRepo_Pin interface {
	isRemoteRepo_Pin()
}

type RemoteRepo_Tag struct {
	Tag string `protobuf:"bytes,3,opt,name=tag,proto3,oneof"`
}

type RemoteRepo_Branch struct {
	Branch string `protobuf:"bytes,4,opt,name=branch,proto3,oneof"`
}

type RemoteRepo_Commit struct {
	Commit string `protobuf:"bytes,5,opt,name=commit,proto3,oneof"`
}

type RemoteRepo_Checksum struct {
	Checksum string `protobuf:"bytes,6,opt,name=checksum,proto3,oneof"` // How to validate the integrity of a file if url leads to an archive
}

func (*RemoteRepo_Tag) isRemoteRepo_Pin() {}

func (*RemoteRepo_Branch) isRemoteRepo_Pin() {}

func (*RemoteRepo_Commit) isRemoteRepo_Pin() {}

func (*RemoteRepo_Checksum) isRemoteRepo_Pin() {}

type ModuleServiceConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProtoconfPath string `protobuf:"bytes,1,opt,name=protoconf_path,json=protoconfPath,proto3" json:"protoconf_path,omitempty"`
	CacheDir      string `protobuf:"bytes,2,opt,name=cache_dir,json=cacheDir,proto3" json:"cache_dir,omitempty"`
	LockFile      string `protobuf:"bytes,3,opt,name=lock_file,json=lockFile,proto3" json:"lock_file,omitempty"`
}

func (x *ModuleServiceConfig) Reset() {
	*x = ModuleServiceConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_remote_repo_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleServiceConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleServiceConfig) ProtoMessage() {}

func (x *ModuleServiceConfig) ProtoReflect() protoreflect.Message {
	mi := &file_remote_repo_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleServiceConfig.ProtoReflect.Descriptor instead.
func (*ModuleServiceConfig) Descriptor() ([]byte, []int) {
	return file_remote_repo_proto_rawDescGZIP(), []int{1}
}

func (x *ModuleServiceConfig) GetProtoconfPath() string {
	if x != nil {
		return x.ProtoconfPath
	}
	return ""
}

func (x *ModuleServiceConfig) GetCacheDir() string {
	if x != nil {
		return x.CacheDir
	}
	return ""
}

func (x *ModuleServiceConfig) GetLockFile() string {
	if x != nil {
		return x.LockFile
	}
	return ""
}

var File_remote_repo_proto protoreflect.FileDescriptor

var file_remote_repo_proto_rawDesc = []byte{
	0x0a, 0x11, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2e, 0x63,
	0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x76,
	0x31, 0x22, 0x95, 0x04, 0x0a, 0x0a, 0x52, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6f,
	0x12, 0x14, 0x0a, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12, 0x12, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x03, 0x74, 0x61, 0x67, 0x12, 0x18, 0x0a, 0x06,
	0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x06,
	0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x12, 0x18, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x12, 0x1c, 0x0a, 0x08, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x00, 0x52, 0x08, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x12, 0x1f,
	0x0a, 0x0b, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12,
	0x32, 0x0a, 0x15, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x5f, 0x64, 0x69, 0x72, 0x73, 0x18, 0x08, 0x20, 0x03, 0x28, 0x09, 0x52, 0x13,
	0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x44,
	0x69, 0x72, 0x73, 0x12, 0x46, 0x0a, 0x04, 0x64, 0x65, 0x70, 0x73, 0x18, 0x09, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x32, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2e, 0x63, 0x6f,
	0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x76, 0x31,
	0x2e, 0x52, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x2e, 0x44, 0x65, 0x70, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x64, 0x65, 0x70, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x69,
	0x6e, 0x74, 0x65, 0x67, 0x72, 0x69, 0x74, 0x79, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x69, 0x74, 0x79, 0x12, 0x1d, 0x0a, 0x0a, 0x67, 0x65, 0x74,
	0x74, 0x65, 0x72, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x67,
	0x65, 0x74, 0x74, 0x65, 0x72, 0x55, 0x72, 0x6c, 0x12, 0x35, 0x0a, 0x17, 0x66, 0x69, 0x6c, 0x65,
	0x5f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x5f, 0x73, 0x65, 0x74, 0x5f,
	0x73, 0x75, 0x6d, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x66, 0x69, 0x6c, 0x65, 0x44,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x53, 0x65, 0x74, 0x53, 0x75, 0x6d, 0x1a,
	0x61, 0x0a, 0x09, 0x44, 0x65, 0x70, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x3e,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2e, 0x63, 0x6f, 0x6d, 0x70, 0x69, 0x6c,
	0x65, 0x72, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x6d,
	0x6f, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6f, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x42, 0x05, 0x0a, 0x03, 0x70, 0x69, 0x6e, 0x22, 0x76, 0x0a, 0x13, 0x4d, 0x6f, 0x64,
	0x75, 0x6c, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x12, 0x25, 0x0a, 0x0e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x5f, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6e, 0x66, 0x50, 0x61, 0x74, 0x68, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x61, 0x63, 0x68, 0x65,
	0x5f, 0x64, 0x69, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x61, 0x63, 0x68,
	0x65, 0x44, 0x69, 0x72, 0x12, 0x1b, 0x0a, 0x09, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x66, 0x69, 0x6c,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6c, 0x6f, 0x63, 0x6b, 0x46, 0x69, 0x6c,
	0x65, 0x42, 0x3a, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6e, 0x66, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63,
	0x6f, 0x6e, 0x66, 0x2f, 0x63, 0x6f, 0x6d, 0x70, 0x69, 0x6c, 0x65, 0x72, 0x2f, 0x6d, 0x6f, 0x64,
	0x75, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_remote_repo_proto_rawDescOnce sync.Once
	file_remote_repo_proto_rawDescData = file_remote_repo_proto_rawDesc
)

func file_remote_repo_proto_rawDescGZIP() []byte {
	file_remote_repo_proto_rawDescOnce.Do(func() {
		file_remote_repo_proto_rawDescData = protoimpl.X.CompressGZIP(file_remote_repo_proto_rawDescData)
	})
	return file_remote_repo_proto_rawDescData
}

var file_remote_repo_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_remote_repo_proto_goTypes = []interface{}{
	(*RemoteRepo)(nil),          // 0: protoconf.compiler.module.v1.RemoteRepo
	(*ModuleServiceConfig)(nil), // 1: protoconf.compiler.module.v1.ModuleServiceConfig
	nil,                         // 2: protoconf.compiler.module.v1.RemoteRepo.DepsEntry
}
var file_remote_repo_proto_depIdxs = []int32{
	2, // 0: protoconf.compiler.module.v1.RemoteRepo.deps:type_name -> protoconf.compiler.module.v1.RemoteRepo.DepsEntry
	0, // 1: protoconf.compiler.module.v1.RemoteRepo.DepsEntry.value:type_name -> protoconf.compiler.module.v1.RemoteRepo
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_remote_repo_proto_init() }
func file_remote_repo_proto_init() {
	if File_remote_repo_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_remote_repo_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoteRepo); i {
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
		file_remote_repo_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModuleServiceConfig); i {
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
	file_remote_repo_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*RemoteRepo_Tag)(nil),
		(*RemoteRepo_Branch)(nil),
		(*RemoteRepo_Commit)(nil),
		(*RemoteRepo_Checksum)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_remote_repo_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_remote_repo_proto_goTypes,
		DependencyIndexes: file_remote_repo_proto_depIdxs,
		MessageInfos:      file_remote_repo_proto_msgTypes,
	}.Build()
	File_remote_repo_proto = out.File
	file_remote_repo_proto_rawDesc = nil
	file_remote_repo_proto_goTypes = nil
	file_remote_repo_proto_depIdxs = nil
}