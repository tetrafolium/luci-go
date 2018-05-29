// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/buildbucket/proto/build.proto

/*
Package buildbucketpb is a generated protocol buffer package.

It is generated from these files:
	go.chromium.org/luci/buildbucket/proto/build.proto
	go.chromium.org/luci/buildbucket/proto/common.proto
	go.chromium.org/luci/buildbucket/proto/rpc.proto
	go.chromium.org/luci/buildbucket/proto/step.proto

It has these top-level messages:
	Build
	CancelReason
	InfraFailureReason
	BuildInfra
	Builder
	GerritChange
	GitilesCommit
	StringPair
	TimeRange
	GetBuildRequest
	SearchBuildsRequest
	SearchBuildsResponse
	BuildPredicate
	Step
*/
package buildbucketpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"
import google_protobuf1 "github.com/golang/protobuf/ptypes/struct"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// A single build, identified by an int64 id.
// Belongs to a builder.
//
// RPC: see Builds service for build creation and retrieval.
// Some Build fields are marked as excluded from responses by default.
// Use build_fields request field to specify that a field must be included.
//
// BigQuery: this message also defines schema of a BigQuery table of completed builds.
// A BigQuery row is inserted soon after build ends, i.e. a row represents a state of
// a build at completion time and does not change after that.
// All fields are included.
type Build struct {
	// Identifier of the build, unique per LUCI deployment.
	// IDs are monotonically decreasing.
	Id int64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	// Required. The builder this build belongs to.
	//
	// Tuple (builder.project, builder.bucket) defines build ACL
	// which may change after build has ended.
	Builder *Builder_ID `protobuf:"bytes,2,opt,name=builder" json:"builder,omitempty"`
	// Human-oriented identifier of the build with the following properties:
	// - unique within the builder
	// - a monotonically increasing number
	// - mostly contiguous
	// - much shorter than id
	//
	// Caution: populated (positive number) iff build numbers were enabled
	// in the builder configuration at the time of build creation.
	//
	// Caution: Build numbers are not guaranteed to be contiguous.
	// There may be gaps during outages.
	//
	// Caution: Build numbers, while monotonically increasing, do not
	// necessarily reflect source-code order. For example, force builds
	// or rebuilds can allocate new, higher, numbers, but build an older-
	// than-HEAD version of the source.
	Number int32 `protobuf:"varint,3,opt,name=number" json:"number,omitempty"`
	// Verified identity which created this build.
	CreatedBy string `protobuf:"bytes,4,opt,name=created_by,json=createdBy" json:"created_by,omitempty"`
	// URL of a human-oriented build page.
	// Always populated.
	ViewUrl string `protobuf:"bytes,5,opt,name=view_url,json=viewUrl" json:"view_url,omitempty"`
	// When the build was created.
	CreateTime *google_protobuf.Timestamp `protobuf:"bytes,6,opt,name=create_time,json=createTime" json:"create_time,omitempty"`
	// When the build started.
	StartTime *google_protobuf.Timestamp `protobuf:"bytes,7,opt,name=start_time,json=startTime" json:"start_time,omitempty"`
	// When the build ended.
	EndTime *google_protobuf.Timestamp `protobuf:"bytes,8,opt,name=end_time,json=endTime" json:"end_time,omitempty"`
	// When the build was most recently updated.
	//
	// RPC: can be > end_time if, e.g. new tags were attached to a completed
	// build.
	UpdateTime *google_protobuf.Timestamp `protobuf:"bytes,9,opt,name=update_time,json=updateTime" json:"update_time,omitempty"`
	// Status of the build.
	// Must be specified, i.e. not STATUS_UNSPECIFIED.
	//
	// RPC: Responses have most current status.
	//
	// BigQuery: Final status of the build. Cannot be SCHEDULED or STARTED.
	Status Status `protobuf:"varint,12,opt,name=status,enum=buildbucket.v2.Status" json:"status,omitempty"`
	// Explanation of the current status.
	//
	// Types that are valid to be assigned to StatusReason:
	//	*Build_InfraFailureReason
	//	*Build_CancelReason
	StatusReason isBuild_StatusReason `protobuf_oneof:"status_reason"`
	// Input to the build script / recipe.
	Input *Build_Input `protobuf:"bytes,15,opt,name=input" json:"input,omitempty"`
	// Output of the build script / recipe.
	// SHOULD depend only on input field and NOT other fields.
	//
	// RPC: By default, this field is excluded from responses.
	// Updated while the build is running and finalized when the build ends.
	Output *Build_Output `protobuf:"bytes,16,opt,name=output" json:"output,omitempty"`
	// Current list of build steps.
	// Updated as build runs.
	//
	// RPC: By default, this field is excluded from responses.
	Steps []*Step `protobuf:"bytes,17,rep,name=steps" json:"steps,omitempty"`
	// Build infrastructure used by the build.
	//
	// RPC: By default, this field is excluded from responses.
	Infra *BuildInfra `protobuf:"bytes,18,opt,name=infra" json:"infra,omitempty"`
	// Arbitrary annotations for the build.
	// One key may have multiple values, which is why this is not a map<string,string>.
	// Indexed by the server, see also BuildFilter.tags.
	Tags []*StringPair `protobuf:"bytes,19,rep,name=tags" json:"tags,omitempty"`
}

