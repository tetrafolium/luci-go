// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/resultdb/proto/sink/v1/sink.proto

package sinkpb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

// A container of a message to a ResultSink server.
// The server accepts a sequence of these messages in JSON format.
type SinkMessageContainer struct {
	// Types that are valid to be assigned to Msg:
	//	*SinkMessageContainer_Handshake
	//	*SinkMessageContainer_TestResult
	//	*SinkMessageContainer_TestResultFile
	Msg                  isSinkMessageContainer_Msg `protobuf_oneof:"msg"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *SinkMessageContainer) Reset()         { *m = SinkMessageContainer{} }
func (m *SinkMessageContainer) String() string { return proto.CompactTextString(m) }
func (*SinkMessageContainer) ProtoMessage()    {}
func (*SinkMessageContainer) Descriptor() ([]byte, []int) {
	return fileDescriptor_67e05c474f1f0646, []int{0}
}

func (m *SinkMessageContainer) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SinkMessageContainer.Unmarshal(m, b)
}
func (m *SinkMessageContainer) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SinkMessageContainer.Marshal(b, m, deterministic)
}
func (m *SinkMessageContainer) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SinkMessageContainer.Merge(m, src)
}
func (m *SinkMessageContainer) XXX_Size() int {
	return xxx_messageInfo_SinkMessageContainer.Size(m)
}
func (m *SinkMessageContainer) XXX_DiscardUnknown() {
	xxx_messageInfo_SinkMessageContainer.DiscardUnknown(m)
}

var xxx_messageInfo_SinkMessageContainer proto.InternalMessageInfo

type isSinkMessageContainer_Msg interface {
	isSinkMessageContainer_Msg()
}

type SinkMessageContainer_Handshake struct {
	Handshake *Handshake `protobuf:"bytes,1,opt,name=handshake,proto3,oneof"`
}

type SinkMessageContainer_TestResult struct {
	TestResult *TestResult `protobuf:"bytes,2,opt,name=test_result,json=testResult,proto3,oneof"`
}

type SinkMessageContainer_TestResultFile struct {
	TestResultFile *TestResultFile `protobuf:"bytes,3,opt,name=test_result_file,json=testResultFile,proto3,oneof"`
}

func (*SinkMessageContainer_Handshake) isSinkMessageContainer_Msg() {}

func (*SinkMessageContainer_TestResult) isSinkMessageContainer_Msg() {}

func (*SinkMessageContainer_TestResultFile) isSinkMessageContainer_Msg() {}

func (m *SinkMessageContainer) GetMsg() isSinkMessageContainer_Msg {
	if m != nil {
		return m.Msg
	}
	return nil
}

func (m *SinkMessageContainer) GetHandshake() *Handshake {
	if x, ok := m.GetMsg().(*SinkMessageContainer_Handshake); ok {
		return x.Handshake
	}
	return nil
}

func (m *SinkMessageContainer) GetTestResult() *TestResult {
	if x, ok := m.GetMsg().(*SinkMessageContainer_TestResult); ok {
		return x.TestResult
	}
	return nil
}

func (m *SinkMessageContainer) GetTestResultFile() *TestResultFile {
	if x, ok := m.GetMsg().(*SinkMessageContainer_TestResultFile); ok {
		return x.TestResultFile
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*SinkMessageContainer) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*SinkMessageContainer_Handshake)(nil),
		(*SinkMessageContainer_TestResult)(nil),
		(*SinkMessageContainer_TestResultFile)(nil),
	}
}

// The very first message in a ResultSink TCP connection.
type Handshake struct {
	// The auth token is available to eligible subprocesses via
	// test_results.uploader.auth_token LUCI_CONTEXT value.
	// More about LUCI_CONTEXT: https://chromium.googlesource.com/infra/luci/luci-py/+/6b6dad7aef994b96d3bb5b6f13fae8168938560f/client/LUCI_CONTEXT.md
	// If the value is unexpected, the server will close the connection.
	AuthToken            string   `protobuf:"bytes,1,opt,name=auth_token,json=authToken,proto3" json:"auth_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Handshake) Reset()         { *m = Handshake{} }
func (m *Handshake) String() string { return proto.CompactTextString(m) }
func (*Handshake) ProtoMessage()    {}
func (*Handshake) Descriptor() ([]byte, []int) {
	return fileDescriptor_67e05c474f1f0646, []int{1}
}

func (m *Handshake) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Handshake.Unmarshal(m, b)
}
func (m *Handshake) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Handshake.Marshal(b, m, deterministic)
}
func (m *Handshake) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Handshake.Merge(m, src)
}
func (m *Handshake) XXX_Size() int {
	return xxx_messageInfo_Handshake.Size(m)
}
func (m *Handshake) XXX_DiscardUnknown() {
	xxx_messageInfo_Handshake.DiscardUnknown(m)
}

