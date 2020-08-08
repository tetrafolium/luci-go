// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/cq/api/bigquery/attempt.proto

package bigquery

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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

type Mode int32

const (
	// Default, never set.
	Mode_MODE_UNSPECIFIED Mode = 0
	// Run all tests but do not submit.
	Mode_DRY_RUN Mode = 1
	// Run all tests and potentially submit.
	Mode_FULL_RUN Mode = 2
)

var Mode_name = map[int32]string{
	0: "MODE_UNSPECIFIED",
	1: "DRY_RUN",
	2: "FULL_RUN",
}

var Mode_value = map[string]int32{
	"MODE_UNSPECIFIED": 0,
	"DRY_RUN":          1,
	"FULL_RUN":         2,
}

func (x Mode) String() string {
	return proto.EnumName(Mode_name, int32(x))
}

func (Mode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{0}
}

type AttemptStatus int32

const (
	// Default, never set.
	AttemptStatus_ATTEMPT_STATUS_UNSPECIFIED AttemptStatus = 0
	// Started but not completed. Used by CQ API, TBD.
	AttemptStatus_STARTED AttemptStatus = 1
	// Ready to submit, all checks passed.
	AttemptStatus_SUCCESS AttemptStatus = 2
	// Attempt stopped before completion, due to some external event and not
	// a failure of the CLs to pass all tests. For example, this may happen
	// when a new patchset is uploaded, a CL is deleted, etc.
	AttemptStatus_ABORTED AttemptStatus = 3
	// Completed and failed some check. This may happen when a build failed,
	// footer syntax was incorrect, or CL was not approved.
	AttemptStatus_FAILURE AttemptStatus = 4
	// Failure in CQ itself caused the Attempt to be dropped.
	AttemptStatus_INFRA_FAILURE AttemptStatus = 5
)

var AttemptStatus_name = map[int32]string{
	0: "ATTEMPT_STATUS_UNSPECIFIED",
	1: "STARTED",
	2: "SUCCESS",
	3: "ABORTED",
	4: "FAILURE",
	5: "INFRA_FAILURE",
}

var AttemptStatus_value = map[string]int32{
	"ATTEMPT_STATUS_UNSPECIFIED": 0,
	"STARTED":                    1,
	"SUCCESS":                    2,
	"ABORTED":                    3,
	"FAILURE":                    4,
	"INFRA_FAILURE":              5,
}

func (x AttemptStatus) String() string {
	return proto.EnumName(AttemptStatus_name, int32(x))
}

func (AttemptStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{1}
}

type AttemptSubstatus int32

const (
	// Default, never set.
	AttemptSubstatus_ATTEMPT_SUBSTATUS_UNSPECIFIED AttemptSubstatus = 0
	// There is no more detailed status set.
	AttemptSubstatus_NO_SUBSTATUS AttemptSubstatus = 1
	// Failed at least one critical tryjob.
	AttemptSubstatus_FAILED_TRYJOBS AttemptSubstatus = 2
	// Failed an initial quick check of CL and CL description state.
	AttemptSubstatus_FAILED_LINT AttemptSubstatus = 3
	// A CL didn't get sufficient approval for submitting via CQ.
	AttemptSubstatus_UNAPPROVED AttemptSubstatus = 4
	// A CQ triggerer doesn't have permission to trigger CQ.
	AttemptSubstatus_PERMISSION_DENIED AttemptSubstatus = 5
	// There was a problem with a dependency CL, e.g. some dependencies
	// were not submitted or not grouped together in this attempt.
	AttemptSubstatus_UNSATISFIED_DEPENDENCY AttemptSubstatus = 6
	// Aborted because of a manual cancelation.
	AttemptSubstatus_MANUAL_CANCEL AttemptSubstatus = 7
	// A request to buildbucket failed because CQ didn't have permission to
	// trigger builds.
	AttemptSubstatus_BUILDBUCKET_MISCONFIGURATION AttemptSubstatus = 8
)

var AttemptSubstatus_name = map[int32]string{
	0: "ATTEMPT_SUBSTATUS_UNSPECIFIED",
	1: "NO_SUBSTATUS",
	2: "FAILED_TRYJOBS",
	3: "FAILED_LINT",
	4: "UNAPPROVED",
	5: "PERMISSION_DENIED",
	6: "UNSATISFIED_DEPENDENCY",
	7: "MANUAL_CANCEL",
	8: "BUILDBUCKET_MISCONFIGURATION",
}

