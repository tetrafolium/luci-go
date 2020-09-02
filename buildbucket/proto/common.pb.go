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
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.1
// source: github.com/tetrafolium/luci-go/buildbucket/proto/common.proto

package buildbucketpb

import (
	proto "github.com/golang/protobuf/proto"
	duration "github.com/golang/protobuf/ptypes/duration"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

// Status of a build or a step.
type Status int32

const (
	// Unspecified state. Meaning depends on the context.
	Status_STATUS_UNSPECIFIED Status = 0
	// Build was scheduled, but did not start or end yet.
	Status_SCHEDULED Status = 1
	// Build/step has started.
	Status_STARTED Status = 2
	// A union of all terminal statuses.
	// Can be used in BuildPredicate.status.
	// A concrete build/step cannot have this status.
	// Can be used as a bitmask to check that a build/step ended.
	Status_ENDED_MASK Status = 4
	// A build/step ended successfully.
	// This is a terminal status. It may not transition to another status.
	Status_SUCCESS Status = 12 // 8 | ENDED
	// A build/step ended unsuccessfully due to its Build.Input,
	// e.g. tests failed, and NOT due to a build infrastructure failure.
	// This is a terminal status. It may not transition to another status.
	Status_FAILURE Status = 20 // 16 | ENDED
	// A build/step ended unsuccessfully due to a failure independent of the
	// input, e.g. swarming failed, not enough capacity or the recipe was unable
	// to read the patch from gerrit.
	// start_time is not required for this status.
	// This is a terminal status. It may not transition to another status.
	Status_INFRA_FAILURE Status = 36 // 32 | ENDED
	// A build was cancelled explicitly, e.g. via an RPC.
	// This is a terminal status. It may not transition to another status.
	Status_CANCELED Status = 68 // 64 | ENDED
)

// Enum value maps for Status.
var (
	Status_name = map[int32]string{
		0:  "STATUS_UNSPECIFIED",
		1:  "SCHEDULED",
		2:  "STARTED",
		4:  "ENDED_MASK",
		12: "SUCCESS",
		20: "FAILURE",
		36: "INFRA_FAILURE",
		68: "CANCELED",
	}
	Status_value = map[string]int32{
		"STATUS_UNSPECIFIED": 0,
		"SCHEDULED":          1,
		"STARTED":            2,
		"ENDED_MASK":         4,
		"SUCCESS":            12,
		"FAILURE":            20,
		"INFRA_FAILURE":      36,
		"CANCELED":           68,
	}
)

func (x Status) Enum() *Status {
	p := new(Status)
	*p = x
	return p
}

func (x Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Status) Descriptor() protoreflect.EnumDescriptor {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_enumTypes[0].Descriptor()
}

func (Status) Type() protoreflect.EnumType {
	return &file_go_chromium_org_luci_buildbucket_proto_common_proto_enumTypes[0]
}

func (x Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Status.Descriptor instead.
func (Status) EnumDescriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{0}
}

// A boolean with an undefined value.
type Trinary int32

const (
	Trinary_UNSET Trinary = 0
	Trinary_YES   Trinary = 1
	Trinary_NO    Trinary = 2
)

// Enum value maps for Trinary.
var (
	Trinary_name = map[int32]string{
		0: "UNSET",
		1: "YES",
		2: "NO",
	}
	Trinary_value = map[string]int32{
		"UNSET": 0,
		"YES":   1,
		"NO":    2,
	}
)

func (x Trinary) Enum() *Trinary {
	p := new(Trinary)
	*p = x
	return p
}

func (x Trinary) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Trinary) Descriptor() protoreflect.EnumDescriptor {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_enumTypes[1].Descriptor()
}

func (Trinary) Type() protoreflect.EnumType {
	return &file_go_chromium_org_luci_buildbucket_proto_common_proto_enumTypes[1]
}

func (x Trinary) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Trinary.Descriptor instead.
func (Trinary) EnumDescriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{1}
}

