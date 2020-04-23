// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/resultdb/proto/rpc/v1/artifact.proto

package rpcpb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

// A file produced during a build/test, typically a test artifact.
// The parent resource is either a TestResult or an Invocation.
type Artifact struct {
	// Can be used to refer to this artifact.
	// Format:
	// - For invocation-level artifacts:
	//   "invocations/{INVOCATION_ID}/artifacts/{ARTIFACT_ID}".
	// - For test-result-level artifacts:
	//   "invocations/{INVOCATION_ID}/tests/{URL_ESCAPED_TEST_ID}/results/{RESULT_ID}/artifacts/{ARTIFACT_ID}".
	// where URL_ESCAPED_TEST_ID is the test_id escaped with
	// https://golang.org/pkg/net/url/#PathEscape (see also https://aip.dev/122),
	// and ARTIFACT_ID is documented below.
	// Examples: "screenshot.png", "traces/a.txt".
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// A local identifier of the artifact, unique within the parent resource.
	// MAY have slashes, but MUST NOT start with a slash.
	// Regex: ^[[:word:]]([[:print:]]{0,254}[[:word:]])?$
	ArtifactId string `protobuf:"bytes,2,opt,name=artifact_id,json=artifactId,proto3" json:"artifact_id,omitempty"`
	// A signed short-lived URL to fetch the contents of the artifact.
	// See also fetch_url_expiration.
	//
	// Internally, this may have format "isolate://{host}/{ns}/{digest}" at the
	// storage level, but it is converted to an HTTPS URL before serving.
	FetchUrl string `protobuf:"bytes,3,opt,name=fetch_url,json=fetchUrl,proto3" json:"fetch_url,omitempty"`
	// When fetch_url expires. If expired, re-request this Artifact.
	FetchUrlExpiration *timestamp.Timestamp `protobuf:"bytes,4,opt,name=fetch_url_expiration,json=fetchUrlExpiration,proto3" json:"fetch_url_expiration,omitempty"`
	// Media type of the artifact.
	// Logs are typically "text/plain" and screenshots are typically "image/png".
	// Optional.
	ContentType string `protobuf:"bytes,5,opt,name=content_type,json=contentType,proto3" json:"content_type,omitempty"`
	// Size of the file.
	// Can be used in UI to decide between displaying the artifact inline or only
	// showing a link if it is too large.
	SizeBytes            int64    `protobuf:"varint,6,opt,name=size_bytes,json=sizeBytes,proto3" json:"size_bytes,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Artifact) Reset()         { *m = Artifact{} }
func (m *Artifact) String() string { return proto.CompactTextString(m) }
func (*Artifact) ProtoMessage()    {}
func (*Artifact) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad04eb79912d3886, []int{0}
}

func (m *Artifact) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Artifact.Unmarshal(m, b)
}
func (m *Artifact) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Artifact.Marshal(b, m, deterministic)
}
func (m *Artifact) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Artifact.Merge(m, src)
}
func (m *Artifact) XXX_Size() int {
	return xxx_messageInfo_Artifact.Size(m)
}
func (m *Artifact) XXX_DiscardUnknown() {
	xxx_messageInfo_Artifact.DiscardUnknown(m)
}

var xxx_messageInfo_Artifact proto.InternalMessageInfo

func (m *Artifact) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Artifact) GetArtifactId() string {
	if m != nil {
		return m.ArtifactId
	}
	return ""
}

func (m *Artifact) GetFetchUrl() string {
	if m != nil {
		return m.FetchUrl
	}
	return ""
}

func (m *Artifact) GetFetchUrlExpiration() *timestamp.Timestamp {
	if m != nil {
		return m.FetchUrlExpiration
	}
	return nil
}

func (m *Artifact) GetContentType() string {
	if m != nil {
		return m.ContentType
	}
	return ""
}

func (m *Artifact) GetSizeBytes() int64 {
	if m != nil {
		return m.SizeBytes
	}
	return 0
}

func init() {
	proto.RegisterType((*Artifact)(nil), "luci.resultdb.rpc.v1.Artifact")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/resultdb/proto/rpc/v1/artifact.proto", fileDescriptor_ad04eb79912d3886)
}

var fileDescriptor_ad04eb79912d3886 = []byte{
	// 282 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x3c, 0x8e, 0x31, 0x4f, 0xc3, 0x30,
	0x14, 0x84, 0x15, 0x5a, 0xaa, 0xd6, 0x65, 0xb2, 0x3a, 0x58, 0x45, 0xa8, 0x85, 0xa9, 0x93, 0xad,
	0xc2, 0xc4, 0x08, 0x12, 0x03, 0x12, 0x53, 0x55, 0x66, 0xcb, 0x71, 0x5e, 0x52, 0x4b, 0x4e, 0x6c,
	0x39, 0x2f, 0x15, 0xe5, 0x37, 0xf3, 0x23, 0x50, 0x9c, 0x3a, 0x9b, 0xfd, 0xdd, 0xdd, 0xbb, 0x23,
	0xaf, 0x95, 0xe3, 0xfa, 0x14, 0x5c, 0x6d, 0xba, 0x9a, 0xbb, 0x50, 0x09, 0xdb, 0x69, 0x23, 0x02,
	0xb4, 0x9d, 0xc5, 0x22, 0x17, 0x3e, 0x38, 0x74, 0x22, 0x78, 0x2d, 0xce, 0x7b, 0xa1, 0x02, 0x9a,
	0x52, 0x69, 0xe4, 0x91, 0xd2, 0x55, 0x6f, 0xe5, 0xc9, 0xca, 0x83, 0xd7, 0xfc, 0xbc, 0x5f, 0x6f,
	0x2a, 0xe7, 0x2a, 0x0b, 0x42, 0x79, 0x23, 0x4a, 0x03, 0xb6, 0x90, 0x39, 0x9c, 0xd4, 0xd9, 0xb8,
	0x30, 0xc4, 0x46, 0x43, 0xfc, 0xe5, 0x5d, 0x29, 0xd0, 0xd4, 0xd0, 0xa2, 0xaa, 0xfd, 0x60, 0x78,
	0xfa, 0xcb, 0xc8, 0xfc, 0xed, 0x5a, 0x45, 0x29, 0x99, 0x36, 0xaa, 0x06, 0x96, 0x6d, 0xb3, 0xdd,
	0xe2, 0x10, 0xdf, 0x74, 0x43, 0x96, 0x69, 0x8a, 0x34, 0x05, 0xbb, 0x89, 0x12, 0x49, 0xe8, 0xb3,
	0xa0, 0xf7, 0x64, 0x51, 0x02, 0xea, 0x93, 0xec, 0x82, 0x65, 0x93, 0x28, 0xcf, 0x23, 0xf8, 0x0e,
	0x96, 0x7e, 0x91, 0xd5, 0x28, 0x4a, 0xf8, 0xf1, 0x26, 0x28, 0x34, 0xae, 0x61, 0xd3, 0x6d, 0xb6,
	0x5b, 0x3e, 0xaf, 0xf9, 0x30, 0x8f, 0xa7, 0x79, 0xfc, 0x98, 0xe6, 0x1d, 0x68, 0xba, 0xf1, 0x31,
	0xa6, 0xe8, 0x23, 0xb9, 0xd3, 0xae, 0x41, 0x68, 0x50, 0xe2, 0xc5, 0x03, 0xbb, 0x8d, 0x6d, 0xcb,
	0x2b, 0x3b, 0x5e, 0x3c, 0xd0, 0x07, 0x42, 0x5a, 0xf3, 0x0b, 0x32, 0xbf, 0x20, 0xb4, 0x6c, 0xb6,
	0xcd, 0x76, 0x93, 0xc3, 0xa2, 0x27, 0xef, 0x3d, 0xc8, 0x67, 0xb1, 0xe9, 0xe5, 0x3f, 0x00, 0x00,
	0xff, 0xff, 0xbd, 0x94, 0xac, 0x32, 0x8a, 0x01, 0x00, 0x00,
}