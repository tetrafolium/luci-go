// Code generated by protoc-gen-go.
// source: github.com/luci/luci-go/common/proto/google/empty.proto
// DO NOT EDIT!

package google

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// A generic empty message that you can re-use to avoid defining duplicated
// empty messages in your APIs. A typical example is to use it as the request
// or the response type of an API method. For instance:
//
//     service Foo {
//       rpc Bar(google.protobuf.Empty) returns (google.protobuf.Empty);
//     }
//
// The JSON representation for `Empty` is empty JSON object `{}`.
type Empty struct {
}

func (m *Empty) Reset()                    { *m = Empty{} }
func (m *Empty) String() string            { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()               {}
func (*Empty) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }
func (*Empty) XXX_WellKnownType() string   { return "Empty" }

func init() {
	proto.RegisterType((*Empty)(nil), "google.protobuf.Empty")
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/common/proto/google/empty.proto", fileDescriptor1)
}

var fileDescriptor1 = []byte{
	// 150 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0x32, 0x4f, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0xcf, 0x29, 0x4d, 0xce, 0x04, 0x13, 0xba, 0xe9, 0xf9,
	0xfa, 0xc9, 0xf9, 0xb9, 0xb9, 0xf9, 0x79, 0xfa, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0xfa, 0xe9, 0xf9,
	0xf9, 0xe9, 0x39, 0xa9, 0xfa, 0xa9, 0xb9, 0x05, 0x25, 0x95, 0x7a, 0x60, 0x21, 0x21, 0x7e, 0x88,
	0x18, 0x84, 0x97, 0x54, 0x9a, 0xa6, 0xc4, 0xce, 0xc5, 0xea, 0x0a, 0x92, 0x77, 0x0a, 0xe0, 0x12,
	0x4e, 0xce, 0xcf, 0xd5, 0x43, 0x93, 0x77, 0xe2, 0x02, 0xcb, 0x06, 0x80, 0xb8, 0x01, 0x8c, 0x0b,
	0x18, 0x19, 0x7f, 0x30, 0x32, 0x2e, 0x62, 0x62, 0x76, 0x0f, 0x70, 0x5a, 0xc5, 0x24, 0xe7, 0x0e,
	0x51, 0x1b, 0x00, 0x55, 0xab, 0x17, 0x9e, 0x9a, 0x93, 0xe3, 0x9d, 0x97, 0x5f, 0x9e, 0x17, 0x52,
	0x59, 0x90, 0x5a, 0x9c, 0xc4, 0x06, 0x36, 0xc4, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x3a, 0x58,
	0x5e, 0xfd, 0xad, 0x00, 0x00, 0x00,
}