func (m *Build) Reset()                    { *m = Build{} }
func (m *Build) String() string            { return proto.CompactTextString(m) }
func (*Build) ProtoMessage()               {}
func (*Build) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type isBuild_StatusReason interface {
	isBuild_StatusReason()
}

type Build_InfraFailureReason struct {
	InfraFailureReason *InfraFailureReason `protobuf:"bytes,13,opt,name=infra_failure_reason,json=infraFailureReason,oneof"`
}
type Build_CancelReason struct {
	CancelReason *CancelReason `protobuf:"bytes,14,opt,name=cancel_reason,json=cancelReason,oneof"`
}

func (*Build_InfraFailureReason) isBuild_StatusReason() {}
func (*Build_CancelReason) isBuild_StatusReason()       {}

func (m *Build) GetStatusReason() isBuild_StatusReason {
	if m != nil {
		return m.StatusReason
	}
	return nil
}

func (m *Build) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Build) GetBuilder() *Builder_ID {
	if m != nil {
		return m.Builder
	}
	return nil
}

func (m *Build) GetNumber() int32 {
	if m != nil {
		return m.Number
	}
	return 0
}

func (m *Build) GetCreatedBy() string {
	if m != nil {
		return m.CreatedBy
	}
	return ""
}

func (m *Build) GetViewUrl() string {
	if m != nil {
		return m.ViewUrl
	}
	return ""
}

func (m *Build) GetCreateTime() *google_protobuf.Timestamp {
	if m != nil {
		return m.CreateTime
	}
	return nil
}

func (m *Build) GetStartTime() *google_protobuf.Timestamp {
	if m != nil {
		return m.StartTime
	}
	return nil
}

func (m *Build) GetEndTime() *google_protobuf.Timestamp {
	if m != nil {
		return m.EndTime
	}
	return nil
}

func (m *Build) GetUpdateTime() *google_protobuf.Timestamp {
	if m != nil {
		return m.UpdateTime
	}
	return nil
}

func (m *Build) GetStatus() Status {
	if m != nil {
		return m.Status
	}
	return Status_STATUS_UNSPECIFIED
}

func (m *Build) GetInfraFailureReason() *InfraFailureReason {
	if x, ok := m.GetStatusReason().(*Build_InfraFailureReason); ok {
		return x.InfraFailureReason
	}
	return nil
}

func (m *Build) GetCancelReason() *CancelReason {
	if x, ok := m.GetStatusReason().(*Build_CancelReason); ok {
		return x.CancelReason
	}
	return nil
}

func (m *Build) GetInput() *Build_Input {
	if m != nil {
		return m.Input
	}
	return nil
}

func (m *Build) GetOutput() *Build_Output {
	if m != nil {
		return m.Output
	}
	return nil
}

func (m *Build) GetSteps() []*Step {
	if m != nil {
		return m.Steps
	}
	return nil
}

func (m *Build) GetInfra() *BuildInfra {
	if m != nil {
		return m.Infra
	}
	return nil
}

func (m *Build) GetTags() []*StringPair {
	if m != nil {
		return m.Tags
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Build) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Build_OneofMarshaler, _Build_OneofUnmarshaler, _Build_OneofSizer, []interface{}{
		(*Build_InfraFailureReason)(nil),
		(*Build_CancelReason)(nil),
	}
}

func _Build_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Build)
	// status_reason
	switch x := m.StatusReason.(type) {
	case *Build_InfraFailureReason:
		b.EncodeVarint(13<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.InfraFailureReason); err != nil {
			return err
		}
	case *Build_CancelReason:
		b.EncodeVarint(14<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.CancelReason); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("Build.StatusReason has unexpected type %T", x)
	}
	return nil
}

func _Build_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Build)
	switch tag {
	case 13: // status_reason.infra_failure_reason
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(InfraFailureReason)
		err := b.DecodeMessage(msg)
		m.StatusReason = &Build_InfraFailureReason{msg}
		return true, err
	case 14: // status_reason.cancel_reason
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(CancelReason)
		err := b.DecodeMessage(msg)
		m.StatusReason = &Build_CancelReason{msg}
		return true, err
	default:
		return false, nil
	}
}

