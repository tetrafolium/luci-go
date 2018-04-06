// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/machine-db/api/crimson/v1/vm_slots.proto

package crimson

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// A request to find available VM slots in the database.
type FindVMSlotsRequest struct {
	// The number of available VM slots to find.
	// Values < 1 return all available VM slots.
	Slots int32 `protobuf:"varint,1,opt,name=slots" json:"slots,omitempty"`
	// The platform manufacturers to filter found VM slots on.
	Manufacturers []string `protobuf:"bytes,2,rep,name=manufacturers" json:"manufacturers,omitempty"`
}

func (m *FindVMSlotsRequest) Reset()                    { *m = FindVMSlotsRequest{} }
func (m *FindVMSlotsRequest) String() string            { return proto.CompactTextString(m) }
func (*FindVMSlotsRequest) ProtoMessage()               {}
func (*FindVMSlotsRequest) Descriptor() ([]byte, []int) { return fileDescriptor14, []int{0} }

func (m *FindVMSlotsRequest) GetSlots() int32 {
	if m != nil {
		return m.Slots
	}
	return 0
}

func (m *FindVMSlotsRequest) GetManufacturers() []string {
	if m != nil {
		return m.Manufacturers
	}
	return nil
}

// A response containing a list of available VM slots in the database.
type FindVMSlotsResponse struct {
	// The hosts with available VM slots.
	// Only includes name, vlan_id, and vm_slots.
	// vm_slots in this context means the number of available VM slots.
	Hosts []*PhysicalHost `protobuf:"bytes,1,rep,name=hosts" json:"hosts,omitempty"`
}

func (m *FindVMSlotsResponse) Reset()                    { *m = FindVMSlotsResponse{} }
func (m *FindVMSlotsResponse) String() string            { return proto.CompactTextString(m) }
func (*FindVMSlotsResponse) ProtoMessage()               {}
func (*FindVMSlotsResponse) Descriptor() ([]byte, []int) { return fileDescriptor14, []int{1} }

func (m *FindVMSlotsResponse) GetHosts() []*PhysicalHost {
	if m != nil {
		return m.Hosts
	}
	return nil
}

func init() {
	proto.RegisterType((*FindVMSlotsRequest)(nil), "crimson.FindVMSlotsRequest")
	proto.RegisterType((*FindVMSlotsResponse)(nil), "crimson.FindVMSlotsResponse")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/machine-db/api/crimson/v1/vm_slots.proto", fileDescriptor14)
}

var fileDescriptor14 = []byte{
	// 213 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x8e, 0xbf, 0x4b, 0xc4, 0x30,
	0x14, 0x80, 0xa9, 0x47, 0x15, 0x23, 0x2e, 0x51, 0xa1, 0x38, 0x95, 0xc3, 0xa1, 0x20, 0x26, 0xa8,
	0xb3, 0x8b, 0x82, 0xb8, 0x08, 0x47, 0x04, 0xd7, 0x23, 0x4d, 0x63, 0x13, 0x68, 0xf2, 0x62, 0x5e,
	0x52, 0xf0, 0xbf, 0x17, 0xdb, 0x3a, 0x74, 0xbc, 0xf1, 0xfd, 0xe0, 0xfb, 0x3e, 0xf2, 0xd4, 0x03,
	0x53, 0x26, 0x82, 0xb3, 0xd9, 0x31, 0x88, 0x3d, 0x1f, 0xb2, 0xb2, 0xdc, 0x49, 0x65, 0xac, 0xd7,
	0x77, 0x5d, 0xcb, 0x65, 0xb0, 0x5c, 0x45, 0xeb, 0x10, 0x3c, 0x1f, 0xef, 0xf9, 0xe8, 0xf6, 0x38,
	0x40, 0x42, 0x16, 0x22, 0x24, 0xa0, 0x27, 0xcb, 0xe9, 0xfa, 0xe5, 0x40, 0x4e, 0x30, 0x3f, 0x68,
	0x95, 0x1c, 0xf6, 0x06, 0xf0, 0x9f, 0xb6, 0xdd, 0x11, 0xfa, 0x6a, 0x7d, 0xf7, 0xf9, 0xfe, 0xf1,
	0xa7, 0x10, 0xfa, 0x3b, 0x6b, 0x4c, 0xf4, 0x92, 0x94, 0x93, 0xb2, 0x2a, 0xea, 0xa2, 0x29, 0xc5,
	0x3c, 0xd0, 0x1b, 0x72, 0xee, 0xa4, 0xcf, 0x5f, 0x52, 0xa5, 0x1c, 0x75, 0xc4, 0xea, 0xa8, 0xde,
	0x34, 0xa7, 0x62, 0xbd, 0xdc, 0x3e, 0x93, 0x8b, 0x15, 0x11, 0x03, 0x78, 0xd4, 0xf4, 0x96, 0x94,
	0x93, 0xb7, 0x2a, 0xea, 0x4d, 0x73, 0xf6, 0x70, 0xc5, 0x96, 0x32, 0xb6, 0x5b, 0xb2, 0xde, 0x00,
	0x93, 0x98, 0x7f, 0xda, 0xe3, 0x29, 0xee, 0xf1, 0x37, 0x00, 0x00, 0xff, 0xff, 0x75, 0x8f, 0x62,
	0x9b, 0x2b, 0x01, 0x00, 0x00,
}
