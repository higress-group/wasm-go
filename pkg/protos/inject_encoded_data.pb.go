// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.14.0
// source: inject_encoded_data.proto

package envoy_source_extensions_common_wasm

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

type InjectEncodedDataToFilterChainArguments struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Body      string `protobuf:"bytes,1,opt,name=body,proto3" json:"body,omitempty"`
	Endstream bool   `protobuf:"varint,2,opt,name=endstream,proto3" json:"endstream,omitempty"`
}

func (x *InjectEncodedDataToFilterChainArguments) Reset() {
	*x = InjectEncodedDataToFilterChainArguments{}
	if protoimpl.UnsafeEnabled {
		mi := &file_inject_encoded_data_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InjectEncodedDataToFilterChainArguments) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InjectEncodedDataToFilterChainArguments) ProtoMessage() {}

func (x *InjectEncodedDataToFilterChainArguments) ProtoReflect() protoreflect.Message {
	mi := &file_inject_encoded_data_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InjectEncodedDataToFilterChainArguments.ProtoReflect.Descriptor instead.
func (*InjectEncodedDataToFilterChainArguments) Descriptor() ([]byte, []int) {
	return file_inject_encoded_data_proto_rawDescGZIP(), []int{0}
}

func (x *InjectEncodedDataToFilterChainArguments) GetBody() string {
	if x != nil {
		return x.Body
	}
	return ""
}

func (x *InjectEncodedDataToFilterChainArguments) GetEndstream() bool {
	if x != nil {
		return x.Endstream
	}
	return false
}

var File_inject_encoded_data_proto protoreflect.FileDescriptor

var file_inject_encoded_data_proto_rawDesc = []byte{
	0x0a, 0x19, 0x69, 0x6e, 0x6a, 0x65, 0x63, 0x74, 0x5f, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x65, 0x64,
	0x5f, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x23, 0x65, 0x6e, 0x76,
	0x6f, 0x79, 0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x77, 0x61, 0x73, 0x6d,
	0x22, 0x5b, 0x0a, 0x27, 0x49, 0x6e, 0x6a, 0x65, 0x63, 0x74, 0x45, 0x6e, 0x63, 0x6f, 0x64, 0x65,
	0x64, 0x44, 0x61, 0x74, 0x61, 0x54, 0x6f, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x43, 0x68, 0x61,
	0x69, 0x6e, 0x41, 0x72, 0x67, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x62,
	0x6f, 0x64, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x62, 0x6f, 0x64, 0x79, 0x12,
	0x1c, 0x0a, 0x09, 0x65, 0x6e, 0x64, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x09, 0x65, 0x6e, 0x64, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_inject_encoded_data_proto_rawDescOnce sync.Once
	file_inject_encoded_data_proto_rawDescData = file_inject_encoded_data_proto_rawDesc
)

func file_inject_encoded_data_proto_rawDescGZIP() []byte {
	file_inject_encoded_data_proto_rawDescOnce.Do(func() {
		file_inject_encoded_data_proto_rawDescData = protoimpl.X.CompressGZIP(file_inject_encoded_data_proto_rawDescData)
	})
	return file_inject_encoded_data_proto_rawDescData
}

var file_inject_encoded_data_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_inject_encoded_data_proto_goTypes = []interface{}{
	(*InjectEncodedDataToFilterChainArguments)(nil), // 0: envoy.source.extensions.common.wasm.InjectEncodedDataToFilterChainArguments
}
var file_inject_encoded_data_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_inject_encoded_data_proto_init() }
func file_inject_encoded_data_proto_init() {
	if File_inject_encoded_data_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_inject_encoded_data_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InjectEncodedDataToFilterChainArguments); i {
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
			RawDescriptor: file_inject_encoded_data_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_inject_encoded_data_proto_goTypes,
		DependencyIndexes: file_inject_encoded_data_proto_depIdxs,
		MessageInfos:      file_inject_encoded_data_proto_msgTypes,
	}.Build()
	File_inject_encoded_data_proto = out.File
	file_inject_encoded_data_proto_rawDesc = nil
	file_inject_encoded_data_proto_goTypes = nil
	file_inject_encoded_data_proto_depIdxs = nil
}
