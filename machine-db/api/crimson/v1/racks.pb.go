// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/tetrafolium/luci-go/machine-db/api/crimson/v1/racks.proto

package crimson

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	v1 "github.com/tetrafolium/luci-go/machine-db/api/common/v1"
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

// A rack in the database.
type Rack struct {
	// The name of this rack. Uniquely identifies this rack.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// A description of this rack.
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	// The datacenter this rack belongs to.
	Datacenter string `protobuf:"bytes,3,opt,name=datacenter,proto3" json:"datacenter,omitempty"`
	// The state of this rack.
	State v1.State `protobuf:"varint,4,opt,name=state,proto3,enum=common.State" json:"state,omitempty"`
	// The KVM serving this rack.
	Kvm                  string   `protobuf:"bytes,5,opt,name=kvm,proto3" json:"kvm,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Rack) Reset()         { *m = Rack{} }
func (m *Rack) String() string { return proto.CompactTextString(m) }
func (*Rack) ProtoMessage()    {}
func (*Rack) Descriptor() ([]byte, []int) {
	return fileDescriptor_5e7bc6de9b3dbb6d, []int{0}
}

func (m *Rack) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Rack.Unmarshal(m, b)
}
func (m *Rack) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Rack.Marshal(b, m, deterministic)
}
func (m *Rack) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Rack.Merge(m, src)
}
func (m *Rack) XXX_Size() int {
	return xxx_messageInfo_Rack.Size(m)
}
func (m *Rack) XXX_DiscardUnknown() {
	xxx_messageInfo_Rack.DiscardUnknown(m)
}

var xxx_messageInfo_Rack proto.InternalMessageInfo

func (m *Rack) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Rack) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *Rack) GetDatacenter() string {
	if m != nil {
		return m.Datacenter
	}
	return ""
}

func (m *Rack) GetState() v1.State {
	if m != nil {
		return m.State
	}
	return v1.State_STATE_UNSPECIFIED
}

func (m *Rack) GetKvm() string {
	if m != nil {
		return m.Kvm
	}
	return ""
}

// A request to list racks in the database.
type ListRacksRequest struct {
	// The names of racks to retrieve.
	Names []string `protobuf:"bytes,1,rep,name=names,proto3" json:"names,omitempty"`
	// The datacenters to filter retrieved racks on.
	Datacenters []string `protobuf:"bytes,2,rep,name=datacenters,proto3" json:"datacenters,omitempty"`
	// The KVMs to filter retrieved racks on.
	Kvms                 []string `protobuf:"bytes,3,rep,name=kvms,proto3" json:"kvms,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListRacksRequest) Reset()         { *m = ListRacksRequest{} }
func (m *ListRacksRequest) String() string { return proto.CompactTextString(m) }
func (*ListRacksRequest) ProtoMessage()    {}
func (*ListRacksRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_5e7bc6de9b3dbb6d, []int{1}
}

func (m *ListRacksRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListRacksRequest.Unmarshal(m, b)
}
func (m *ListRacksRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListRacksRequest.Marshal(b, m, deterministic)
}
func (m *ListRacksRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListRacksRequest.Merge(m, src)
}
func (m *ListRacksRequest) XXX_Size() int {
	return xxx_messageInfo_ListRacksRequest.Size(m)
}
func (m *ListRacksRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListRacksRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListRacksRequest proto.InternalMessageInfo

func (m *ListRacksRequest) GetNames() []string {
	if m != nil {
		return m.Names
	}
	return nil
}

func (m *ListRacksRequest) GetDatacenters() []string {
	if m != nil {
		return m.Datacenters
	}
	return nil
}

func (m *ListRacksRequest) GetKvms() []string {
	if m != nil {
		return m.Kvms
	}
	return nil
}

// A response containing a list of racks in the database.
type ListRacksResponse struct {
	// The racks matching the request.
	Racks                []*Rack  `protobuf:"bytes,1,rep,name=racks,proto3" json:"racks,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListRacksResponse) Reset()         { *m = ListRacksResponse{} }
func (m *ListRacksResponse) String() string { return proto.CompactTextString(m) }
func (*ListRacksResponse) ProtoMessage()    {}
func (*ListRacksResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_5e7bc6de9b3dbb6d, []int{2}
}

func (m *ListRacksResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListRacksResponse.Unmarshal(m, b)
}
func (m *ListRacksResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListRacksResponse.Marshal(b, m, deterministic)
}
func (m *ListRacksResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListRacksResponse.Merge(m, src)
}
func (m *ListRacksResponse) XXX_Size() int {
	return xxx_messageInfo_ListRacksResponse.Size(m)
}
func (m *ListRacksResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListRacksResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListRacksResponse proto.InternalMessageInfo

func (m *ListRacksResponse) GetRacks() []*Rack {
	if m != nil {
		return m.Racks
	}
	return nil
}

func init() {
	proto.RegisterType((*Rack)(nil), "crimson.Rack")
	proto.RegisterType((*ListRacksRequest)(nil), "crimson.ListRacksRequest")
	proto.RegisterType((*ListRacksResponse)(nil), "crimson.ListRacksResponse")
}

func init() {
	proto.RegisterFile("github.com/tetrafolium/luci-go/machine-db/api/crimson/v1/racks.proto", fileDescriptor_5e7bc6de9b3dbb6d)
}

var fileDescriptor_5e7bc6de9b3dbb6d = []byte{
	// 277 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x90, 0x31, 0x4f, 0xc3, 0x30,
	0x10, 0x85, 0x95, 0xa6, 0x05, 0xd5, 0x15, 0xa8, 0x58, 0x0c, 0x16, 0x03, 0x8a, 0xc2, 0x92, 0x05,
	0x5b, 0xc0, 0x82, 0xf8, 0x0d, 0x4c, 0x66, 0x47, 0x72, 0x1d, 0xab, 0xb5, 0x82, 0xed, 0xe0, 0x73,
	0xf2, 0x47, 0xf8, 0xc3, 0xc8, 0xe7, 0x0a, 0x65, 0x64, 0xbb, 0xbc, 0x77, 0x79, 0xf7, 0xf9, 0x91,
	0xb7, 0x63, 0xe0, 0xfa, 0x14, 0x83, 0xb3, 0x93, 0xe3, 0x21, 0x1e, 0xc5, 0xd7, 0xa4, 0xad, 0x70,
	0x4a, 0x9f, 0xac, 0x37, 0x8f, 0xfd, 0x41, 0xa8, 0xd1, 0x0a, 0x1d, 0xad, 0x83, 0xe0, 0xc5, 0xfc,
	0x24, 0xa2, 0xd2, 0x03, 0xf0, 0x31, 0x86, 0x14, 0xe8, 0xe5, 0x59, 0xbf, 0xfb, 0x5f, 0x48, 0x70,
	0xae, 0x64, 0x40, 0x52, 0xc9, 0x9c, 0x43, 0xda, 0x9f, 0x8a, 0xac, 0xa5, 0xd2, 0x03, 0xa5, 0x64,
	0xed, 0x95, 0x33, 0xac, 0x6a, 0xaa, 0x6e, 0x2b, 0x71, 0xa6, 0x0d, 0xd9, 0xf5, 0x06, 0x74, 0xb4,
	0x63, 0xb2, 0xc1, 0xb3, 0x15, 0x5a, 0x4b, 0x89, 0xde, 0x13, 0xd2, 0xab, 0xa4, 0xb4, 0xf1, 0xc9,
	0x44, 0x56, 0xe3, 0xc2, 0x42, 0xa1, 0x0f, 0x64, 0x83, 0xe7, 0xd8, 0xba, 0xa9, 0xba, 0xeb, 0xe7,
	0x2b, 0x5e, 0x30, 0xf8, 0x47, 0x16, 0x65, 0xf1, 0xe8, 0x9e, 0xd4, 0xc3, 0xec, 0xd8, 0x06, 0xff,
	0xce, 0x63, 0xfb, 0x49, 0xf6, 0xef, 0x16, 0x52, 0x06, 0x03, 0x69, 0xbe, 0x27, 0x03, 0x89, 0xde,
	0x92, 0x4d, 0x86, 0x02, 0x56, 0x35, 0x75, 0xb7, 0x95, 0xe5, 0x03, 0x11, 0xff, 0xce, 0x01, 0x5b,
	0xa1, 0xb7, 0x94, 0xf2, 0xc3, 0x86, 0xd9, 0x01, 0xab, 0xd1, 0xc2, 0xb9, 0x7d, 0x25, 0x37, 0x8b,
	0x7c, 0x18, 0x83, 0x07, 0x93, 0x59, 0xb1, 0x5e, 0x3c, 0xb0, 0xcb, 0xac, 0xa5, 0x5f, 0x9e, 0xd7,
	0x64, 0xf1, 0x0e, 0x17, 0x58, 0xdb, 0xcb, 0x6f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x86, 0xec, 0x2f,
	0xd1, 0xb9, 0x01, 0x00, 0x00,
}
