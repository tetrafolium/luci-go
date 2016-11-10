// Code generated by protoc-gen-go.
// source: github.com/luci/luci-go/cipd/client/cipd/internal/messages/messages.proto
// DO NOT EDIT!

/*
Package messages is a generated protocol buffer package.

It is generated from these files:
	github.com/luci/luci-go/cipd/client/cipd/internal/messages/messages.proto

It has these top-level messages:
	BlobWithSHA1
	TagCache
	InstanceCache
*/
package messages

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/luci/luci-go/common/proto/google"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// BlobWithSHA1 is a wrapper around a binary blob with SHA1 hash to verify
// its integrity.
type BlobWithSHA1 struct {
	Blob []byte `protobuf:"bytes,1,opt,name=blob,proto3" json:"blob,omitempty"`
	Sha1 []byte `protobuf:"bytes,2,opt,name=sha1,proto3" json:"sha1,omitempty"`
}

func (m *BlobWithSHA1) Reset()                    { *m = BlobWithSHA1{} }
func (m *BlobWithSHA1) String() string            { return proto.CompactTextString(m) }
func (*BlobWithSHA1) ProtoMessage()               {}
func (*BlobWithSHA1) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// TagCache stores a mapping (package name, tag) -> instance ID to speed up
// subsequence ResolveVersion calls when tags are used.
type TagCache struct {
	// Capped list of entries, most recently resolved is last.
	Entries []*TagCache_Entry `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty"`
}

func (m *TagCache) Reset()                    { *m = TagCache{} }
func (m *TagCache) String() string            { return proto.CompactTextString(m) }
func (*TagCache) ProtoMessage()               {}
func (*TagCache) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *TagCache) GetEntries() []*TagCache_Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

type TagCache_Entry struct {
	Package    string `protobuf:"bytes,1,opt,name=package" json:"package,omitempty"`
	Tag        string `protobuf:"bytes,2,opt,name=tag" json:"tag,omitempty"`
	InstanceId string `protobuf:"bytes,3,opt,name=instance_id,json=instanceId" json:"instance_id,omitempty"`
}

func (m *TagCache_Entry) Reset()                    { *m = TagCache_Entry{} }
func (m *TagCache_Entry) String() string            { return proto.CompactTextString(m) }
func (*TagCache_Entry) ProtoMessage()               {}
func (*TagCache_Entry) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1, 0} }

