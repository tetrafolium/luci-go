// Code generated by protoc-gen-go.
// source: github.com/luci/luci-go/logdog/api/config/svcconfig/project.proto
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

// ProjectConfig is a set of per-project configuration parameters. Each
// luci-config project must include one of these configs in order to register
// or view log streams in that project's log stream space.
//
// A project's configuration should reside in the "projects/<project>" config
// set and be named "<app-id>.cfg".
//
// Many of the parameters here can be bounded by GlobalConfig parameters.
type ProjectConfig struct {
	// The set of auth service groups that are permitted READ access to this
	// project's log streams.
	ReaderAuthGroups []string `protobuf:"bytes,2,rep,name=reader_auth_groups,json=readerAuthGroups" json:"reader_auth_groups,omitempty"`
	// The set of chrome-infra-auth groups that are permitted WRITE access to this
	// project's log streams.
	WriterAuthGroups []string `protobuf:"bytes,3,rep,name=writer_auth_groups,json=writerAuthGroups" json:"writer_auth_groups,omitempty"`
	// The maximum lifetime of a log stream.
	//
	// If a stream has not terminated after this period of time, it will be
	// forcefully archived, and additional stream data will be discarded.
	//
	// This is upper-bounded by the global "archive_delay_max" parameter.
	MaxStreamAge *google_protobuf.Duration `protobuf:"bytes,4,opt,name=max_stream_age,json=maxStreamAge" json:"max_stream_age,omitempty"`
	// The maximum amount of time after a prefix has been registered when log
	// streams may also be registered under that prefix.
	//
	// See Config's "prefix_expiration" for more information.
	PrefixExpiration *google_protobuf.Duration `protobuf:"bytes,5,opt,name=prefix_expiration,json=prefixExpiration" json:"prefix_expiration,omitempty"`
	// The archival Google Storage bucket name.
	//
	// Log streams artifacts will be stored in a subdirectory of this bucket:
	// gs://<archive_gs_bucket>/<app-id>/<project-name>/<log-path>/artifact...
	//
	// Note that the Archivist microservice must have WRITE access to this
	// bucket, and the Coordinator must have READ access.
	//
	// If this is not set, the logs will be archived in a project-named
	// subdirectory in the global "archive_gs_base" location.
	ArchiveGsBucket string `protobuf:"bytes,10,opt,name=archive_gs_bucket,json=archiveGsBucket" json:"archive_gs_bucket,omitempty"`
	// If true, always create an additional data file that is the rendered content
	// of the stream data. By default, only streams that explicitly register a
	// binary file extension must be rendered.
	//
	// See Config's "always_create_binary" for more information.
	RenderAllStreams bool `protobuf:"varint,11,opt,name=render_all_streams,json=renderAllStreams" json:"render_all_streams,omitempty"`
	// Project-specific archive index configuration.
	//
	// Any unspecified index configuration will default to the service archival
	// config.
	ArchiveIndexConfig *ArchiveIndexConfig `protobuf:"bytes,12,opt,name=archive_index_config,json=archiveIndexConfig" json:"archive_index_config,omitempty"`
}

func (m *ProjectConfig) Reset()                    { *m = ProjectConfig{} }
func (m *ProjectConfig) String() string            { return proto.CompactTextString(m) }
func (*ProjectConfig) ProtoMessage()               {}
func (*ProjectConfig) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{0} }

func (m *ProjectConfig) GetMaxStreamAge() *google_protobuf.Duration {
	if m != nil {
		return m.MaxStreamAge
	}
	return nil
}

func (m *ProjectConfig) GetPrefixExpiration() *google_protobuf.Duration {
	if m != nil {
		return m.PrefixExpiration
	}
	return nil
}

func (m *ProjectConfig) GetArchiveIndexConfig() *ArchiveIndexConfig {
	if m != nil {
		return m.ArchiveIndexConfig
	}
	return nil
}

func init() {
	proto.RegisterType((*ProjectConfig)(nil), "svcconfig.ProjectConfig")
}

func init() {
	proto.RegisterFile("github.com/luci/luci-go/logdog/api/config/svcconfig/project.proto", fileDescriptor2)
}