// An executable to run when the build is ready to start.
//
// Please refer to github.com/tetrafolium/luci-go/luciexe for the protocol this executable
// is expected to implement.
//
// In addition to the "Host Application" responsibilities listed there,
// buildbucket will also ensure that $CWD points to an empty directory when it
// starts the build.
type Executable struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The CIPD package containing the executable.
	//
	// See the `cmd` field below for how the executable will be located within the
	// package.
	CipdPackage string `protobuf:"bytes,1,opt,name=cipd_package,json=cipdPackage,proto3" json:"cipd_package,omitempty"`
	// The CIPD version to fetch.
	//
	// Optional. If omitted, this defaults to `latest`.
	CipdVersion string `protobuf:"bytes,2,opt,name=cipd_version,json=cipdVersion,proto3" json:"cipd_version,omitempty"`
	// The command to invoke within the package.
	//
	// The 0th argument is taken as relative to the cipd_package root (a.k.a.
	// BBAgentArgs.payload_path), so "foo" would invoke the binary called "foo" in
	// the root of the package. On Windows, this will automatically look
	// first for ".exe" and ".bat" variants. Similarly, "subdir/foo" would
	// look for "foo" in "subdir" of the CIPD package.
	//
	// The other arguments are passed verbatim to the executable.
	//
	// The 'build.proto' binary message will always be passed to stdin, even when
	// this command has arguments (see github.com/tetrafolium/luci-go/luciexe).
	//
	// RECOMMENDATION: It's advised to rely on the build.proto's Input.Properties
	// field for passing task-specific data. Properties are JSON-typed and can be
	// modeled with a protobuf (using JSONPB). However, supplying additional args
	// can be useful to, e.g., increase logging verbosity, or similar
	// 'system level' settings within the binary.
	//
	// Optional. If omitted, defaults to `['luciexe']`.
	Cmd []string `protobuf:"bytes,3,rep,name=cmd,proto3" json:"cmd,omitempty"`
}

func (x *Executable) Reset() {
	*x = Executable{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Executable) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Executable) ProtoMessage() {}

func (x *Executable) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Executable.ProtoReflect.Descriptor instead.
func (*Executable) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{0}
}

func (x *Executable) GetCipdPackage() string {
	if x != nil {
		return x.CipdPackage
	}
	return ""
}

func (x *Executable) GetCipdVersion() string {
	if x != nil {
		return x.CipdVersion
	}
	return ""
}

func (x *Executable) GetCmd() []string {
	if x != nil {
		return x.Cmd
	}
	return nil
}

// Machine-readable details of a status.
// Human-readble details are present in a sibling summary_markdown field.
type StatusDetails struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// If set, indicates that the failure was due to a resource exhaustion / quota
	// denial.
	// Applicable in FAILURE and INFRA_FAILURE statuses.
	ResourceExhaustion *StatusDetails_ResourceExhaustion `protobuf:"bytes,3,opt,name=resource_exhaustion,json=resourceExhaustion,proto3" json:"resource_exhaustion,omitempty"`
	// If set, indicates that the failure was due to a timeout.
	// Applicable in FAILURE and INFRA_FAILURE statuses.
	Timeout *StatusDetails_Timeout `protobuf:"bytes,4,opt,name=timeout,proto3" json:"timeout,omitempty"`
}

func (x *StatusDetails) Reset() {
	*x = StatusDetails{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusDetails) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusDetails) ProtoMessage() {}

func (x *StatusDetails) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusDetails.ProtoReflect.Descriptor instead.
func (*StatusDetails) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{1}
}

func (x *StatusDetails) GetResourceExhaustion() *StatusDetails_ResourceExhaustion {
	if x != nil {
		return x.ResourceExhaustion
	}
	return nil
}

func (x *StatusDetails) GetTimeout() *StatusDetails_Timeout {
	if x != nil {
		return x.Timeout
	}
	return nil
}