func _Build_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Build)
	// status_reason
	switch x := m.StatusReason.(type) {
	case *Build_InfraFailureReason:
		s := proto.Size(x.InfraFailureReason)
		n += proto.SizeVarint(13<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Build_CancelReason:
		s := proto.Size(x.CancelReason)
		n += proto.SizeVarint(14<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// Defines what to build/test.
type Build_Input struct {
	// Arbitrary JSON object. Available at build run time.
	//
	// RPC: By default, this field is excluded from responses.
	//
	// V1 equivalent: corresponds to "properties" key in "parameters_json".
	Properties *google_protobuf1.Struct `protobuf:"bytes,1,opt,name=properties" json:"properties,omitempty"`
	// Gitiles commits to run against.
	// Usually present in CI builds, set by LUCI Scheduler.
	// If not present, the build may checkout "refs/heads/master".
	// NOT a blamelist.
	//
	// V1 equivalent: supersedes "revision" property and "buildset"
	// tag that starts with "commit/gitiles/".
	GitilesCommit *GitilesCommit `protobuf:"bytes,2,opt,name=gitiles_commit,json=gitilesCommit" json:"gitiles_commit,omitempty"`
	// Gerrit patchsets to run against.
	// Usually present in tryjobs, set by CQ, Gerrit, git-cl-try.
	// Applied on top of gitiles_commit if specified, otherwise tip of the tree.
	//
	// V1 equivalent: supersedes patch_* properties and "buildset"
	// tag that starts with "patch/gerrit/".
	GerritChanges []*GerritChange `protobuf:"bytes,3,rep,name=gerrit_changes,json=gerritChanges" json:"gerrit_changes,omitempty"`
	// If true, the build does not affect prod. In recipe land, runtime.is_experimental will
	// return true and recipes should not make prod-visible side effects.
	// By default, experimental builds are not surfaced in RPCs, PubSub
	// notifications (unless it is callback), and reported in metrics / BigQuery tables
	// under a different name.
	// See also include_experimental fields to in request messages.
	Experimental bool `protobuf:"varint,5,opt,name=experimental" json:"experimental,omitempty"`
}

func (m *Build_Input) Reset()                    { *m = Build_Input{} }
func (m *Build_Input) String() string            { return proto.CompactTextString(m) }
func (*Build_Input) ProtoMessage()               {}
func (*Build_Input) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

func (m *Build_Input) GetProperties() *google_protobuf1.Struct {
	if m != nil {
		return m.Properties
	}
	return nil
}

func (m *Build_Input) GetGitilesCommit() *GitilesCommit {
	if m != nil {
		return m.GitilesCommit
	}
	return nil
}

func (m *Build_Input) GetGerritChanges() []*GerritChange {
	if m != nil {
		return m.GerritChanges
	}
	return nil
}

func (m *Build_Input) GetExperimental() bool {
	if m != nil {
		return m.Experimental
	}
	return false
}

// Output of the build script / recipe.
type Build_Output struct {
	// Arbitrary JSON object produced by the build.
	//
	// V1 equivalent: corresponds to "properties" key in
	// "result_details_json".
	// In V1 output properties are not populated until build ends.
	Properties *google_protobuf1.Struct `protobuf:"bytes,1,opt,name=properties" json:"properties,omitempty"`
	// Human-oriented summary of the build provided by the build itself,
	// in Markdown format (https://spec.commonmark.org/0.28/).
	//
	// BigQuery: excluded from rows.
	SummaryMarkdown string `protobuf:"bytes,2,opt,name=summary_markdown,json=summaryMarkdown" json:"summary_markdown,omitempty"`
}

func (m *Build_Output) Reset()                    { *m = Build_Output{} }
func (m *Build_Output) String() string            { return proto.CompactTextString(m) }
func (*Build_Output) ProtoMessage()               {}
func (*Build_Output) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 1} }

func (m *Build_Output) GetProperties() *google_protobuf1.Struct {
	if m != nil {
		return m.Properties
	}
	return nil
}

func (m *Build_Output) GetSummaryMarkdown() string {
	if m != nil {
		return m.SummaryMarkdown
	}
	return ""
}

// Explains why status is CANCELED.
type CancelReason struct {
	// Human-oriented reasoning.
	Message string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	// Verified identity who canceled this build.
	CanceledBy string `protobuf:"bytes,2,opt,name=canceled_by,json=canceledBy" json:"canceled_by,omitempty"`
}

func (m *CancelReason) Reset()                    { *m = CancelReason{} }
func (m *CancelReason) String() string            { return proto.CompactTextString(m) }
func (*CancelReason) ProtoMessage()               {}
func (*CancelReason) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *CancelReason) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *CancelReason) GetCanceledBy() string {
	if m != nil {
		return m.CanceledBy
	}
	return ""
}