var fileDescriptor2 = []byte{
	// 344 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x9c, 0x91, 0xcd, 0x4a, 0xfb, 0x40,
	0x14, 0xc5, 0xe9, 0xbf, 0x7f, 0xc5, 0x4e, 0xab, 0xb6, 0xc1, 0x45, 0x2c, 0x28, 0xc1, 0x55, 0x10,
	0x4d, 0x40, 0x1f, 0x40, 0x52, 0x3f, 0x8a, 0x2b, 0x25, 0x3e, 0xc0, 0x30, 0x49, 0xa6, 0x93, 0xd1,
	0x49, 0x26, 0xcc, 0x47, 0xcd, 0xdb, 0xf8, 0xaa, 0x92, 0xb9, 0x69, 0x11, 0x5d, 0x08, 0x6e, 0x42,
	0x72, 0xcf, 0xef, 0x1e, 0x4e, 0xce, 0x45, 0x09, 0xe3, 0xa6, 0xb4, 0x59, 0x94, 0xcb, 0x2a, 0x16,
	0x36, 0xe7, 0xee, 0x71, 0xc9, 0x64, 0x2c, 0x24, 0x2b, 0x24, 0x8b, 0x49, 0xc3, 0xe3, 0x5c, 0xd6,
	0x2b, 0xce, 0x62, 0xbd, 0xce, 0xfb, 0xb7, 0x46, 0xc9, 0x57, 0x9a, 0x9b, 0xa8, 0x51, 0xd2, 0x48,
	0x6f, 0xb4, 0x15, 0xe6, 0x8b, 0xbf, 0xb8, 0x11, 0x95, 0x97, 0x7c, 0x4d, 0x04, 0xd8, 0xcd, 0x4f,
	0x99, 0x94, 0x4c, 0xd0, 0xd8, 0x7d, 0x65, 0x76, 0x15, 0x17, 0x56, 0x11, 0xc3, 0x65, 0x0d, 0xfa,
	0xd9, 0xc7, 0x10, 0xed, 0x3f, 0x43, 0x80, 0x5b, 0x67, 0xe0, 0x5d, 0x20, 0x4f, 0x51, 0x52, 0x50,
	0x85, 0x89, 0x35, 0x25, 0x66, 0x4a, 0xda, 0x46, 0xfb, 0xff, 0x82, 0x61, 0x38, 0x4a, 0xa7, 0xa0,
	0x24, 0xd6, 0x94, 0x4b, 0x37, 0xef, 0xe8, 0x77, 0xc5, 0xcd, 0x37, 0x7a, 0x08, 0x34, 0x28, 0x5f,
	0xe8, 0x1b, 0x74, 0x50, 0x91, 0x16, 0x6b, 0xa3, 0x28, 0xa9, 0x30, 0x61, 0xd4, 0xff, 0x1f, 0x0c,
	0xc2, 0xf1, 0xd5, 0x71, 0x04, 0x31, 0xa3, 0x4d, 0xcc, 0xe8, 0xae, 0x8f, 0x99, 0x4e, 0x2a, 0xd2,
	0xbe, 0x38, 0x3e, 0x61, 0xd4, 0x7b, 0x40, 0xb3, 0x46, 0xd1, 0x15, 0x6f, 0x31, 0x6d, 0x1b, 0x0e,
	0x88, 0xbf, 0xf3, 0x9b, 0xc7, 0x14, 0x76, 0xee, 0xb7, 0x2b, 0xde, 0x39, 0x9a, 0x41, 0x51, 0x14,
	0x33, 0x8d, 0x33, 0x9b, 0xbf, 0x51, 0xe3, 0xa3, 0x60, 0x10, 0x8e, 0xd2, 0xc3, 0x5e, 0x58, 0xea,
	0x85, 0x1b, 0x43, 0x21, 0xb5, 0x2b, 0x44, 0x88, 0x3e, 0xbb, 0xf6, 0xc7, 0xc1, 0x20, 0xdc, 0xeb,
	0x0a, 0xe9, 0x94, 0x44, 0x08, 0xc8, 0xa8, 0xbd, 0x27, 0x74, 0xb4, 0x71, 0xe6, 0x75, 0x41, 0x5b,
	0x0c, 0x77, 0xf1, 0x27, 0x2e, 0xe4, 0x49, 0xb4, 0xbd, 0x54, 0x94, 0x00, 0xf6, 0xd8, 0x51, 0xd0,
	0x7d, 0xea, 0x91, 0x1f, 0xb3, 0x6c, 0xd7, 0xfd, 0xcf, 0xf5, 0x67, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x58, 0x02, 0x29, 0xb9, 0x5c, 0x02, 0x00, 0x00,
}
