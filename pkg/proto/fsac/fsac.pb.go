// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.31.1
// source: pkg/proto/fsac/fsac.proto

package fsac

import (
	v2 "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FileSystemAccessProfile struct {
	state                    protoimpl.MessageState `protogen:"open.v1"`
	BloomFilter              []byte                 `protobuf:"bytes,1,opt,name=bloom_filter,json=bloomFilter,proto3" json:"bloom_filter,omitempty"`
	BloomFilterHashFunctions uint32                 `protobuf:"varint,2,opt,name=bloom_filter_hash_functions,json=bloomFilterHashFunctions,proto3" json:"bloom_filter_hash_functions,omitempty"`
	unknownFields            protoimpl.UnknownFields
	sizeCache                protoimpl.SizeCache
}

func (x *FileSystemAccessProfile) Reset() {
	*x = FileSystemAccessProfile{}
	mi := &file_pkg_proto_fsac_fsac_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileSystemAccessProfile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileSystemAccessProfile) ProtoMessage() {}

func (x *FileSystemAccessProfile) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_fsac_fsac_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileSystemAccessProfile.ProtoReflect.Descriptor instead.
func (*FileSystemAccessProfile) Descriptor() ([]byte, []int) {
	return file_pkg_proto_fsac_fsac_proto_rawDescGZIP(), []int{0}
}

func (x *FileSystemAccessProfile) GetBloomFilter() []byte {
	if x != nil {
		return x.BloomFilter
	}
	return nil
}

func (x *FileSystemAccessProfile) GetBloomFilterHashFunctions() uint32 {
	if x != nil {
		return x.BloomFilterHashFunctions
	}
	return 0
}

type GetFileSystemAccessProfileRequest struct {
	state               protoimpl.MessageState  `protogen:"open.v1"`
	InstanceName        string                  `protobuf:"bytes,1,opt,name=instance_name,json=instanceName,proto3" json:"instance_name,omitempty"`
	DigestFunction      v2.DigestFunction_Value `protobuf:"varint,2,opt,name=digest_function,json=digestFunction,proto3,enum=build.bazel.remote.execution.v2.DigestFunction_Value" json:"digest_function,omitempty"`
	ReducedActionDigest *v2.Digest              `protobuf:"bytes,3,opt,name=reduced_action_digest,json=reducedActionDigest,proto3" json:"reduced_action_digest,omitempty"`
	unknownFields       protoimpl.UnknownFields
	sizeCache           protoimpl.SizeCache
}

func (x *GetFileSystemAccessProfileRequest) Reset() {
	*x = GetFileSystemAccessProfileRequest{}
	mi := &file_pkg_proto_fsac_fsac_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetFileSystemAccessProfileRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetFileSystemAccessProfileRequest) ProtoMessage() {}

func (x *GetFileSystemAccessProfileRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_fsac_fsac_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetFileSystemAccessProfileRequest.ProtoReflect.Descriptor instead.
func (*GetFileSystemAccessProfileRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_fsac_fsac_proto_rawDescGZIP(), []int{1}
}

func (x *GetFileSystemAccessProfileRequest) GetInstanceName() string {
	if x != nil {
		return x.InstanceName
	}
	return ""
}

func (x *GetFileSystemAccessProfileRequest) GetDigestFunction() v2.DigestFunction_Value {
	if x != nil {
		return x.DigestFunction
	}
	return v2.DigestFunction_Value(0)
}

func (x *GetFileSystemAccessProfileRequest) GetReducedActionDigest() *v2.Digest {
	if x != nil {
		return x.ReducedActionDigest
	}
	return nil
}

type UpdateFileSystemAccessProfileRequest struct {
	state                   protoimpl.MessageState   `protogen:"open.v1"`
	InstanceName            string                   `protobuf:"bytes,1,opt,name=instance_name,json=instanceName,proto3" json:"instance_name,omitempty"`
	DigestFunction          v2.DigestFunction_Value  `protobuf:"varint,2,opt,name=digest_function,json=digestFunction,proto3,enum=build.bazel.remote.execution.v2.DigestFunction_Value" json:"digest_function,omitempty"`
	ReducedActionDigest     *v2.Digest               `protobuf:"bytes,3,opt,name=reduced_action_digest,json=reducedActionDigest,proto3" json:"reduced_action_digest,omitempty"`
	FileSystemAccessProfile *FileSystemAccessProfile `protobuf:"bytes,4,opt,name=file_system_access_profile,json=fileSystemAccessProfile,proto3" json:"file_system_access_profile,omitempty"`
	unknownFields           protoimpl.UnknownFields
	sizeCache               protoimpl.SizeCache
}

func (x *UpdateFileSystemAccessProfileRequest) Reset() {
	*x = UpdateFileSystemAccessProfileRequest{}
	mi := &file_pkg_proto_fsac_fsac_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateFileSystemAccessProfileRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateFileSystemAccessProfileRequest) ProtoMessage() {}

func (x *UpdateFileSystemAccessProfileRequest) ProtoReflect() protoreflect.Message {
	mi := &file_pkg_proto_fsac_fsac_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateFileSystemAccessProfileRequest.ProtoReflect.Descriptor instead.
func (*UpdateFileSystemAccessProfileRequest) Descriptor() ([]byte, []int) {
	return file_pkg_proto_fsac_fsac_proto_rawDescGZIP(), []int{2}
}

func (x *UpdateFileSystemAccessProfileRequest) GetInstanceName() string {
	if x != nil {
		return x.InstanceName
	}
	return ""
}

func (x *UpdateFileSystemAccessProfileRequest) GetDigestFunction() v2.DigestFunction_Value {
	if x != nil {
		return x.DigestFunction
	}
	return v2.DigestFunction_Value(0)
}

func (x *UpdateFileSystemAccessProfileRequest) GetReducedActionDigest() *v2.Digest {
	if x != nil {
		return x.ReducedActionDigest
	}
	return nil
}

func (x *UpdateFileSystemAccessProfileRequest) GetFileSystemAccessProfile() *FileSystemAccessProfile {
	if x != nil {
		return x.FileSystemAccessProfile
	}
	return nil
}

var File_pkg_proto_fsac_fsac_proto protoreflect.FileDescriptor

const file_pkg_proto_fsac_fsac_proto_rawDesc = "" +
	"\n" +
	"\x19pkg/proto/fsac/fsac.proto\x12\x0ebuildbarn.fsac\x1a6build/bazel/remote/execution/v2/remote_execution.proto\x1a\x1bgoogle/protobuf/empty.proto\"{\n" +
	"\x17FileSystemAccessProfile\x12!\n" +
	"\fbloom_filter\x18\x01 \x01(\fR\vbloomFilter\x12=\n" +
	"\x1bbloom_filter_hash_functions\x18\x02 \x01(\rR\x18bloomFilterHashFunctions\"\x85\x02\n" +
	"!GetFileSystemAccessProfileRequest\x12#\n" +
	"\rinstance_name\x18\x01 \x01(\tR\finstanceName\x12^\n" +
	"\x0fdigest_function\x18\x02 \x01(\x0e25.build.bazel.remote.execution.v2.DigestFunction.ValueR\x0edigestFunction\x12[\n" +
	"\x15reduced_action_digest\x18\x03 \x01(\v2'.build.bazel.remote.execution.v2.DigestR\x13reducedActionDigest\"\xee\x02\n" +
	"$UpdateFileSystemAccessProfileRequest\x12#\n" +
	"\rinstance_name\x18\x01 \x01(\tR\finstanceName\x12^\n" +
	"\x0fdigest_function\x18\x02 \x01(\x0e25.build.bazel.remote.execution.v2.DigestFunction.ValueR\x0edigestFunction\x12[\n" +
	"\x15reduced_action_digest\x18\x03 \x01(\v2'.build.bazel.remote.execution.v2.DigestR\x13reducedActionDigest\x12d\n" +
	"\x1afile_system_access_profile\x18\x04 \x01(\v2'.buildbarn.fsac.FileSystemAccessProfileR\x17fileSystemAccessProfile2\x80\x02\n" +
	"\x15FileSystemAccessCache\x12x\n" +
	"\x1aGetFileSystemAccessProfile\x121.buildbarn.fsac.GetFileSystemAccessProfileRequest\x1a'.buildbarn.fsac.FileSystemAccessProfile\x12m\n" +
	"\x1dUpdateFileSystemAccessProfile\x124.buildbarn.fsac.UpdateFileSystemAccessProfileRequest\x1a\x16.google.protobuf.EmptyB0Z.github.com/buildbarn/bb-storage/pkg/proto/fsacb\x06proto3"

var (
	file_pkg_proto_fsac_fsac_proto_rawDescOnce sync.Once
	file_pkg_proto_fsac_fsac_proto_rawDescData []byte
)

func file_pkg_proto_fsac_fsac_proto_rawDescGZIP() []byte {
	file_pkg_proto_fsac_fsac_proto_rawDescOnce.Do(func() {
		file_pkg_proto_fsac_fsac_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_pkg_proto_fsac_fsac_proto_rawDesc), len(file_pkg_proto_fsac_fsac_proto_rawDesc)))
	})
	return file_pkg_proto_fsac_fsac_proto_rawDescData
}

