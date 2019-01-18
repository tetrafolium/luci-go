// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/gce/api/tasks/v1/tasks.proto

package tasks

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	v1 "go.chromium.org/luci/gce/api/config/v1"
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

// A task to create a GCE instance from a VM entity.
type CreateInstance struct {
	// The ID of the VM entity to create a GCE instance from.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateInstance) Reset()         { *m = CreateInstance{} }
func (m *CreateInstance) String() string { return proto.CompactTextString(m) }
func (*CreateInstance) ProtoMessage()    {}
func (*CreateInstance) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{0}
}

func (m *CreateInstance) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateInstance.Unmarshal(m, b)
}
func (m *CreateInstance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateInstance.Marshal(b, m, deterministic)
}
func (m *CreateInstance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateInstance.Merge(m, src)
}
func (m *CreateInstance) XXX_Size() int {
	return xxx_messageInfo_CreateInstance.Size(m)
}
func (m *CreateInstance) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateInstance.DiscardUnknown(m)
}

var xxx_messageInfo_CreateInstance proto.InternalMessageInfo

func (m *CreateInstance) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

// A task to destroy a GCE instance created from a VM entity.
type DestroyInstance struct {
	// The ID of the VM entity to destroy a GCE instance for.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The URL of the GCE instance to destroy.
	Url                  string   `protobuf:"bytes,2,opt,name=url,proto3" json:"url,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DestroyInstance) Reset()         { *m = DestroyInstance{} }
func (m *DestroyInstance) String() string { return proto.CompactTextString(m) }
func (*DestroyInstance) ProtoMessage()    {}
func (*DestroyInstance) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{1}
}

func (m *DestroyInstance) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DestroyInstance.Unmarshal(m, b)
}
func (m *DestroyInstance) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DestroyInstance.Marshal(b, m, deterministic)
}
func (m *DestroyInstance) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DestroyInstance.Merge(m, src)
}
func (m *DestroyInstance) XXX_Size() int {
	return xxx_messageInfo_DestroyInstance.Size(m)
}
func (m *DestroyInstance) XXX_DiscardUnknown() {
	xxx_messageInfo_DestroyInstance.DiscardUnknown(m)
}

var xxx_messageInfo_DestroyInstance proto.InternalMessageInfo

func (m *DestroyInstance) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *DestroyInstance) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

// A task to drain a particular VM entity.
type DrainVM struct {
	// The ID of the VM entity to drain.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DrainVM) Reset()         { *m = DrainVM{} }
func (m *DrainVM) String() string { return proto.CompactTextString(m) }
func (*DrainVM) ProtoMessage()    {}
func (*DrainVM) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{2}
}

func (m *DrainVM) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DrainVM.Unmarshal(m, b)
}
func (m *DrainVM) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DrainVM.Marshal(b, m, deterministic)
}
func (m *DrainVM) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DrainVM.Merge(m, src)
}
func (m *DrainVM) XXX_Size() int {
	return xxx_messageInfo_DrainVM.Size(m)
}
func (m *DrainVM) XXX_DiscardUnknown() {
	xxx_messageInfo_DrainVM.DiscardUnknown(m)
}

var xxx_messageInfo_DrainVM proto.InternalMessageInfo

func (m *DrainVM) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

// A task to create or update a particular VM entity.
type EnsureVM struct {
	// The index of the VM entity to create or update.
	Index int32 `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"`
	// The attributes of the VM.
	Attributes *v1.VM `protobuf:"bytes,2,opt,name=attributes,proto3" json:"attributes,omitempty"`
	// The ID of the config this VM entity belongs to.
	Config string `protobuf:"bytes,3,opt,name=config,proto3" json:"config,omitempty"`
	// The lifetime of the VM in seconds.
	Lifetime int64 `protobuf:"varint,4,opt,name=lifetime,proto3" json:"lifetime,omitempty"`
	// The prefix to use when naming this VM.
	Prefix string `protobuf:"bytes,5,opt,name=prefix,proto3" json:"prefix,omitempty"`
	// The hostname of the Swarming server this VM connects to.
	Swarming             string   `protobuf:"bytes,6,opt,name=swarming,proto3" json:"swarming,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EnsureVM) Reset()         { *m = EnsureVM{} }
func (m *EnsureVM) String() string { return proto.CompactTextString(m) }
func (*EnsureVM) ProtoMessage()    {}
func (*EnsureVM) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{3}
}

func (m *EnsureVM) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnsureVM.Unmarshal(m, b)
}
func (m *EnsureVM) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnsureVM.Marshal(b, m, deterministic)
}
func (m *EnsureVM) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnsureVM.Merge(m, src)
}
func (m *EnsureVM) XXX_Size() int {
	return xxx_messageInfo_EnsureVM.Size(m)
}
func (m *EnsureVM) XXX_DiscardUnknown() {
	xxx_messageInfo_EnsureVM.DiscardUnknown(m)
}

var xxx_messageInfo_EnsureVM proto.InternalMessageInfo

func (m *EnsureVM) GetIndex() int32 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *EnsureVM) GetAttributes() *v1.VM {
	if m != nil {
		return m.Attributes
	}
	return nil
}

func (m *EnsureVM) GetConfig() string {
	if m != nil {
		return m.Config
	}
	return ""
}

func (m *EnsureVM) GetLifetime() int64 {
	if m != nil {
		return m.Lifetime
	}
	return 0
}

func (m *EnsureVM) GetPrefix() string {
	if m != nil {
		return m.Prefix
	}
	return ""
}

func (m *EnsureVM) GetSwarming() string {
	if m != nil {
		return m.Swarming
	}
	return ""
}

// A task to manage a Swarming bot associated with a VM entity.
type ManageBot struct {
	// The ID of the VM entity to manage a Swarming bot for.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ManageBot) Reset()         { *m = ManageBot{} }
func (m *ManageBot) String() string { return proto.CompactTextString(m) }
func (*ManageBot) ProtoMessage()    {}
func (*ManageBot) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{4}
}

func (m *ManageBot) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ManageBot.Unmarshal(m, b)
}
func (m *ManageBot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ManageBot.Marshal(b, m, deterministic)
}
func (m *ManageBot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ManageBot.Merge(m, src)
}
func (m *ManageBot) XXX_Size() int {
	return xxx_messageInfo_ManageBot.Size(m)
}
func (m *ManageBot) XXX_DiscardUnknown() {
	xxx_messageInfo_ManageBot.DiscardUnknown(m)
}

var xxx_messageInfo_ManageBot proto.InternalMessageInfo

func (m *ManageBot) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

// A task to process a config.
type ProcessConfig struct {
	// The ID of the config to process.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ProcessConfig) Reset()         { *m = ProcessConfig{} }
func (m *ProcessConfig) String() string { return proto.CompactTextString(m) }
func (*ProcessConfig) ProtoMessage()    {}
func (*ProcessConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{5}
}

func (m *ProcessConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProcessConfig.Unmarshal(m, b)
}
func (m *ProcessConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProcessConfig.Marshal(b, m, deterministic)
}
func (m *ProcessConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProcessConfig.Merge(m, src)
}
func (m *ProcessConfig) XXX_Size() int {
	return xxx_messageInfo_ProcessConfig.Size(m)
}
func (m *ProcessConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_ProcessConfig.DiscardUnknown(m)
}

var xxx_messageInfo_ProcessConfig proto.InternalMessageInfo

func (m *ProcessConfig) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

// A task to report GCE quota utilization.
type ReportQuota struct {
	// The ID of the project entity to report quota utilization for.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReportQuota) Reset()         { *m = ReportQuota{} }
func (m *ReportQuota) String() string { return proto.CompactTextString(m) }
func (*ReportQuota) ProtoMessage()    {}
func (*ReportQuota) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{6}
}

func (m *ReportQuota) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ReportQuota.Unmarshal(m, b)
}
func (m *ReportQuota) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ReportQuota.Marshal(b, m, deterministic)
}
func (m *ReportQuota) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReportQuota.Merge(m, src)
}
func (m *ReportQuota) XXX_Size() int {
	return xxx_messageInfo_ReportQuota.Size(m)
}
func (m *ReportQuota) XXX_DiscardUnknown() {
	xxx_messageInfo_ReportQuota.DiscardUnknown(m)
}

var xxx_messageInfo_ReportQuota proto.InternalMessageInfo

func (m *ReportQuota) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

// A task to terminate a Swarming bot associated with a VM entity.
type TerminateBot struct {
	// The ID of the VM entity to terminate a Swarming bot for.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The hostname of the Swarming bot to terminate.
	Hostname             string   `protobuf:"bytes,2,opt,name=hostname,proto3" json:"hostname,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TerminateBot) Reset()         { *m = TerminateBot{} }
func (m *TerminateBot) String() string { return proto.CompactTextString(m) }
func (*TerminateBot) ProtoMessage()    {}
func (*TerminateBot) Descriptor() ([]byte, []int) {
	return fileDescriptor_f63d8744087b0bbc, []int{7}
}

func (m *TerminateBot) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TerminateBot.Unmarshal(m, b)
}
func (m *TerminateBot) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TerminateBot.Marshal(b, m, deterministic)
}
func (m *TerminateBot) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TerminateBot.Merge(m, src)
}
func (m *TerminateBot) XXX_Size() int {
	return xxx_messageInfo_TerminateBot.Size(m)
}
func (m *TerminateBot) XXX_DiscardUnknown() {
	xxx_messageInfo_TerminateBot.DiscardUnknown(m)
}

