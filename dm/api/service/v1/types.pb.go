// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/tetrafolium/luci-go/dm/api/service/v1/types.proto

package dm

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type MultiPropertyValue struct {
	Values               []*PropertyValue `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *MultiPropertyValue) Reset()         { *m = MultiPropertyValue{} }
func (m *MultiPropertyValue) String() string { return proto.CompactTextString(m) }
func (*MultiPropertyValue) ProtoMessage()    {}
func (*MultiPropertyValue) Descriptor() ([]byte, []int) {
	return fileDescriptor_1def6b5c0f81b24d, []int{0}
}

func (m *MultiPropertyValue) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MultiPropertyValue.Unmarshal(m, b)
}
func (m *MultiPropertyValue) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MultiPropertyValue.Marshal(b, m, deterministic)
}
func (m *MultiPropertyValue) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MultiPropertyValue.Merge(m, src)
}
func (m *MultiPropertyValue) XXX_Size() int {
	return xxx_messageInfo_MultiPropertyValue.Size(m)
}
func (m *MultiPropertyValue) XXX_DiscardUnknown() {
	xxx_messageInfo_MultiPropertyValue.DiscardUnknown(m)
}

var xxx_messageInfo_MultiPropertyValue proto.InternalMessageInfo

func (m *MultiPropertyValue) GetValues() []*PropertyValue {
	if m != nil {
		return m.Values
	}
	return nil
}

type PropertyValue struct {
	// Types that are valid to be assigned to Value:
	//	*PropertyValue_Str
	//	*PropertyValue_Dat
	//	*PropertyValue_Num
	//	*PropertyValue_Bin
	//	*PropertyValue_Time
	//	*PropertyValue_Null
	Value                isPropertyValue_Value `protobuf_oneof:"value"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *PropertyValue) Reset()         { *m = PropertyValue{} }
func (m *PropertyValue) String() string { return proto.CompactTextString(m) }
func (*PropertyValue) ProtoMessage()    {}
func (*PropertyValue) Descriptor() ([]byte, []int) {
	return fileDescriptor_1def6b5c0f81b24d, []int{1}
}

func (m *PropertyValue) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PropertyValue.Unmarshal(m, b)
}
func (m *PropertyValue) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PropertyValue.Marshal(b, m, deterministic)
}
func (m *PropertyValue) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PropertyValue.Merge(m, src)
}
func (m *PropertyValue) XXX_Size() int {
	return xxx_messageInfo_PropertyValue.Size(m)
}
func (m *PropertyValue) XXX_DiscardUnknown() {
	xxx_messageInfo_PropertyValue.DiscardUnknown(m)
}

var xxx_messageInfo_PropertyValue proto.InternalMessageInfo

type isPropertyValue_Value interface {
	isPropertyValue_Value()
}

type PropertyValue_Str struct {
	Str string `protobuf:"bytes,1,opt,name=str,proto3,oneof"`
}

type PropertyValue_Dat struct {
	Dat []byte `protobuf:"bytes,2,opt,name=dat,proto3,oneof"`
}

type PropertyValue_Num struct {
	Num int64 `protobuf:"varint,3,opt,name=num,proto3,oneof"`
}

type PropertyValue_Bin struct {
	Bin bool `protobuf:"varint,5,opt,name=bin,proto3,oneof"`
}

type PropertyValue_Time struct {
	Time *timestamp.Timestamp `protobuf:"bytes,6,opt,name=time,proto3,oneof"`
}

type PropertyValue_Null struct {
	Null *empty.Empty `protobuf:"bytes,7,opt,name=null,proto3,oneof"`
}

func (*PropertyValue_Str) isPropertyValue_Value() {}

func (*PropertyValue_Dat) isPropertyValue_Value() {}

func (*PropertyValue_Num) isPropertyValue_Value() {}

func (*PropertyValue_Bin) isPropertyValue_Value() {}

func (*PropertyValue_Time) isPropertyValue_Value() {}

func (*PropertyValue_Null) isPropertyValue_Value() {}

func (m *PropertyValue) GetValue() isPropertyValue_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *PropertyValue) GetStr() string {
	if x, ok := m.GetValue().(*PropertyValue_Str); ok {
		return x.Str
	}
	return ""
}

func (m *PropertyValue) GetDat() []byte {
	if x, ok := m.GetValue().(*PropertyValue_Dat); ok {
		return x.Dat
	}
	return nil
}

func (m *PropertyValue) GetNum() int64 {
	if x, ok := m.GetValue().(*PropertyValue_Num); ok {
		return x.Num
	}
	return 0
}

func (m *PropertyValue) GetBin() bool {
	if x, ok := m.GetValue().(*PropertyValue_Bin); ok {
		return x.Bin
	}
	return false
}

