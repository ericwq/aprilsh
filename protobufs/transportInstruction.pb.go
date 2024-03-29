// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: protobufs/transportInstruction.proto

package protobufs

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

type Instruction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ProtocolVersion uint32 `protobuf:"varint,1,opt,name=protocol_version,json=protocolVersion,proto3" json:"protocol_version,omitempty"`
	OldNum          uint64 `protobuf:"varint,2,opt,name=old_num,json=oldNum,proto3" json:"old_num,omitempty"`
	NewNum          uint64 `protobuf:"varint,3,opt,name=new_num,json=newNum,proto3" json:"new_num,omitempty"`
	AckNum          uint64 `protobuf:"varint,4,opt,name=ack_num,json=ackNum,proto3" json:"ack_num,omitempty"`
	ThrowawayNum    uint64 `protobuf:"varint,5,opt,name=throwaway_num,json=throwawayNum,proto3" json:"throwaway_num,omitempty"`
	Diff            []byte `protobuf:"bytes,6,opt,name=diff,proto3,oneof" json:"diff,omitempty"`
	Chaff           []byte `protobuf:"bytes,7,opt,name=chaff,proto3,oneof" json:"chaff,omitempty"`
}

func (x *Instruction) Reset() {
	*x = Instruction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobufs_transportInstruction_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Instruction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Instruction) ProtoMessage() {}

func (x *Instruction) ProtoReflect() protoreflect.Message {
	mi := &file_protobufs_transportInstruction_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Instruction.ProtoReflect.Descriptor instead.
func (*Instruction) Descriptor() ([]byte, []int) {
	return file_protobufs_transportInstruction_proto_rawDescGZIP(), []int{0}
}

func (x *Instruction) GetProtocolVersion() uint32 {
	if x != nil {
		return x.ProtocolVersion
	}
	return 0
}

func (x *Instruction) GetOldNum() uint64 {
	if x != nil {
		return x.OldNum
	}
	return 0
}

func (x *Instruction) GetNewNum() uint64 {
	if x != nil {
		return x.NewNum
	}
	return 0
}

func (x *Instruction) GetAckNum() uint64 {
	if x != nil {
		return x.AckNum
	}
	return 0
}

func (x *Instruction) GetThrowawayNum() uint64 {
	if x != nil {
		return x.ThrowawayNum
	}
	return 0
}

func (x *Instruction) GetDiff() []byte {
	if x != nil {
		return x.Diff
	}
	return nil
}

func (x *Instruction) GetChaff() []byte {
	if x != nil {
		return x.Chaff
	}
	return nil
}

var File_protobufs_transportInstruction_proto protoreflect.FileDescriptor

var file_protobufs_transportInstruction_proto_rawDesc = []byte{
	0x0a, 0x24, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x73, 0x2f, 0x74, 0x72, 0x61, 0x6e,
	0x73, 0x70, 0x6f, 0x72, 0x74, 0x49, 0x6e, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x10, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72,
	0x74, 0x42, 0x75, 0x66, 0x66, 0x65, 0x72, 0x73, 0x22, 0xef, 0x01, 0x0a, 0x0b, 0x49, 0x6e, 0x73,
	0x74, 0x72, 0x75, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x29, 0x0a, 0x10, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x63, 0x6f, 0x6c, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x0f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x56, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x17, 0x0a, 0x07, 0x6f, 0x6c, 0x64, 0x5f, 0x6e, 0x75, 0x6d, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6f, 0x6c, 0x64, 0x4e, 0x75, 0x6d, 0x12, 0x17, 0x0a, 0x07,
	0x6e, 0x65, 0x77, 0x5f, 0x6e, 0x75, 0x6d, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6e,
	0x65, 0x77, 0x4e, 0x75, 0x6d, 0x12, 0x17, 0x0a, 0x07, 0x61, 0x63, 0x6b, 0x5f, 0x6e, 0x75, 0x6d,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x61, 0x63, 0x6b, 0x4e, 0x75, 0x6d, 0x12, 0x23,
	0x0a, 0x0d, 0x74, 0x68, 0x72, 0x6f, 0x77, 0x61, 0x77, 0x61, 0x79, 0x5f, 0x6e, 0x75, 0x6d, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0c, 0x74, 0x68, 0x72, 0x6f, 0x77, 0x61, 0x77, 0x61, 0x79,
	0x4e, 0x75, 0x6d, 0x12, 0x17, 0x0a, 0x04, 0x64, 0x69, 0x66, 0x66, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x0c, 0x48, 0x00, 0x52, 0x04, 0x64, 0x69, 0x66, 0x66, 0x88, 0x01, 0x01, 0x12, 0x19, 0x0a, 0x05,
	0x63, 0x68, 0x61, 0x66, 0x66, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x48, 0x01, 0x52, 0x05, 0x63,
	0x68, 0x61, 0x66, 0x66, 0x88, 0x01, 0x01, 0x42, 0x07, 0x0a, 0x05, 0x5f, 0x64, 0x69, 0x66, 0x66,
	0x42, 0x08, 0x0a, 0x06, 0x5f, 0x63, 0x68, 0x61, 0x66, 0x66, 0x42, 0x0c, 0x5a, 0x0a, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protobufs_transportInstruction_proto_rawDescOnce sync.Once
	file_protobufs_transportInstruction_proto_rawDescData = file_protobufs_transportInstruction_proto_rawDesc
)

func file_protobufs_transportInstruction_proto_rawDescGZIP() []byte {
	file_protobufs_transportInstruction_proto_rawDescOnce.Do(func() {
		file_protobufs_transportInstruction_proto_rawDescData = protoimpl.X.CompressGZIP(file_protobufs_transportInstruction_proto_rawDescData)
	})
	return file_protobufs_transportInstruction_proto_rawDescData
}

var file_protobufs_transportInstruction_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_protobufs_transportInstruction_proto_goTypes = []interface{}{
	(*Instruction)(nil), // 0: TransportBuffers.Instruction
}
var file_protobufs_transportInstruction_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_protobufs_transportInstruction_proto_init() }
func file_protobufs_transportInstruction_proto_init() {
	if File_protobufs_transportInstruction_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protobufs_transportInstruction_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Instruction); i {
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
	file_protobufs_transportInstruction_proto_msgTypes[0].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_protobufs_transportInstruction_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protobufs_transportInstruction_proto_goTypes,
		DependencyIndexes: file_protobufs_transportInstruction_proto_depIdxs,
		MessageInfos:      file_protobufs_transportInstruction_proto_msgTypes,
	}.Build()
	File_protobufs_transportInstruction_proto = out.File
	file_protobufs_transportInstruction_proto_rawDesc = nil
	file_protobufs_transportInstruction_proto_goTypes = nil
	file_protobufs_transportInstruction_proto_depIdxs = nil
}