// Explains why status is INFRA_FAILURE.
type InfraFailureReason struct {
	// Human-oriented explanation of the infrastructure failure.
	Message string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
	// Indicates that the failure was due to a resource exhaustion / quota denial.
	ResourceExhaustion bool `protobuf:"varint,2,opt,name=resource_exhaustion,json=resourceExhaustion" json:"resource_exhaustion,omitempty"`
}

func (m *InfraFailureReason) Reset()                    { *m = InfraFailureReason{} }
func (m *InfraFailureReason) String() string            { return proto.CompactTextString(m) }
func (*InfraFailureReason) ProtoMessage()               {}
func (*InfraFailureReason) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *InfraFailureReason) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *InfraFailureReason) GetResourceExhaustion() bool {
	if m != nil {
		return m.ResourceExhaustion
	}
	return false
}

// Build infrastructure that was used for a particular build.
type BuildInfra struct {
	Buildbucket *BuildInfra_Buildbucket `protobuf:"bytes,1,opt,name=buildbucket" json:"buildbucket,omitempty"`
	Swarming    *BuildInfra_Swarming    `protobuf:"bytes,2,opt,name=swarming" json:"swarming,omitempty"`
	Logdog      *BuildInfra_LogDog      `protobuf:"bytes,3,opt,name=logdog" json:"logdog,omitempty"`
}

func (m *BuildInfra) Reset()                    { *m = BuildInfra{} }
func (m *BuildInfra) String() string            { return proto.CompactTextString(m) }
func (*BuildInfra) ProtoMessage()               {}
func (*BuildInfra) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *BuildInfra) GetBuildbucket() *BuildInfra_Buildbucket {
	if m != nil {
		return m.Buildbucket
	}
	return nil
}

func (m *BuildInfra) GetSwarming() *BuildInfra_Swarming {
	if m != nil {
		return m.Swarming
	}
	return nil
}

func (m *BuildInfra) GetLogdog() *BuildInfra_LogDog {
	if m != nil {
		return m.Logdog
	}
	return nil
}

// Buildbucket-specific information, captured at the build creation time.
type BuildInfra_Buildbucket struct {
	// Version of swarming task template. Defines
	// versions of kitchen, git, git wrapper, python, vpython, etc.
	ServiceConfigRevision string `protobuf:"bytes,2,opt,name=service_config_revision,json=serviceConfigRevision" json:"service_config_revision,omitempty"`
	// Whether canary version of the swarming task template was used for this
	// build.
	Canary bool `protobuf:"varint,4,opt,name=canary" json:"canary,omitempty"`
}

func (m *BuildInfra_Buildbucket) Reset()                    { *m = BuildInfra_Buildbucket{} }
func (m *BuildInfra_Buildbucket) String() string            { return proto.CompactTextString(m) }
func (*BuildInfra_Buildbucket) ProtoMessage()               {}
func (*BuildInfra_Buildbucket) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 0} }

func (m *BuildInfra_Buildbucket) GetServiceConfigRevision() string {
	if m != nil {
		return m.ServiceConfigRevision
	}
	return ""
}

func (m *BuildInfra_Buildbucket) GetCanary() bool {
	if m != nil {
		return m.Canary
	}
	return false
}

// Swarming-specific information.
type BuildInfra_Swarming struct {
	// Swarming hostname, e.g. "chromium-swarm.appspot.com".
	// Populated at the build creation time.
	Hostname string `protobuf:"bytes,1,opt,name=hostname" json:"hostname,omitempty"`
	// Swarming task id.
	// Not guaranteed to be populated at the build creation time.
	TaskId string `protobuf:"bytes,2,opt,name=task_id,json=taskId" json:"task_id,omitempty"`
	// Task service account email address.
	// This is the service account used for all authenticated requests by the
	// build.
	TaskServiceAccount string `protobuf:"bytes,3,opt,name=task_service_account,json=taskServiceAccount" json:"task_service_account,omitempty"`
	// Priority of the task. The lower the more important.
	// Valid values are [1..255].
	Priority int32 `protobuf:"varint,4,opt,name=priority" json:"priority,omitempty"`
	// Swarming dimensions for the task.
	TaskDimensions []*StringPair `protobuf:"bytes,5,rep,name=task_dimensions,json=taskDimensions" json:"task_dimensions,omitempty"`
	// Swarming dimensions of the bot used for the task.
	BotDimensions []*StringPair `protobuf:"bytes,6,rep,name=bot_dimensions,json=botDimensions" json:"bot_dimensions,omitempty"`
}

func (m *BuildInfra_Swarming) Reset()                    { *m = BuildInfra_Swarming{} }
func (m *BuildInfra_Swarming) String() string            { return proto.CompactTextString(m) }
func (*BuildInfra_Swarming) ProtoMessage()               {}
func (*BuildInfra_Swarming) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 1} }