func (m *PropertyValue) GetTime() *timestamp.Timestamp {
	if x, ok := m.GetValue().(*PropertyValue_Time); ok {
		return x.Time
	}
	return nil
}

func (m *PropertyValue) GetNull() *empty.Empty {
	if x, ok := m.GetValue().(*PropertyValue_Null); ok {
		return x.Null
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*PropertyValue) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*PropertyValue_Str)(nil),
		(*PropertyValue_Dat)(nil),
		(*PropertyValue_Num)(nil),
		(*PropertyValue_Bin)(nil),
		(*PropertyValue_Time)(nil),
		(*PropertyValue_Null)(nil),
	}
}

// AttemptList is logically a listing of unique attempts, which has a compact
// representation in the common scenario of listing multiple attempts of the
// same quest(s).
type AttemptList struct {
	// To is a map of quests-to-attempts to depend on. So if you want to depend
	// on the attempt "foo|1", "foo|2" and "bar|1", this would look like:
	//   {
	//     "foo": [1, 2],
	//     "bar": [1],
	//   }
	To                   map[string]*AttemptList_Nums `protobuf:"bytes,2,rep,name=to,proto3" json:"to,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *AttemptList) Reset()         { *m = AttemptList{} }
func (m *AttemptList) String() string { return proto.CompactTextString(m) }
func (*AttemptList) ProtoMessage()    {}
func (*AttemptList) Descriptor() ([]byte, []int) {
	return fileDescriptor_1def6b5c0f81b24d, []int{2}
}

func (m *AttemptList) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AttemptList.Unmarshal(m, b)
}
func (m *AttemptList) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AttemptList.Marshal(b, m, deterministic)
}
func (m *AttemptList) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AttemptList.Merge(m, src)
}
func (m *AttemptList) XXX_Size() int {
	return xxx_messageInfo_AttemptList.Size(m)
}
func (m *AttemptList) XXX_DiscardUnknown() {
	xxx_messageInfo_AttemptList.DiscardUnknown(m)
}

var xxx_messageInfo_AttemptList proto.InternalMessageInfo

func (m *AttemptList) GetTo() map[string]*AttemptList_Nums {
	if m != nil {
		return m.To
	}
	return nil
}

type AttemptList_Nums struct {
	Nums                 []uint32 `protobuf:"varint,1,rep,packed,name=nums,proto3" json:"nums,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AttemptList_Nums) Reset()         { *m = AttemptList_Nums{} }
func (m *AttemptList_Nums) String() string { return proto.CompactTextString(m) }
func (*AttemptList_Nums) ProtoMessage()    {}
func (*AttemptList_Nums) Descriptor() ([]byte, []int) {
	return fileDescriptor_1def6b5c0f81b24d, []int{2, 0}
}

func (m *AttemptList_Nums) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AttemptList_Nums.Unmarshal(m, b)
}
func (m *AttemptList_Nums) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AttemptList_Nums.Marshal(b, m, deterministic)
}
func (m *AttemptList_Nums) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AttemptList_Nums.Merge(m, src)
}
func (m *AttemptList_Nums) XXX_Size() int {
	return xxx_messageInfo_AttemptList_Nums.Size(m)
}
func (m *AttemptList_Nums) XXX_DiscardUnknown() {
	xxx_messageInfo_AttemptList_Nums.DiscardUnknown(m)
}

var xxx_messageInfo_AttemptList_Nums proto.InternalMessageInfo

func (m *AttemptList_Nums) GetNums() []uint32 {
	if m != nil {
		return m.Nums
	}
	return nil
}

func init() {
	proto.RegisterType((*MultiPropertyValue)(nil), "dm.MultiPropertyValue")
	proto.RegisterType((*PropertyValue)(nil), "dm.PropertyValue")
	proto.RegisterType((*AttemptList)(nil), "dm.AttemptList")
	proto.RegisterMapType((map[string]*AttemptList_Nums)(nil), "dm.AttemptList.ToEntry")
	proto.RegisterType((*AttemptList_Nums)(nil), "dm.AttemptList.Nums")
}

func init() {
	proto.RegisterFile("github.com/tetrafolium/luci-go/dm/api/service/v1/types.proto", fileDescriptor_1def6b5c0f81b24d)
}

