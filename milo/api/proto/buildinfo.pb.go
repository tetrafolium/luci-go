// Code generated by protoc-gen-go.
// source: github.com/luci/luci-go/milo/api/proto/buildinfo.proto
// DO NOT EDIT!

package milo

import prpc "github.com/luci/luci-go/grpc/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import milo1 "github.com/luci/luci-go/common/proto/milo"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type BuildInfoRequest struct {
	// Types that are valid to be assigned to Build:
	//	*BuildInfoRequest_Buildbot
	//	*BuildInfoRequest_Swarming_
	Build isBuildInfoRequest_Build `protobuf_oneof:"build"`
	// Project hint is a LUCI project suggestion for this build. Some builds,
	// notably older ones, may not contain enough metadata to resolve their
	// project. Resolution may succeed if this hint is provided and correct.
	//
	// This field is optional, and its use is discouraged unless necessary.
	ProjectHint string `protobuf:"bytes,11,opt,name=project_hint,json=projectHint" json:"project_hint,omitempty"`
}

func (m *BuildInfoRequest) Reset()                    { *m = BuildInfoRequest{} }
func (m *BuildInfoRequest) String() string            { return proto.CompactTextString(m) }
func (*BuildInfoRequest) ProtoMessage()               {}
func (*BuildInfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

type isBuildInfoRequest_Build interface {
	isBuildInfoRequest_Build()
}

type BuildInfoRequest_Buildbot struct {
	Buildbot *BuildInfoRequest_BuildBot `protobuf:"bytes,1,opt,name=buildbot,oneof"`
}
type BuildInfoRequest_Swarming_ struct {
	Swarming *BuildInfoRequest_Swarming `protobuf:"bytes,2,opt,name=swarming,oneof"`
}

func (*BuildInfoRequest_Buildbot) isBuildInfoRequest_Build()  {}
func (*BuildInfoRequest_Swarming_) isBuildInfoRequest_Build() {}

func (m *BuildInfoRequest) GetBuild() isBuildInfoRequest_Build {
	if m != nil {
		return m.Build
	}
	return nil
}

func (m *BuildInfoRequest) GetBuildbot() *BuildInfoRequest_BuildBot {
	if x, ok := m.GetBuild().(*BuildInfoRequest_Buildbot); ok {
		return x.Buildbot
	}
	return nil
}

func (m *BuildInfoRequest) GetSwarming() *BuildInfoRequest_Swarming {
	if x, ok := m.GetBuild().(*BuildInfoRequest_Swarming_); ok {
		return x.Swarming
	}
	return nil
}

func (m *BuildInfoRequest) GetProjectHint() string {
	if m != nil {
		return m.ProjectHint
	}
	return ""
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*BuildInfoRequest) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _BuildInfoRequest_OneofMarshaler, _BuildInfoRequest_OneofUnmarshaler, _BuildInfoRequest_OneofSizer, []interface{}{
		(*BuildInfoRequest_Buildbot)(nil),
		(*BuildInfoRequest_Swarming_)(nil),
	}
}

func _BuildInfoRequest_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*BuildInfoRequest)
	// build
	switch x := m.Build.(type) {
	case *BuildInfoRequest_Buildbot:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Buildbot); err != nil {
			return err
		}
	case *BuildInfoRequest_Swarming_:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Swarming); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("BuildInfoRequest.Build has unexpected type %T", x)
	}
	return nil
}

func _BuildInfoRequest_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*BuildInfoRequest)
	switch tag {
	case 1: // build.buildbot
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(BuildInfoRequest_BuildBot)
		err := b.DecodeMessage(msg)
		m.Build = &BuildInfoRequest_Buildbot{msg}
		return true, err
	case 2: // build.swarming
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(BuildInfoRequest_Swarming)
		err := b.DecodeMessage(msg)
		m.Build = &BuildInfoRequest_Swarming_{msg}
		return true, err
	default:
		return false, nil
	}
}