var AttemptSubstatus_value = map[string]int32{
	"ATTEMPT_SUBSTATUS_UNSPECIFIED": 0,
	"NO_SUBSTATUS":                  1,
	"FAILED_TRYJOBS":                2,
	"FAILED_LINT":                   3,
	"UNAPPROVED":                    4,
	"PERMISSION_DENIED":             5,
	"UNSATISFIED_DEPENDENCY":        6,
	"MANUAL_CANCEL":                 7,
	"BUILDBUCKET_MISCONFIGURATION":  8,
}

func (x AttemptSubstatus) String() string {
	return proto.EnumName(AttemptSubstatus_name, int32(x))
}

func (AttemptSubstatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{2}
}

type GerritChange_SubmitStatus int32

const (
	// Default. Never set.
	GerritChange_SUBMIT_STATUS_UNSPECIFIED GerritChange_SubmitStatus = 0
	// CQ didn't try submitting this CL.
	//
	// Includes a case where CQ tried submitting the CL, but submission failed
	// due to transient error leaving CL as is, and CQ didn't try again.
	GerritChange_PENDING GerritChange_SubmitStatus = 1
	// CQ tried to submit, but got presumably transient errors and couldn't
	// ascertain whether submission was successful.
	//
	// It's possible that change was actually submitted, but CQ didn't receive
	// a confirmation from Gerrit and follow up checks of the change status
	// failed, too.
	GerritChange_UNKNOWN GerritChange_SubmitStatus = 2
	// CQ tried to submit, but Gerrit rejected the submission because this
	// Change can't be submitted.
	// Typically, this is because a rebase conflict needs to be resolved,
	// or rarely because the change needs some kind of approval.
	GerritChange_FAILURE GerritChange_SubmitStatus = 3
	// CQ submitted this change (aka "merged" in Gerrit jargon).
	//
	// Submission of Gerrit CLs in an Attempt is not an atomic operation,
	// so it's possible that only some of the GerritChanges are submitted.
	GerritChange_SUCCESS GerritChange_SubmitStatus = 4
)

var GerritChange_SubmitStatus_name = map[int32]string{
	0: "SUBMIT_STATUS_UNSPECIFIED",
	1: "PENDING",
	2: "UNKNOWN",
	3: "FAILURE",
	4: "SUCCESS",
}

var GerritChange_SubmitStatus_value = map[string]int32{
	"SUBMIT_STATUS_UNSPECIFIED": 0,
	"PENDING":                   1,
	"UNKNOWN":                   2,
	"FAILURE":                   3,
	"SUCCESS":                   4,
}

func (x GerritChange_SubmitStatus) String() string {
	return proto.EnumName(GerritChange_SubmitStatus_name, int32(x))
}

func (GerritChange_SubmitStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{1, 0}
}

type Build_Origin int32

const (
	// Default. Never set.
	Build_ORIGIN_UNSPECIFIED Build_Origin = 0
	// Build was triggered as part of this attempt
	// because reuse was disabled for its builder.
	Build_NOT_REUSABLE Build_Origin = 1
	// Build was triggered as part of this attempt,
	// but if there was an already existing build it would have been reused.
	Build_NOT_REUSED Build_Origin = 2
	// Build was reused.
	Build_REUSED Build_Origin = 3
)

var Build_Origin_name = map[int32]string{
	0: "ORIGIN_UNSPECIFIED",
	1: "NOT_REUSABLE",
	2: "NOT_REUSED",
	3: "REUSED",
}

var Build_Origin_value = map[string]int32{
	"ORIGIN_UNSPECIFIED": 0,
	"NOT_REUSABLE":       1,
	"NOT_REUSED":         2,
	"REUSED":             3,
}

func (x Build_Origin) String() string {
	return proto.EnumName(Build_Origin_name, int32(x))
}

func (Build_Origin) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{2, 0}
}

