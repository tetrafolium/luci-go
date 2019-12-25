// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/resultdb/proto/rpc/v1/predicate.proto

package rpcpb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_type "go.chromium.org/luci/resultdb/proto/type"
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

// Filters test results based on TestResult.expected field.
type TestResultPredicate_Expectancy int32

const (
	// All test results satisfiy this.
	// WARNING: using this significantly increases response size and latency.
	TestResultPredicate_ALL TestResultPredicate_Expectancy = 0
	// A test result must belong to a test variant that has one or more
	// unexpected results. It can be used to fetch both unexpected and flakily
	// expected results.
	//
	// Note that the predicate is defined at the test variant level.
	// For example, if a test variant expects a PASS and has results
	// [FAIL, FAIL, PASS], then all results satisfy the predicate because
	// the variant satisfies the predicate.
	TestResultPredicate_VARIANTS_WITH_UNEXPECTED_RESULTS TestResultPredicate_Expectancy = 1
)

var TestResultPredicate_Expectancy_name = map[int32]string{
	0: "ALL",
	1: "VARIANTS_WITH_UNEXPECTED_RESULTS",
}

var TestResultPredicate_Expectancy_value = map[string]int32{
	"ALL":                              0,
	"VARIANTS_WITH_UNEXPECTED_RESULTS": 1,
}

func (x TestResultPredicate_Expectancy) String() string {
	return proto.EnumName(TestResultPredicate_Expectancy_name, int32(x))
}

func (TestResultPredicate_Expectancy) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_c5e4555b96213a1b, []int{0, 0}
}

