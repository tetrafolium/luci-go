// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/tetrafolium/luci-go/machine-db/api/crimson/v1/vms.proto

package crimson

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	v1 "github.com/tetrafolium/luci-go/machine-db/api/common/v1"
	field_mask "google.golang.org/genproto/protobuf/field_mask"
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

// A VM in the database.
type VM struct {
	// The name of this VM on the network. Uniquely identifies this VM.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The VLAN this VM belongs to.
	// When creating a VM, omit this field. It will be inferred from the IPv4 address.
	Vlan int64 `protobuf:"varint,2,opt,name=vlan,proto3" json:"vlan,omitempty"`
	// The physical host this VM is running on.
	Host string `protobuf:"bytes,3,opt,name=host,proto3" json:"host,omitempty"`
	// The VLAN this VM's physical host belongs to.
	// When creating a VM, omit this field. It will be inferred from the host.
	HostVlan int64 `protobuf:"varint,4,opt,name=host_vlan,json=hostVlan,proto3" json:"host_vlan,omitempty"`
	// The operating system running on this VM.
	Os string `protobuf:"bytes,5,opt,name=os,proto3" json:"os,omitempty"`
	// A description of this VM.
	Description string `protobuf:"bytes,6,opt,name=description,proto3" json:"description,omitempty"`
	// The deployment ticket associated with this VM.
	DeploymentTicket string `protobuf:"bytes,7,opt,name=deployment_ticket,json=deploymentTicket,proto3" json:"deployment_ticket,omitempty"`
	// The IPv4 address associated with this host.
	Ipv4 string `protobuf:"bytes,8,opt,name=ipv4,proto3" json:"ipv4,omitempty"`
	// The state of this VM.
	State                v1.State `protobuf:"varint,9,opt,name=state,proto3,enum=common.State" json:"state,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *VM) Reset()         { *m = VM{} }
func (m *VM) String() string { return proto.CompactTextString(m) }
func (*VM) ProtoMessage()    {}
func (*VM) Descriptor() ([]byte, []int) {
	return fileDescriptor_ac68362bdfdec1d3, []int{0}
}

func (m *VM) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VM.Unmarshal(m, b)
}
func (m *VM) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VM.Marshal(b, m, deterministic)
}
func (m *VM) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VM.Merge(m, src)
}
func (m *VM) XXX_Size() int {
	return xxx_messageInfo_VM.Size(m)
}
func (m *VM) XXX_DiscardUnknown() {
	xxx_messageInfo_VM.DiscardUnknown(m)
}

var xxx_messageInfo_VM proto.InternalMessageInfo

func (m *VM) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *VM) GetVlan() int64 {
	if m != nil {
		return m.Vlan
	}
	return 0
}

func (m *VM) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *VM) GetHostVlan() int64 {
	if m != nil {
		return m.HostVlan
	}
	return 0
}

func (m *VM) GetOs() string {
	if m != nil {
		return m.Os
	}
	return ""
}

func (m *VM) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *VM) GetDeploymentTicket() string {
	if m != nil {
		return m.DeploymentTicket
	}
	return ""
}

func (m *VM) GetIpv4() string {
	if m != nil {
		return m.Ipv4
	}
	return ""
}

func (m *VM) GetState() v1.State {
	if m != nil {
		return m.State
	}
	return v1.State_STATE_UNSPECIFIED
}

// A request to create a new VM in the database.
type CreateVMRequest struct {
	// The VM to create in the database.
	Vm                   *VM      `protobuf:"bytes,1,opt,name=vm,proto3" json:"vm,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateVMRequest) Reset()         { *m = CreateVMRequest{} }
func (m *CreateVMRequest) String() string { return proto.CompactTextString(m) }
func (*CreateVMRequest) ProtoMessage()    {}
func (*CreateVMRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ac68362bdfdec1d3, []int{1}
}

func (m *CreateVMRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateVMRequest.Unmarshal(m, b)
}
func (m *CreateVMRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateVMRequest.Marshal(b, m, deterministic)
}
func (m *CreateVMRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateVMRequest.Merge(m, src)
}
func (m *CreateVMRequest) XXX_Size() int {
	return xxx_messageInfo_CreateVMRequest.Size(m)
}
func (m *CreateVMRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateVMRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateVMRequest proto.InternalMessageInfo

func (m *CreateVMRequest) GetVm() *VM {
	if m != nil {
		return m.Vm
	}
	return nil
}

// A request to list VMs in the database.
type ListVMsRequest struct {
	// The names of VMs to get.
	Names []string `protobuf:"bytes,1,rep,name=names,proto3" json:"names,omitempty"`
	// The VLANs to filter retrieved VMs on.
	Vlans []int64 `protobuf:"varint,2,rep,packed,name=vlans,proto3" json:"vlans,omitempty"`
	// The IPv4 addresses to filter retrieved VMs on.
	Ipv4S []string `protobuf:"bytes,3,rep,name=ipv4s,proto3" json:"ipv4s,omitempty"`
	// The physical hosts to filter retrieved VMs on.
	Hosts []string `protobuf:"bytes,4,rep,name=hosts,proto3" json:"hosts,omitempty"`
	// The physical host VLANs to filter retrieved VMs on.
	HostVlans []int64 `protobuf:"varint,5,rep,packed,name=host_vlans,json=hostVlans,proto3" json:"host_vlans,omitempty"`
	// The operating system to filter retrieved VMs on.
	Oses []string `protobuf:"bytes,6,rep,name=oses,proto3" json:"oses,omitempty"`
	// The states to filter retrieved VMs on.
	States               []v1.State `protobuf:"varint,10,rep,packed,name=states,proto3,enum=common.State" json:"states,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *ListVMsRequest) Reset()         { *m = ListVMsRequest{} }
func (m *ListVMsRequest) String() string { return proto.CompactTextString(m) }
func (*ListVMsRequest) ProtoMessage()    {}
func (*ListVMsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ac68362bdfdec1d3, []int{2}
}

func (m *ListVMsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListVMsRequest.Unmarshal(m, b)
}
func (m *ListVMsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListVMsRequest.Marshal(b, m, deterministic)
}
func (m *ListVMsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListVMsRequest.Merge(m, src)
}
func (m *ListVMsRequest) XXX_Size() int {
	return xxx_messageInfo_ListVMsRequest.Size(m)
}
func (m *ListVMsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListVMsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListVMsRequest proto.InternalMessageInfo

func (m *ListVMsRequest) GetNames() []string {
	if m != nil {
		return m.Names
	}
	return nil
}

func (m *ListVMsRequest) GetVlans() []int64 {
	if m != nil {
		return m.Vlans
	}
	return nil
}

func (m *ListVMsRequest) GetIpv4S() []string {
	if m != nil {
		return m.Ipv4S
	}
	return nil
}

func (m *ListVMsRequest) GetHosts() []string {
	if m != nil {
		return m.Hosts
	}
	return nil
}

func (m *ListVMsRequest) GetHostVlans() []int64 {
	if m != nil {
		return m.HostVlans
	}
	return nil
}

func (m *ListVMsRequest) GetOses() []string {
	if m != nil {
		return m.Oses
	}
	return nil
}

func (m *ListVMsRequest) GetStates() []v1.State {
	if m != nil {
		return m.States
	}
	return nil
}

// A response containing a list of VMs in the database.
type ListVMsResponse struct {
	// The VMs matching this request.
	Vms                  []*VM    `protobuf:"bytes,1,rep,name=vms,proto3" json:"vms,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListVMsResponse) Reset()         { *m = ListVMsResponse{} }
func (m *ListVMsResponse) String() string { return proto.CompactTextString(m) }
func (*ListVMsResponse) ProtoMessage()    {}
func (*ListVMsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_ac68362bdfdec1d3, []int{3}
}

func (m *ListVMsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListVMsResponse.Unmarshal(m, b)
}
func (m *ListVMsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListVMsResponse.Marshal(b, m, deterministic)
}
func (m *ListVMsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListVMsResponse.Merge(m, src)
}
func (m *ListVMsResponse) XXX_Size() int {
	return xxx_messageInfo_ListVMsResponse.Size(m)
}
func (m *ListVMsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListVMsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListVMsResponse proto.InternalMessageInfo

func (m *ListVMsResponse) GetVms() []*VM {
	if m != nil {
		return m.Vms
	}
	return nil
}

// A request to update a VM in the database.
type UpdateVMRequest struct {
	// The VM to update in the database.
	Vm *VM `protobuf:"bytes,1,opt,name=vm,proto3" json:"vm,omitempty"`
	// The fields to update in the VM.
	UpdateMask           *field_mask.FieldMask `protobuf:"bytes,2,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *UpdateVMRequest) Reset()         { *m = UpdateVMRequest{} }
func (m *UpdateVMRequest) String() string { return proto.CompactTextString(m) }
func (*UpdateVMRequest) ProtoMessage()    {}
func (*UpdateVMRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ac68362bdfdec1d3, []int{4}
}

func (m *UpdateVMRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UpdateVMRequest.Unmarshal(m, b)
}
func (m *UpdateVMRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UpdateVMRequest.Marshal(b, m, deterministic)
}
func (m *UpdateVMRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UpdateVMRequest.Merge(m, src)
}
func (m *UpdateVMRequest) XXX_Size() int {
	return xxx_messageInfo_UpdateVMRequest.Size(m)
}
func (m *UpdateVMRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_UpdateVMRequest.DiscardUnknown(m)
}

var xxx_messageInfo_UpdateVMRequest proto.InternalMessageInfo

func (m *UpdateVMRequest) GetVm() *VM {
	if m != nil {
		return m.Vm
	}
	return nil
}

func (m *UpdateVMRequest) GetUpdateMask() *field_mask.FieldMask {
	if m != nil {
		return m.UpdateMask
	}
	return nil
}

func init() {
	proto.RegisterType((*VM)(nil), "crimson.VM")
	proto.RegisterType((*CreateVMRequest)(nil), "crimson.CreateVMRequest")
	proto.RegisterType((*ListVMsRequest)(nil), "crimson.ListVMsRequest")
	proto.RegisterType((*ListVMsResponse)(nil), "crimson.ListVMsResponse")
	proto.RegisterType((*UpdateVMRequest)(nil), "crimson.UpdateVMRequest")
}

func init() {
	proto.RegisterFile("github.com/tetrafolium/luci-go/machine-db/api/crimson/v1/vms.proto", fileDescriptor_ac68362bdfdec1d3)
}

var fileDescriptor_ac68362bdfdec1d3 = []byte{
	// 462 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x92, 0x41, 0x8f, 0xd3, 0x30,
	0x10, 0x85, 0x95, 0xa4, 0xed, 0x6e, 0xa7, 0xa2, 0x05, 0x8b, 0x83, 0x55, 0xb4, 0x52, 0x54, 0x84,
	0x54, 0x09, 0xe1, 0x40, 0xe1, 0x80, 0xe0, 0x88, 0xc4, 0x89, 0x5e, 0x02, 0xf4, 0x5a, 0xa5, 0xa9,
	0xb7, 0xb5, 0x1a, 0xc7, 0x21, 0xe3, 0x44, 0xe2, 0xef, 0xf1, 0x9f, 0xb8, 0xa3, 0x19, 0xb7, 0x0b,
	0x62, 0x2f, 0x7b, 0xea, 0xcc, 0x37, 0xcf, 0xb5, 0xdf, 0xcb, 0xc0, 0xfb, 0x83, 0x53, 0xe5, 0xb1,
	0x75, 0xd6, 0x74, 0x56, 0xb9, 0xf6, 0x90, 0x55, 0x5d, 0x69, 0x32, 0x5b, 0x94, 0x47, 0x53, 0xeb,
	0x57, 0xfb, 0x5d, 0x56, 0x34, 0x26, 0x2b, 0x5b, 0x63, 0xd1, 0xd5, 0x59, 0xff, 0x26, 0xeb, 0x2d,
	0xaa, 0xa6, 0x75, 0xde, 0x89, 0xab, 0x33, 0x9d, 0xa7, 0x07, 0xe7, 0x0e, 0x95, 0xce, 0x18, 0xef,
	0xba, 0xdb, 0xec, 0xd6, 0xe8, 0x6a, 0xbf, 0xb5, 0x05, 0x9e, 0x82, 0x74, 0xfe, 0xe1, 0x41, 0x97,
	0x38, 0x6b, 0xc3, 0x1d, 0xe8, 0x0b, 0xaf, 0xcf, 0xd7, 0x2c, 0x7e, 0x47, 0x10, 0x6f, 0xd6, 0x42,
	0xc0, 0xa0, 0x2e, 0xac, 0x96, 0x51, 0x1a, 0x2d, 0xc7, 0x39, 0xd7, 0xc4, 0xfa, 0xaa, 0xa8, 0x65,
	0x9c, 0x46, 0xcb, 0x24, 0xe7, 0x9a, 0xd8, 0xd1, 0xa1, 0x97, 0x49, 0xd0, 0x51, 0x2d, 0x9e, 0xc1,
	0x98, 0x7e, 0xb7, 0x2c, 0x1e, 0xb0, 0xf8, 0x9a, 0xc0, 0x86, 0x0e, 0x4c, 0x21, 0x76, 0x28, 0x87,
	0x2c, 0x8f, 0x1d, 0x8a, 0x14, 0x26, 0x7b, 0x8d, 0x65, 0x6b, 0x1a, 0x6f, 0x5c, 0x2d, 0x47, 0x3c,
	0xf8, 0x17, 0x89, 0x97, 0xf0, 0x64, 0xaf, 0x9b, 0xca, 0xfd, 0xb4, 0xba, 0xf6, 0x5b, 0x6f, 0xca,
	0x93, 0xf6, 0xf2, 0x8a, 0x75, 0x8f, 0xff, 0x0e, 0xbe, 0x31, 0xa7, 0xf7, 0x98, 0xa6, 0x7f, 0x27,
	0xaf, 0xc3, 0x7b, 0xa8, 0x16, 0xcf, 0x61, 0xc8, 0x16, 0xe5, 0x38, 0x8d, 0x96, 0xd3, 0xd5, 0x23,
	0x15, 0xac, 0xab, 0xaf, 0x04, 0xf3, 0x30, 0x5b, 0x28, 0x98, 0x7d, 0x6a, 0x75, 0xe1, 0xf5, 0x66,
	0x9d, 0xeb, 0x1f, 0x9d, 0x66, 0x1f, 0x71, 0x6f, 0x39, 0x81, 0xc9, 0x6a, 0xa2, 0xce, 0xf1, 0xab,
	0xcd, 0x3a, 0x8f, 0x7b, 0xbb, 0xf8, 0x15, 0xc1, 0xf4, 0x8b, 0x41, 0xbf, 0x59, 0xe3, 0x45, 0xff,
	0x14, 0x86, 0x94, 0x13, 0xca, 0x28, 0x4d, 0x96, 0xe3, 0x3c, 0x34, 0x44, 0x29, 0x08, 0x94, 0x71,
	0x9a, 0x2c, 0x93, 0x3c, 0x34, 0x44, 0xe9, 0x6d, 0x28, 0x93, 0xa0, 0xe5, 0x86, 0x28, 0x05, 0x85,
	0x72, 0x10, 0x28, 0x37, 0xe2, 0x06, 0xe0, 0x2e, 0x4f, 0x8a, 0x8e, 0xfe, 0x66, 0x7c, 0x09, 0x14,
	0xc9, 0xb2, 0x43, 0x8d, 0x72, 0xc4, 0x67, 0xb8, 0x16, 0x2f, 0x60, 0x14, 0xbe, 0xaa, 0x84, 0x34,
	0xb9, 0xef, 0xf9, 0x3c, 0x5c, 0xbc, 0x86, 0xd9, 0x9d, 0x07, 0x6c, 0x5c, 0x8d, 0x5a, 0xdc, 0x40,
	0xd2, 0xdb, 0x60, 0xe1, 0x3f, 0xd7, 0xc4, 0x17, 0x27, 0x98, 0x7d, 0x6f, 0xf6, 0x0f, 0x8e, 0x49,
	0x7c, 0x84, 0x49, 0xc7, 0x7a, 0xde, 0x4f, 0x5e, 0x9d, 0xc9, 0x6a, 0xae, 0xc2, 0x0a, 0xab, 0xcb,
	0x0a, 0xab, 0xcf, 0xb4, 0xc2, 0xeb, 0x02, 0x4f, 0x39, 0x04, 0x39, 0xd5, 0xbb, 0x11, 0xcf, 0xdf,
	0xfe, 0x09, 0x00, 0x00, 0xff, 0xff, 0x44, 0x6f, 0x15, 0x63, 0x35, 0x03, 0x00, 0x00,
}