// Attempt includes the state of one CQ attempt.
//
// An attempt involves doing checks for one or more CLs that could
// potentially be submitted together.
//
// Next ID: 13.
type Attempt struct {
	// The opaque key unique to this Attempt.
	Key string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	// The LUCI project that this Attempt belongs to.
	LuciProject string `protobuf:"bytes,2,opt,name=luci_project,json=luciProject,proto3" json:"luci_project,omitempty"`
	// The name of the config group that this Attempt belongs to.
	ConfigGroup string `protobuf:"bytes,11,opt,name=config_group,json=configGroup,proto3" json:"config_group,omitempty"`
	// An opaque key that is unique for a given set of Gerrit change patchsets.
	// (or, equivalently, buildsets). The same cl_group_key will be used if
	// another Attempt is made for the same set of changes at a different time.
	ClGroupKey string `protobuf:"bytes,3,opt,name=cl_group_key,json=clGroupKey,proto3" json:"cl_group_key,omitempty"`
	// Similar to cl_group_key, except the key will be the same when the
	// earliest_equivalent_patchset values are the same, even if the patchset
	// values are different.
	//
	// For example, when a new "trivial" patchset is uploaded, then the
	// cl_group_key will change but the equivalent_cl_group_key will stay the
	// same.
	EquivalentClGroupKey string `protobuf:"bytes,4,opt,name=equivalent_cl_group_key,json=equivalentClGroupKey,proto3" json:"equivalent_cl_group_key,omitempty"`
	// The time when the Attempt started (trigger time of the last CL triggered).
	StartTime *timestamp.Timestamp `protobuf:"bytes,5,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	// The time when the Attempt ended (released by CQ).
	EndTime *timestamp.Timestamp `protobuf:"bytes,6,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
	// Gerrit changes, with specific patchsets, in this Attempt.
	// There should be one or more.
	GerritChanges []*GerritChange `protobuf:"bytes,7,rep,name=gerrit_changes,json=gerritChanges,proto3" json:"gerrit_changes,omitempty"`
	// Relevant builds as of this Attempt's end time.
	//
	// While Attempt is processed, CQ may consider more builds than included here.
	//
	// For example, the following builds will be not be included:
	//   * builds triggered before this Attempt started, considered temporarily by
	//     CQ, but then ignored because they ultimately failed such that CQ had to
	//     trigger new builds instead.
	//   * successful builds which were fresh enough at the Attempt start time,
	//     but which were ignored after they became too old for consideration such
	//     that CQ had to trigger new builds instead.
	//   * builds triggered as part of this Attempt, which were later removed from
	//     project CQ config and hence were no longer required by CQ by Attempt
	//     end time.
	Builds []*Build `protobuf:"bytes,8,rep,name=builds,proto3" json:"builds,omitempty"`
	// Final status of the Attempt.
	Status AttemptStatus `protobuf:"varint,9,opt,name=status,proto3,enum=bigquery.AttemptStatus" json:"status,omitempty"`
	// A more fine-grained status the explains more details about the status.
	Substatus AttemptSubstatus `protobuf:"varint,10,opt,name=substatus,proto3,enum=bigquery.AttemptSubstatus" json:"substatus,omitempty"`
	// Whether or not the required builds for this attempt include additional
	// "opted-in" builders by the user via the `Cq-Include-Trybots` footer.
	HasCustomRequirement bool     `protobuf:"varint,12,opt,name=has_custom_requirement,json=hasCustomRequirement,proto3" json:"has_custom_requirement,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Attempt) Reset()         { *m = Attempt{} }
func (m *Attempt) String() string { return proto.CompactTextString(m) }
func (*Attempt) ProtoMessage()    {}
func (*Attempt) Descriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{0}
}

func (m *Attempt) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Attempt.Unmarshal(m, b)
}
func (m *Attempt) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Attempt.Marshal(b, m, deterministic)
}
func (m *Attempt) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Attempt.Merge(m, src)
}
func (m *Attempt) XXX_Size() int {
	return xxx_messageInfo_Attempt.Size(m)
}
func (m *Attempt) XXX_DiscardUnknown() {
	xxx_messageInfo_Attempt.DiscardUnknown(m)
}

var xxx_messageInfo_Attempt proto.InternalMessageInfo

func (m *Attempt) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *Attempt) GetLuciProject() string {
	if m != nil {
		return m.LuciProject
	}
	return ""
}

func (m *Attempt) GetConfigGroup() string {
	if m != nil {
		return m.ConfigGroup
	}
	return ""
}

func (m *Attempt) GetClGroupKey() string {
	if m != nil {
		return m.ClGroupKey
	}
	return ""
}

func (m *Attempt) GetEquivalentClGroupKey() string {
	if m != nil {
		return m.EquivalentClGroupKey
	}
	return ""
}

func (m *Attempt) GetStartTime() *timestamp.Timestamp {
	if m != nil {
		return m.StartTime
	}
	return nil
}

func (m *Attempt) GetEndTime() *timestamp.Timestamp {
	if m != nil {
		return m.EndTime
	}
	return nil
}

func (m *Attempt) GetGerritChanges() []*GerritChange {
	if m != nil {
		return m.GerritChanges
	}
	return nil
}

func (m *Attempt) GetBuilds() []*Build {
	if m != nil {
		return m.Builds
	}
	return nil
}

func (m *Attempt) GetStatus() AttemptStatus {
	if m != nil {
		return m.Status
	}
	return AttemptStatus_ATTEMPT_STATUS_UNSPECIFIED
}

func (m *Attempt) GetSubstatus() AttemptSubstatus {
	if m != nil {
		return m.Substatus
	}
	return AttemptSubstatus_ATTEMPT_SUBSTATUS_UNSPECIFIED
}

func (m *Attempt) GetHasCustomRequirement() bool {
	if m != nil {
		return m.HasCustomRequirement
	}
	return false
}

// GerritChange represents one revision (patchset) of one Gerrit change
// in an Attempt.
//
// See also: GerritChange in buildbucket/proto/common.proto.
type GerritChange struct {
	// Gerrit hostname, e.g. "chromium-review.googlesource.com".
	Host string `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	// Gerrit project, e.g. "chromium/src".
	Project string `protobuf:"bytes,2,opt,name=project,proto3" json:"project,omitempty"`
	// Change number, e.g. 12345.
	Change int64 `protobuf:"varint,3,opt,name=change,proto3" json:"change,omitempty"`
	// Patch set number, e.g. 1.
	Patchset int64 `protobuf:"varint,4,opt,name=patchset,proto3" json:"patchset,omitempty"`
	// The earliest patchset of the CL that is considered equivalent to the
	// patchset above.
	EarliestEquivalentPatchset int64 `protobuf:"varint,5,opt,name=earliest_equivalent_patchset,json=earliestEquivalentPatchset,proto3" json:"earliest_equivalent_patchset,omitempty"`
	// The time that the CQ was triggered for this CL in this Attempt.
	TriggerTime *timestamp.Timestamp `protobuf:"bytes,6,opt,name=trigger_time,json=triggerTime,proto3" json:"trigger_time,omitempty"`
	// CQ Mode for this CL, e.g. dry run or full run.
	Mode Mode `protobuf:"varint,7,opt,name=mode,proto3,enum=bigquery.Mode" json:"mode,omitempty"`
	// Whether CQ tried to submit this change and the result of the operation.
	SubmitStatus         GerritChange_SubmitStatus `protobuf:"varint,8,opt,name=submit_status,json=submitStatus,proto3,enum=bigquery.GerritChange_SubmitStatus" json:"submit_status,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                  `json:"-"`
	XXX_unrecognized     []byte                    `json:"-"`
	XXX_sizecache        int32                     `json:"-"`
}

func (m *GerritChange) Reset()         { *m = GerritChange{} }
func (m *GerritChange) String() string { return proto.CompactTextString(m) }
func (*GerritChange) ProtoMessage()    {}
func (*GerritChange) Descriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{1}
}

func (m *GerritChange) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GerritChange.Unmarshal(m, b)
}
func (m *GerritChange) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GerritChange.Marshal(b, m, deterministic)
}
func (m *GerritChange) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GerritChange.Merge(m, src)
}
func (m *GerritChange) XXX_Size() int {
	return xxx_messageInfo_GerritChange.Size(m)
}
func (m *GerritChange) XXX_DiscardUnknown() {
	xxx_messageInfo_GerritChange.DiscardUnknown(m)
}

var xxx_messageInfo_GerritChange proto.InternalMessageInfo

func (m *GerritChange) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *GerritChange) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *GerritChange) GetChange() int64 {
	if m != nil {
		return m.Change
	}
	return 0
}

func (m *GerritChange) GetPatchset() int64 {
	if m != nil {
		return m.Patchset
	}
	return 0
}

func (m *GerritChange) GetEarliestEquivalentPatchset() int64 {
	if m != nil {
		return m.EarliestEquivalentPatchset
	}
	return 0
}

func (m *GerritChange) GetTriggerTime() *timestamp.Timestamp {
	if m != nil {
		return m.TriggerTime
	}
	return nil
}

func (m *GerritChange) GetMode() Mode {
	if m != nil {
		return m.Mode
	}
	return Mode_MODE_UNSPECIFIED
}

func (m *GerritChange) GetSubmitStatus() GerritChange_SubmitStatus {
	if m != nil {
		return m.SubmitStatus
	}
	return GerritChange_SUBMIT_STATUS_UNSPECIFIED
}

// Build represents one tryjob Buildbucket build.
//
// See also: Build in buildbucket/proto/build.proto.
type Build struct {
	// Buildbucket build ID, unique per Buildbucket instance.
	Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	// Buildbucket host, e.g. "cr-buildbucket.appspot.com".
	Host string `protobuf:"bytes,2,opt,name=host,proto3" json:"host,omitempty"`
	// Information about whether this build was triggered previously and reused,
	// or triggered because there was no reusable build, or because builds by
	// this builder are all not reusable.
	Origin Build_Origin `protobuf:"varint,3,opt,name=origin,proto3,enum=bigquery.Build_Origin" json:"origin,omitempty"`
	// Whether the CQ must wait for this build to pass in order for the CLs to be
	// considered ready to submit. True means this builder must pass, false means
	// this builder is "optional", and so this build should not be used to assess
	// the correctness of the CLs in the Attempt. For example, builds added
	// because of the Cq-Include-Trybots footer are still critical; experimental
	// builders are not.
	//
	// Tip: join this with the Buildbucket BigQuery table to figure out which
	// builder this build belongs to.
	Critical             bool     `protobuf:"varint,4,opt,name=critical,proto3" json:"critical,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Build) Reset()         { *m = Build{} }
func (m *Build) String() string { return proto.CompactTextString(m) }
func (*Build) ProtoMessage()    {}
func (*Build) Descriptor() ([]byte, []int) {
	return fileDescriptor_8792fe122a6ce934, []int{2}
}

func (m *Build) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Build.Unmarshal(m, b)
}
func (m *Build) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Build.Marshal(b, m, deterministic)
}
func (m *Build) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Build.Merge(m, src)
}
func (m *Build) XXX_Size() int {
	return xxx_messageInfo_Build.Size(m)
}
func (m *Build) XXX_DiscardUnknown() {
	xxx_messageInfo_Build.DiscardUnknown(m)
}