// A named log of a step or build.
type Log struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Log name, standard ("stdout", "stderr") or custom (e.g. "json.output").
	// Unique within the containing message (step or build).
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// URL of a Human-readable page that displays log contents.
	ViewUrl string `protobuf:"bytes,2,opt,name=view_url,json=viewUrl,proto3" json:"view_url,omitempty"`
	// URL of the log content.
	// As of 2018-09-06, the only supported scheme is "logdog".
	// Typically it has form
	// "logdog://<host>/<project>/<prefix>/+/<stream_name>".
	// See also
	// https://godoc.org/github.com/tetrafolium/luci-go/logdog/common/types#ParseURL
	Url string `protobuf:"bytes,3,opt,name=url,proto3" json:"url,omitempty"`
}

func (x *Log) Reset() {
	*x = Log{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Log) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Log) ProtoMessage() {}

func (x *Log) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Log.ProtoReflect.Descriptor instead.
func (*Log) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{2}
}

func (x *Log) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Log) GetViewUrl() string {
	if x != nil {
		return x.ViewUrl
	}
	return ""
}

func (x *Log) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

// A Gerrit patchset.
type GerritChange struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Gerrit hostname, e.g. "chromium-review.googlesource.com".
	Host string `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	// Gerrit project, e.g. "chromium/src".
	Project string `protobuf:"bytes,2,opt,name=project,proto3" json:"project,omitempty"`
	// Change number, e.g. 12345.
	Change int64 `protobuf:"varint,3,opt,name=change,proto3" json:"change,omitempty"`
	// Patch set number, e.g. 1.
	Patchset int64 `protobuf:"varint,4,opt,name=patchset,proto3" json:"patchset,omitempty"`
}

func (x *GerritChange) Reset() {
	*x = GerritChange{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GerritChange) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GerritChange) ProtoMessage() {}

func (x *GerritChange) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GerritChange.ProtoReflect.Descriptor instead.
func (*GerritChange) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{3}
}

func (x *GerritChange) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *GerritChange) GetProject() string {
	if x != nil {
		return x.Project
	}
	return ""
}

func (x *GerritChange) GetChange() int64 {
	if x != nil {
		return x.Change
	}
	return 0
}

func (x *GerritChange) GetPatchset() int64 {
	if x != nil {
		return x.Patchset
	}
	return 0
}

// A landed Git commit hosted on Gitiles.
type GitilesCommit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Gitiles hostname, e.g. "chromium.googlesource.com".
	Host string `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	// Repository name on the host, e.g. "chromium/src".
	Project string `protobuf:"bytes,2,opt,name=project,proto3" json:"project,omitempty"`
	// Commit HEX SHA1.
	Id string `protobuf:"bytes,3,opt,name=id,proto3" json:"id,omitempty"`
	// Commit ref, e.g. "refs/heads/master".
	// NOT a branch name: if specified, must start with "refs/".
	Ref string `protobuf:"bytes,4,opt,name=ref,proto3" json:"ref,omitempty"`
	// Defines a total order of commits on the ref. Requires ref field.
	// Typically 1-based, monotonically increasing, contiguous integer
	// defined by a Gerrit plugin, goto.google.com/git-numberer.
	// TODO(tandrii): make it a public doc.
	Position uint32 `protobuf:"varint,5,opt,name=position,proto3" json:"position,omitempty"`
}

func (x *GitilesCommit) Reset() {
	*x = GitilesCommit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GitilesCommit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitilesCommit) ProtoMessage() {}

func (x *GitilesCommit) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitilesCommit.ProtoReflect.Descriptor instead.
func (*GitilesCommit) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{4}
}

func (x *GitilesCommit) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *GitilesCommit) GetProject() string {
	if x != nil {
		return x.Project
	}
	return ""
}

func (x *GitilesCommit) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *GitilesCommit) GetRef() string {
	if x != nil {
		return x.Ref
	}
	return ""
}

func (x *GitilesCommit) GetPosition() uint32 {
	if x != nil {
		return x.Position
	}
	return 0
}

// A key-value pair of strings.
type StringPair struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *StringPair) Reset() {
	*x = StringPair{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StringPair) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StringPair) ProtoMessage() {}

func (x *StringPair) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StringPair.ProtoReflect.Descriptor instead.
func (*StringPair) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{5}
}

func (x *StringPair) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *StringPair) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

