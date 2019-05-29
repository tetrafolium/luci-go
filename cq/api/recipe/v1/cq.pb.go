// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/cq/api/recipe/v1/cq.proto

package recipe

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	proto1 "go.chromium.org/luci/buildbucket/proto"
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

// Input provides CQ metadata for CQ-triggered tryjob.
type Input struct {
	// If active is false, CQ isn't active for the current build.
	Active bool `protobuf:"varint,1,opt,name=active,proto3" json:"active,omitempty"`
	// If false, CQ would try to submit CL(s) if all other checks pass.
	// If true, CQ won't try to submit.
	DryRun bool `protobuf:"varint,2,opt,name=dry_run,json=dryRun,proto3" json:"dry_run,omitempty"`
	// If true, CQ will not take this build into account while deciding whether CL
	// is good or not. See also `experiment_percentage` of CQ's config file.
	Experimental bool `protobuf:"varint,3,opt,name=experimental,proto3" json:"experimental,omitempty"`
	// If true, CQ triggered this build directly, otherwise typically indicates
	// a child build triggered by a CQ triggered one (possibly indirectly).
	//
	// Can be spoofed. *DO NOT USE FOR SECURITY CHECKS.*
	//
	// One possible use is to distinguish which builds must be cancelled manually,
	// and which (top_level=True) CQ would cancel itself.
	TopLevel bool `protobuf:"varint,4,opt,name=top_level,json=topLevel,proto3" json:"top_level,omitempty"`
	// List of CLs constituting CQ attempt for which this build was triggered.
	//
	// The CLs are ordered s.t. applying them in this order minimizes number of CLs
	// that will be applied before their dependencies *for the same repository*.
	//
	// For example, with 5 CLs spanning 2 projects like this:
	//         A2 -> A1
	//         ^     |       ("X -> Y" denotes X depends on Y)
	//         |     v
	//   B3 -> B2 -> B1
	//
	// [A1, A2, B1, B2, B3] and [B1, A1, A2, B2, B3] are among many possible
	// orders.
	//
	// In case of loops within the same repo, (e.g., A1 <-> A2), the loop is broken
	// off arbitrarily but deterministically, meaning both [A1, A2] and [A2, A1]
	// orders are valid though CQ would choose the same one for each build within
	// the same attempt.
	Cls                  []*CL    `protobuf:"bytes,5,rep,name=cls,proto3" json:"cls,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Input) Reset()         { *m = Input{} }
func (m *Input) String() string { return proto.CompactTextString(m) }
func (*Input) ProtoMessage()    {}
func (*Input) Descriptor() ([]byte, []int) {
	return fileDescriptor_5310ea7d0dc1a356, []int{0}
}

func (m *Input) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Input.Unmarshal(m, b)
}
func (m *Input) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Input.Marshal(b, m, deterministic)
}
func (m *Input) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Input.Merge(m, src)
}
func (m *Input) XXX_Size() int {
	return xxx_messageInfo_Input.Size(m)
}
func (m *Input) XXX_DiscardUnknown() {
	xxx_messageInfo_Input.DiscardUnknown(m)
}

var xxx_messageInfo_Input proto.InternalMessageInfo

func (m *Input) GetActive() bool {
	if m != nil {
		return m.Active
	}
	return false
}

func (m *Input) GetDryRun() bool {
	if m != nil {
		return m.DryRun
	}
	return false
}

func (m *Input) GetExperimental() bool {
	if m != nil {
		return m.Experimental
	}
	return false
}

func (m *Input) GetTopLevel() bool {
	if m != nil {
		return m.TopLevel
	}
	return false
}

func (m *Input) GetCls() []*CL {
	if m != nil {
		return m.Cls
	}
	return nil
}

type CL struct {
	// Source of this CL. Currently, only Gerrit is supported.
	//
	// Types that are valid to be assigned to Source:
	//	*CL_Gerrit
	Source isCL_Source `protobuf_oneof:"source"`
	// List of CLs on which this one depends. Each integer here is an index to the
	// dependency among `Input.cls`.
	Deps                 []int32  `protobuf:"varint,11,rep,packed,name=deps,proto3" json:"deps,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CL) Reset()         { *m = CL{} }
func (m *CL) String() string { return proto.CompactTextString(m) }
func (*CL) ProtoMessage()    {}
func (*CL) Descriptor() ([]byte, []int) {
	return fileDescriptor_5310ea7d0dc1a356, []int{1}
}

func (m *CL) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CL.Unmarshal(m, b)
}
func (m *CL) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CL.Marshal(b, m, deterministic)
}
func (m *CL) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CL.Merge(m, src)
}
func (m *CL) XXX_Size() int {
	return xxx_messageInfo_CL.Size(m)
}
func (m *CL) XXX_DiscardUnknown() {
	xxx_messageInfo_CL.DiscardUnknown(m)
}

var xxx_messageInfo_CL proto.InternalMessageInfo

type isCL_Source interface {
	isCL_Source()
}

