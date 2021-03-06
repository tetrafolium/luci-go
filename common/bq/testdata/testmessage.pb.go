// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/common/bq/testdata/testmessage.proto

package testdata

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	duration "github.com/golang/protobuf/ptypes/duration"
	empty "github.com/golang/protobuf/ptypes/empty"
	_struct "github.com/golang/protobuf/ptypes/struct"
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

type TestMessage_FOO int32

const (
	TestMessage_X TestMessage_FOO = 0
	TestMessage_Y TestMessage_FOO = 1
	TestMessage_Z TestMessage_FOO = 2
)

var TestMessage_FOO_name = map[int32]string{
	0: "X",
	1: "Y",
	2: "Z",
}

var TestMessage_FOO_value = map[string]int32{
	"X": 0,
	"Y": 1,
	"Z": 2,
}

func (x TestMessage_FOO) String() string {
	return proto.EnumName(TestMessage_FOO_name, int32(x))
}

func (TestMessage_FOO) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_bc9d446aa0b4e493, []int{0, 0}
}

type TestMessage struct {
	Name           string               `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Timestamp      *timestamp.Timestamp `protobuf:"bytes,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Nested         *NestedTestMessage   `protobuf:"bytes,3,opt,name=nested,proto3" json:"nested,omitempty"`
	RepeatedNested []*NestedTestMessage `protobuf:"bytes,4,rep,name=repeated_nested,json=repeatedNested,proto3" json:"repeated_nested,omitempty"`
	Struct         *_struct.Struct      `protobuf:"bytes,5,opt,name=struct,proto3" json:"struct,omitempty"`
	Foo            TestMessage_FOO      `protobuf:"varint,6,opt,name=foo,proto3,enum=testdata.TestMessage_FOO" json:"foo,omitempty"`
	FooRepeated    []TestMessage_FOO    `protobuf:"varint,7,rep,packed,name=foo_repeated,json=fooRepeated,proto3,enum=testdata.TestMessage_FOO" json:"foo_repeated,omitempty"`
	Empty          *empty.Empty         `protobuf:"bytes,8,opt,name=empty,proto3" json:"empty,omitempty"`
	Empties        []*empty.Empty       `protobuf:"bytes,9,rep,name=empties,proto3" json:"empties,omitempty"`
	Duration       *duration.Duration   `protobuf:"bytes,10,opt,name=duration,proto3" json:"duration,omitempty"`
	// Types that are valid to be assigned to OneOf:
	//	*TestMessage_First
	//	*TestMessage_Second
	OneOf                isTestMessage_OneOf `protobuf_oneof:"one_of"`
	StringMap            map[string]string   `protobuf:"bytes,13,rep,name=string_map,json=stringMap,proto3" json:"string_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *TestMessage) Reset()         { *m = TestMessage{} }
func (m *TestMessage) String() string { return proto.CompactTextString(m) }
func (*TestMessage) ProtoMessage()    {}
func (*TestMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_bc9d446aa0b4e493, []int{0}
}

func (m *TestMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TestMessage.Unmarshal(m, b)
}
func (m *TestMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TestMessage.Marshal(b, m, deterministic)
}
func (m *TestMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TestMessage.Merge(m, src)
}
func (m *TestMessage) XXX_Size() int {
	return xxx_messageInfo_TestMessage.Size(m)
}
func (m *TestMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_TestMessage.DiscardUnknown(m)
}

var xxx_messageInfo_TestMessage proto.InternalMessageInfo

func (m *TestMessage) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *TestMessage) GetTimestamp() *timestamp.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return nil
}

func (m *TestMessage) GetNested() *NestedTestMessage {
	if m != nil {
		return m.Nested
	}
	return nil
}

func (m *TestMessage) GetRepeatedNested() []*NestedTestMessage {
	if m != nil {
		return m.RepeatedNested
	}
	return nil
}

func (m *TestMessage) GetStruct() *_struct.Struct {
	if m != nil {
		return m.Struct
	}
	return nil
}

func (m *TestMessage) GetFoo() TestMessage_FOO {
	if m != nil {
		return m.Foo
	}
	return TestMessage_X
}

func (m *TestMessage) GetFooRepeated() []TestMessage_FOO {
	if m != nil {
		return m.FooRepeated
	}
	return nil
}

func (m *TestMessage) GetEmpty() *empty.Empty {
	if m != nil {
		return m.Empty
	}
	return nil
}

func (m *TestMessage) GetEmpties() []*empty.Empty {
	if m != nil {
		return m.Empties
	}
	return nil
}

func (m *TestMessage) GetDuration() *duration.Duration {
	if m != nil {
		return m.Duration
	}
	return nil
}

type isTestMessage_OneOf interface {
	isTestMessage_OneOf()
}

type TestMessage_First struct {
	First *NestedTestMessage `protobuf:"bytes,11,opt,name=first,proto3,oneof"`
}

type TestMessage_Second struct {
	Second *NestedTestMessage `protobuf:"bytes,12,opt,name=second,proto3,oneof"`
}

func (*TestMessage_First) isTestMessage_OneOf() {}

func (*TestMessage_Second) isTestMessage_OneOf() {}

func (m *TestMessage) GetOneOf() isTestMessage_OneOf {
	if m != nil {
		return m.OneOf
	}
	return nil
}

func (m *TestMessage) GetFirst() *NestedTestMessage {
	if x, ok := m.GetOneOf().(*TestMessage_First); ok {
		return x.First
	}
	return nil
}

func (m *TestMessage) GetSecond() *NestedTestMessage {
	if x, ok := m.GetOneOf().(*TestMessage_Second); ok {
		return x.Second
	}
	return nil
}

func (m *TestMessage) GetStringMap() map[string]string {
	if m != nil {
		return m.StringMap
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*TestMessage) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*TestMessage_First)(nil),
		(*TestMessage_Second)(nil),
	}
}

type NestedTestMessage struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NestedTestMessage) Reset()         { *m = NestedTestMessage{} }
func (m *NestedTestMessage) String() string { return proto.CompactTextString(m) }
func (*NestedTestMessage) ProtoMessage()    {}
func (*NestedTestMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_bc9d446aa0b4e493, []int{1}
}

func (m *NestedTestMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NestedTestMessage.Unmarshal(m, b)
}
func (m *NestedTestMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NestedTestMessage.Marshal(b, m, deterministic)
}
func (m *NestedTestMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NestedTestMessage.Merge(m, src)
}
func (m *NestedTestMessage) XXX_Size() int {
	return xxx_messageInfo_NestedTestMessage.Size(m)
}
func (m *NestedTestMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_NestedTestMessage.DiscardUnknown(m)
}

var xxx_messageInfo_NestedTestMessage proto.InternalMessageInfo

func (m *NestedTestMessage) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func init() {
	proto.RegisterEnum("testdata.TestMessage_FOO", TestMessage_FOO_name, TestMessage_FOO_value)
	proto.RegisterType((*TestMessage)(nil), "testdata.TestMessage")
	proto.RegisterMapType((map[string]string)(nil), "testdata.TestMessage.StringMapEntry")
	proto.RegisterType((*NestedTestMessage)(nil), "testdata.NestedTestMessage")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/common/bq/testdata/testmessage.proto", fileDescriptor_bc9d446aa0b4e493)
}

var fileDescriptor_bc9d446aa0b4e493 = []byte{
	// 494 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x93, 0x51, 0x6f, 0xd3, 0x30,
	0x10, 0xc7, 0x97, 0x66, 0xcd, 0xda, 0xeb, 0x28, 0xc5, 0x42, 0xe0, 0x65, 0x08, 0xa2, 0x0a, 0x89,
	0x4a, 0xa0, 0x04, 0xad, 0x9a, 0x34, 0xd0, 0x9e, 0x60, 0x9b, 0x78, 0x19, 0x95, 0xbc, 0x3d, 0x00,
	0x2f, 0x95, 0x9b, 0x38, 0x21, 0xa2, 0xce, 0x85, 0xd8, 0x41, 0xea, 0x97, 0xe1, 0xb3, 0xa2, 0x38,
	0xc9, 0x36, 0xda, 0xc1, 0xf6, 0x94, 0xb3, 0xff, 0xbf, 0xbf, 0xef, 0xce, 0xbe, 0xc0, 0xbb, 0x04,
	0xfd, 0xf0, 0x7b, 0x81, 0x32, 0x2d, 0xa5, 0x8f, 0x45, 0x12, 0x2c, 0xcb, 0x30, 0x0d, 0x42, 0x94,
	0x12, 0xb3, 0x60, 0xf1, 0x33, 0xd0, 0x42, 0xe9, 0x88, 0x6b, 0x6e, 0x02, 0x29, 0x94, 0xe2, 0x89,
	0xf0, 0xf3, 0x02, 0x35, 0x92, 0x5e, 0xab, 0xb9, 0xcf, 0x13, 0xc4, 0x64, 0x29, 0x02, 0xb3, 0xbf,
	0x28, 0xe3, 0x20, 0x2a, 0x0b, 0xae, 0x53, 0xcc, 0x6a, 0xd2, 0xdd, 0x5f, 0xd7, 0x85, 0xcc, 0xf5,
	0xaa, 0x11, 0x5f, 0xac, 0x8b, 0x3a, 0x95, 0x42, 0x69, 0x2e, 0xf3, 0x06, 0x78, 0xb6, 0x0e, 0x28,
	0x5d, 0x94, 0xa1, 0xae, 0xd5, 0xf1, 0x6f, 0x07, 0x06, 0x97, 0x42, 0xe9, 0xf3, 0xba, 0x36, 0x42,
	0x60, 0x3b, 0xe3, 0x52, 0x50, 0xcb, 0xb3, 0x26, 0x7d, 0x66, 0x62, 0x72, 0x04, 0xfd, 0xab, 0x43,
	0x69, 0xc7, 0xb3, 0x26, 0x83, 0x03, 0xd7, 0xaf, 0x4f, 0xf5, 0xdb, 0x53, 0xfd, 0xcb, 0x96, 0x60,
	0xd7, 0x30, 0x99, 0x82, 0x93, 0x09, 0xa5, 0x45, 0x44, 0x6d, 0x63, 0xdb, 0xf7, 0xdb, 0xa6, 0xfd,
	0xcf, 0x66, 0xff, 0x46, 0x6a, 0xd6, 0xa0, 0xe4, 0x04, 0x1e, 0x16, 0x22, 0x17, 0x5c, 0x8b, 0x68,
	0xde, 0xb8, 0xb7, 0x3d, 0xfb, 0x2e, 0xf7, 0xb0, 0xf5, 0xd4, 0x12, 0x09, 0xc0, 0xa9, 0x1b, 0xa5,
	0x5d, 0x93, 0xfa, 0xe9, 0x46, 0xc5, 0x17, 0x46, 0x66, 0x0d, 0x46, 0x5e, 0x83, 0x1d, 0x23, 0x52,
	0xc7, 0xb3, 0x26, 0xc3, 0x83, 0xbd, 0xeb, 0x54, 0x37, 0x92, 0xf8, 0x67, 0xb3, 0x19, 0xab, 0x28,
	0x72, 0x0c, 0xbb, 0x31, 0xe2, 0xbc, 0xcd, 0x49, 0x77, 0x3c, 0xfb, 0xff, 0xae, 0x41, 0x8c, 0xc8,
	0x1a, 0x9a, 0xbc, 0x81, 0xae, 0x79, 0x42, 0xda, 0x33, 0xa5, 0x3d, 0xd9, 0x28, 0xed, 0xb4, 0x52,
	0x59, 0x0d, 0x91, 0xb7, 0xb0, 0x53, 0x05, 0xa9, 0x50, 0xb4, 0x6f, 0xee, 0xe1, 0x5f, 0x7c, 0x8b,
	0x91, 0x43, 0xe8, 0xb5, 0x23, 0x44, 0xc1, 0xa4, 0xd8, 0xdb, 0xb0, 0x9c, 0x34, 0x00, 0xbb, 0x42,
	0xc9, 0x14, 0xba, 0x71, 0x5a, 0x28, 0x4d, 0x07, 0x77, 0x3e, 0xd6, 0xa7, 0x2d, 0x56, 0xb3, 0xe4,
	0x10, 0x1c, 0x25, 0x42, 0xcc, 0x22, 0xba, 0x7b, 0x1f, 0x57, 0x03, 0x93, 0x8f, 0x00, 0x4a, 0x17,
	0x69, 0x96, 0xcc, 0x25, 0xcf, 0xe9, 0x03, 0xd3, 0xd7, 0xcb, 0xdb, 0xaf, 0xef, 0xc2, 0x70, 0xe7,
	0x3c, 0x3f, 0xcd, 0x74, 0xb1, 0x62, 0x7d, 0xd5, 0xae, 0xdd, 0x63, 0x18, 0xfe, 0x2d, 0x92, 0x11,
	0xd8, 0x3f, 0xc4, 0xaa, 0x99, 0xde, 0x2a, 0x24, 0x8f, 0xa1, 0xfb, 0x8b, 0x2f, 0x4b, 0x61, 0x06,
	0xb7, 0xcf, 0xea, 0xc5, 0xfb, 0xce, 0x91, 0x35, 0x76, 0xc1, 0x3e, 0x9b, 0xcd, 0x48, 0x17, 0xac,
	0x2f, 0xa3, 0xad, 0xea, 0xf3, 0x75, 0x64, 0x55, 0x9f, 0x6f, 0xa3, 0xce, 0x87, 0x1e, 0x38, 0x98,
	0x89, 0x39, 0xc6, 0xe3, 0x57, 0xf0, 0x68, 0xa3, 0x8f, 0xdb, 0xfe, 0x92, 0x85, 0x63, 0xae, 0x76,
	0xfa, 0x27, 0x00, 0x00, 0xff, 0xff, 0x0c, 0xc3, 0xc1, 0x41, 0x13, 0x04, 0x00, 0x00,
}