// Half-open time range.
type TimeRange struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Inclusive lower boundary. Optional.
	StartTime *timestamp.Timestamp `protobuf:"bytes,1,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	// Exclusive upper boundary. Optional.
	EndTime *timestamp.Timestamp `protobuf:"bytes,2,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
}

func (x *TimeRange) Reset() {
	*x = TimeRange{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TimeRange) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TimeRange) ProtoMessage() {}

func (x *TimeRange) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TimeRange.ProtoReflect.Descriptor instead.
func (*TimeRange) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{6}
}

func (x *TimeRange) GetStartTime() *timestamp.Timestamp {
	if x != nil {
		return x.StartTime
	}
	return nil
}

func (x *TimeRange) GetEndTime() *timestamp.Timestamp {
	if x != nil {
		return x.EndTime
	}
	return nil
}

// A requested dimension. Looks like StringPair, but also has an expiration.
type RequestedDimension struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	// If set, ignore this dimension after this duration.
	Expiration *duration.Duration `protobuf:"bytes,3,opt,name=expiration,proto3" json:"expiration,omitempty"`
}

func (x *RequestedDimension) Reset() {
	*x = RequestedDimension{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RequestedDimension) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RequestedDimension) ProtoMessage() {}

func (x *RequestedDimension) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RequestedDimension.ProtoReflect.Descriptor instead.
func (*RequestedDimension) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{7}
}

func (x *RequestedDimension) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *RequestedDimension) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *RequestedDimension) GetExpiration() *duration.Duration {
	if x != nil {
		return x.Expiration
	}
	return nil
}

type StatusDetails_ResourceExhaustion struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StatusDetails_ResourceExhaustion) Reset() {
	*x = StatusDetails_ResourceExhaustion{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusDetails_ResourceExhaustion) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusDetails_ResourceExhaustion) ProtoMessage() {}

func (x *StatusDetails_ResourceExhaustion) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusDetails_ResourceExhaustion.ProtoReflect.Descriptor instead.
func (*StatusDetails_ResourceExhaustion) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{1, 0}
}

type StatusDetails_Timeout struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StatusDetails_Timeout) Reset() {
	*x = StatusDetails_Timeout{}
	if protoimpl.UnsafeEnabled {
		mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StatusDetails_Timeout) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusDetails_Timeout) ProtoMessage() {}

