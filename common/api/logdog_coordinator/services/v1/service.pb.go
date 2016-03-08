// Code generated by protoc-gen-go.
// source: service.proto
// DO NOT EDIT!

/*
Package logdog is a generated protocol buffer package.

It is generated from these files:
	service.proto
	state.proto

It has these top-level messages:
	GetConfigResponse
	RegisterStreamRequest
	RegisterStreamResponse
	LoadStreamRequest
	LoadStreamResponse
	TerminateStreamRequest
	LogStreamState
*/
package logdog

import prpccommon "github.com/luci/luci-go/common/prpc"
import prpc "github.com/luci/luci-go/server/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import logpb "github.com/luci/luci-go/common/proto/logdog/logpb"
import google_protobuf2 "github.com/luci/luci-go/common/proto/google"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// GetConfigResponse is the response structure for the user
// "GetConfig" endpoint.
type GetConfigResponse struct {
	// The API URL of the base "luci-config" service. If empty, the default
	// service URL will be used.
	ConfigServiceUrl string `protobuf:"bytes,1,opt,name=config_service_url" json:"config_service_url,omitempty"`
	// The name of the configuration set to load from.
	ConfigSet string `protobuf:"bytes,2,opt,name=config_set" json:"config_set,omitempty"`
	// The path of the text-serialized configuration protobuf.
	ConfigPath string `protobuf:"bytes,3,opt,name=config_path" json:"config_path,omitempty"`
}