var xxx_messageInfo_Build proto.InternalMessageInfo

func (m *Build) GetId() int64 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Build) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *Build) GetOrigin() Build_Origin {
	if m != nil {
		return m.Origin
	}
	return Build_ORIGIN_UNSPECIFIED
}

func (m *Build) GetCritical() bool {
	if m != nil {
		return m.Critical
	}
	return false
}

func init() {
	proto.RegisterEnum("bigquery.Mode", Mode_name, Mode_value)
	proto.RegisterEnum("bigquery.AttemptStatus", AttemptStatus_name, AttemptStatus_value)
	proto.RegisterEnum("bigquery.AttemptSubstatus", AttemptSubstatus_name, AttemptSubstatus_value)
	proto.RegisterEnum("bigquery.GerritChange_SubmitStatus", GerritChange_SubmitStatus_name, GerritChange_SubmitStatus_value)
	proto.RegisterEnum("bigquery.Build_Origin", Build_Origin_name, Build_Origin_value)
	proto.RegisterType((*Attempt)(nil), "bigquery.Attempt")
	proto.RegisterType((*GerritChange)(nil), "bigquery.GerritChange")
	proto.RegisterType((*Build)(nil), "bigquery.Build")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/cq/api/bigquery/attempt.proto", fileDescriptor_8792fe122a6ce934)
}

