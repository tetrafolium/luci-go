// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.24.0-devel
// 	protoc        v3.12.1
// source: github.com/tetrafolium/luci-go/cipd/appengine/impl/repo/tasks/tasks.proto

package tasks

import (
	proto "github.com/golang/protobuf/proto"
	v1 "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

// RunProcessors task runs a processing step on an uploaded package instance.
type RunProcessors struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Instance *v1.Instance `protobuf:"bytes,1,opt,name=instance,proto3" json:"instance,omitempty"` // an instance to process
}

func (x *RunProcessors) Reset() {
	*x = RunProcessors{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunProcessors) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunProcessors) ProtoMessage() {}

func (x *RunProcessors) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunProcessors.ProtoReflect.Descriptor instead.
func (*RunProcessors) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescGZIP(), []int{0}
}

func (x *RunProcessors) GetInstance() *v1.Instance {
	if x != nil {
		return x.Instance
	}
	return nil
}

var File_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto protoreflect.FileDescriptor

var file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDesc = []byte{
	0x0a, 0x3f, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72,
	0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x63, 0x69, 0x70, 0x64, 0x2f, 0x61, 0x70, 0x70, 0x65,
	0x6e, 0x67, 0x69, 0x6e, 0x65, 0x2f, 0x69, 0x6d, 0x70, 0x6c, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x2f,
	0x74, 0x61, 0x73, 0x6b, 0x73, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x05, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x1a, 0x30, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72,
	0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x63,
	0x69, 0x70, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x63, 0x69, 0x70, 0x64, 0x2f, 0x76, 0x31, 0x2f,
	0x72, 0x65, 0x70, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3b, 0x0a, 0x0d, 0x52, 0x75,
	0x6e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x6f, 0x72, 0x73, 0x12, 0x2a, 0x0a, 0x08, 0x69,
	0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x63, 0x69, 0x70, 0x64, 0x2e, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x08, 0x69,
	0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x6f, 0x2e, 0x63, 0x68,
	0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f,
	0x63, 0x69, 0x70, 0x64, 0x2f, 0x61, 0x70, 0x70, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65, 0x2f, 0x69,
	0x6d, 0x70, 0x6c, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x2f, 0x74, 0x61, 0x73, 0x6b, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescOnce sync.Once
	file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescData = file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDesc
)

func file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescGZIP() []byte {
	file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescOnce.Do(func() {
		file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescData = protoimpl.X.CompressGZIP(file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescData)
	})
	return file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDescData
}

var file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_goTypes = []interface{}{
	(*RunProcessors)(nil), // 0: tasks.RunProcessors
	(*v1.Instance)(nil),   // 1: cipd.Instance
}
var file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_depIdxs = []int32{
	1, // 0: tasks.RunProcessors.instance:type_name -> cipd.Instance
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_init() }
func file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_init() {
	if File_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunProcessors); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_goTypes,
		DependencyIndexes: file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_depIdxs,
		MessageInfos:      file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_msgTypes,
	}.Build()
	File_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto = out.File
	file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_rawDesc = nil
	file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_goTypes = nil
	file_go_chromium_org_luci_cipd_appengine_impl_repo_tasks_tasks_proto_depIdxs = nil
}
