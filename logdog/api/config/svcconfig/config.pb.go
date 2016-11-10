// Code generated by protoc-gen-go.
// source: github.com/luci/luci-go/logdog/api/config/svcconfig/config.proto
// DO NOT EDIT!

package svcconfig

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/luci/luci-go/common/proto/google"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Config is the overall instance configuration.
type Config struct {
	// Configuration for the Butler's log transport.
	Transport *Transport `protobuf:"bytes,10,opt,name=transport" json:"transport,omitempty"`
	// Configuration for intermediate Storage.
	Storage *Storage `protobuf:"bytes,11,opt,name=storage" json:"storage,omitempty"`
	// Coordinator is the coordinator service configuration.
	Coordinator *Coordinator `protobuf:"bytes,20,opt,name=coordinator" json:"coordinator,omitempty"`
	// Collector is the collector fleet configuration.
	Collector *Collector `protobuf:"bytes,21,opt,name=collector" json:"collector,omitempty"`
	// Archivist microservice configuration.
	Archivist *Archivist `protobuf:"bytes,22,opt,name=archivist" json:"archivist,omitempty"`
}

func (m *Config) Reset()                    { *m = Config{} }
func (m *Config) String() string            { return proto.CompactTextString(m) }
func (*Config) ProtoMessage()               {}
func (*Config) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *Config) GetTransport() *Transport {
	if m != nil {
		return m.Transport
	}
	return nil
}

func (m *Config) GetStorage() *Storage {
	if m != nil {
		return m.Storage
	}
	return nil
}

func (m *Config) GetCoordinator() *Coordinator {
	if m != nil {
		return m.Coordinator
	}
	return nil
}

func (m *Config) GetCollector() *Collector {
	if m != nil {
		return m.Collector
	}
	return nil
}

func (m *Config) GetArchivist() *Archivist {
	if m != nil {
		return m.Archivist
	}
	return nil
}

// Coordinator is the Coordinator service configuration.
type Coordinator struct {
	// The name of the authentication group for administrators.
	AdminAuthGroup string `protobuf:"bytes,10,opt,name=admin_auth_group,json=adminAuthGroup" json:"admin_auth_group,omitempty"`
	// The name of the authentication group for backend services.
	ServiceAuthGroup string `protobuf:"bytes,11,opt,name=service_auth_group,json=serviceAuthGroup" json:"service_auth_group,omitempty"`
	// A list of origin URLs that are allowed to perform CORS RPC calls.
	RpcAllowOrigins []string `protobuf:"bytes,20,rep,name=rpc_allow_origins,json=rpcAllowOrigins" json:"rpc_allow_origins,omitempty"`
	// The maximum amount of time after a prefix has been registered when log
	// streams may also be registered under that prefix.
	//
	// After the expiration period has passed, new log stream registration will
	// fail.
	//
	// Project configurations or stream prefix regitrations may override this by
	// providing >= 0 values for prefix expiration. The smallest configured
	// expiration will be applied.
	PrefixExpiration *google_protobuf.Duration `protobuf:"bytes,21,opt,name=prefix_expiration,json=prefixExpiration" json:"prefix_expiration,omitempty"`
	// The full path of the archival Pub/Sub topic.
	//
	// The Coordinator must have permission to publish to this topic.
	ArchiveTopic string `protobuf:"bytes,30,opt,name=archive_topic,json=archiveTopic" json:"archive_topic,omitempty"`
	// The amount of time after an archive request has been dispatched before it
	// should be executed.
	//
	// Since terminal messages can arrive out of order, the archival request may
	// be kicked off before all of the log stream data has been loaded into
	// intermediate storage. If this happens, the Archivist will retry archival
	// later autometically.
	//
	// This parameter is an optimization to stop the archivist from wasting its
	// time until the log stream has a reasonable expectation of being available.
	ArchiveSettleDelay *google_protobuf.Duration `protobuf:"bytes,31,opt,name=archive_settle_delay,json=archiveSettleDelay" json:"archive_settle_delay,omitempty"`
	// The amount of time before a log stream is candidate for archival regardless
	// of whether or not it's been terminated or complete.
	//
	// This is a failsafe designed to ensure that log streams with missing records
	// or no terminal record (e.g., Butler crashed) are eventually archived.
	//
	// This should be fairly large (days) to avoid prematurely archiving
	// long-running streams, but should be considerably smaller than the
	// intermediate storage data retention period.
	//
	// If a project's "max_stream_age" is smaller than this value, it will be used
	// on that project's streams.
	ArchiveDelayMax *google_protobuf.Duration `protobuf:"bytes,32,opt,name=archive_delay_max,json=archiveDelayMax" json:"archive_delay_max,omitempty"`
}