func (m *BuildInfra_Swarming) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *BuildInfra_Swarming) GetTaskId() string {
	if m != nil {
		return m.TaskId
	}
	return ""
}

func (m *BuildInfra_Swarming) GetTaskServiceAccount() string {
	if m != nil {
		return m.TaskServiceAccount
	}
	return ""
}

func (m *BuildInfra_Swarming) GetPriority() int32 {
	if m != nil {
		return m.Priority
	}
	return 0
}

func (m *BuildInfra_Swarming) GetTaskDimensions() []*StringPair {
	if m != nil {
		return m.TaskDimensions
	}
	return nil
}

func (m *BuildInfra_Swarming) GetBotDimensions() []*StringPair {
	if m != nil {
		return m.BotDimensions
	}
	return nil
}

// LogDog-specific information.
type BuildInfra_LogDog struct {
	// LogDog hostname, e.g. "logs.chromium.org".
	Hostname string `protobuf:"bytes,1,opt,name=hostname" json:"hostname,omitempty"`
	// LogDog project, e.g. "chromium".
	// Typically matches Build.builder.project.
	Project string `protobuf:"bytes,2,opt,name=project" json:"project,omitempty"`
	// A slash-separated path prefix shared by all logs and artifacts of this
	// build.
	// No other build can have the same prefix.
	// Can be used to discover logs and/or load log contents.
	//
	// Can be used to construct URL of a page that displays stdout/stderr of all
	// steps of a build. In pseudo-JS:
	//   q_stdout = `${project}/${prefix}/+/**/stdout`;
	//   q_stderr = `${project}/${prefix}/+/**/stderr`;
	//   url = `https://${host}/v/?s=${urlquote(q_stdout)}&s=${urlquote(q_stderr)}`;
	Prefix string `protobuf:"bytes,3,opt,name=prefix" json:"prefix,omitempty"`
}

func (m *BuildInfra_LogDog) Reset()                    { *m = BuildInfra_LogDog{} }
func (m *BuildInfra_LogDog) String() string            { return proto.CompactTextString(m) }
func (*BuildInfra_LogDog) ProtoMessage()               {}
func (*BuildInfra_LogDog) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 2} }

func (m *BuildInfra_LogDog) GetHostname() string {
	if m != nil {
		return m.Hostname
	}
	return ""
}

func (m *BuildInfra_LogDog) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *BuildInfra_LogDog) GetPrefix() string {
	if m != nil {
		return m.Prefix
	}
	return ""
}

type Builder struct {
}

func (m *Builder) Reset()                    { *m = Builder{} }
func (m *Builder) String() string            { return proto.CompactTextString(m) }
func (*Builder) ProtoMessage()               {}
func (*Builder) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

// Identifies a builder.
// Canonical string representation: “{project}/{bucket}/{builder}”.
type Builder_ID struct {
	// Project ID, e.g. "chromium". Unique within a LUCI deployment.
	Project string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	// Bucket name, e.g. "try". Unique within the project.
	// Together with project, defines an ACL.
	Bucket string `protobuf:"bytes,2,opt,name=bucket" json:"bucket,omitempty"`
	// Builder name, e.g. "linux-rel". Unique within the bucket.
	Builder string `protobuf:"bytes,3,opt,name=builder" json:"builder,omitempty"`
}

func (m *Builder_ID) Reset()                    { *m = Builder_ID{} }
func (m *Builder_ID) String() string            { return proto.CompactTextString(m) }
func (*Builder_ID) ProtoMessage()               {}
func (*Builder_ID) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4, 0} }

func (m *Builder_ID) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *Builder_ID) GetBucket() string {
	if m != nil {
		return m.Bucket
	}
	return ""
}

func (m *Builder_ID) GetBuilder() string {
	if m != nil {
		return m.Builder
	}
	return ""
}

func init() {
	proto.RegisterType((*Build)(nil), "buildbucket.v2.Build")
	proto.RegisterType((*Build_Input)(nil), "buildbucket.v2.Build.Input")
	proto.RegisterType((*Build_Output)(nil), "buildbucket.v2.Build.Output")
	proto.RegisterType((*CancelReason)(nil), "buildbucket.v2.CancelReason")
	proto.RegisterType((*InfraFailureReason)(nil), "buildbucket.v2.InfraFailureReason")
	proto.RegisterType((*BuildInfra)(nil), "buildbucket.v2.BuildInfra")
	proto.RegisterType((*BuildInfra_Buildbucket)(nil), "buildbucket.v2.BuildInfra.Buildbucket")
	proto.RegisterType((*BuildInfra_Swarming)(nil), "buildbucket.v2.BuildInfra.Swarming")
	proto.RegisterType((*BuildInfra_LogDog)(nil), "buildbucket.v2.BuildInfra.LogDog")
	proto.RegisterType((*Builder)(nil), "buildbucket.v2.Builder")
	proto.RegisterType((*Builder_ID)(nil), "buildbucket.v2.Builder.ID")
}