var fileDescriptor_1def6b5c0f81b24d = []byte{
	// 380 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x90, 0xdf, 0x8a, 0xd4, 0x30,
	0x14, 0xc6, 0x37, 0x6d, 0x67, 0x66, 0x4d, 0x5d, 0x18, 0x83, 0xac, 0xa1, 0x82, 0x96, 0xb9, 0xb1,
	0x8a, 0x24, 0x5a, 0x6f, 0xc4, 0x1b, 0x71, 0x61, 0x61, 0xf1, 0x1f, 0x52, 0x16, 0xef, 0xdb, 0x69,
	0xac, 0xc1, 0xa6, 0x29, 0xe9, 0xc9, 0x40, 0x9f, 0xc7, 0xf7, 0xf1, 0x99, 0xe4, 0xa4, 0xb3, 0xe0,
	0xae, 0x77, 0xe7, 0x7c, 0xe7, 0x77, 0x72, 0xf2, 0x7d, 0xb4, 0xec, 0xac, 0xd8, 0xff, 0x74, 0xd6,
	0x68, 0x6f, 0x84, 0x75, 0x9d, 0xec, 0xfd, 0x5e, 0xcb, 0xd6, 0xc8, 0x7a, 0xd4, 0x72, 0x52, 0xee,
	0xa0, 0xf7, 0x4a, 0x1e, 0x5e, 0x4b, 0x98, 0x47, 0x35, 0x89, 0xd1, 0x59, 0xb0, 0x2c, 0x6a, 0x4d,
	0xf6, 0xb4, 0xb3, 0xb6, 0xeb, 0x95, 0x0c, 0x4a, 0xe3, 0x7f, 0x48, 0xd0, 0x46, 0x4d, 0x50, 0x9b,
	0x71, 0x81, 0xb2, 0xc7, 0x77, 0x01, 0x65, 0x46, 0x98, 0x97, 0xe1, 0xee, 0x3d, 0x65, 0x5f, 0x7c,
	0x0f, 0xfa, 0x9b, 0xb3, 0xa3, 0x72, 0x30, 0x7f, 0xaf, 0x7b, 0xaf, 0xd8, 0x73, 0xba, 0x3e, 0x60,
	0x31, 0x71, 0x92, 0xc7, 0x45, 0x5a, 0x3e, 0x10, 0xad, 0x11, 0xb7, 0x90, 0xea, 0x08, 0xec, 0xfe,
	0x10, 0x7a, 0x76, 0x7b, 0x99, 0xd1, 0x78, 0x02, 0xc7, 0x49, 0x4e, 0x8a, 0x7b, 0x57, 0x27, 0x15,
	0x36, 0xa8, 0xb5, 0x35, 0xf0, 0x28, 0x27, 0xc5, 0x7d, 0xd4, 0xda, 0x1a, 0x50, 0x1b, 0xbc, 0xe1,
	0x71, 0x4e, 0x8a, 0x18, 0xb5, 0xc1, 0x1b, 0xd4, 0x1a, 0x3d, 0xf0, 0x55, 0x4e, 0x8a, 0x53, 0xd4,
	0x1a, 0x3d, 0xb0, 0x57, 0x34, 0x41, 0x4b, 0x7c, 0x9d, 0x93, 0x22, 0x2d, 0x33, 0xb1, 0xd8, 0x11,
	0x37, 0x76, 0xc4, 0xf5, 0x8d, 0xdf, 0xab, 0x93, 0x2a, 0x90, 0xec, 0x25, 0x4d, 0x06, 0xdf, 0xf7,
	0x7c, 0x13, 0x36, 0xce, 0xff, 0xdb, 0xb8, 0xc4, 0x00, 0x90, 0x46, 0xea, 0x62, 0x43, 0x57, 0xc1,
	0xcb, 0xc7, 0xe4, 0x34, 0xd9, 0xae, 0x76, 0xbf, 0x09, 0x4d, 0x3f, 0x00, 0x60, 0x48, 0x9f, 0xf5,
	0x04, 0xec, 0x19, 0x8d, 0xc0, 0xf2, 0x28, 0xe4, 0xf0, 0x08, 0x73, 0xf8, 0x67, 0x28, 0xae, 0xed,
	0xe5, 0x00, 0x6e, 0xae, 0x22, 0xb0, 0xd9, 0x13, 0x9a, 0x7c, 0xf5, 0x66, 0x62, 0xe7, 0x78, 0xdd,
	0x2c, 0xd1, 0x9d, 0x5d, 0x44, 0x5b, 0x52, 0x85, 0x3e, 0xfb, 0x44, 0x37, 0x47, 0x9c, 0x6d, 0x69,
	0xfc, 0x4b, 0xcd, 0x4b, 0x44, 0x15, 0x96, 0xec, 0xc5, 0xf1, 0x13, 0x21, 0xa2, 0xb4, 0x7c, 0x78,
	0xf7, 0x10, 0xbe, 0x5c, 0x2d, 0xc8, 0xbb, 0xe8, 0x2d, 0x69, 0xd6, 0xc1, 0xcc, 0x9b, 0xbf, 0x01,
	0x00, 0x00, 0xff, 0xff, 0x4b, 0x1c, 0xe8, 0xca, 0x36, 0x02, 0x00, 0x00,
}