func (m *Coordinator) Reset()                    { *m = Coordinator{} }
func (m *Coordinator) String() string            { return proto.CompactTextString(m) }
func (*Coordinator) ProtoMessage()               {}
func (*Coordinator) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *Coordinator) GetPrefixExpiration() *google_protobuf.Duration {
	if m != nil {
		return m.PrefixExpiration
	}
	return nil
}

func (m *Coordinator) GetArchiveSettleDelay() *google_protobuf.Duration {
	if m != nil {
		return m.ArchiveSettleDelay
	}
	return nil
}

func (m *Coordinator) GetArchiveDelayMax() *google_protobuf.Duration {
	if m != nil {
		return m.ArchiveDelayMax
	}
	return nil
}

// Collector is the set of configuration parameters for Collector instances.
type Collector struct {
	// The maximum number of concurrent transport messages to process. If <= 0,
	// a default will be chosen based on the transport.
	MaxConcurrentMessages int32 `protobuf:"varint,1,opt,name=max_concurrent_messages,json=maxConcurrentMessages" json:"max_concurrent_messages,omitempty"`
	// The maximum number of concurrent workers to process each ingested message.
	// If <= 0, collector.DefaultMaxMessageWorkers will be used.
	MaxMessageWorkers int32 `protobuf:"varint,2,opt,name=max_message_workers,json=maxMessageWorkers" json:"max_message_workers,omitempty"`
	// The maximum number of log stream states to cache locally. If <= 0, a
	// default will be used.
	StateCacheSize int32 `protobuf:"varint,3,opt,name=state_cache_size,json=stateCacheSize" json:"state_cache_size,omitempty"`
	// The maximum amount of time that cached stream state is valid. If <= 0, a
	// default will be used.
	StateCacheExpiration *google_protobuf.Duration `protobuf:"bytes,4,opt,name=state_cache_expiration,json=stateCacheExpiration" json:"state_cache_expiration,omitempty"`
}

func (m *Collector) Reset()                    { *m = Collector{} }
func (m *Collector) String() string            { return proto.CompactTextString(m) }
func (*Collector) ProtoMessage()               {}
func (*Collector) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

func (m *Collector) GetStateCacheExpiration() *google_protobuf.Duration {
	if m != nil {
		return m.StateCacheExpiration
	}
	return nil
}

// Configuration for the Archivist microservice.
type Archivist struct {
	// The name of the archival Pub/Sub subscription.
	//
	// This should be connected to "archive_topic", and the Archivist must have
	// permission to consume from this subscription.
	Subscription string `protobuf:"bytes,1,opt,name=subscription" json:"subscription,omitempty"`
	// The number of tasks to run at a time. If blank, the archivist will choose a
	// default value.
	Tasks int32 `protobuf:"varint,2,opt,name=tasks" json:"tasks,omitempty"`
	// The name of the staging storage bucket. All projects will share the same
	// staging bucket. Logs for a project will be staged under:
	//
	// gs://<gs_staging_bucket>/<app-id>/<project-name>/...
	GsStagingBucket string `protobuf:"bytes,3,opt,name=gs_staging_bucket,json=gsStagingBucket" json:"gs_staging_bucket,omitempty"`
	// Service-wide index configuration. This is used if per-project configuration
	// is not specified.
	ArchiveIndexConfig *ArchiveIndexConfig `protobuf:"bytes,10,opt,name=archive_index_config,json=archiveIndexConfig" json:"archive_index_config,omitempty"`
	// If true, always render the log entries as a binary file during archival,
	// regardless of whether a specific stream has a binary file extension.
	//
	// By default, a stream will only be rendered as a binary if its descriptor
	// includes a non-empty binary file extension field.
	//
	// The binary stream consists of each log entry's data rendered back-to-back.
	//   - For text streams, this produces a text document similar to the source
	//     text.
	//   - For binary streams and datagram streams, this reproduces the source
	//     contiguous binary file.
	//   - For datagram streams, the size-prefixed datagrams are written back-to-
	//     back.
	//
	// Enabling this option will consume roughly twice the archival space, as each
	// stream's data will be archived once as a series of log entries and once as
	// a binary file.
	//
	// Streams without an explicit binary file extension will default to ".bin" if
	// this is enabled.
	RenderAllStreams bool `protobuf:"varint,13,opt,name=render_all_streams,json=renderAllStreams" json:"render_all_streams,omitempty"`
}

func (m *Archivist) Reset()                    { *m = Archivist{} }
func (m *Archivist) String() string            { return proto.CompactTextString(m) }
func (*Archivist) ProtoMessage()               {}
func (*Archivist) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

func (m *Archivist) GetArchiveIndexConfig() *ArchiveIndexConfig {
	if m != nil {
		return m.ArchiveIndexConfig
	}
	return nil
}