func (x *StatusDetails_Timeout) ProtoReflect() protoreflect.Message {
	mi := &file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusDetails_Timeout.ProtoReflect.Descriptor instead.
func (*StatusDetails_Timeout) Descriptor() ([]byte, []int) {
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP(), []int{1, 1}
}

var File_go_chromium_org_luci_buildbucket_proto_common_proto protoreflect.FileDescriptor

var file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDesc = []byte{
	0x0a, 0x33, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f, 0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72,
	0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x62, 0x75, 0x63, 0x6b,
	0x65, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x62, 0x75, 0x63, 0x6b,
	0x65, 0x74, 0x2e, 0x76, 0x32, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x64, 0x0a, 0x0a, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74,
	0x61, 0x62, 0x6c, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x69, 0x70, 0x64, 0x5f, 0x70, 0x61, 0x63,
	0x6b, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63, 0x69, 0x70, 0x64,
	0x50, 0x61, 0x63, 0x6b, 0x61, 0x67, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x63, 0x69, 0x70, 0x64, 0x5f,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x63,
	0x69, 0x70, 0x64, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x6d,
	0x64, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x03, 0x63, 0x6d, 0x64, 0x22, 0xe0, 0x01, 0x0a,
	0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x44, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x12, 0x61,
	0x0a, 0x13, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x65, 0x78, 0x68, 0x61, 0x75,
	0x73, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x62, 0x75,
	0x69, 0x6c, 0x64, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x2e, 0x76, 0x32, 0x2e, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x44, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x45, 0x78, 0x68, 0x61, 0x75, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x12, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x45, 0x78, 0x68, 0x61, 0x75, 0x73, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x3f, 0x0a, 0x07, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74,
	0x2e, 0x76, 0x32, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x44, 0x65, 0x74, 0x61, 0x69, 0x6c,
	0x73, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x52, 0x07, 0x74, 0x69, 0x6d, 0x65, 0x6f,
	0x75, 0x74, 0x1a, 0x14, 0x0a, 0x12, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x45, 0x78,
	0x68, 0x61, 0x75, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x09, 0x0a, 0x07, 0x54, 0x69, 0x6d, 0x65,
	0x6f, 0x75, 0x74, 0x4a, 0x04, 0x08, 0x01, 0x10, 0x02, 0x4a, 0x04, 0x08, 0x02, 0x10, 0x03, 0x22,
	0x46, 0x0a, 0x03, 0x4c, 0x6f, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x76, 0x69,
	0x65, 0x77, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x69,
	0x65, 0x77, 0x55, 0x72, 0x6c, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x22, 0x70, 0x0a, 0x0c, 0x47, 0x65, 0x72, 0x72, 0x69,
	0x74, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x70,
	0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x72,
	0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x12, 0x1a, 0x0a,
	0x08, 0x70, 0x61, 0x74, 0x63, 0x68, 0x73, 0x65, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x08, 0x70, 0x61, 0x74, 0x63, 0x68, 0x73, 0x65, 0x74, 0x22, 0x7b, 0x0a, 0x0d, 0x47, 0x69, 0x74,
	0x69, 0x6c, 0x65, 0x73, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x6f,
	0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x12, 0x18,
	0x0a, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x72, 0x65, 0x66, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x72, 0x65, 0x66, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x34, 0x0a, 0x0a, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x50, 0x61, 0x69, 0x72, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x7d, 0x0a, 0x09,
	0x54, 0x69, 0x6d, 0x65, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x12, 0x39, 0x0a, 0x0a, 0x73, 0x74, 0x61,
	0x72, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x73, 0x74, 0x61, 0x72, 0x74,
	0x54, 0x69, 0x6d, 0x65, 0x12, 0x35, 0x0a, 0x08, 0x65, 0x6e, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x52, 0x07, 0x65, 0x6e, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x22, 0x77, 0x0a, 0x12, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x65, 0x64, 0x44, 0x69, 0x6d, 0x65, 0x6e, 0x73, 0x69, 0x6f,
	0x6e, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x39, 0x0a, 0x0a, 0x65, 0x78, 0x70,
	0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x2a, 0x87, 0x01, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x16, 0x0a, 0x12, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43,
	0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x53, 0x43, 0x48, 0x45, 0x44,
	0x55, 0x4c, 0x45, 0x44, 0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x54, 0x41, 0x52, 0x54, 0x45,
	0x44, 0x10, 0x02, 0x12, 0x0e, 0x0a, 0x0a, 0x45, 0x4e, 0x44, 0x45, 0x44, 0x5f, 0x4d, 0x41, 0x53,
	0x4b, 0x10, 0x04, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43, 0x45, 0x53, 0x53, 0x10, 0x0c,
	0x12, 0x0b, 0x0a, 0x07, 0x46, 0x41, 0x49, 0x4c, 0x55, 0x52, 0x45, 0x10, 0x14, 0x12, 0x11, 0x0a,
	0x0d, 0x49, 0x4e, 0x46, 0x52, 0x41, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x55, 0x52, 0x45, 0x10, 0x24,
	0x12, 0x0c, 0x0a, 0x08, 0x43, 0x41, 0x4e, 0x43, 0x45, 0x4c, 0x45, 0x44, 0x10, 0x44, 0x2a, 0x25,
	0x0a, 0x07, 0x54, 0x72, 0x69, 0x6e, 0x61, 0x72, 0x79, 0x12, 0x09, 0x0a, 0x05, 0x55, 0x4e, 0x53,
	0x45, 0x54, 0x10, 0x00, 0x12, 0x07, 0x0a, 0x03, 0x59, 0x45, 0x53, 0x10, 0x01, 0x12, 0x06, 0x0a,
	0x02, 0x4e, 0x4f, 0x10, 0x02, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x6f, 0x2e, 0x63, 0x68, 0x72, 0x6f,
	0x6d, 0x69, 0x75, 0x6d, 0x2e, 0x6f, 0x72, 0x67, 0x2f, 0x6c, 0x75, 0x63, 0x69, 0x2f, 0x62, 0x75,
	0x69, 0x6c, 0x64, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x3b,
	0x62, 0x75, 0x69, 0x6c, 0x64, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescOnce sync.Once
	file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescData = file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDesc
)

func file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescGZIP() []byte {
	file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescOnce.Do(func() {
		file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescData)
	})
	return file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDescData
}