func _BuildInfoRequest_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*BuildInfoRequest)
	// build
	switch x := m.Build.(type) {
	case *BuildInfoRequest_Buildbot:
		s := proto.Size(x.Buildbot)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *BuildInfoRequest_Swarming_:
		s := proto.Size(x.Swarming)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// The request for the name of a BuildBot built.
type BuildInfoRequest_BuildBot struct {
	// The master name.
	MasterName string `protobuf:"bytes,1,opt,name=master_name,json=masterName" json:"master_name,omitempty"`
	// The builder name server.
	BuilderName string `protobuf:"bytes,2,opt,name=builder_name,json=builderName" json:"builder_name,omitempty"`
	// The build number.
	BuildNumber int64 `protobuf:"varint,3,opt,name=build_number,json=buildNumber" json:"build_number,omitempty"`
}

func (m *BuildInfoRequest_BuildBot) Reset()                    { *m = BuildInfoRequest_BuildBot{} }
func (m *BuildInfoRequest_BuildBot) String() string            { return proto.CompactTextString(m) }
func (*BuildInfoRequest_BuildBot) ProtoMessage()               {}
func (*BuildInfoRequest_BuildBot) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0, 0} }

func (m *BuildInfoRequest_BuildBot) GetMasterName() string {
	if m != nil {
		return m.MasterName
	}
	return ""
}

func (m *BuildInfoRequest_BuildBot) GetBuilderName() string {
	if m != nil {
		return m.BuilderName
	}
	return ""
}

func (m *BuildInfoRequest_BuildBot) GetBuildNumber() int64 {
	if m != nil {
		return m.BuildNumber
	}
	return 0
}

// The request containing a Swarming task.
type BuildInfoRequest_Swarming struct {
	// Host is the hostname of the Swarming server to connect to
	// (e.g., "swarming.example.com").
	//
	// This is optional. If omitted or empty, Milo's default Swarming server
	// will be used.
	Host string `protobuf:"bytes,1,opt,name=host" json:"host,omitempty"`
	// The Swarming task name.
	Task string `protobuf:"bytes,2,opt,name=task" json:"task,omitempty"`
}

func (m *BuildInfoRequest_Swarming) Reset()                    { *m = BuildInfoRequest_Swarming{} }
func (m *BuildInfoRequest_Swarming) String() string            { return proto.CompactTextString(m) }
func (*BuildInfoRequest_Swarming) ProtoMessage()               {}
func (*BuildInfoRequest_Swarming) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0, 1} }

func (m *BuildInfoRequest_Swarming) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *BuildInfoRequest_Swarming) GetTask() string {
	if m != nil {
		return m.Task
	}
	return ""
}

// The request containing the name of the master.
type BuildInfoResponse struct {
	// The LUCI project that this build belongs to.
	Project string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	// The main build step.
	Step *milo1.Step `protobuf:"bytes,2,opt,name=step" json:"step,omitempty"`
	// The LogDog annotation stream for this build. The Prefix will be populated
	// and can be used as the prefix for any un-prefixed LogdogStream in "step".
	AnnotationStream *milo1.LogdogStream `protobuf:"bytes,3,opt,name=annotation_stream,json=annotationStream" json:"annotation_stream,omitempty"`
}

func (m *BuildInfoResponse) Reset()                    { *m = BuildInfoResponse{} }
func (m *BuildInfoResponse) String() string            { return proto.CompactTextString(m) }
func (*BuildInfoResponse) ProtoMessage()               {}
func (*BuildInfoResponse) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *BuildInfoResponse) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *BuildInfoResponse) GetStep() *milo1.Step {
	if m != nil {
		return m.Step
	}
	return nil
}

func (m *BuildInfoResponse) GetAnnotationStream() *milo1.LogdogStream {
	if m != nil {
		return m.AnnotationStream
	}
	return nil
}

