// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.17.3
// source: block/block.proto

package block

import (
	transaction "github.com/filefilego/filefilego/internal/transaction"
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

// ProtoBlock is the proto representation of a block.
type ProtoBlock struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// hash represents the block hash.
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	// signature of the block.
	Signature []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	// timestamp represents the block time.
	Timestamp int64 `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// data includes arbitrary data from the sealer.
	Data []byte `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
	// previous_block_hash is the hash of the previous block.
	PreviousBlockHash []byte `protobuf:"bytes,5,opt,name=previous_block_hash,json=previousBlockHash,proto3" json:"previous_block_hash,omitempty"`
	// transactions contain a list of transactions in the block.
	Transactions []*transaction.ProtoTransaction `protobuf:"bytes,6,rep,name=transactions,proto3" json:"transactions,omitempty"`
	// number represents the block number.
	Number uint64 `protobuf:"varint,7,opt,name=number,proto3" json:"number,omitempty"`
}

func (x *ProtoBlock) Reset() {
	*x = ProtoBlock{}
	if protoimpl.UnsafeEnabled {
		mi := &file_block_block_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProtoBlock) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProtoBlock) ProtoMessage() {}

func (x *ProtoBlock) ProtoReflect() protoreflect.Message {
	mi := &file_block_block_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProtoBlock.ProtoReflect.Descriptor instead.
func (*ProtoBlock) Descriptor() ([]byte, []int) {
	return file_block_block_proto_rawDescGZIP(), []int{0}
}

func (x *ProtoBlock) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

func (x *ProtoBlock) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

func (x *ProtoBlock) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *ProtoBlock) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *ProtoBlock) GetPreviousBlockHash() []byte {
	if x != nil {
		return x.PreviousBlockHash
	}
	return nil
}

func (x *ProtoBlock) GetTransactions() []*transaction.ProtoTransaction {
	if x != nil {
		return x.Transactions
	}
	return nil
}

func (x *ProtoBlock) GetNumber() uint64 {
	if x != nil {
		return x.Number
	}
	return 0
}

var File_block_block_proto protoreflect.FileDescriptor

var file_block_block_proto_rawDesc = []byte{
	0x0a, 0x11, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x2f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x05, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x1a, 0x1d, 0x74, 0x72, 0x61, 0x6e,
	0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xfb, 0x01, 0x0a, 0x0a, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x1c, 0x0a, 0x09,
	0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x2e, 0x0a, 0x13,
	0x70, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x68,
	0x61, 0x73, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x11, 0x70, 0x72, 0x65, 0x76, 0x69,
	0x6f, 0x75, 0x73, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x48, 0x61, 0x73, 0x68, 0x12, 0x41, 0x0a, 0x0c,
	0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x06, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x0c, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12,
	0x16, 0x0a, 0x06, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x07, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x06, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x66, 0x69, 0x6c, 0x65, 0x67, 0x6f,
	0x2f, 0x66, 0x69, 0x6c, 0x65, 0x66, 0x69, 0x6c, 0x65, 0x67, 0x6f, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_block_block_proto_rawDescOnce sync.Once
	file_block_block_proto_rawDescData = file_block_block_proto_rawDesc
)

func file_block_block_proto_rawDescGZIP() []byte {
	file_block_block_proto_rawDescOnce.Do(func() {
		file_block_block_proto_rawDescData = protoimpl.X.CompressGZIP(file_block_block_proto_rawDescData)
	})
	return file_block_block_proto_rawDescData
}

var file_block_block_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_block_block_proto_goTypes = []interface{}{
	(*ProtoBlock)(nil),                   // 0: block.ProtoBlock
	(*transaction.ProtoTransaction)(nil), // 1: transaction.ProtoTransaction
}
var file_block_block_proto_depIdxs = []int32{
	1, // 0: block.ProtoBlock.transactions:type_name -> transaction.ProtoTransaction
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_block_block_proto_init() }
func file_block_block_proto_init() {
	if File_block_block_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_block_block_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProtoBlock); i {
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
			RawDescriptor: file_block_block_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_block_block_proto_goTypes,
		DependencyIndexes: file_block_block_proto_depIdxs,
		MessageInfos:      file_block_block_proto_msgTypes,
	}.Build()
	File_block_block_proto = out.File
	file_block_block_proto_rawDesc = nil
	file_block_block_proto_goTypes = nil
	file_block_block_proto_depIdxs = nil
}