func init() {
	proto.RegisterType((*Config)(nil), "svcconfig.Config")
	proto.RegisterType((*Coordinator)(nil), "svcconfig.Coordinator")
	proto.RegisterType((*Collector)(nil), "svcconfig.Collector")
	proto.RegisterType((*Archivist)(nil), "svcconfig.Archivist")
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/logdog/api/config/svcconfig/config.proto", fileDescriptor1)
}

var fileDescriptor1 = []byte{
	// 660 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x9c, 0x54, 0x4b, 0x6b, 0xdb, 0x30,
	0x1c, 0x27, 0xed, 0xda, 0x2d, 0x4a, 0x1f, 0x89, 0x96, 0x76, 0x5e, 0x61, 0x5d, 0xc8, 0x2e, 0x61,
	0x74, 0x0e, 0x74, 0x30, 0x76, 0x5c, 0x9a, 0x76, 0x63, 0x8c, 0x52, 0x70, 0x0a, 0x3b, 0x0a, 0x45,
	0x56, 0x14, 0x51, 0xdb, 0x32, 0x92, 0xdc, 0x7a, 0xfd, 0x0c, 0x3b, 0xed, 0x83, 0x8e, 0x7d, 0x84,
	0xa1, 0x87, 0x1f, 0xd0, 0x43, 0xa1, 0x97, 0x24, 0xfe, 0xbd, 0xf2, 0x97, 0x7e, 0xb2, 0xc0, 0x17,
	0xc6, 0xf5, 0xba, 0x58, 0x86, 0x44, 0xa4, 0xd3, 0xa4, 0x20, 0xdc, 0x7e, 0x7c, 0x60, 0x62, 0x9a,
	0x08, 0x16, 0x0b, 0x36, 0xc5, 0x39, 0x9f, 0x12, 0x91, 0xad, 0x38, 0x9b, 0xaa, 0x5b, 0xe2, 0x7f,
	0xb9, 0xaf, 0x30, 0x97, 0x42, 0x0b, 0xd8, 0xad, 0xf1, 0xa3, 0xb3, 0xa7, 0x84, 0x61, 0x49, 0xd6,
	0xfc, 0x16, 0x27, 0x2e, 0xee, 0x68, 0xf6, 0x94, 0x0c, 0xa5, 0x85, 0xc4, 0x8c, 0xfa, 0x88, 0xf9,
	0x53, 0x22, 0xb4, 0xc4, 0x99, 0xca, 0x85, 0xd4, 0x3e, 0xe4, 0x98, 0x09, 0xc1, 0x12, 0x3a, 0xb5,
	0x4f, 0xcb, 0x62, 0x35, 0x8d, 0x0b, 0x89, 0x35, 0x17, 0x99, 0xe3, 0xc7, 0xbf, 0x37, 0xc0, 0xf6,
	0xdc, 0x5a, 0xe1, 0x29, 0xe8, 0xd6, 0xee, 0x00, 0x8c, 0x3a, 0x93, 0xde, 0xe9, 0x30, 0xac, 0x93,
	0xc3, 0xeb, 0x8a, 0x8b, 0x1a, 0x19, 0x3c, 0x01, 0xcf, 0xfd, 0xd0, 0x41, 0xcf, 0x3a, 0x60, 0xcb,
	0xb1, 0x70, 0x4c, 0x54, 0x49, 0xe0, 0x67, 0xd0, 0x23, 0x42, 0xc8, 0x98, 0x67, 0x58, 0x0b, 0x19,
	0x0c, 0xad, 0xe3, 0xb0, 0xe5, 0x98, 0x37, 0x6c, 0xd4, 0x96, 0x9a, 0xd9, 0x88, 0x48, 0x12, 0x4a,
	0x8c, 0xef, 0xe0, 0xc1, 0x6c, 0xf3, 0x8a, 0x8b, 0x1a, 0x99, 0xf1, 0xb8, 0x52, 0xb8, 0xd2, 0xc1,
	0xe1, 0x03, 0xcf, 0xac, 0xe2, 0xa2, 0x46, 0x36, 0xfe, 0xb3, 0x09, 0x7a, 0xad, 0x21, 0xe0, 0x04,
	0xf4, 0x71, 0x9c, 0xf2, 0x0c, 0xe1, 0x42, 0xaf, 0x11, 0x93, 0xa2, 0xc8, 0xed, 0xd6, 0x74, 0xa3,
	0x3d, 0x8b, 0xcf, 0x0a, 0xbd, 0xfe, 0x66, 0x50, 0x78, 0x02, 0xa0, 0xa2, 0xf2, 0x96, 0x13, 0xda,
	0xd6, 0xf6, 0xac, 0xb6, 0xef, 0x99, 0x46, 0xfd, 0x1e, 0x0c, 0x64, 0x4e, 0x10, 0x4e, 0x12, 0x71,
	0x87, 0x84, 0xe4, 0x8c, 0x67, 0x2a, 0x18, 0x8e, 0x36, 0x27, 0xdd, 0x68, 0x5f, 0xe6, 0x64, 0x66,
	0xf0, 0x2b, 0x07, 0xc3, 0xaf, 0x60, 0x90, 0x4b, 0xba, 0xe2, 0x25, 0xa2, 0x65, 0xce, 0x5d, 0x7b,
	0x7e, 0x0f, 0x5e, 0x87, 0xae, 0xde, 0xb0, 0xaa, 0x37, 0x3c, 0xf7, 0xf5, 0x46, 0x7d, 0xe7, 0xb9,
	0xa8, 0x2d, 0xf0, 0x1d, 0xd8, 0x75, 0x0b, 0xa5, 0x48, 0x8b, 0x9c, 0x93, 0xe0, 0xd8, 0x0e, 0xb7,
	0xe3, 0xc1, 0x6b, 0x83, 0xc1, 0x1f, 0x60, 0x58, 0x89, 0x14, 0xd5, 0x3a, 0xa1, 0x28, 0xa6, 0x09,
	0xfe, 0x15, 0xbc, 0x7d, 0xec, 0xff, 0xa0, 0xb7, 0x2d, 0xac, 0xeb, 0xdc, 0x98, 0xe0, 0x05, 0x18,
	0x54, 0x61, 0x36, 0x05, 0xa5, 0xb8, 0x0c, 0x46, 0x8f, 0x25, 0xed, 0x7b, 0x8f, 0xcd, 0xb8, 0xc4,
	0xe5, 0xf8, 0x6f, 0x07, 0x74, 0xeb, 0x86, 0xe1, 0x27, 0xf0, 0x2a, 0xc5, 0x25, 0x22, 0x22, 0x23,
	0x85, 0x94, 0x34, 0xd3, 0x28, 0xa5, 0x4a, 0x61, 0x46, 0x55, 0xd0, 0x19, 0x75, 0x26, 0x5b, 0xd1,
	0x41, 0x8a, 0xcb, 0x79, 0xcd, 0x5e, 0x7a, 0x12, 0x86, 0xe0, 0xa5, 0xf1, 0x79, 0x31, 0xba, 0x13,
	0xf2, 0x86, 0x4a, 0x15, 0x6c, 0x58, 0xcf, 0x20, 0xc5, 0xa5, 0x57, 0xfe, 0x74, 0x84, 0xa9, 0x5e,
	0x69, 0xac, 0x29, 0x22, 0x98, 0xac, 0x29, 0x52, 0xfc, 0x9e, 0x06, 0x9b, 0x56, 0xbc, 0x67, 0xf1,
	0xb9, 0x81, 0x17, 0xfc, 0x9e, 0xc2, 0x2b, 0x70, 0xd8, 0x56, 0xb6, 0x5a, 0x7a, 0xf6, 0xd8, 0x5a,
	0x87, 0x4d, 0x54, 0xd3, 0xd4, 0xf8, 0x5f, 0x07, 0x74, 0xeb, 0xe3, 0x09, 0xc7, 0x60, 0x47, 0x15,
	0x4b, 0x45, 0x24, 0xcf, 0x6d, 0x68, 0xc7, 0xd5, 0xd6, 0xc6, 0xe0, 0x10, 0x6c, 0x69, 0xac, 0x6e,
	0xaa, 0xe5, 0xb8, 0x07, 0x73, 0xca, 0x98, 0x42, 0x4a, 0x63, 0xc6, 0x33, 0x86, 0x96, 0x05, 0xb9,
	0xa1, 0xda, 0xae, 0xa1, 0x1b, 0xed, 0x33, 0xb5, 0x70, 0xf8, 0x99, 0x85, 0xe1, 0x55, 0x53, 0x3c,
	0xcf, 0x62, 0x6a, 0x37, 0x78, 0xc5, 0x99, 0xbf, 0x08, 0xde, 0x3c, 0x78, 0x71, 0xe8, 0x77, 0xa3,
	0x72, 0x57, 0x47, 0x5d, 0x7e, 0x0b, 0x33, 0x2f, 0x84, 0xa4, 0x59, 0x4c, 0xa5, 0x39, 0xe5, 0x48,
	0x69, 0x49, 0x71, 0xaa, 0x82, 0xdd, 0x51, 0x67, 0xf2, 0x22, 0xea, 0x3b, 0x66, 0x96, 0x24, 0x0b,
	0x87, 0x2f, 0xb7, 0xed, 0xde, 0x7c, 0xfc, 0x1f, 0x00, 0x00, 0xff, 0xff, 0xc2, 0x11, 0xee, 0x77,
	0xc9, 0x05, 0x00, 0x00,
}
