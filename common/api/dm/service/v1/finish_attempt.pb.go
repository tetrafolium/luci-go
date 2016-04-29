// Code generated by protoc-gen-go.
// source: finish_attempt.proto
// DO NOT EDIT!

package dm

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/luci/luci-go/common/proto/google"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// FinishAttemptReq sets the final result of an Attempt.
type FinishAttemptReq struct {
	// required
	Auth       *Execution_Auth            `protobuf:"bytes,1,opt,name=auth" json:"auth,omitempty"`
	JsonResult string                     `protobuf:"bytes,2,opt,name=json_result,json=jsonResult" json:"json_result,omitempty"`
	Expiration *google_protobuf.Timestamp `protobuf:"bytes,3,opt,name=expiration" json:"expiration,omitempty"`
}

func (m *FinishAttemptReq) Reset()                    { *m = FinishAttemptReq{} }
func (m *FinishAttemptReq) String() string            { return proto.CompactTextString(m) }
func (*FinishAttemptReq) ProtoMessage()               {}
func (*FinishAttemptReq) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }

func (m *FinishAttemptReq) GetAuth() *Execution_Auth {
	if m != nil {
		return m.Auth
	}
	return nil
}

func (m *FinishAttemptReq) GetExpiration() *google_protobuf.Timestamp {
	if m != nil {
		return m.Expiration
	}
	return nil
}

func init() {
	proto.RegisterType((*FinishAttemptReq)(nil), "dm.FinishAttemptReq")
}

var fileDescriptor3 = []byte{
	// 199 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x4c, 0x8e, 0xbf, 0x8a, 0x83, 0x30,
	0x1c, 0xc7, 0xd1, 0x3b, 0x0e, 0x2e, 0x2e, 0x12, 0x6e, 0x10, 0x17, 0x8f, 0x0e, 0xa5, 0x53, 0x84,
	0x76, 0xeb, 0xe6, 0xd0, 0x3e, 0x40, 0xe8, 0x2e, 0xb1, 0x46, 0x4d, 0x31, 0x26, 0x4d, 0x7e, 0x01,
	0xdf, 0xa4, 0xaf, 0x5b, 0x4d, 0x10, 0xba, 0x7e, 0xbe, 0x7f, 0xf8, 0xa0, 0xbf, 0x4e, 0x4c, 0xc2,
	0x0e, 0x35, 0x03, 0xe0, 0x52, 0x03, 0xd1, 0x46, 0x81, 0xc2, 0x71, 0x2b, 0xf3, 0xa2, 0x57, 0xaa,
	0x1f, 0x79, 0xe9, 0x49, 0xe3, 0xba, 0x12, 0x84, 0xe4, 0x16, 0x98, 0xd4, 0xa1, 0x94, 0xa7, 0xbd,
	0x61, 0x7a, 0xa8, 0x5b, 0x06, 0x2c, 0x90, 0xdd, 0x2b, 0x42, 0xe9, 0xd5, 0xff, 0x55, 0xe1, 0x8e,
	0xf2, 0x27, 0xde, 0xa3, 0x6f, 0xe6, 0x60, 0xc8, 0xa2, 0xff, 0xe8, 0x90, 0x1c, 0x31, 0x69, 0x25,
	0xb9, 0xcc, 0xfc, 0xee, 0x40, 0xa8, 0x89, 0x54, 0x4b, 0x42, 0x7d, 0x8e, 0x0b, 0x94, 0x3c, 0xac,
	0x9a, 0x6a, 0xc3, 0xad, 0x1b, 0x21, 0x8b, 0x97, 0xfa, 0x2f, 0x45, 0x2b, 0xa2, 0x9e, 0xe0, 0x33,
	0x42, 0x7c, 0xd6, 0xc2, 0xb0, 0x75, 0x99, 0x7d, 0xf9, 0xbb, 0x9c, 0x04, 0x4b, 0xb2, 0x59, 0x92,
	0xdb, 0x66, 0x49, 0x3f, 0xda, 0xcd, 0x8f, 0xcf, 0x4f, 0xef, 0x00, 0x00, 0x00, 0xff, 0xff, 0x3f,
	0x90, 0xe9, 0x84, 0xef, 0x00, 0x00, 0x00,
}
