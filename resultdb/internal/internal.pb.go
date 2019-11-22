// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/resultdb/internal/internal.proto

package internal

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	v1 "go.chromium.org/luci/resultdb/proto/rpc/v1"
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

type Artifacts struct {
	Artifacts            []*v1.Artifact `protobuf:"bytes,1,rep,name=artifacts,proto3" json:"artifacts,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Artifacts) Reset()         { *m = Artifacts{} }
func (m *Artifacts) String() string { return proto.CompactTextString(m) }
func (*Artifacts) ProtoMessage()    {}
func (*Artifacts) Descriptor() ([]byte, []int) {
	return fileDescriptor_1c7a6dcb33711786, []int{0}
}

func (m *Artifacts) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Artifacts.Unmarshal(m, b)
}
func (m *Artifacts) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Artifacts.Marshal(b, m, deterministic)
}
func (m *Artifacts) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Artifacts.Merge(m, src)
}
func (m *Artifacts) XXX_Size() int {
	return xxx_messageInfo_Artifacts.Size(m)
}
func (m *Artifacts) XXX_DiscardUnknown() {
	xxx_messageInfo_Artifacts.DiscardUnknown(m)
}

var xxx_messageInfo_Artifacts proto.InternalMessageInfo

func (m *Artifacts) GetArtifacts() []*v1.Artifact {
	if m != nil {
		return m.Artifacts
	}
	return nil
}

func init() {
	proto.RegisterType((*Artifacts)(nil), "luci.resultdb.internal.Artifacts")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/resultdb/internal/internal.proto", fileDescriptor_1c7a6dcb33711786)
}

var fileDescriptor_1c7a6dcb33711786 = []byte{
	// 163 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x32, 0x4d, 0xcf, 0xd7, 0x4b,
	0xce, 0x28, 0xca, 0xcf, 0xcd, 0x2c, 0xcd, 0xd5, 0xcb, 0x2f, 0x4a, 0xd7, 0xcf, 0x29, 0x4d, 0xce,
	0xd4, 0x2f, 0x4a, 0x2d, 0x2e, 0xcd, 0x29, 0x49, 0x49, 0xd2, 0xcf, 0xcc, 0x2b, 0x49, 0x2d, 0xca,
	0x4b, 0xcc, 0x81, 0x33, 0xf4, 0x0a, 0x8a, 0xf2, 0x4b, 0xf2, 0x85, 0xc4, 0x40, 0xca, 0xf4, 0x60,
	0xca, 0xf4, 0x60, 0xb2, 0x52, 0x36, 0xf8, 0x8d, 0x03, 0x6b, 0xd6, 0x2f, 0x2a, 0x48, 0xd6, 0x2f,
	0x33, 0xd4, 0x2f, 0x49, 0x2d, 0x2e, 0x89, 0x87, 0x48, 0x41, 0x4c, 0x55, 0xf2, 0xe4, 0xe2, 0x74,
	0x2c, 0x2a, 0xc9, 0x4c, 0x4b, 0x4c, 0x2e, 0x29, 0x16, 0xb2, 0xe1, 0xe2, 0x4c, 0x84, 0x71, 0x24,
	0x18, 0x15, 0x98, 0x35, 0xb8, 0x8d, 0xe4, 0xf4, 0x50, 0xad, 0x2d, 0x2a, 0x48, 0xd6, 0x2b, 0x33,
	0xd4, 0x83, 0xe9, 0x09, 0x42, 0x68, 0x70, 0x32, 0x8c, 0xd2, 0x27, 0xce, 0x67, 0xd6, 0x30, 0x46,
	0x12, 0x1b, 0xd8, 0x11, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0x48, 0x80, 0x2e, 0x4a, 0x13,
	0x01, 0x00, 0x00,
}