// Represents a function TestResult -> bool.
// Empty message matches all test results.
//
// Most clients would want to set expected_results to
// VARIANTS_WITH_UNEXPECTED_RESULTS.
type TestResultPredicate struct {
	// A test result must have a test id matching this regular expression
	// entirely, i.e. the expression is implicitly wrapped with ^ and $.
	TestIdRegexp string `protobuf:"bytes,1,opt,name=test_id_regexp,json=testIdRegexp,proto3" json:"test_id_regexp,omitempty"`
	// A test result must have a variant satisfying this predicate.
	Variant *VariantPredicate `protobuf:"bytes,2,opt,name=variant,proto3" json:"variant,omitempty"`
	// A test result must match this predicate based on TestResult.expected field.
	// Most clients would want to override this field because the default
	// typically causes a large response size.
	Expectancy           TestResultPredicate_Expectancy `protobuf:"varint,3,opt,name=expectancy,proto3,enum=luci.resultdb.rpc.v1.TestResultPredicate_Expectancy" json:"expectancy,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *TestResultPredicate) Reset()         { *m = TestResultPredicate{} }
func (m *TestResultPredicate) String() string { return proto.CompactTextString(m) }
func (*TestResultPredicate) ProtoMessage()    {}
func (*TestResultPredicate) Descriptor() ([]byte, []int) {
	return fileDescriptor_c5e4555b96213a1b, []int{0}
}

func (m *TestResultPredicate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TestResultPredicate.Unmarshal(m, b)
}
func (m *TestResultPredicate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TestResultPredicate.Marshal(b, m, deterministic)
}
func (m *TestResultPredicate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TestResultPredicate.Merge(m, src)
}
func (m *TestResultPredicate) XXX_Size() int {
	return xxx_messageInfo_TestResultPredicate.Size(m)
}
func (m *TestResultPredicate) XXX_DiscardUnknown() {
	xxx_messageInfo_TestResultPredicate.DiscardUnknown(m)
}

var xxx_messageInfo_TestResultPredicate proto.InternalMessageInfo

func (m *TestResultPredicate) GetTestIdRegexp() string {
	if m != nil {
		return m.TestIdRegexp
	}
	return ""
}

func (m *TestResultPredicate) GetVariant() *VariantPredicate {
	if m != nil {
		return m.Variant
	}
	return nil
}

func (m *TestResultPredicate) GetExpectancy() TestResultPredicate_Expectancy {
	if m != nil {
		return m.Expectancy
	}
	return TestResultPredicate_ALL
}

// Represents a function TestExoneration -> bool.
// Empty message matches all test exonerations.
type TestExonerationPredicate struct {
	// A test exoneration must have a test id matching this regular expression
	// entirely, i.e. the expression is implicitly wrapped with ^ and $.
	TestIdRegexp string `protobuf:"bytes,1,opt,name=test_id_regexp,json=testIdRegexp,proto3" json:"test_id_regexp,omitempty"`
	// A test exoneration must have a variant satisfying this predicate.
	Variant              *VariantPredicate `protobuf:"bytes,2,opt,name=variant,proto3" json:"variant,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *TestExonerationPredicate) Reset()         { *m = TestExonerationPredicate{} }
func (m *TestExonerationPredicate) String() string { return proto.CompactTextString(m) }
func (*TestExonerationPredicate) ProtoMessage()    {}
func (*TestExonerationPredicate) Descriptor() ([]byte, []int) {
	return fileDescriptor_c5e4555b96213a1b, []int{1}
}

func (m *TestExonerationPredicate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TestExonerationPredicate.Unmarshal(m, b)
}
func (m *TestExonerationPredicate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TestExonerationPredicate.Marshal(b, m, deterministic)
}
func (m *TestExonerationPredicate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TestExonerationPredicate.Merge(m, src)
}
func (m *TestExonerationPredicate) XXX_Size() int {
	return xxx_messageInfo_TestExonerationPredicate.Size(m)
}
func (m *TestExonerationPredicate) XXX_DiscardUnknown() {
	xxx_messageInfo_TestExonerationPredicate.DiscardUnknown(m)
}

var xxx_messageInfo_TestExonerationPredicate proto.InternalMessageInfo

func (m *TestExonerationPredicate) GetTestIdRegexp() string {
	if m != nil {
		return m.TestIdRegexp
	}
	return ""
}

func (m *TestExonerationPredicate) GetVariant() *VariantPredicate {
	if m != nil {
		return m.Variant
	}
	return nil
}

// Represents a function Variant -> bool.
type VariantPredicate struct {
	// Types that are valid to be assigned to Predicate:
	//	*VariantPredicate_Exact
	//	*VariantPredicate_Contains
	Predicate            isVariantPredicate_Predicate `protobuf_oneof:"predicate"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *VariantPredicate) Reset()         { *m = VariantPredicate{} }
func (m *VariantPredicate) String() string { return proto.CompactTextString(m) }
func (*VariantPredicate) ProtoMessage()    {}
func (*VariantPredicate) Descriptor() ([]byte, []int) {
	return fileDescriptor_c5e4555b96213a1b, []int{2}
}

func (m *VariantPredicate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_VariantPredicate.Unmarshal(m, b)
}
func (m *VariantPredicate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_VariantPredicate.Marshal(b, m, deterministic)
}
func (m *VariantPredicate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VariantPredicate.Merge(m, src)
}
func (m *VariantPredicate) XXX_Size() int {
	return xxx_messageInfo_VariantPredicate.Size(m)
}
func (m *VariantPredicate) XXX_DiscardUnknown() {
	xxx_messageInfo_VariantPredicate.DiscardUnknown(m)
}

var xxx_messageInfo_VariantPredicate proto.InternalMessageInfo

type isVariantPredicate_Predicate interface {
	isVariantPredicate_Predicate()
}

type VariantPredicate_Exact struct {
	Exact *_type.Variant `protobuf:"bytes,1,opt,name=exact,proto3,oneof"`
}

type VariantPredicate_Contains struct {
	Contains *_type.Variant `protobuf:"bytes,2,opt,name=contains,proto3,oneof"`
}

func (*VariantPredicate_Exact) isVariantPredicate_Predicate() {}

func (*VariantPredicate_Contains) isVariantPredicate_Predicate() {}

func (m *VariantPredicate) GetPredicate() isVariantPredicate_Predicate {
	if m != nil {
		return m.Predicate
	}
	return nil
}

func (m *VariantPredicate) GetExact() *_type.Variant {
	if x, ok := m.GetPredicate().(*VariantPredicate_Exact); ok {
		return x.Exact
	}
	return nil
}

func (m *VariantPredicate) GetContains() *_type.Variant {
	if x, ok := m.GetPredicate().(*VariantPredicate_Contains); ok {
		return x.Contains
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*VariantPredicate) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*VariantPredicate_Exact)(nil),
		(*VariantPredicate_Contains)(nil),
	}
}

func init() {
	proto.RegisterEnum("luci.resultdb.rpc.v1.TestResultPredicate_Expectancy", TestResultPredicate_Expectancy_name, TestResultPredicate_Expectancy_value)
	proto.RegisterType((*TestResultPredicate)(nil), "luci.resultdb.rpc.v1.TestResultPredicate")
	proto.RegisterType((*TestExonerationPredicate)(nil), "luci.resultdb.rpc.v1.TestExonerationPredicate")
	proto.RegisterType((*VariantPredicate)(nil), "luci.resultdb.rpc.v1.VariantPredicate")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/resultdb/proto/rpc/v1/predicate.proto", fileDescriptor_c5e4555b96213a1b)
}

var fileDescriptor_c5e4555b96213a1b = []byte{
	// 388 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x92, 0xc1, 0x8b, 0xd3, 0x40,
	0x18, 0xc5, 0x37, 0xbb, 0xe8, 0xba, 0x53, 0x59, 0xca, 0xe8, 0x21, 0xac, 0x07, 0x43, 0x58, 0xa4,
	0xa7, 0x19, 0x37, 0xab, 0x07, 0xdd, 0x8b, 0xad, 0x06, 0x5a, 0x28, 0xa5, 0x4c, 0xd3, 0x2a, 0x5e,
	0xc2, 0x64, 0x32, 0xa6, 0x03, 0x49, 0x66, 0x98, 0x4c, 0x43, 0x7a, 0xf5, 0x1f, 0xd0, 0x3f, 0x59,
	0x92, 0x98, 0x28, 0xa5, 0x60, 0x4f, 0x5e, 0xdf, 0xf7, 0x7b, 0xef, 0x7b, 0xcc, 0x7c, 0xe0, 0x7d,
	0x22, 0x11, 0xdb, 0x6a, 0x99, 0x89, 0x5d, 0x86, 0xa4, 0x4e, 0x70, 0xba, 0x63, 0x02, 0x6b, 0x5e,
	0xec, 0x52, 0x13, 0x47, 0x58, 0x69, 0x69, 0x24, 0xd6, 0x8a, 0xe1, 0xf2, 0x0e, 0x2b, 0xcd, 0x63,
	0xc1, 0xa8, 0xe1, 0xa8, 0x91, 0xe1, 0xf3, 0x9a, 0x45, 0x1d, 0x8b, 0xb4, 0x62, 0xa8, 0xbc, 0xbb,
	0x79, 0x99, 0x48, 0x99, 0xa4, 0x1c, 0x53, 0x25, 0xf0, 0x37, 0xc1, 0xd3, 0x38, 0x8c, 0xf8, 0x96,
	0x96, 0x42, 0xea, 0xd6, 0x76, 0xf3, 0xf6, 0x94, 0x95, 0x66, 0xaf, 0x38, 0x66, 0x32, 0xcb, 0x64,
	0xde, 0xda, 0xdc, 0x9f, 0xe7, 0xe0, 0x59, 0xc0, 0x0b, 0x43, 0x1a, 0x70, 0xd9, 0x75, 0x81, 0xb7,
	0xe0, 0xda, 0xf0, 0xc2, 0x84, 0x22, 0x0e, 0x35, 0x4f, 0x78, 0xa5, 0x6c, 0xcb, 0xb1, 0x46, 0x57,
	0xe4, 0x69, 0xad, 0xce, 0x62, 0xd2, 0x68, 0xf0, 0x03, 0xb8, 0x2c, 0xa9, 0x16, 0x34, 0x37, 0xf6,
	0xb9, 0x63, 0x8d, 0x06, 0xde, 0x2b, 0x74, 0xac, 0x3d, 0xda, 0xb4, 0x50, 0x1f, 0x4f, 0x3a, 0x1b,
	0x0c, 0x00, 0xe0, 0x95, 0xe2, 0xcc, 0xd0, 0x9c, 0xed, 0xed, 0x0b, 0xc7, 0x1a, 0x5d, 0x7b, 0x6f,
	0x8e, 0x87, 0x1c, 0xa9, 0x89, 0xfc, 0xde, 0x4b, 0xfe, 0xca, 0x71, 0x1f, 0x00, 0xf8, 0x33, 0x81,
	0x97, 0xe0, 0x62, 0x3c, 0x9f, 0x0f, 0xcf, 0xe0, 0x2d, 0x70, 0x36, 0x63, 0x32, 0x1b, 0x2f, 0x82,
	0x55, 0xf8, 0x79, 0x16, 0x4c, 0xc3, 0xf5, 0xc2, 0xff, 0xb2, 0xf4, 0x3f, 0x06, 0xfe, 0xa7, 0x90,
	0xf8, 0xab, 0xf5, 0x3c, 0x58, 0x0d, 0x2d, 0xf7, 0xbb, 0x05, 0xec, 0x7a, 0x97, 0x5f, 0xc9, 0x9c,
	0x6b, 0x6a, 0x84, 0xcc, 0xff, 0xfb, 0xbb, 0xb8, 0x3f, 0x2c, 0x30, 0x3c, 0x9c, 0xc2, 0x7b, 0xf0,
	0x88, 0x57, 0x94, 0x99, 0x66, 0xe7, 0xc0, 0x7b, 0x71, 0x10, 0x5a, 0xff, 0x6e, 0x17, 0x39, 0x3d,
	0x23, 0x2d, 0x0b, 0xdf, 0x81, 0x27, 0x4c, 0xe6, 0x86, 0x8a, 0xbc, 0xf8, 0x5d, 0xe6, 0x1f, 0xbe,
	0x1e, 0x9f, 0x0c, 0xc0, 0x55, 0x7f, 0x9d, 0x13, 0xef, 0xeb, 0xeb, 0xd3, 0xaf, 0xfa, 0x41, 0x2b,
	0xa6, 0xa2, 0xe8, 0x71, 0xa3, 0xdd, 0xff, 0x0a, 0x00, 0x00, 0xff, 0xff, 0x4d, 0x62, 0x66, 0x83,
	0x10, 0x03, 0x00, 0x00,
}