func init() {
	proto.RegisterType((*BuildInfoRequest)(nil), "milo.BuildInfoRequest")
	proto.RegisterType((*BuildInfoRequest_BuildBot)(nil), "milo.BuildInfoRequest.BuildBot")
	proto.RegisterType((*BuildInfoRequest_Swarming)(nil), "milo.BuildInfoRequest.Swarming")
	proto.RegisterType((*BuildInfoResponse)(nil), "milo.BuildInfoResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for BuildInfo service

type BuildInfoClient interface {
	Get(ctx context.Context, in *BuildInfoRequest, opts ...grpc.CallOption) (*BuildInfoResponse, error)
}
type buildInfoPRPCClient struct {
	client *prpc.Client
}

func NewBuildInfoPRPCClient(client *prpc.Client) BuildInfoClient {
	return &buildInfoPRPCClient{client}
}

func (c *buildInfoPRPCClient) Get(ctx context.Context, in *BuildInfoRequest, opts ...grpc.CallOption) (*BuildInfoResponse, error) {
	out := new(BuildInfoResponse)
	err := c.client.Call(ctx, "milo.BuildInfo", "Get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type buildInfoClient struct {
	cc *grpc.ClientConn
}

func NewBuildInfoClient(cc *grpc.ClientConn) BuildInfoClient {
	return &buildInfoClient{cc}
}

func (c *buildInfoClient) Get(ctx context.Context, in *BuildInfoRequest, opts ...grpc.CallOption) (*BuildInfoResponse, error) {
	out := new(BuildInfoResponse)
	err := grpc.Invoke(ctx, "/milo.BuildInfo/Get", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for BuildInfo service

type BuildInfoServer interface {
	Get(context.Context, *BuildInfoRequest) (*BuildInfoResponse, error)
}

func RegisterBuildInfoServer(s prpc.Registrar, srv BuildInfoServer) {
	s.RegisterService(&_BuildInfo_serviceDesc, srv)
}

func _BuildInfo_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BuildInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildInfoServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/milo.BuildInfo/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildInfoServer).Get(ctx, req.(*BuildInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _BuildInfo_serviceDesc = grpc.ServiceDesc{
	ServiceName: "milo.BuildInfo",
	HandlerType: (*BuildInfoServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Get",
			Handler:    _BuildInfo_Get_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/luci/luci-go/milo/api/proto/buildinfo.proto",
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/milo/api/proto/buildinfo.proto", fileDescriptor1)
}

var fileDescriptor1 = []byte{
	// 383 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x7c, 0x92, 0xbf, 0xae, 0xd3, 0x30,
	0x18, 0xc5, 0x49, 0x5b, 0xb8, 0xed, 0x17, 0x86, 0x7b, 0x3d, 0x40, 0x94, 0x81, 0x96, 0x4e, 0x5d,
	0x48, 0xa4, 0x20, 0x75, 0x41, 0x08, 0xa9, 0x0c, 0x14, 0x09, 0x75, 0x48, 0x1f, 0x20, 0x72, 0x52,
	0x37, 0x35, 0xd4, 0xfe, 0xd2, 0xf8, 0x8b, 0x78, 0x0b, 0x5e, 0x92, 0x17, 0x41, 0xb1, 0x9d, 0x96,
	0xff, 0x4b, 0x64, 0xff, 0x7c, 0xce, 0xf1, 0xb1, 0x1d, 0x58, 0xd7, 0x92, 0x4e, 0x5d, 0x99, 0x54,
	0xa8, 0xd2, 0x73, 0x57, 0x49, 0xfb, 0x79, 0x55, 0x63, 0xaa, 0xe4, 0x19, 0x53, 0xde, 0xc8, 0xb4,
	0x69, 0x91, 0x30, 0x2d, 0x3b, 0x79, 0x3e, 0x48, 0x7d, 0xc4, 0xc4, 0xce, 0xd9, 0xa4, 0x5f, 0x8f,
	0xdf, 0xfc, 0xcb, 0x5d, 0xa1, 0x52, 0xa8, 0xbd, 0xd7, 0x45, 0x69, 0x8d, 0xc4, 0x49, 0xa2, 0x36,
	0x2e, 0x62, 0xf9, 0x7d, 0x04, 0xf7, 0x9b, 0x3e, 0xf6, 0xa3, 0x3e, 0x62, 0x2e, 0x2e, 0x9d, 0x30,
	0xc4, 0xde, 0xc2, 0xd4, 0x6e, 0x55, 0x22, 0x45, 0xc1, 0x22, 0x58, 0x85, 0xd9, 0x3c, 0xe9, 0xfd,
	0xc9, 0xef, 0x4a, 0x07, 0x36, 0x48, 0xdb, 0x47, 0xf9, 0xd5, 0xd2, 0xdb, 0xcd, 0x57, 0xde, 0x2a,
	0xa9, 0xeb, 0x68, 0xf4, 0x5f, 0xfb, 0xde, 0xcb, 0x7a, 0xfb, 0x60, 0x61, 0x2f, 0xe1, 0x69, 0xd3,
	0xe2, 0x67, 0x51, 0x51, 0x71, 0x92, 0x9a, 0xa2, 0x70, 0x11, 0xac, 0x66, 0x79, 0xe8, 0xd9, 0x56,
	0x6a, 0x8a, 0x2f, 0x30, 0x1d, 0x76, 0x66, 0x73, 0x08, 0x15, 0x37, 0x24, 0xda, 0x42, 0x73, 0x25,
	0x6c, 0xdf, 0x59, 0x0e, 0x0e, 0xed, 0xb8, 0x12, 0x7d, 0x9e, 0xad, 0x36, 0x28, 0x46, 0x2e, 0xcf,
	0xb3, 0x5f, 0x24, 0x85, 0xee, 0x54, 0x29, 0xda, 0x68, 0xbc, 0x08, 0x56, 0x63, 0x2f, 0xd9, 0x59,
	0x14, 0x67, 0x30, 0x1d, 0xda, 0x32, 0x06, 0x93, 0x13, 0x1a, 0xf2, 0x7b, 0xd9, 0x71, 0xcf, 0x88,
	0x9b, 0x2f, 0x3e, 0xdd, 0x8e, 0x37, 0x77, 0xf0, 0xd8, 0x46, 0x2c, 0xbf, 0x05, 0xf0, 0xf0, 0xd3,
	0xe1, 0x4d, 0x83, 0xda, 0x08, 0x16, 0xc1, 0x9d, 0x3f, 0x94, 0x4f, 0x1a, 0xa6, 0xec, 0x05, 0x4c,
	0x0c, 0x89, 0xc6, 0xdf, 0x1e, 0xb8, 0xdb, 0xdb, 0x93, 0x68, 0x72, 0xcb, 0xd9, 0x3b, 0x78, 0xb8,
	0x3d, 0x65, 0x61, 0xa8, 0x15, 0x5c, 0xd9, 0xd2, 0x61, 0xc6, 0x9c, 0xf8, 0x13, 0xd6, 0x07, 0xac,
	0xf7, 0x76, 0x25, 0xbf, 0xbf, 0x89, 0x1d, 0xc9, 0xde, 0xc3, 0xec, 0xda, 0x87, 0xad, 0x61, 0xfc,
	0x41, 0x10, 0x7b, 0xf6, 0xf7, 0x47, 0x8a, 0x9f, 0xff, 0xc1, 0x5d, 0xff, 0xf2, 0x89, 0xfd, 0x85,
	0x5e, 0xff, 0x08, 0x00, 0x00, 0xff, 0xff, 0xb0, 0x29, 0xcb, 0x8a, 0xbf, 0x02, 0x00, 0x00,
}