// InstanceCache stores a list of instances in cache
// and their last access time.
type InstanceCache struct {
	// Entries is a map of {instance id -> information about instance}.
	Entries map[string]*InstanceCache_Entry `protobuf:"bytes,1,rep,name=entries" json:"entries,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// LastSynced is timestamp when we synchronized Entries with actual
	// instance files.
	LastSynced *google_protobuf.Timestamp `protobuf:"bytes,2,opt,name=last_synced,json=lastSynced" json:"last_synced,omitempty"`
}

func (m *InstanceCache) Reset()                    { *m = InstanceCache{} }
func (m *InstanceCache) String() string            { return proto.CompactTextString(m) }
func (*InstanceCache) ProtoMessage()               {}
func (*InstanceCache) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *InstanceCache) GetEntries() map[string]*InstanceCache_Entry {
	if m != nil {
		return m.Entries
	}
	return nil
}

func (m *InstanceCache) GetLastSynced() *google_protobuf.Timestamp {
	if m != nil {
		return m.LastSynced
	}
	return nil
}

// Entry stores info about an instance.
type InstanceCache_Entry struct {
	// LastAccess is last time this instance was retrieved from or put to the
	// cache.
	LastAccess *google_protobuf.Timestamp `protobuf:"bytes,2,opt,name=last_access,json=lastAccess" json:"last_access,omitempty"`
}

func (m *InstanceCache_Entry) Reset()                    { *m = InstanceCache_Entry{} }
func (m *InstanceCache_Entry) String() string            { return proto.CompactTextString(m) }
func (*InstanceCache_Entry) ProtoMessage()               {}
func (*InstanceCache_Entry) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2, 0} }

func (m *InstanceCache_Entry) GetLastAccess() *google_protobuf.Timestamp {
	if m != nil {
		return m.LastAccess
	}
	return nil
}

func init() {
	proto.RegisterType((*BlobWithSHA1)(nil), "messages.BlobWithSHA1")
	proto.RegisterType((*TagCache)(nil), "messages.TagCache")
	proto.RegisterType((*TagCache_Entry)(nil), "messages.TagCache.Entry")
	proto.RegisterType((*InstanceCache)(nil), "messages.InstanceCache")
	proto.RegisterType((*InstanceCache_Entry)(nil), "messages.InstanceCache.Entry")
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/cipd/client/cipd/internal/messages/messages.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 366 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x8c, 0x90, 0x51, 0x6b, 0xdb, 0x30,
	0x14, 0x85, 0x71, 0xb2, 0x2c, 0x89, 0x9c, 0xc1, 0xd0, 0x93, 0x31, 0x8c, 0x84, 0xb0, 0x87, 0xbc,
	0xcc, 0x26, 0x09, 0x8c, 0xb1, 0xc1, 0x20, 0x5b, 0x0b, 0xcd, 0xab, 0x13, 0x28, 0x7d, 0x0a, 0xb2,
	0x7c, 0x2b, 0x8b, 0xc8, 0x92, 0xb1, 0xe4, 0x82, 0xff, 0x47, 0xff, 0x46, 0xff, 0x63, 0xb1, 0x1c,
	0xa5, 0x4d, 0x1f, 0x4a, 0x5f, 0xcc, 0xb9, 0xc7, 0x87, 0x7b, 0x3f, 0x1d, 0xb4, 0x65, 0xdc, 0xe4,
	0x75, 0x1a, 0x51, 0x55, 0xc4, 0xa2, 0xa6, 0xdc, 0x7e, 0x7e, 0x30, 0x15, 0x53, 0x5e, 0x66, 0x31,
	0x15, 0x1c, 0xa4, 0xe9, 0x34, 0x97, 0x06, 0x2a, 0x49, 0x44, 0x5c, 0x80, 0xd6, 0x84, 0x81, 0x3e,
	0x8b, 0xa8, 0xac, 0x94, 0x51, 0x78, 0xe4, 0xe6, 0x70, 0xca, 0x94, 0x62, 0x02, 0x62, 0xeb, 0xa7,
	0xf5, 0x7d, 0x6c, 0x78, 0x01, 0xda, 0x90, 0xa2, 0xec, 0xa2, 0xf3, 0x9f, 0x68, 0xf2, 0x4f, 0xa8,
	0xf4, 0x96, 0x9b, 0x7c, 0x77, 0xb3, 0x59, 0x62, 0x8c, 0x3e, 0xa5, 0x42, 0xa5, 0x81, 0x37, 0xf3,
	0x16, 0x93, 0xc4, 0xea, 0xd6, 0xd3, 0x39, 0x59, 0x06, 0xbd, 0xce, 0x6b, 0xf5, 0xfc, 0xd1, 0x43,
	0xa3, 0x3d, 0x61, 0xff, 0x09, 0xcd, 0x01, 0xaf, 0xd0, 0x10, 0xa4, 0xa9, 0x38, 0xe8, 0xc0, 0x9b,
	0xf5, 0x17, 0xfe, 0x2a, 0x88, 0xce, 0x44, 0x2e, 0x14, 0x5d, 0x4b, 0x53, 0x35, 0x89, 0x0b, 0x86,
	0x7b, 0x34, 0xb0, 0x0e, 0x0e, 0xd0, 0xb0, 0x24, 0xf4, 0x48, 0x18, 0xd8, 0xa3, 0xe3, 0xc4, 0x8d,
	0xf8, 0x2b, 0xea, 0x1b, 0xc2, 0xec, 0xd9, 0x71, 0xd2, 0x4a, 0x3c, 0x45, 0x3e, 0x97, 0xda, 0x10,
	0x49, 0xe1, 0xc0, 0xb3, 0xa0, 0x6f, 0xff, 0x20, 0x67, 0x6d, 0xb3, 0xf9, 0x53, 0x0f, 0x7d, 0xd9,
	0x9e, 0xc6, 0x8e, 0xed, 0xef, 0x5b, 0xb6, 0xef, 0x2f, 0x6c, 0x17, 0x49, 0x0b, 0xc8, 0x41, 0x5f,
	0x72, 0xe2, 0x3f, 0xc8, 0x17, 0x44, 0x9b, 0x83, 0x6e, 0x24, 0x85, 0xcc, 0xc2, 0xf8, 0xab, 0x30,
	0xea, 0x7a, 0x8d, 0x5c, 0xaf, 0xd1, 0xde, 0xf5, 0x9a, 0xa0, 0x36, 0xbe, 0xb3, 0xe9, 0xf0, 0xca,
	0x3d, 0xd2, 0x6d, 0x21, 0x94, 0x82, 0xd6, 0x1f, 0xdd, 0xb2, 0xb1, 0xe9, 0xf0, 0x0e, 0x4d, 0x5e,
	0xb3, 0xb5, 0xbd, 0x1c, 0xa1, 0x39, 0xb5, 0xd5, 0x4a, 0xbc, 0x46, 0x83, 0x07, 0x22, 0x6a, 0x38,
	0x2d, 0xfe, 0xf6, 0xde, 0x13, 0x9b, 0xa4, 0xcb, 0xfe, 0xee, 0xfd, 0xf2, 0xd2, 0xcf, 0xf6, 0xf4,
	0xfa, 0x39, 0x00, 0x00, 0xff, 0xff, 0xe8, 0xbb, 0x67, 0xc8, 0x7d, 0x02, 0x00, 0x00,
}