func init() { proto.RegisterFile("go.chromium.org/luci/buildbucket/proto/build.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 1002 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x56, 0xdf, 0x6e, 0xdb, 0xb6,
	0x17, 0xfe, 0x39, 0xa9, 0x65, 0xfb, 0x38, 0x76, 0xfa, 0x63, 0xb3, 0x46, 0xd3, 0x5a, 0xd4, 0xf3,
	0x80, 0xc1, 0xdb, 0x85, 0xd2, 0xba, 0x59, 0x87, 0x22, 0x17, 0x43, 0xed, 0x6c, 0xad, 0x81, 0x0d,
	0x2b, 0x98, 0xad, 0x17, 0x1b, 0x06, 0x81, 0x96, 0x68, 0x85, 0x8b, 0x24, 0x0a, 0x24, 0x95, 0x3f,
	0x0f, 0xb2, 0x07, 0xd8, 0x9b, 0xec, 0x8d, 0xf6, 0x0a, 0x83, 0x48, 0xca, 0x51, 0xed, 0xce, 0x0e,
	0x76, 0xa7, 0x73, 0xce, 0xf7, 0x7d, 0x3c, 0xfc, 0x78, 0x28, 0x09, 0xc6, 0x31, 0xf7, 0xc3, 0x73,
	0xc1, 0x53, 0x56, 0xa4, 0x3e, 0x17, 0xf1, 0x51, 0x52, 0x84, 0xec, 0x68, 0x5e, 0xb0, 0x24, 0x9a,
	0x17, 0xe1, 0x05, 0x55, 0x47, 0xb9, 0xe0, 0x8a, 0x9b, 0x8c, 0xaf, 0x9f, 0x51, 0xbf, 0x56, 0xf6,
	0x2f, 0xc7, 0xde, 0x93, 0x98, 0xf3, 0x38, 0xa1, 0x06, 0x39, 0x2f, 0x16, 0x47, 0x8a, 0xa5, 0x54,
	0x2a, 0x92, 0xe6, 0x86, 0xe0, 0x3d, 0x5a, 0x05, 0x48, 0x25, 0x8a, 0x50, 0xd9, 0xea, 0xf3, 0x3b,
	0xb6, 0x10, 0xf2, 0x34, 0xe5, 0x99, 0x25, 0x3d, 0xbb, 0x23, 0x49, 0x2a, 0x6a, 0xbb, 0x18, 0xfe,
	0xd5, 0x81, 0xe6, 0xa4, 0x04, 0xa0, 0x3e, 0xec, 0xb0, 0xc8, 0x6d, 0x0c, 0x1a, 0xa3, 0x5d, 0xbc,
	0xc3, 0x22, 0x74, 0x0c, 0x2d, 0xcd, 0xa4, 0xc2, 0xdd, 0x19, 0x34, 0x46, 0xdd, 0xb1, 0xe7, 0xbf,
	0xbf, 0x45, 0x7f, 0x62, 0xca, 0xfe, 0xec, 0x14, 0x57, 0x50, 0xf4, 0x10, 0x9c, 0xac, 0x48, 0xe7,
	0x54, 0xb8, 0xbb, 0x83, 0xc6, 0xa8, 0x89, 0x6d, 0x84, 0x1e, 0x03, 0x84, 0x82, 0x12, 0x45, 0xa3,
	0x60, 0x7e, 0xe3, 0xde, 0x1b, 0x34, 0x46, 0x1d, 0xdc, 0xb1, 0x99, 0xc9, 0x0d, 0xfa, 0x18, 0xda,
	0x97, 0x8c, 0x5e, 0x05, 0x85, 0x48, 0xdc, 0xa6, 0x2e, 0xb6, 0xca, 0xf8, 0x67, 0x91, 0xa0, 0x13,
	0xe8, 0x1a, 0x5c, 0x50, 0x3a, 0xe8, 0x3a, 0xb6, 0x17, 0xe3, 0x9e, 0x5f, 0xb9, 0xe7, 0xff, 0x54,
	0xd9, 0x8b, 0xed, 0x42, 0x65, 0x02, 0xbd, 0x04, 0x90, 0x8a, 0x08, 0x65, 0xb8, 0xad, 0xad, 0xdc,
	0x8e, 0x46, 0x6b, 0xea, 0x57, 0xd0, 0xa6, 0x59, 0x64, 0x88, 0xed, 0xad, 0xc4, 0x16, 0xcd, 0x22,
	0x4d, 0x3b, 0x81, 0x6e, 0x91, 0x47, 0xcb, 0x76, 0x3b, 0xdb, 0xdb, 0x35, 0x70, 0x4d, 0xf6, 0xc1,
	0x91, 0x8a, 0xa8, 0x42, 0xba, 0x7b, 0x83, 0xc6, 0xa8, 0x3f, 0x7e, 0xb8, 0x6a, 0xf9, 0x99, 0xae,
	0x62, 0x8b, 0x42, 0xef, 0xe0, 0x80, 0x65, 0x0b, 0x41, 0x82, 0x05, 0x61, 0x49, 0x21, 0x68, 0x20,
	0x28, 0x91, 0x3c, 0x73, 0x7b, 0x7a, 0xd5, 0xe1, 0x2a, 0x7b, 0x56, 0x62, 0xbf, 0x33, 0x50, 0xac,
	0x91, 0x6f, 0xfe, 0x87, 0x11, 0x5b, 0xcb, 0xa2, 0x29, 0xf4, 0x42, 0x92, 0x85, 0x34, 0xa9, 0x04,
	0xfb, 0x5a, 0xf0, 0xd1, 0xaa, 0xe0, 0x54, 0x83, 0x96, 0x52, 0x7b, 0x61, 0x2d, 0x46, 0xcf, 0xa0,
	0xc9, 0xb2, 0xbc, 0x50, 0xee, 0xbe, 0x26, 0x7f, 0xf2, 0xc1, 0xf1, 0xf1, 0x67, 0x25, 0x04, 0x1b,
	0x24, 0x3a, 0x06, 0x87, 0x17, 0xaa, 0xe4, 0xdc, 0xff, 0xf0, 0x82, 0x86, 0xf3, 0xa3, 0xc6, 0x60,
	0x8b, 0x45, 0x5f, 0x42, 0xb3, 0x9c, 0x68, 0xe9, 0xfe, 0x7f, 0xb0, 0x3b, 0xea, 0x8e, 0x0f, 0xd6,
	0x4d, 0xa3, 0x39, 0x36, 0x10, 0xf4, 0xb4, 0x6c, 0x6a, 0x21, 0x88, 0x8b, 0x36, 0xcc, 0xb4, 0xf6,
	0x09, 0x1b, 0x20, 0xf2, 0xe1, 0x9e, 0x22, 0xb1, 0x74, 0x1f, 0x68, 0x71, 0x6f, 0x5d, 0x5c, 0xb0,
	0x2c, 0x7e, 0x4b, 0x98, 0xc0, 0x1a, 0xe7, 0xfd, 0xdd, 0x80, 0xa6, 0xde, 0x14, 0xfa, 0x1a, 0x20,
	0x17, 0x3c, 0xa7, 0x42, 0x31, 0x2a, 0xf5, 0xcd, 0xea, 0x8e, 0x0f, 0xd7, 0x26, 0xe1, 0x4c, 0x5f,
	0x7b, 0x5c, 0x83, 0xa2, 0x53, 0xe8, 0xc7, 0x4c, 0xb1, 0x84, 0xca, 0xa0, 0xbc, 0xdf, 0x4c, 0xd9,
	0x1b, 0xf8, 0x78, 0x75, 0xf1, 0xd7, 0x06, 0x35, 0xd5, 0x20, 0xdc, 0x8b, 0xeb, 0x21, 0x9a, 0x42,
	0x3f, 0xa6, 0x42, 0x30, 0x15, 0x84, 0xe7, 0x24, 0x8b, 0xa9, 0x74, 0x77, 0xf5, 0x16, 0xd6, 0x4c,
	0x7d, 0xad, 0x51, 0x53, 0x0d, 0xc2, 0xbd, 0xb8, 0x16, 0x49, 0x34, 0x84, 0x3d, 0x7a, 0x9d, 0x53,
	0xc1, 0x52, 0x9a, 0x29, 0x62, 0x2e, 0x67, 0x1b, 0xbf, 0x97, 0xf3, 0x12, 0x70, 0xcc, 0x89, 0xfc,
	0xf7, 0x1d, 0x7f, 0x01, 0xf7, 0x65, 0x91, 0xa6, 0x44, 0xdc, 0x04, 0x29, 0x11, 0x17, 0x11, 0xbf,
	0xca, 0xf4, 0x9e, 0x3b, 0x78, 0xdf, 0xe6, 0x7f, 0xb0, 0xe9, 0xc9, 0x3e, 0xf4, 0xcc, 0xf4, 0xdb,
	0xd9, 0x1c, 0xce, 0x60, 0xaf, 0x3e, 0x87, 0xc8, 0x85, 0x56, 0x4a, 0xa5, 0x24, 0x31, 0xd5, 0x1d,
	0x74, 0x70, 0x15, 0xa2, 0x27, 0xd0, 0x35, 0x13, 0x6a, 0xde, 0x42, 0x66, 0x01, 0xa8, 0x52, 0x93,
	0x9b, 0x61, 0x00, 0x68, 0xfd, 0x8e, 0x6c, 0x10, 0x3c, 0x82, 0x07, 0x82, 0x4a, 0x5e, 0x88, 0x90,
	0x06, 0xf4, 0xfa, 0x9c, 0x14, 0x52, 0x31, 0x6e, 0x3a, 0x6f, 0x63, 0x54, 0x95, 0xbe, 0x5d, 0x56,
	0x86, 0x7f, 0x36, 0x01, 0x6e, 0x47, 0x0c, 0xbd, 0x81, 0x6e, 0xed, 0x2c, 0xac, 0x61, 0x9f, 0xff,
	0xfb, 0x4c, 0x9a, 0x47, 0x53, 0xc1, 0x75, 0x2a, 0xfa, 0x06, 0xda, 0xf2, 0x8a, 0x88, 0x94, 0x65,
	0xb1, 0x1d, 0x96, 0xcf, 0x36, 0xc8, 0x9c, 0x59, 0x28, 0x5e, 0x92, 0xd0, 0x4b, 0x70, 0x12, 0x1e,
	0x47, 0x3c, 0xd6, 0x2f, 0xee, 0xee, 0xf8, 0xd3, 0x0d, 0xf4, 0xef, 0x79, 0x7c, 0xca, 0x63, 0x6c,
	0x09, 0xde, 0x6f, 0xd0, 0xad, 0xf5, 0x85, 0x5e, 0xc0, 0xa1, 0xa4, 0xe2, 0x92, 0x85, 0x34, 0x08,
	0x79, 0xb6, 0x60, 0x71, 0x20, 0xe8, 0x25, 0x93, 0x95, 0x31, 0x1d, 0xfc, 0x91, 0x2d, 0x4f, 0x75,
	0x15, 0xdb, 0x62, 0xf9, 0xe9, 0x08, 0x49, 0x46, 0x84, 0xf9, 0x3c, 0xb4, 0xb1, 0x8d, 0xbc, 0x3f,
	0x76, 0xa0, 0x5d, 0x35, 0x8c, 0x3c, 0x68, 0x9f, 0x73, 0xa9, 0x32, 0x92, 0x56, 0x87, 0xb1, 0x8c,
	0xd1, 0x21, 0xb4, 0x14, 0x91, 0x17, 0x01, 0x8b, 0xec, 0x42, 0x4e, 0x19, 0xce, 0x22, 0xf4, 0x14,
	0x0e, 0x74, 0xa1, 0x6a, 0x8b, 0x84, 0x21, 0x2f, 0x32, 0xa5, 0x77, 0xda, 0xc1, 0xa8, 0xac, 0x9d,
	0x99, 0xd2, 0x2b, 0x53, 0x29, 0x97, 0xc9, 0x05, 0xe3, 0x82, 0x29, 0xd3, 0x4d, 0x13, 0x2f, 0x63,
	0x34, 0x85, 0x7d, 0xad, 0x16, 0x95, 0xf3, 0x5f, 0x76, 0x2e, 0xdd, 0xe6, 0xd6, 0x77, 0x43, 0xbf,
	0xa4, 0x9c, 0x2e, 0x19, 0xe8, 0x15, 0xf4, 0xe7, 0x5c, 0xd5, 0x35, 0x9c, 0xad, 0x1a, 0xbd, 0x39,
	0x57, 0xb7, 0x12, 0xde, 0x3b, 0x70, 0xcc, 0x41, 0x6c, 0x34, 0xc5, 0x85, 0x56, 0x2e, 0xf8, 0xef,
	0x34, 0x54, 0xd6, 0x94, 0x2a, 0x2c, 0xfd, 0xce, 0x05, 0x5d, 0xb0, 0x6b, 0xeb, 0x83, 0x8d, 0x86,
	0xbf, 0x42, 0xcb, 0x7e, 0xd9, 0xbd, 0xb7, 0xb0, 0x33, 0x3b, 0xad, 0x4b, 0x34, 0xd6, 0x24, 0xec,
	0xe8, 0x5a, 0xc3, 0xed, 0x08, 0xb8, 0xb7, 0xff, 0x0e, 0x46, 0xbb, 0x0a, 0x27, 0x2f, 0x7e, 0x39,
	0xbe, 0xdb, 0x4f, 0xca, 0x49, 0x2d, 0x93, 0xcf, 0xe7, 0x8e, 0x4e, 0x3e, 0xff, 0x27, 0x00, 0x00,
	0xff, 0xff, 0x59, 0x70, 0xaf, 0xd2, 0x9b, 0x09, 0x00, 0x00,
}