var file_pkg_proto_fsac_fsac_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_pkg_proto_fsac_fsac_proto_goTypes = []any{
	(*FileSystemAccessProfile)(nil),              // 0: buildbarn.fsac.FileSystemAccessProfile
	(*GetFileSystemAccessProfileRequest)(nil),    // 1: buildbarn.fsac.GetFileSystemAccessProfileRequest
	(*UpdateFileSystemAccessProfileRequest)(nil), // 2: buildbarn.fsac.UpdateFileSystemAccessProfileRequest
	(v2.DigestFunction_Value)(0),                 // 3: build.bazel.remote.execution.v2.DigestFunction.Value
	(*v2.Digest)(nil),                            // 4: build.bazel.remote.execution.v2.Digest
	(*emptypb.Empty)(nil),                        // 5: google.protobuf.Empty
}
var file_pkg_proto_fsac_fsac_proto_depIdxs = []int32{
	3, // 0: buildbarn.fsac.GetFileSystemAccessProfileRequest.digest_function:type_name -> build.bazel.remote.execution.v2.DigestFunction.Value
	4, // 1: buildbarn.fsac.GetFileSystemAccessProfileRequest.reduced_action_digest:type_name -> build.bazel.remote.execution.v2.Digest
	3, // 2: buildbarn.fsac.UpdateFileSystemAccessProfileRequest.digest_function:type_name -> build.bazel.remote.execution.v2.DigestFunction.Value
	4, // 3: buildbarn.fsac.UpdateFileSystemAccessProfileRequest.reduced_action_digest:type_name -> build.bazel.remote.execution.v2.Digest
	0, // 4: buildbarn.fsac.UpdateFileSystemAccessProfileRequest.file_system_access_profile:type_name -> buildbarn.fsac.FileSystemAccessProfile
	1, // 5: buildbarn.fsac.FileSystemAccessCache.GetFileSystemAccessProfile:input_type -> buildbarn.fsac.GetFileSystemAccessProfileRequest
	2, // 6: buildbarn.fsac.FileSystemAccessCache.UpdateFileSystemAccessProfile:input_type -> buildbarn.fsac.UpdateFileSystemAccessProfileRequest
	0, // 7: buildbarn.fsac.FileSystemAccessCache.GetFileSystemAccessProfile:output_type -> buildbarn.fsac.FileSystemAccessProfile
	5, // 8: buildbarn.fsac.FileSystemAccessCache.UpdateFileSystemAccessProfile:output_type -> google.protobuf.Empty
	7, // [7:9] is the sub-list for method output_type
	5, // [5:7] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_pkg_proto_fsac_fsac_proto_init() }
func file_pkg_proto_fsac_fsac_proto_init() {
	if File_pkg_proto_fsac_fsac_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_pkg_proto_fsac_fsac_proto_rawDesc), len(file_pkg_proto_fsac_fsac_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_pkg_proto_fsac_fsac_proto_goTypes,
		DependencyIndexes: file_pkg_proto_fsac_fsac_proto_depIdxs,
		MessageInfos:      file_pkg_proto_fsac_fsac_proto_msgTypes,
	}.Build()
	File_pkg_proto_fsac_fsac_proto = out.File
	file_pkg_proto_fsac_fsac_proto_goTypes = nil
	file_pkg_proto_fsac_fsac_proto_depIdxs = nil
}