var fileDescriptor_8792fe122a6ce934 = []byte{
	// 927 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0x51, 0x6f, 0xa3, 0x46,
	0x17, 0xfd, 0x30, 0x8e, 0xed, 0x5c, 0x3b, 0x5e, 0x76, 0x94, 0x2f, 0x4b, 0xad, 0xdd, 0xd6, 0xeb,
	0x3e, 0x34, 0xca, 0x03, 0x96, 0xd2, 0xae, 0xda, 0x3e, 0xac, 0x54, 0x0c, 0x24, 0xa5, 0xb1, 0x07,
	0x6b, 0x80, 0x56, 0x79, 0x42, 0x18, 0xcf, 0x62, 0x5a, 0xdb, 0x38, 0x30, 0xac, 0x94, 0x1f, 0xd6,
	0x3f, 0xd0, 0x97, 0xfe, 0x91, 0xfe, 0x90, 0x6a, 0x06, 0xb0, 0x9d, 0xdd, 0xad, 0xda, 0x37, 0xce,
	0x3d, 0xe7, 0xce, 0x70, 0xcf, 0xb9, 0x03, 0xd7, 0x71, 0xaa, 0x45, 0xab, 0x2c, 0xdd, 0x24, 0xc5,
	0x46, 0x4b, 0xb3, 0x78, 0xbc, 0x2e, 0xa2, 0x64, 0x1c, 0x3d, 0x8c, 0xc3, 0x5d, 0x32, 0x5e, 0x24,
	0xf1, 0x43, 0x41, 0xb3, 0xc7, 0x71, 0xc8, 0x18, 0xdd, 0xec, 0x98, 0xb6, 0xcb, 0x52, 0x96, 0xa2,
	0x4e, 0x5d, 0x1f, 0x7c, 0x11, 0xa7, 0x69, 0xbc, 0xa6, 0x63, 0x51, 0x5f, 0x14, 0xef, 0xc6, 0x2c,
	0xd9, 0xd0, 0x9c, 0x85, 0x9b, 0x5d, 0x29, 0x1d, 0xfd, 0xde, 0x84, 0xb6, 0x5e, 0x36, 0x23, 0x05,
	0xe4, 0xdf, 0xe8, 0xa3, 0x2a, 0x0d, 0xa5, 0xcb, 0x53, 0xc2, 0x3f, 0xd1, 0x6b, 0xe8, 0xf1, 0xeb,
	0x82, 0x5d, 0x96, 0xfe, 0x4a, 0x23, 0xa6, 0x36, 0x04, 0xd5, 0xe5, 0xb5, 0x79, 0x59, 0xe2, 0x92,
	0x28, 0xdd, 0xbe, 0x4b, 0xe2, 0x20, 0xce, 0xd2, 0x62, 0xa7, 0x76, 0x4b, 0x49, 0x59, 0xbb, 0xe5,
	0x25, 0x34, 0x84, 0x5e, 0xb4, 0x2e, 0xe9, 0x80, 0x5f, 0x20, 0x0b, 0x09, 0x44, 0x6b, 0x41, 0xdf,
	0xd1, 0x47, 0xf4, 0x06, 0x5e, 0xd0, 0x87, 0x22, 0x79, 0x1f, 0xae, 0xe9, 0x96, 0x05, 0x4f, 0xc4,
	0x4d, 0x21, 0x3e, 0x3f, 0xd0, 0xc6, 0xa1, 0xed, 0x7b, 0x80, 0x9c, 0x85, 0x19, 0x0b, 0xf8, 0x54,
	0xea, 0xc9, 0x50, 0xba, 0xec, 0x5e, 0x0f, 0xb4, 0x72, 0x64, 0xad, 0x1e, 0x59, 0xf3, 0xea, 0x91,
	0xc9, 0xa9, 0x50, 0x73, 0x8c, 0xde, 0x40, 0x87, 0x6e, 0x97, 0x65, 0x63, 0xeb, 0x5f, 0x1b, 0xdb,
	0x74, 0xbb, 0x14, 0x6d, 0x6f, 0xa1, 0x1f, 0xd3, 0x2c, 0x4b, 0x58, 0x10, 0xad, 0xc2, 0x6d, 0x4c,
	0x73, 0xb5, 0x3d, 0x94, 0x2f, 0xbb, 0xd7, 0x17, 0x5a, 0x6d, 0xb9, 0x76, 0x2b, 0x78, 0x43, 0xd0,
	0xe4, 0x2c, 0x3e, 0x42, 0x39, 0xfa, 0x0a, 0x5a, 0x8b, 0x22, 0x59, 0x2f, 0x73, 0xb5, 0x23, 0xda,
	0x9e, 0x1d, 0xda, 0x26, 0xbc, 0x4e, 0x2a, 0x1a, 0x8d, 0xa1, 0x95, 0xb3, 0x90, 0x15, 0xb9, 0x7a,
	0x3a, 0x94, 0x2e, 0xfb, 0xd7, 0x2f, 0x0e, 0xc2, 0x2a, 0x2d, 0x57, 0xd0, 0xa4, 0x92, 0xa1, 0xef,
	0xe0, 0x34, 0x2f, 0x16, 0x55, 0x0f, 0x88, 0x9e, 0xc1, 0xc7, 0x3d, 0xb5, 0x82, 0x1c, 0xc4, 0xe8,
	0x1b, 0xb8, 0x58, 0x85, 0x79, 0x10, 0x15, 0x39, 0x4b, 0x37, 0x41, 0xc6, 0x8d, 0xce, 0xe8, 0x86,
	0x6e, 0x99, 0xda, 0x1b, 0x4a, 0x97, 0x1d, 0x72, 0xbe, 0x0a, 0x73, 0x43, 0x90, 0xe4, 0xc0, 0x8d,
	0xfe, 0x90, 0xa1, 0x77, 0x3c, 0x29, 0x42, 0xd0, 0x5c, 0xa5, 0x39, 0xab, 0xb6, 0x47, 0x7c, 0x23,
	0x15, 0xda, 0x4f, 0x37, 0xa7, 0x86, 0xe8, 0x02, 0x5a, 0xa5, 0x81, 0x62, 0x19, 0x64, 0x52, 0x21,
	0x34, 0x80, 0xce, 0x2e, 0x64, 0xd1, 0x2a, 0xa7, 0x4c, 0x24, 0x2f, 0x93, 0x3d, 0x46, 0x3f, 0xc0,
	0x4b, 0x1a, 0x66, 0xeb, 0x84, 0xe6, 0x2c, 0x38, 0xda, 0x96, 0xbd, 0xfe, 0x44, 0xe8, 0x07, 0xb5,
	0xc6, 0xda, 0x4b, 0xe6, 0xf5, 0x09, 0x6f, 0xa1, 0xc7, 0xb2, 0x24, 0x8e, 0x69, 0xf6, 0x5f, 0x83,
	0xef, 0x56, 0x7a, 0x11, 0xfe, 0x08, 0x9a, 0x9b, 0x74, 0x49, 0xd5, 0xb6, 0xb0, 0xb7, 0x7f, 0xb0,
	0x77, 0x96, 0x2e, 0x29, 0x11, 0x1c, 0xfa, 0x11, 0xce, 0xf2, 0x62, 0xb1, 0x49, 0x58, 0x50, 0x65,
	0xd1, 0x11, 0xe2, 0x2f, 0x3f, 0xbd, 0x1f, 0x9a, 0x2b, 0xb4, 0x55, 0x96, 0xbd, 0xfc, 0x08, 0x8d,
	0x42, 0xe8, 0x1d, 0xb3, 0xe8, 0x15, 0x7c, 0xe6, 0xfa, 0x93, 0x99, 0xed, 0x05, 0xae, 0xa7, 0x7b,
	0xbe, 0x1b, 0xf8, 0xd8, 0x9d, 0x5b, 0x86, 0x7d, 0x63, 0x5b, 0xa6, 0xf2, 0x3f, 0xd4, 0x85, 0xf6,
	0xdc, 0xc2, 0xa6, 0x8d, 0x6f, 0x15, 0x89, 0x03, 0x1f, 0xdf, 0x61, 0xe7, 0x17, 0xac, 0x34, 0x38,
	0xb8, 0xd1, 0xed, 0xa9, 0x4f, 0x2c, 0x45, 0xe6, 0xc0, 0xf5, 0x0d, 0xc3, 0x72, 0x5d, 0xa5, 0x39,
	0xfa, 0x53, 0x82, 0x13, 0xb1, 0x77, 0xa8, 0x0f, 0x8d, 0x64, 0x29, 0xb2, 0x93, 0x49, 0x23, 0x59,
	0xee, 0xd3, 0x6c, 0x1c, 0xa5, 0xa9, 0x41, 0x2b, 0xcd, 0x92, 0x38, 0xd9, 0x8a, 0xcc, 0xfa, 0xc7,
	0x3b, 0x2f, 0x0e, 0xd1, 0x1c, 0xc1, 0x92, 0x4a, 0xc5, 0xb3, 0x8c, 0xb2, 0x84, 0x25, 0x51, 0xb8,
	0x16, 0x59, 0x76, 0xc8, 0x1e, 0x8f, 0x30, 0xb4, 0x4a, 0x35, 0xba, 0x00, 0xe4, 0x10, 0xfb, 0xd6,
	0xc6, 0x1f, 0xcc, 0xa3, 0x40, 0x0f, 0x3b, 0x5e, 0x40, 0x2c, 0xdf, 0xd5, 0x27, 0x53, 0x4b, 0x91,
	0x50, 0x1f, 0xa0, 0xae, 0x58, 0xa6, 0xd2, 0x40, 0x00, 0xad, 0xea, 0x5b, 0xbe, 0xfa, 0x16, 0x9a,
	0x3c, 0x04, 0x74, 0x0e, 0xca, 0xcc, 0x31, 0xad, 0x8f, 0xbd, 0x31, 0xc9, 0x7d, 0x40, 0x7c, 0xac,
	0x48, 0xa8, 0x07, 0x9d, 0x1b, 0x7f, 0x3a, 0x15, 0xa8, 0x71, 0xf5, 0x1e, 0xce, 0x9e, 0x3c, 0x28,
	0xf4, 0x39, 0x0c, 0x74, 0xcf, 0xb3, 0x66, 0xf3, 0x7f, 0xf6, 0xd9, 0xf5, 0x74, 0xe2, 0x59, 0x66,
	0xe9, 0x73, 0xed, 0xa6, 0xf0, 0x59, 0x9f, 0x38, 0x82, 0x91, 0x8f, 0x4d, 0x6f, 0xa2, 0xe7, 0x70,
	0x66, 0xe3, 0x1b, 0xa2, 0x07, 0x75, 0xe9, 0xe4, 0xea, 0x2f, 0x09, 0x94, 0x0f, 0x5f, 0x25, 0x7a,
	0x0d, 0xaf, 0xf6, 0x77, 0xfb, 0x93, 0x4f, 0x5e, 0x2f, 0x6c, 0x39, 0xb0, 0x8a, 0x84, 0x10, 0xf4,
	0xf9, 0xb1, 0x96, 0x19, 0x78, 0xe4, 0xfe, 0x27, 0x67, 0xc2, 0x7f, 0xe5, 0x19, 0x74, 0xab, 0xda,
	0xd4, 0xc6, 0x9e, 0x22, 0x73, 0xef, 0x7c, 0xac, 0xcf, 0xe7, 0xc4, 0xf9, 0xd9, 0x32, 0x95, 0x26,
	0xfa, 0x3f, 0x3c, 0x9f, 0x5b, 0x64, 0x66, 0xbb, 0xae, 0xed, 0xe0, 0xc0, 0xb4, 0x30, 0x3f, 0xfd,
	0x04, 0x0d, 0xe0, 0xc2, 0xc7, 0xae, 0xee, 0xd9, 0x2e, 0xbf, 0x2e, 0x30, 0x2d, 0xbe, 0x52, 0x16,
	0x36, 0xee, 0x95, 0x16, 0x1f, 0x62, 0xa6, 0x63, 0x5f, 0x9f, 0x06, 0x86, 0x8e, 0x0d, 0x6b, 0xaa,
	0xb4, 0xd1, 0x10, 0x5e, 0x4e, 0x7c, 0x7b, 0x6a, 0x4e, 0x7c, 0xe3, 0xce, 0xf2, 0x82, 0x99, 0xed,
	0x1a, 0x0e, 0xbe, 0xb1, 0x6f, 0x7d, 0xa2, 0x7b, 0xb6, 0x83, 0x95, 0xce, 0xa2, 0x25, 0xde, 0xd4,
	0xd7, 0x7f, 0x07, 0x00, 0x00, 0xff, 0xff, 0xf2, 0xfa, 0x4a, 0xbd, 0xc6, 0x06, 0x00, 0x00,
}