type CL_Gerrit struct {
	Gerrit *proto1.GerritChange `protobuf:"bytes,1,opt,name=gerrit,proto3,oneof"`
}

func (*CL_Gerrit) isCL_Source() {}

func (m *CL) GetSource() isCL_Source {
	if m != nil {
		return m.Source
	}
	return nil
}

func (m *CL) GetGerrit() *proto1.GerritChange {
	if x, ok := m.GetSource().(*CL_Gerrit); ok {
		return x.Gerrit
	}
	return nil
}

func (m *CL) GetDeps() []int32 {
	if m != nil {
		return m.Deps
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*CL) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*CL_Gerrit)(nil),
	}
}

func init() {
	proto.RegisterType((*Input)(nil), "cq.recipe.Input")
	proto.RegisterType((*CL)(nil), "cq.recipe.CL")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/cq/api/recipe/v1/cq.proto", fileDescriptor_5310ea7d0dc1a356)
}

var fileDescriptor_5310ea7d0dc1a356 = []byte{
	// 292 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x90, 0x31, 0x4f, 0xc3, 0x30,
	0x10, 0x85, 0x69, 0xd3, 0x86, 0xd6, 0x85, 0xc5, 0x03, 0x58, 0x80, 0x44, 0xd5, 0xa9, 0x03, 0xb2,
	0x45, 0x2b, 0xb1, 0xb0, 0x35, 0x03, 0x20, 0x65, 0xca, 0xd8, 0xa5, 0x4a, 0x9c, 0x53, 0x6a, 0xe1,
	0xd8, 0x8e, 0x63, 0x47, 0xf4, 0xaf, 0xf0, 0x6b, 0x51, 0x9d, 0x0a, 0x81, 0xc4, 0xc0, 0x76, 0x77,
	0xdf, 0x7b, 0xd2, 0xbd, 0x87, 0x68, 0xa5, 0x29, 0xdf, 0x5b, 0x5d, 0x0b, 0x5f, 0x53, 0x6d, 0x2b,
	0x26, 0x3d, 0x17, 0x8c, 0x37, 0x2c, 0x37, 0x82, 0x59, 0xe0, 0xc2, 0x00, 0xeb, 0x1e, 0x19, 0x6f,
	0xa8, 0xb1, 0xda, 0x69, 0x3c, 0xe5, 0x0d, 0xed, 0xcf, 0x37, 0xeb, 0x3f, 0xad, 0x85, 0x17, 0xb2,
	0x2c, 0x3c, 0x7f, 0x07, 0xc7, 0x82, 0x85, 0x71, 0x5d, 0xd7, 0x5a, 0xf5, 0xfe, 0xc5, 0xe7, 0x00,
	0x8d, 0xdf, 0x94, 0xf1, 0x0e, 0x5f, 0xa1, 0x38, 0xe7, 0x4e, 0x74, 0x40, 0x06, 0xf3, 0xc1, 0x72,
	0x92, 0x9d, 0x36, 0x7c, 0x8d, 0xce, 0x4b, 0x7b, 0xd8, 0x59, 0xaf, 0xc8, 0xb0, 0x07, 0xa5, 0x3d,
	0x64, 0x5e, 0xe1, 0x05, 0xba, 0x80, 0x0f, 0x03, 0x56, 0xd4, 0xa0, 0x5c, 0x2e, 0x49, 0x14, 0xe8,
	0xaf, 0x1b, 0xbe, 0x45, 0x53, 0xa7, 0xcd, 0x4e, 0x42, 0x07, 0x92, 0x8c, 0x82, 0x60, 0xe2, 0xb4,
	0x49, 0x8f, 0x3b, 0xbe, 0x47, 0x11, 0x97, 0x2d, 0x19, 0xcf, 0xa3, 0xe5, 0x6c, 0x75, 0x49, 0xbf,
	0x93, 0xd0, 0x24, 0xcd, 0x8e, 0x64, 0xb1, 0x45, 0xc3, 0x24, 0xc5, 0x4f, 0x28, 0xae, 0xc0, 0x5a,
	0xe1, 0xc2, 0x63, 0xb3, 0xd5, 0x1d, 0xfd, 0x91, 0x89, 0x76, 0x2b, 0xfa, 0x12, 0x68, 0xb2, 0xcf,
	0x55, 0x05, 0xaf, 0x67, 0xd9, 0x49, 0x8d, 0x31, 0x1a, 0x95, 0x60, 0x5a, 0x32, 0x9b, 0x47, 0xcb,
	0x71, 0x16, 0xe6, 0xcd, 0x04, 0xc5, 0xad, 0xf6, 0x96, 0xc3, 0x86, 0x6e, 0x1f, 0xfe, 0x55, 0xf5,
	0x73, 0x3f, 0x15, 0x71, 0xe8, 0x6b, 0xfd, 0x15, 0x00, 0x00, 0xff, 0xff, 0x7a, 0xeb, 0xc4, 0xdd,
	0xa1, 0x01, 0x00, 0x00,
}