func (m *GetConfigResponse) Reset()                    { *m = GetConfigResponse{} }
func (m *GetConfigResponse) String() string            { return proto.CompactTextString(m) }
func (*GetConfigResponse) ProtoMessage()               {}
func (*GetConfigResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// RegisterStreamRequest is the set of caller-supplied data for the
// RegisterStream Coordinator service endpoint.
type RegisterStreamRequest struct {
	// The log stream's path.
	Path string `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	// The log stream's secret.
	Secret []byte `protobuf:"bytes,2,opt,name=secret,proto3" json:"secret,omitempty"`
	// The protobuf version string for this stream.
	ProtoVersion string `protobuf:"bytes,3,opt,name=proto_version" json:"proto_version,omitempty"`
	// The serialized LogStreamDescriptor protobuf for this stream.
	Desc *logpb.LogStreamDescriptor `protobuf:"bytes,4,opt,name=desc" json:"desc,omitempty"`
}

func (m *RegisterStreamRequest) Reset()                    { *m = RegisterStreamRequest{} }
func (m *RegisterStreamRequest) String() string            { return proto.CompactTextString(m) }
func (*RegisterStreamRequest) ProtoMessage()               {}
func (*RegisterStreamRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *RegisterStreamRequest) GetDesc() *logpb.LogStreamDescriptor {
	if m != nil {
		return m.Desc
	}
	return nil
}

// The response message for the RegisterStream RPC.
type RegisterStreamResponse struct {
	// The state of the requested log stream.
	State *LogStreamState `protobuf:"bytes,1,opt,name=state" json:"state,omitempty"`
	// The log stream's secret.
	//
	// Note that the secret is returned! This is okay, since this endpoint is only
	// accessible to trusted services. The secret can be cached by services to
	// validate stream information without needing to ping the Coordinator in
	// between each update.
	Secret []byte `protobuf:"bytes,2,opt,name=secret,proto3" json:"secret,omitempty"`
}

func (m *RegisterStreamResponse) Reset()                    { *m = RegisterStreamResponse{} }
func (m *RegisterStreamResponse) String() string            { return proto.CompactTextString(m) }
func (*RegisterStreamResponse) ProtoMessage()               {}
func (*RegisterStreamResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *RegisterStreamResponse) GetState() *LogStreamState {
	if m != nil {
		return m.State
	}
	return nil
}

// LoadStreamRequest loads the current state of a log stream.
type LoadStreamRequest struct {
	// The log stream's path.
	Path string `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	// If true, include the log stream descriptor.
	Desc bool `protobuf:"varint,2,opt,name=desc" json:"desc,omitempty"`
}

func (m *LoadStreamRequest) Reset()                    { *m = LoadStreamRequest{} }
func (m *LoadStreamRequest) String() string            { return proto.CompactTextString(m) }
func (*LoadStreamRequest) ProtoMessage()               {}
func (*LoadStreamRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

// The response message for the LoadStream RPC.
type LoadStreamResponse struct {
	// The state of the requested log stream.
	State *LogStreamState `protobuf:"bytes,1,opt,name=state" json:"state,omitempty"`
	// If requested, the serialized log stream descriptor. The protobuf version
	// of this descriptor will match the "proto_version" field in "state".
	Desc []byte `protobuf:"bytes,2,opt,name=desc,proto3" json:"desc,omitempty"`
}

func (m *LoadStreamResponse) Reset()                    { *m = LoadStreamResponse{} }
func (m *LoadStreamResponse) String() string            { return proto.CompactTextString(m) }
func (*LoadStreamResponse) ProtoMessage()               {}
func (*LoadStreamResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *LoadStreamResponse) GetState() *LogStreamState {
	if m != nil {
		return m.State
	}
	return nil
}

// TerminateStreamRequest is the set of caller-supplied data for the
// TerminateStream service endpoint.
type TerminateStreamRequest struct {
	// The log stream's path.
	Path string `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	// The log stream's secret.
	Secret []byte `protobuf:"bytes,2,opt,name=secret,proto3" json:"secret,omitempty"`
	// The terminal index of the stream.
	TerminalIndex int64 `protobuf:"varint,3,opt,name=terminal_index" json:"terminal_index,omitempty"`
}

func (m *TerminateStreamRequest) Reset()                    { *m = TerminateStreamRequest{} }
func (m *TerminateStreamRequest) String() string            { return proto.CompactTextString(m) }
func (*TerminateStreamRequest) ProtoMessage()               {}
func (*TerminateStreamRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func init() {
	proto.RegisterType((*GetConfigResponse)(nil), "logdog.GetConfigResponse")
	proto.RegisterType((*RegisterStreamRequest)(nil), "logdog.RegisterStreamRequest")
	proto.RegisterType((*RegisterStreamResponse)(nil), "logdog.RegisterStreamResponse")
	proto.RegisterType((*LoadStreamRequest)(nil), "logdog.LoadStreamRequest")
	proto.RegisterType((*LoadStreamResponse)(nil), "logdog.LoadStreamResponse")
	proto.RegisterType((*TerminateStreamRequest)(nil), "logdog.TerminateStreamRequest")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// Client API for Services service

type ServicesClient interface {
	// GetConfig allows a service to retrieve the current service configuration
	// parameters.
	GetConfig(ctx context.Context, in *google_protobuf2.Empty, opts ...grpc.CallOption) (*GetConfigResponse, error)
	// RegisterStream is an idempotent stream state register operation.
	RegisterStream(ctx context.Context, in *RegisterStreamRequest, opts ...grpc.CallOption) (*RegisterStreamResponse, error)
	// LoadStream loads the current state of a log stream.
	LoadStream(ctx context.Context, in *LoadStreamRequest, opts ...grpc.CallOption) (*LoadStreamResponse, error)
	// TerminateStream is an idempotent operation to update the stream's terminal
	// index.
	TerminateStream(ctx context.Context, in *TerminateStreamRequest, opts ...grpc.CallOption) (*google_protobuf2.Empty, error)
}
type servicesPRPCClient struct {
	client *prpccommon.Client
}

func NewServicesPRPCClient(client *prpccommon.Client) ServicesClient {
	return &servicesPRPCClient{client}
}

func (c *servicesPRPCClient) GetConfig(ctx context.Context, in *google_protobuf2.Empty, opts ...grpc.CallOption) (*GetConfigResponse, error) {
	out := new(GetConfigResponse)
	err := c.client.Call(ctx, "logdog.Services", "GetConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servicesPRPCClient) RegisterStream(ctx context.Context, in *RegisterStreamRequest, opts ...grpc.CallOption) (*RegisterStreamResponse, error) {
	out := new(RegisterStreamResponse)
	err := c.client.Call(ctx, "logdog.Services", "RegisterStream", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servicesPRPCClient) LoadStream(ctx context.Context, in *LoadStreamRequest, opts ...grpc.CallOption) (*LoadStreamResponse, error) {
	out := new(LoadStreamResponse)
	err := c.client.Call(ctx, "logdog.Services", "LoadStream", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servicesPRPCClient) TerminateStream(ctx context.Context, in *TerminateStreamRequest, opts ...grpc.CallOption) (*google_protobuf2.Empty, error) {
	out := new(google_protobuf2.Empty)
	err := c.client.Call(ctx, "logdog.Services", "TerminateStream", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type servicesClient struct {
	cc *grpc.ClientConn
}

func NewServicesClient(cc *grpc.ClientConn) ServicesClient {
	return &servicesClient{cc}
}

func (c *servicesClient) GetConfig(ctx context.Context, in *google_protobuf2.Empty, opts ...grpc.CallOption) (*GetConfigResponse, error) {
	out := new(GetConfigResponse)
	err := grpc.Invoke(ctx, "/logdog.Services/GetConfig", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servicesClient) RegisterStream(ctx context.Context, in *RegisterStreamRequest, opts ...grpc.CallOption) (*RegisterStreamResponse, error) {
	out := new(RegisterStreamResponse)
	err := grpc.Invoke(ctx, "/logdog.Services/RegisterStream", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servicesClient) LoadStream(ctx context.Context, in *LoadStreamRequest, opts ...grpc.CallOption) (*LoadStreamResponse, error) {
	out := new(LoadStreamResponse)
	err := grpc.Invoke(ctx, "/logdog.Services/LoadStream", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *servicesClient) TerminateStream(ctx context.Context, in *TerminateStreamRequest, opts ...grpc.CallOption) (*google_protobuf2.Empty, error) {
	out := new(google_protobuf2.Empty)
	err := grpc.Invoke(ctx, "/logdog.Services/TerminateStream", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Services service

type ServicesServer interface {
	// GetConfig allows a service to retrieve the current service configuration
	// parameters.
	GetConfig(context.Context, *google_protobuf2.Empty) (*GetConfigResponse, error)
	// RegisterStream is an idempotent stream state register operation.
	RegisterStream(context.Context, *RegisterStreamRequest) (*RegisterStreamResponse, error)
	// LoadStream loads the current state of a log stream.
	LoadStream(context.Context, *LoadStreamRequest) (*LoadStreamResponse, error)
	// TerminateStream is an idempotent operation to update the stream's terminal
	// index.
	TerminateStream(context.Context, *TerminateStreamRequest) (*google_protobuf2.Empty, error)
}

func RegisterServicesServer(s prpc.Registrar, srv ServicesServer) {
	s.RegisterService(&_Services_serviceDesc, srv)
}

func _Services_GetConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(google_protobuf2.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(ServicesServer).GetConfig(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Services_RegisterStream_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(RegisterStreamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(ServicesServer).RegisterStream(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Services_LoadStream_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(LoadStreamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(ServicesServer).LoadStream(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func _Services_TerminateStream_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(TerminateStreamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	out, err := srv.(ServicesServer).TerminateStream(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

var _Services_serviceDesc = grpc.ServiceDesc{
	ServiceName: "logdog.Services",
	HandlerType: (*ServicesServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetConfig",
			Handler:    _Services_GetConfig_Handler,
		},
		{
			MethodName: "RegisterStream",
			Handler:    _Services_RegisterStream_Handler,
		},
		{
			MethodName: "LoadStream",
			Handler:    _Services_LoadStream_Handler,
		},
		{
			MethodName: "TerminateStream",
			Handler:    _Services_TerminateStream_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var fileDescriptor0 = []byte{
	// 426 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x94, 0x52, 0xdf, 0x6b, 0xd4, 0x40,
	0x10, 0xa6, 0xed, 0x79, 0xb4, 0x73, 0xe9, 0x49, 0x57, 0x1a, 0xd2, 0x15, 0x45, 0x02, 0x42, 0x5f,
	0xdc, 0xc0, 0xf9, 0x28, 0xf8, 0x52, 0x45, 0x0a, 0xc5, 0xc2, 0x9d, 0x0f, 0xbe, 0x85, 0xfc, 0x98,
	0x6e, 0x17, 0x92, 0x6c, 0xdc, 0xdd, 0x1c, 0xfa, 0x37, 0xf9, 0x4f, 0x9a, 0xec, 0xee, 0xdd, 0xd5,
	0x78, 0x15, 0xfa, 0x92, 0xb0, 0xdf, 0x37, 0xf3, 0xcd, 0xcc, 0x37, 0x03, 0xa7, 0x1a, 0xd5, 0x5a,
	0x14, 0xc8, 0x5a, 0x25, 0x8d, 0x24, 0xd3, 0x4a, 0xf2, 0x52, 0x72, 0x3a, 0xd3, 0x26, 0x33, 0x1e,
	0xa4, 0x1f, 0xb8, 0x30, 0xf7, 0x5d, 0xce, 0x0a, 0x59, 0x27, 0x55, 0x57, 0x08, 0xfb, 0x79, 0xc7,
	0x65, 0xd2, 0x03, 0xb5, 0x6c, 0x12, 0x1b, 0x95, 0xb8, 0xcc, 0xe1, 0xd7, 0xe6, 0xc3, 0xd7, 0x27,
	0xbf, 0xe4, 0x52, 0xf2, 0x0a, 0x5d, 0x50, 0xde, 0xdd, 0x25, 0x58, 0xb7, 0xe6, 0x97, 0x23, 0xe3,
	0xef, 0x70, 0xf6, 0x05, 0xcd, 0x95, 0x6c, 0xee, 0x04, 0x5f, 0xa2, 0x6e, 0x65, 0xa3, 0x91, 0x50,
	0x20, 0x85, 0x45, 0x52, 0xdf, 0x5b, 0xda, 0xa9, 0x2a, 0x3a, 0x78, 0x73, 0x70, 0x79, 0x42, 0x08,
	0xc0, 0x96, 0x33, 0xd1, 0xa1, 0xc5, 0x5e, 0xc0, 0xcc, 0x63, 0x6d, 0x66, 0xee, 0xa3, 0xa3, 0x01,
	0x8c, 0xd7, 0x70, 0xbe, 0x44, 0x2e, 0xb4, 0x41, 0xb5, 0x32, 0x0a, 0xb3, 0x7a, 0x89, 0x3f, 0x3a,
	0xd4, 0x86, 0x04, 0x30, 0xb1, 0x61, 0x4e, 0x6f, 0x0e, 0x53, 0x8d, 0x85, 0xf2, 0x5a, 0x01, 0x39,
	0x87, 0x53, 0xdb, 0x59, 0xba, 0x46, 0xa5, 0x85, 0x6c, 0x9c, 0x1a, 0xb9, 0x84, 0x49, 0x89, 0xba,
	0x88, 0x26, 0xfd, 0x6b, 0xb6, 0xa0, 0xcc, 0x0e, 0xc9, 0x6e, 0x24, 0x77, 0xda, 0x9f, 0x7a, 0x4e,
	0x89, 0xd6, 0x48, 0x15, 0xdf, 0x42, 0x38, 0xae, 0xeb, 0xc7, 0x7a, 0x0b, 0xcf, 0xac, 0xa9, 0xb6,
	0xf2, 0x6c, 0x11, 0x32, 0x67, 0xd8, 0x4e, 0x65, 0x35, 0xb0, 0xe3, 0x8e, 0xe2, 0x04, 0xce, 0x6e,
	0x64, 0x56, 0xfe, 0x6f, 0x88, 0xc0, 0x77, 0x37, 0x24, 0x1c, 0xc7, 0xd7, 0x40, 0x1e, 0x26, 0x3c,
	0xad, 0xfa, 0x43, 0xa9, 0x20, 0xfe, 0x0a, 0xe1, 0x37, 0x54, 0xb5, 0x68, 0x7a, 0xea, 0x29, 0x2e,
	0x86, 0x30, 0x37, 0x2e, 0xaf, 0x4a, 0x45, 0x53, 0xe2, 0x4f, 0x6b, 0xe3, 0xd1, 0xe2, 0xf7, 0x21,
	0x1c, 0xaf, 0xdc, 0x4e, 0x35, 0xf9, 0x08, 0x27, 0xdb, 0xdd, 0x93, 0x90, 0xb9, 0x33, 0x61, 0x9b,
	0x33, 0x61, 0x9f, 0x87, 0x33, 0xa1, 0x17, 0x9b, 0x3e, 0xff, 0x3d, 0x93, 0x5b, 0x98, 0xff, 0xed,
	0x34, 0x79, 0xb5, 0x09, 0xde, 0xbb, 0x79, 0xfa, 0xfa, 0x31, 0xda, 0x0b, 0x5e, 0x01, 0xec, 0x8c,
	0x23, 0x17, 0x3b, 0x87, 0x46, 0xee, 0x53, 0xba, 0x8f, 0xf2, 0x22, 0xd7, 0xf0, 0x7c, 0x64, 0x19,
	0xd9, 0xd6, 0xdd, 0xef, 0x25, 0x7d, 0x64, 0xf6, 0x7c, 0x6a, 0xdf, 0xef, 0xff, 0x04, 0x00, 0x00,
	0xff, 0xff, 0xb3, 0xba, 0x89, 0x38, 0xa3, 0x03, 0x00, 0x00,
}