var file_go_chromium_org_luci_buildbucket_proto_common_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_go_chromium_org_luci_buildbucket_proto_common_proto_goTypes = []interface{}{
	(Status)(0),                              // 0: buildbucket.v2.Status
	(Trinary)(0),                             // 1: buildbucket.v2.Trinary
	(*Executable)(nil),                       // 2: buildbucket.v2.Executable
	(*StatusDetails)(nil),                    // 3: buildbucket.v2.StatusDetails
	(*Log)(nil),                              // 4: buildbucket.v2.Log
	(*GerritChange)(nil),                     // 5: buildbucket.v2.GerritChange
	(*GitilesCommit)(nil),                    // 6: buildbucket.v2.GitilesCommit
	(*StringPair)(nil),                       // 7: buildbucket.v2.StringPair
	(*TimeRange)(nil),                        // 8: buildbucket.v2.TimeRange
	(*RequestedDimension)(nil),               // 9: buildbucket.v2.RequestedDimension
	(*StatusDetails_ResourceExhaustion)(nil), // 10: buildbucket.v2.StatusDetails.ResourceExhaustion
	(*StatusDetails_Timeout)(nil),            // 11: buildbucket.v2.StatusDetails.Timeout
	(*timestamp.Timestamp)(nil),              // 12: google.protobuf.Timestamp
	(*duration.Duration)(nil),                // 13: google.protobuf.Duration
}
var file_go_chromium_org_luci_buildbucket_proto_common_proto_depIdxs = []int32{
	10, // 0: buildbucket.v2.StatusDetails.resource_exhaustion:type_name -> buildbucket.v2.StatusDetails.ResourceExhaustion
	11, // 1: buildbucket.v2.StatusDetails.timeout:type_name -> buildbucket.v2.StatusDetails.Timeout
	12, // 2: buildbucket.v2.TimeRange.start_time:type_name -> google.protobuf.Timestamp
	12, // 3: buildbucket.v2.TimeRange.end_time:type_name -> google.protobuf.Timestamp
	13, // 4: buildbucket.v2.RequestedDimension.expiration:type_name -> google.protobuf.Duration
	5,  // [5:5] is the sub-list for method output_type
	5,  // [5:5] is the sub-list for method input_type
	5,  // [5:5] is the sub-list for extension type_name
	5,  // [5:5] is the sub-list for extension extendee
	0,  // [0:5] is the sub-list for field type_name
}

func init() { file_go_chromium_org_luci_buildbucket_proto_common_proto_init() }
func file_go_chromium_org_luci_buildbucket_proto_common_proto_init() {
	if File_go_chromium_org_luci_buildbucket_proto_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Executable); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusDetails); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Log); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GerritChange); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GitilesCommit); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StringPair); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TimeRange); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RequestedDimension); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusDetails_ResourceExhaustion); i {
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
		file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StatusDetails_Timeout); i {
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
			RawDescriptor: file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_go_chromium_org_luci_buildbucket_proto_common_proto_goTypes,
		DependencyIndexes: file_go_chromium_org_luci_buildbucket_proto_common_proto_depIdxs,
		EnumInfos:         file_go_chromium_org_luci_buildbucket_proto_common_proto_enumTypes,
		MessageInfos:      file_go_chromium_org_luci_buildbucket_proto_common_proto_msgTypes,
	}.Build()
	File_go_chromium_org_luci_buildbucket_proto_common_proto = out.File
	file_go_chromium_org_luci_buildbucket_proto_common_proto_rawDesc = nil
	file_go_chromium_org_luci_buildbucket_proto_common_proto_goTypes = nil
	file_go_chromium_org_luci_buildbucket_proto_common_proto_depIdxs = nil
}