var xxx_messageInfo_TerminateBot proto.InternalMessageInfo

func (m *TerminateBot) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *TerminateBot) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func init() {
	proto.RegisterType((*CreateInstance)(nil), "tasks.CreateInstance")
	proto.RegisterType((*DestroyInstance)(nil), "tasks.DestroyInstance")
	proto.RegisterType((*DrainVM)(nil), "tasks.DrainVM")
	proto.RegisterType((*EnsureVM)(nil), "tasks.EnsureVM")
	proto.RegisterType((*ManageBot)(nil), "tasks.ManageBot")
	proto.RegisterType((*ProcessConfig)(nil), "tasks.ProcessConfig")
	proto.RegisterType((*ReportQuota)(nil), "tasks.ReportQuota")
	proto.RegisterType((*TerminateBot)(nil), "tasks.TerminateBot")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/gce/api/tasks/v1/tasks.proto", fileDescriptor_f63d8744087b0bbc)
}

var fileDescriptor_f63d8744087b0bbc = []byte{
	// 330 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x91, 0xc1, 0x4f, 0xc2, 0x30,
	0x14, 0xc6, 0x33, 0x70, 0x08, 0x0f, 0x45, 0xd3, 0x18, 0x33, 0x31, 0x46, 0xb2, 0x13, 0xf1, 0xb0,
	0x05, 0xb9, 0x79, 0x14, 0x3c, 0x78, 0x58, 0xa2, 0x8b, 0xe1, 0x5e, 0xc6, 0x63, 0x34, 0xb2, 0x76,
	0x69, 0xdf, 0x14, 0xff, 0x2f, 0xff, 0x40, 0xb3, 0x96, 0x10, 0x23, 0xea, 0xed, 0xfd, 0xfa, 0xbd,
	0xef, 0x7d, 0xaf, 0x2d, 0x8c, 0x72, 0x15, 0x65, 0x2b, 0xad, 0x0a, 0x51, 0x15, 0x91, 0xd2, 0x79,
	0xbc, 0xae, 0x32, 0x11, 0xe7, 0x19, 0xc6, 0xbc, 0x14, 0x31, 0x71, 0xf3, 0x6a, 0xe2, 0xb7, 0x91,
	0x2b, 0xa2, 0x52, 0x2b, 0x52, 0xcc, 0xb7, 0xd0, 0x1f, 0xff, 0xeb, 0xcc, 0x94, 0x5c, 0x8a, 0xbc,
	0xb6, 0xba, 0xca, 0x79, 0xc3, 0x01, 0xf4, 0x26, 0x1a, 0x39, 0xe1, 0xa3, 0x34, 0xc4, 0x65, 0x86,
	0xac, 0x07, 0x0d, 0xb1, 0x08, 0xbc, 0x81, 0x37, 0xec, 0xa4, 0x0d, 0xb1, 0x08, 0xc7, 0x70, 0x32,
	0x45, 0x43, 0x5a, 0x7d, 0xfc, 0xd5, 0xc2, 0x4e, 0xa1, 0x59, 0xe9, 0x75, 0xd0, 0xb0, 0x07, 0x75,
	0x19, 0x5e, 0xc0, 0xe1, 0x54, 0x73, 0x21, 0x67, 0xc9, 0xde, 0xbc, 0x4f, 0x0f, 0xda, 0x0f, 0xd2,
	0x54, 0x1a, 0x67, 0x09, 0x3b, 0x03, 0x5f, 0xc8, 0x05, 0x6e, 0xac, 0xee, 0xa7, 0x0e, 0xd8, 0x0d,
	0x00, 0x27, 0xd2, 0x62, 0x5e, 0x11, 0x1a, 0x3b, 0xb6, 0x7b, 0x0b, 0xd1, 0x76, 0xef, 0x59, 0x92,
	0x7e, 0x53, 0xd9, 0x39, 0xb4, 0x9c, 0x10, 0x34, 0x6d, 0xc4, 0x96, 0x58, 0x1f, 0xda, 0x6b, 0xb1,
	0x44, 0x12, 0x05, 0x06, 0x07, 0x03, 0x6f, 0xd8, 0x4c, 0x77, 0x5c, 0x7b, 0x4a, 0x8d, 0x4b, 0xb1,
	0x09, 0x7c, 0xe7, 0x71, 0x54, 0x7b, 0xcc, 0x3b, 0xd7, 0x85, 0x90, 0x79, 0xd0, 0xb2, 0xca, 0x8e,
	0xc3, 0x4b, 0xe8, 0x24, 0x5c, 0xf2, 0x1c, 0xef, 0x15, 0xed, 0xdd, 0xe9, 0x1a, 0x8e, 0x9f, 0xb4,
	0xca, 0xd0, 0x98, 0x89, 0x4b, 0xff, 0xd9, 0x70, 0x05, 0xdd, 0x14, 0x4b, 0xa5, 0xe9, 0xb9, 0x52,
	0xc4, 0xf7, 0xe4, 0x3b, 0x38, 0x7a, 0xc1, 0x3a, 0x87, 0xd3, 0x6f, 0xf3, 0xeb, 0xc5, 0x56, 0xca,
	0x90, 0xe4, 0x05, 0x6e, 0x5f, 0x79, 0xc7, 0xf3, 0x96, 0xfd, 0xc8, 0xf1, 0x57, 0x00, 0x00, 0x00,
	0xff, 0xff, 0x10, 0xed, 0x4a, 0xe4, 0x39, 0x02, 0x00, 0x00,
}
