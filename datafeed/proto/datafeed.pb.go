// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0-devel
// 	protoc        v3.15.8
// source: datafeed/proto/datafeed.proto

package proto

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

type DataFeedReport struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// marketHash is the market hash of the repot
	MarketHash string `protobuf:"bytes,1,opt,name=marketHash,proto3" json:"marketHash,omitempty"`
	// outcome is the outcome of the report
	Outcome string `protobuf:"bytes,2,opt,name=outcome,proto3" json:"outcome,omitempty"`
	// signatures is the concatenated list of validator signatures
	Signatures string `protobuf:"bytes,3,opt,name=signatures,proto3" json:"signatures,omitempty"`
	// epoch is the epoch at which the message was initially gossiped at
	Epoch uint64 `protobuf:"varint,4,opt,name=epoch,proto3" json:"epoch,omitempty"`
	// timestamp is the initial signature's server clock
	Timestamp int64 `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *DataFeedReport) Reset() {
	*x = DataFeedReport{}
	if protoimpl.UnsafeEnabled {
		mi := &file_datafeed_proto_datafeed_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataFeedReport) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataFeedReport) ProtoMessage() {}

func (x *DataFeedReport) ProtoReflect() protoreflect.Message {
	mi := &file_datafeed_proto_datafeed_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataFeedReport.ProtoReflect.Descriptor instead.
func (*DataFeedReport) Descriptor() ([]byte, []int) {
	return file_datafeed_proto_datafeed_proto_rawDescGZIP(), []int{0}
}

func (x *DataFeedReport) GetMarketHash() string {
	if x != nil {
		return x.MarketHash
	}
	return ""
}

func (x *DataFeedReport) GetOutcome() string {
	if x != nil {
		return x.Outcome
	}
	return ""
}

func (x *DataFeedReport) GetSignatures() string {
	if x != nil {
		return x.Signatures
	}
	return ""
}

func (x *DataFeedReport) GetEpoch() uint64 {
	if x != nil {
		return x.Epoch
	}
	return 0
}

func (x *DataFeedReport) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

var File_datafeed_proto_datafeed_proto protoreflect.FileDescriptor

var file_datafeed_proto_datafeed_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x64, 0x61, 0x74, 0x61, 0x66, 0x65, 0x65, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x64, 0x61, 0x74, 0x61, 0x66, 0x65, 0x65, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x02, 0x76, 0x31, 0x22, 0x9e, 0x01, 0x0a, 0x0e, 0x44, 0x61, 0x74, 0x61, 0x46, 0x65, 0x65, 0x64,
	0x52, 0x65, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x1e, 0x0a, 0x0a, 0x6d, 0x61, 0x72, 0x6b, 0x65, 0x74,
	0x48, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x6d, 0x61, 0x72, 0x6b,
	0x65, 0x74, 0x48, 0x61, 0x73, 0x68, 0x12, 0x18, 0x0a, 0x07, 0x6f, 0x75, 0x74, 0x63, 0x6f, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6f, 0x75, 0x74, 0x63, 0x6f, 0x6d, 0x65,
	0x12, 0x1e, 0x0a, 0x0a, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73,
	0x12, 0x14, 0x0a, 0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x05, 0x65, 0x70, 0x6f, 0x63, 0x68, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x42, 0x11, 0x5a, 0x0f, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x66, 0x65, 0x65,
	0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_datafeed_proto_datafeed_proto_rawDescOnce sync.Once
	file_datafeed_proto_datafeed_proto_rawDescData = file_datafeed_proto_datafeed_proto_rawDesc
)

func file_datafeed_proto_datafeed_proto_rawDescGZIP() []byte {
	file_datafeed_proto_datafeed_proto_rawDescOnce.Do(func() {
		file_datafeed_proto_datafeed_proto_rawDescData = protoimpl.X.CompressGZIP(file_datafeed_proto_datafeed_proto_rawDescData)
	})
	return file_datafeed_proto_datafeed_proto_rawDescData
}

var file_datafeed_proto_datafeed_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_datafeed_proto_datafeed_proto_goTypes = []interface{}{
	(*DataFeedReport)(nil), // 0: v1.DataFeedReport
}
var file_datafeed_proto_datafeed_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_datafeed_proto_datafeed_proto_init() }
func file_datafeed_proto_datafeed_proto_init() {
	if File_datafeed_proto_datafeed_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_datafeed_proto_datafeed_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataFeedReport); i {
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
			RawDescriptor: file_datafeed_proto_datafeed_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_datafeed_proto_datafeed_proto_goTypes,
		DependencyIndexes: file_datafeed_proto_datafeed_proto_depIdxs,
		MessageInfos:      file_datafeed_proto_datafeed_proto_msgTypes,
	}.Build()
	File_datafeed_proto_datafeed_proto = out.File
	file_datafeed_proto_datafeed_proto_rawDesc = nil
	file_datafeed_proto_datafeed_proto_goTypes = nil
	file_datafeed_proto_datafeed_proto_depIdxs = nil
}