var xxx_messageInfo_Handshake proto.InternalMessageInfo

func (m *Handshake) GetAuthToken() string {
	if m != nil {
		return m.AuthToken
	}
	return ""
}

func init() {
	proto.RegisterType((*SinkMessageContainer)(nil), "luci.resultdb.sink.SinkMessageContainer")
	proto.RegisterType((*Handshake)(nil), "luci.resultdb.sink.Handshake")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/resultdb/proto/sink/v1/sink.proto", fileDescriptor_67e05c474f1f0646)
}

var fileDescriptor_67e05c474f1f0646 = []byte{
	// 264 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x90, 0x41, 0x4b, 0xc3, 0x30,
	0x14, 0xc7, 0x57, 0x87, 0x42, 0xdf, 0x40, 0x24, 0x78, 0x28, 0xc2, 0x44, 0x7a, 0x12, 0x0f, 0x09,
	0x4e, 0xf1, 0x22, 0x3b, 0x38, 0x41, 0x7a, 0xd1, 0x43, 0xdd, 0xc9, 0x4b, 0x49, 0xb7, 0x67, 0x1b,
	0xda, 0x26, 0x23, 0x79, 0xf5, 0x6b, 0xfb, 0x15, 0x24, 0x19, 0x9b, 0x13, 0x45, 0xd8, 0xe9, 0x91,
	0xdf, 0xfb, 0xe7, 0xf7, 0x5e, 0x02, 0x77, 0x95, 0xe1, 0x8b, 0xda, 0x9a, 0x4e, 0xf5, 0x1d, 0x37,
	0xb6, 0x12, 0x6d, 0xbf, 0x50, 0xc2, 0xa2, 0xeb, 0x5b, 0x5a, 0x96, 0x62, 0x65, 0x0d, 0x19, 0xe1,
	0x94, 0x6e, 0xc4, 0xc7, 0x75, 0xa8, 0x3c, 0x20, 0xc6, 0x7c, 0x8e, 0x6f, 0x72, 0xdc, 0x77, 0xce,
	0xa6, 0xfb, 0xb8, 0x08, 0x1d, 0x15, 0xeb, 0xde, 0x5a, 0x99, 0x7e, 0x46, 0x70, 0xfa, 0xaa, 0x74,
	0xf3, 0x8c, 0xce, 0xc9, 0x0a, 0x1f, 0x8d, 0x26, 0xa9, 0x34, 0x5a, 0x36, 0x85, 0xb8, 0x96, 0x7a,
	0xe9, 0x6a, 0xd9, 0x60, 0x12, 0x5d, 0x44, 0x97, 0xa3, 0xc9, 0x98, 0xff, 0x9e, 0xcf, 0xb3, 0x4d,
	0x28, 0x1b, 0xe4, 0xdf, 0x37, 0xd8, 0x03, 0x8c, 0x76, 0x86, 0x25, 0x07, 0x41, 0x70, 0xfe, 0x97,
	0x60, 0x8e, 0x8e, 0xf2, 0x40, 0xb2, 0x41, 0x0e, 0xb4, 0x3d, 0xb1, 0x17, 0x38, 0xd9, 0x51, 0x14,
	0xef, 0xaa, 0xc5, 0x64, 0x18, 0x3c, 0xe9, 0xff, 0x9e, 0x27, 0xd5, 0xfa, 0x6d, 0x8e, 0xe9, 0x07,
	0x99, 0x1d, 0xc2, 0xb0, 0x73, 0x55, 0x7a, 0x05, 0xf1, 0x76, 0x67, 0x36, 0x06, 0x90, 0x3d, 0xd5,
	0x05, 0x99, 0x06, 0x75, 0x78, 0x66, 0x9c, 0xc7, 0x9e, 0xcc, 0x3d, 0x98, 0xdd, 0xbe, 0x4d, 0xf6,
	0xf8, 0xde, 0x7b, 0x5f, 0x57, 0x65, 0x79, 0x14, 0xe8, 0xcd, 0x57, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x95, 0x81, 0x32, 0x4d, 0xe7, 0x01, 0x00, 0x00,
}