// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/tetrafolium/luci-go/dm/api/service/v1/ensure_graph_data.proto

package dm

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	templateproto "github.com/tetrafolium/luci-go/common/data/text/templateproto"
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

type TemplateInstantiation struct {
	// project is the luci-config project which defines the template.
	Project string `protobuf:"bytes,1,opt,name=project,proto3" json:"project,omitempty"`
	// ref is the git ref of the project that defined this template. If omitted,
	// this will use the template definition from the project-wide configuration
	// and not the configuration located on a particular ref (like
	// 'refs/heads/master').
	Ref string `protobuf:"bytes,2,opt,name=ref,proto3" json:"ref,omitempty"`
	// specifier specifies the actual template name, as well as any substitution
	// parameters which that template might require.
	Specifier            *templateproto.Specifier `protobuf:"bytes,4,opt,name=specifier,proto3" json:"specifier,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *TemplateInstantiation) Reset()         { *m = TemplateInstantiation{} }
func (m *TemplateInstantiation) String() string { return proto.CompactTextString(m) }
func (*TemplateInstantiation) ProtoMessage()    {}
func (*TemplateInstantiation) Descriptor() ([]byte, []int) {
	return fileDescriptor_d2be8364c35d3177, []int{0}
}

func (m *TemplateInstantiation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TemplateInstantiation.Unmarshal(m, b)
}
func (m *TemplateInstantiation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TemplateInstantiation.Marshal(b, m, deterministic)
}
func (m *TemplateInstantiation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TemplateInstantiation.Merge(m, src)
}
func (m *TemplateInstantiation) XXX_Size() int {
	return xxx_messageInfo_TemplateInstantiation.Size(m)
}
func (m *TemplateInstantiation) XXX_DiscardUnknown() {
	xxx_messageInfo_TemplateInstantiation.DiscardUnknown(m)
}

var xxx_messageInfo_TemplateInstantiation proto.InternalMessageInfo

func (m *TemplateInstantiation) GetProject() string {
	if m != nil {
		return m.Project
	}
	return ""
}

func (m *TemplateInstantiation) GetRef() string {
	if m != nil {
		return m.Ref
	}
	return ""
}

func (m *TemplateInstantiation) GetSpecifier() *templateproto.Specifier {
	if m != nil {
		return m.Specifier
	}
	return nil
}

// EnsureGraphDataReq allows you to assert the existence of Attempts in DM's
// graph, and allows you to declare dependencies from one Attempt to another.
//
// You can declare Attempts by any combination of:
//   * Providing a quest description plus a list of Attempt numbers for that
//     quest.
//   * Providing a template instantiation (for a project-declared quest
//     template) plus a list of Attempt numbers for that quest.
//   * Providing a raw set of quest_id -> attempt numbers for quests that you
//     already know that DM has a definition for.
//
// In response, DM will tell you what the IDs of all supplied Quests/Attempts
// are.
//
// To create a dependencies, call this method while running as part of an
// execution by filling the for_execution field. All attempts named as described
// above will become dependencies for the indicated execution. It is only
// possible for a currently-running execution to create dependencies for its own
// Attempt. In particular, it is not possible to create dependencies as
// a non-execution user (e.g. a human), nor is it possible for an execution to
// create attempts on behalf of some other execution.
//
// If the attempts were being created as dependencies, and were already in the
// Finished state, this request can also opt to include the AttemptResults
// directly.
type EnsureGraphDataReq struct {
	// Quest is a list of quest descriptors. DM will ensure that the
	// corresponding Quests exist. If they don't, they'll be created.
	Quest []*Quest_Desc `protobuf:"bytes,1,rep,name=quest,proto3" json:"quest,omitempty"`
	// QuestAttempt allows the addition of attempts which are derived from
	// the quest bodies provided above.
	// Each entry here maps 1:1 with the equivalent quest.
	QuestAttempt []*AttemptList_Nums `protobuf:"bytes,2,rep,name=quest_attempt,json=questAttempt,proto3" json:"quest_attempt,omitempty"`
	// TemplateQuest allows the addition of quests which are derived from
	// Templates, as defined on a per-project basis.
	TemplateQuest []*TemplateInstantiation `protobuf:"bytes,3,rep,name=template_quest,json=templateQuest,proto3" json:"template_quest,omitempty"`
	// TemplateAttempt allows the addition of attempts which are derived from
	// Templates. This must be equal in length to template_quest.
	// Each entry here maps 1:1 with the equivalent quest in template_quest.
	TemplateAttempt []*AttemptList_Nums `protobuf:"bytes,4,rep,name=template_attempt,json=templateAttempt,proto3" json:"template_attempt,omitempty"`
	// RawAttempts is a list that asserts that the following attempts should
	// exist. The quest ids in this list must be already-known to DM, NOT
	// included in the quest field above. This is useful when you know the ID of
	// the Quest, but not the actual definition of the quest.
	RawAttempts *AttemptList `protobuf:"bytes,5,opt,name=raw_attempts,json=rawAttempts,proto3" json:"raw_attempts,omitempty"`
	// ForExecution is an authentication pair (Execution_ID, Token).
	//
	// If this is provided then it will serve as authorization for the creation of
	// any `quests` included, and any `attempts` indicated will be set as
	// dependencies for the execution.
	//
	// If this omitted, then the request requires some user/bot authentication,
	// and any quests/attempts provided will be made standalone (e.g. nothing will
	// depend on them).
	ForExecution         *Execution_Auth             `protobuf:"bytes,6,opt,name=for_execution,json=forExecution,proto3" json:"for_execution,omitempty"`
	Limit                *EnsureGraphDataReq_Limit   `protobuf:"bytes,7,opt,name=limit,proto3" json:"limit,omitempty"`
	Include              *EnsureGraphDataReq_Include `protobuf:"bytes,8,opt,name=include,proto3" json:"include,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *EnsureGraphDataReq) Reset()         { *m = EnsureGraphDataReq{} }
func (m *EnsureGraphDataReq) String() string { return proto.CompactTextString(m) }
func (*EnsureGraphDataReq) ProtoMessage()    {}
func (*EnsureGraphDataReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_d2be8364c35d3177, []int{1}
}

func (m *EnsureGraphDataReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnsureGraphDataReq.Unmarshal(m, b)
}
func (m *EnsureGraphDataReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnsureGraphDataReq.Marshal(b, m, deterministic)
}
func (m *EnsureGraphDataReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnsureGraphDataReq.Merge(m, src)
}
func (m *EnsureGraphDataReq) XXX_Size() int {
	return xxx_messageInfo_EnsureGraphDataReq.Size(m)
}
func (m *EnsureGraphDataReq) XXX_DiscardUnknown() {
	xxx_messageInfo_EnsureGraphDataReq.DiscardUnknown(m)
}

var xxx_messageInfo_EnsureGraphDataReq proto.InternalMessageInfo

func (m *EnsureGraphDataReq) GetQuest() []*Quest_Desc {
	if m != nil {
		return m.Quest
	}
	return nil
}

func (m *EnsureGraphDataReq) GetQuestAttempt() []*AttemptList_Nums {
	if m != nil {
		return m.QuestAttempt
	}
	return nil
}

func (m *EnsureGraphDataReq) GetTemplateQuest() []*TemplateInstantiation {
	if m != nil {
		return m.TemplateQuest
	}
	return nil
}

func (m *EnsureGraphDataReq) GetTemplateAttempt() []*AttemptList_Nums {
	if m != nil {
		return m.TemplateAttempt
	}
	return nil
}

func (m *EnsureGraphDataReq) GetRawAttempts() *AttemptList {
	if m != nil {
		return m.RawAttempts
	}
	return nil
}

func (m *EnsureGraphDataReq) GetForExecution() *Execution_Auth {
	if m != nil {
		return m.ForExecution
	}
	return nil
}

func (m *EnsureGraphDataReq) GetLimit() *EnsureGraphDataReq_Limit {
	if m != nil {
		return m.Limit
	}
	return nil
}

func (m *EnsureGraphDataReq) GetInclude() *EnsureGraphDataReq_Include {
	if m != nil {
		return m.Include
	}
	return nil
}

type EnsureGraphDataReq_Limit struct {
	// MaxDataSize sets the maximum amount of 'Data' (in bytes) that can be
	// returned, if include.attempt_result is set. If this limit is hit, then
	// the appropriate 'partial' value will be set for that object, but the base
	// object would still be included in the result.
	//
	// If this limit is 0, a default limit of 16MB will be used. If this limit
	// exceeds 30MB, it will be reduced to 30MB.
	MaxDataSize          uint32   `protobuf:"varint,3,opt,name=max_data_size,json=maxDataSize,proto3" json:"max_data_size,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EnsureGraphDataReq_Limit) Reset()         { *m = EnsureGraphDataReq_Limit{} }
func (m *EnsureGraphDataReq_Limit) String() string { return proto.CompactTextString(m) }
func (*EnsureGraphDataReq_Limit) ProtoMessage()    {}
func (*EnsureGraphDataReq_Limit) Descriptor() ([]byte, []int) {
	return fileDescriptor_d2be8364c35d3177, []int{1, 0}
}

func (m *EnsureGraphDataReq_Limit) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnsureGraphDataReq_Limit.Unmarshal(m, b)
}
func (m *EnsureGraphDataReq_Limit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnsureGraphDataReq_Limit.Marshal(b, m, deterministic)
}
func (m *EnsureGraphDataReq_Limit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnsureGraphDataReq_Limit.Merge(m, src)
}
func (m *EnsureGraphDataReq_Limit) XXX_Size() int {
	return xxx_messageInfo_EnsureGraphDataReq_Limit.Size(m)
}
func (m *EnsureGraphDataReq_Limit) XXX_DiscardUnknown() {
	xxx_messageInfo_EnsureGraphDataReq_Limit.DiscardUnknown(m)
}

var xxx_messageInfo_EnsureGraphDataReq_Limit proto.InternalMessageInfo

func (m *EnsureGraphDataReq_Limit) GetMaxDataSize() uint32 {
	if m != nil {
		return m.MaxDataSize
	}
	return 0
}

type EnsureGraphDataReq_Include struct {
	Attempt              *EnsureGraphDataReq_Include_Options `protobuf:"bytes,4,opt,name=attempt,proto3" json:"attempt,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                            `json:"-"`
	XXX_unrecognized     []byte                              `json:"-"`
	XXX_sizecache        int32                               `json:"-"`
}

func (m *EnsureGraphDataReq_Include) Reset()         { *m = EnsureGraphDataReq_Include{} }
func (m *EnsureGraphDataReq_Include) String() string { return proto.CompactTextString(m) }
func (*EnsureGraphDataReq_Include) ProtoMessage()    {}
func (*EnsureGraphDataReq_Include) Descriptor() ([]byte, []int) {
	return fileDescriptor_d2be8364c35d3177, []int{1, 1}
}

func (m *EnsureGraphDataReq_Include) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnsureGraphDataReq_Include.Unmarshal(m, b)
}
func (m *EnsureGraphDataReq_Include) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnsureGraphDataReq_Include.Marshal(b, m, deterministic)
}
func (m *EnsureGraphDataReq_Include) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnsureGraphDataReq_Include.Merge(m, src)
}
func (m *EnsureGraphDataReq_Include) XXX_Size() int {
	return xxx_messageInfo_EnsureGraphDataReq_Include.Size(m)
}
func (m *EnsureGraphDataReq_Include) XXX_DiscardUnknown() {
	xxx_messageInfo_EnsureGraphDataReq_Include.DiscardUnknown(m)
}

var xxx_messageInfo_EnsureGraphDataReq_Include proto.InternalMessageInfo

func (m *EnsureGraphDataReq_Include) GetAttempt() *EnsureGraphDataReq_Include_Options {
	if m != nil {
		return m.Attempt
	}
	return nil
}

type EnsureGraphDataReq_Include_Options struct {
	// Instructs finished objects to include the Result field.
	Result               bool     `protobuf:"varint,3,opt,name=result,proto3" json:"result,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EnsureGraphDataReq_Include_Options) Reset()         { *m = EnsureGraphDataReq_Include_Options{} }
func (m *EnsureGraphDataReq_Include_Options) String() string { return proto.CompactTextString(m) }
func (*EnsureGraphDataReq_Include_Options) ProtoMessage()    {}
func (*EnsureGraphDataReq_Include_Options) Descriptor() ([]byte, []int) {
	return fileDescriptor_d2be8364c35d3177, []int{1, 1, 0}
}

func (m *EnsureGraphDataReq_Include_Options) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnsureGraphDataReq_Include_Options.Unmarshal(m, b)
}
func (m *EnsureGraphDataReq_Include_Options) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnsureGraphDataReq_Include_Options.Marshal(b, m, deterministic)
}
func (m *EnsureGraphDataReq_Include_Options) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnsureGraphDataReq_Include_Options.Merge(m, src)
}
func (m *EnsureGraphDataReq_Include_Options) XXX_Size() int {
	return xxx_messageInfo_EnsureGraphDataReq_Include_Options.Size(m)
}
func (m *EnsureGraphDataReq_Include_Options) XXX_DiscardUnknown() {
	xxx_messageInfo_EnsureGraphDataReq_Include_Options.DiscardUnknown(m)
}

var xxx_messageInfo_EnsureGraphDataReq_Include_Options proto.InternalMessageInfo

func (m *EnsureGraphDataReq_Include_Options) GetResult() bool {
	if m != nil {
		return m.Result
	}
	return false
}

type EnsureGraphDataRsp struct {
	// accepted is true when all new graph data was journaled successfully. This
	// means that `quests`, `attempts`, `template_quest`, `template_attempt` were
	// all well-formed and are scheduled to be added. They will 'eventually' be
	// readable via other APIs (like WalkGraph), but when they are, they'll have
	// the IDs reflected in this response.
	//
	// If `attempts` referrs to quests that don't exist and weren't provided in
	// `quests`, those quests will be listed in `result` with the DNE flag set.
	//
	// If `template_quest` had errors (missing template, bad params, etc.), the
	// errors will be located in `template_error`. If all of the templates parsed
	// successfully, the quest ids for those rendered `template_quest` will be in
	// `template_ids`.
	Accepted bool `protobuf:"varint,1,opt,name=accepted,proto3" json:"accepted,omitempty"`
	// quest_ids will be populated with the Quest.IDs of any quests defined
	// by quest in the initial request. Its length is guaranteed to match
	// the length of quest, if there were no errors.
	QuestIds []*Quest_ID `protobuf:"bytes,2,rep,name=quest_ids,json=questIds,proto3" json:"quest_ids,omitempty"`
	// template_ids will be populated with the Quest.IDs of any templates defined
	// by template_quest in the initial request. Its length is guaranteed to match
	// the length of template_quest, if there were no errors.
	TemplateIds []*Quest_ID `protobuf:"bytes,3,rep,name=template_ids,json=templateIds,proto3" json:"template_ids,omitempty"`
	// template_error is either empty if there were no template errors, or the
	// length of template_quest. Non-empty strings are errors.
	TemplateError []string `protobuf:"bytes,4,rep,name=template_error,json=templateError,proto3" json:"template_error,omitempty"`
	// result holds the graph data pertaining to the request, containing any
	// graph state that already existed at the time of the call. Any new data
	// that was added to the graph state (accepted==true) will appear with
	// `DNE==true`.
	//
	// Quest data will always be returned for any Quests which exist.
	//
	// If accepted==false, you can inspect this to determine why:
	//   * Quests (without data) mentioned by the `attempts` field that do not
	//     exist will have `DNE==true`.
	//
	// This also can be used to make adding dependencies a stateless
	// single-request action:
	//   * Attempts requested (assuming the corresponding Quest exists) will
	//     contain their current state. If Include.AttemptResult was true, the
	//     results will be populated (with the size limit mentioned in the request
	//     documentation).
	Result *GraphData `protobuf:"bytes,5,opt,name=result,proto3" json:"result,omitempty"`
	// (if `for_execution` was specified) ShouldHalt indicates that the request
	// was accepted by DM, and the execution should halt (DM will re-execute the
	// Attempt when it becomes unblocked). If this is true, then the execution's
	// auth Token is also revoked and will no longer work for futher API calls.
	//
	// If `for_execution` was provided in the request and this is false, it means
	// that the execution may continue executing.
	ShouldHalt           bool     `protobuf:"varint,6,opt,name=should_halt,json=shouldHalt,proto3" json:"should_halt,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EnsureGraphDataRsp) Reset()         { *m = EnsureGraphDataRsp{} }
func (m *EnsureGraphDataRsp) String() string { return proto.CompactTextString(m) }
func (*EnsureGraphDataRsp) ProtoMessage()    {}
func (*EnsureGraphDataRsp) Descriptor() ([]byte, []int) {
	return fileDescriptor_d2be8364c35d3177, []int{2}
}

func (m *EnsureGraphDataRsp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnsureGraphDataRsp.Unmarshal(m, b)
}
func (m *EnsureGraphDataRsp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnsureGraphDataRsp.Marshal(b, m, deterministic)
}
func (m *EnsureGraphDataRsp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnsureGraphDataRsp.Merge(m, src)
}
func (m *EnsureGraphDataRsp) XXX_Size() int {
	return xxx_messageInfo_EnsureGraphDataRsp.Size(m)
}
func (m *EnsureGraphDataRsp) XXX_DiscardUnknown() {
	xxx_messageInfo_EnsureGraphDataRsp.DiscardUnknown(m)
}

var xxx_messageInfo_EnsureGraphDataRsp proto.InternalMessageInfo

func (m *EnsureGraphDataRsp) GetAccepted() bool {
	if m != nil {
		return m.Accepted
	}
	return false
}

func (m *EnsureGraphDataRsp) GetQuestIds() []*Quest_ID {
	if m != nil {
		return m.QuestIds
	}
	return nil
}

func (m *EnsureGraphDataRsp) GetTemplateIds() []*Quest_ID {
	if m != nil {
		return m.TemplateIds
	}
	return nil
}

func (m *EnsureGraphDataRsp) GetTemplateError() []string {
	if m != nil {
		return m.TemplateError
	}
	return nil
}

func (m *EnsureGraphDataRsp) GetResult() *GraphData {
	if m != nil {
		return m.Result
	}
	return nil
}

func (m *EnsureGraphDataRsp) GetShouldHalt() bool {
	if m != nil {
		return m.ShouldHalt
	}
	return false
}

func init() {
	proto.RegisterType((*TemplateInstantiation)(nil), "dm.TemplateInstantiation")
	proto.RegisterType((*EnsureGraphDataReq)(nil), "dm.EnsureGraphDataReq")
	proto.RegisterType((*EnsureGraphDataReq_Limit)(nil), "dm.EnsureGraphDataReq.Limit")
	proto.RegisterType((*EnsureGraphDataReq_Include)(nil), "dm.EnsureGraphDataReq.Include")
	proto.RegisterType((*EnsureGraphDataReq_Include_Options)(nil), "dm.EnsureGraphDataReq.Include.Options")
	proto.RegisterType((*EnsureGraphDataRsp)(nil), "dm.EnsureGraphDataRsp")
}

func init() {
	proto.RegisterFile("github.com/tetrafolium/luci-go/dm/api/service/v1/ensure_graph_data.proto", fileDescriptor_d2be8364c35d3177)
}

var fileDescriptor_d2be8364c35d3177 = []byte{
	// 649 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0x5f, 0x6f, 0xd3, 0x3e,
	0x14, 0x55, 0x9a, 0xa4, 0x49, 0xdd, 0x76, 0x8b, 0xac, 0xdf, 0x0f, 0x85, 0x08, 0xc1, 0x54, 0x31,
	0x54, 0x5e, 0x12, 0x51, 0x24, 0xc6, 0x5e, 0x60, 0x43, 0x9b, 0xa0, 0xd5, 0x04, 0xc2, 0xe3, 0x3d,
	0x32, 0x89, 0xbb, 0x1a, 0x25, 0x75, 0x66, 0x3b, 0x5b, 0xd9, 0x1b, 0x5f, 0x84, 0x6f, 0xc0, 0x27,
	0xe4, 0x05, 0xd9, 0xf9, 0xd3, 0xb1, 0x95, 0x69, 0x6f, 0xb9, 0xc7, 0xe7, 0xdc, 0x73, 0xef, 0xd5,
	0x09, 0x78, 0x73, 0xc6, 0xc2, 0x64, 0xc1, 0x59, 0x4e, 0xcb, 0x3c, 0x64, 0xfc, 0x2c, 0xca, 0xca,
	0x84, 0x46, 0x69, 0x1e, 0xe1, 0x82, 0x46, 0x82, 0xf0, 0x0b, 0x9a, 0x90, 0xe8, 0xe2, 0x45, 0x44,
	0x96, 0xa2, 0xe4, 0x24, 0x3e, 0xe3, 0xb8, 0x58, 0xc4, 0x29, 0x96, 0x38, 0x2c, 0x38, 0x93, 0x0c,
	0x76, 0xd2, 0x3c, 0x78, 0xb7, 0xb1, 0x47, 0xc2, 0xf2, 0x9c, 0x2d, 0x23, 0xc5, 0x8d, 0x24, 0x59,
	0xc9, 0x48, 0x92, 0xbc, 0xc8, 0xb0, 0x24, 0x5a, 0xd8, 0x56, 0x55, 0x9f, 0x60, 0xef, 0x9e, 0x73,
	0xdc, 0x1c, 0x20, 0x98, 0xdc, 0x53, 0x28, 0xbf, 0x17, 0x44, 0x54, 0x9a, 0xd1, 0x0f, 0x03, 0xfc,
	0xff, 0xa5, 0xf6, 0x9f, 0x2e, 0x85, 0xc4, 0x4b, 0x49, 0xb1, 0xa4, 0x6c, 0x09, 0x7d, 0xe0, 0x14,
	0x9c, 0x7d, 0x23, 0x89, 0xf4, 0x8d, 0x1d, 0x63, 0xdc, 0x43, 0x4d, 0x09, 0x3d, 0x60, 0x72, 0x32,
	0xf7, 0x3b, 0x1a, 0x55, 0x9f, 0xf0, 0x15, 0xe8, 0x89, 0x82, 0x24, 0x74, 0x4e, 0x09, 0xf7, 0xad,
	0x1d, 0x63, 0xdc, 0x9f, 0xf8, 0xe1, 0x5f, 0x4b, 0x86, 0xa7, 0xcd, 0x3b, 0x5a, 0x53, 0x67, 0x96,
	0x6b, 0x7a, 0xd6, 0xe8, 0x97, 0x0d, 0xe0, 0xb1, 0x3e, 0xea, 0x7b, 0xb5, 0xd2, 0x11, 0x96, 0x18,
	0x91, 0x73, 0xf8, 0x14, 0xd8, 0xe7, 0x25, 0x11, 0xca, 0xde, 0x1c, 0xf7, 0x27, 0x5b, 0x61, 0x9a,
	0x87, 0x9f, 0x15, 0x10, 0x1e, 0x11, 0x91, 0xa0, 0xea, 0x11, 0xee, 0x83, 0xa1, 0xfe, 0x88, 0xb1,
	0x54, 0x86, 0xd2, 0xef, 0x68, 0xf6, 0x7f, 0x8a, 0x7d, 0x58, 0x41, 0x27, 0x54, 0xc8, 0xf0, 0x63,
	0x99, 0x0b, 0x34, 0xd0, 0xd4, 0x1a, 0x86, 0x07, 0x60, 0xab, 0x99, 0x31, 0xae, 0x9c, 0x4c, 0xad,
	0x7d, 0xa8, 0xb4, 0x1b, 0x8f, 0x82, 0x86, 0x8d, 0x40, 0x0f, 0x02, 0xdf, 0x02, 0xaf, 0xed, 0xd0,
	0xf8, 0x5b, 0x77, 0xf8, 0x6f, 0x37, 0xec, 0x66, 0x84, 0x09, 0x18, 0x70, 0x7c, 0xd9, 0x68, 0x85,
	0x6f, 0xeb, 0xdb, 0x6d, 0xdf, 0x10, 0xa3, 0x3e, 0xc7, 0x97, 0x75, 0x2d, 0xe0, 0x1e, 0x18, 0xce,
	0x19, 0x8f, 0xc9, 0x8a, 0x24, 0xa5, 0x1a, 0xca, 0xef, 0x6a, 0x11, 0x54, 0xa2, 0xe3, 0x06, 0x0c,
	0x0f, 0x4b, 0xb9, 0x40, 0x83, 0x39, 0xe3, 0x2d, 0x04, 0x27, 0xc0, 0xce, 0x68, 0x4e, 0xa5, 0xef,
	0x68, 0xc1, 0x23, 0x2d, 0xb8, 0x75, 0xf7, 0xf0, 0x44, 0x71, 0x50, 0x45, 0x85, 0xaf, 0x81, 0x43,
	0x97, 0x49, 0x56, 0xa6, 0xc4, 0x77, 0xb5, 0xea, 0xf1, 0x3f, 0x54, 0xd3, 0x8a, 0x85, 0x1a, 0x7a,
	0xb0, 0x07, 0x6c, 0xdd, 0x09, 0x8e, 0xc0, 0x30, 0xc7, 0x2b, 0x1d, 0xd4, 0x58, 0xd0, 0x2b, 0xe2,
	0x9b, 0x3b, 0xc6, 0x78, 0x88, 0xfa, 0x39, 0x5e, 0x29, 0xf1, 0x29, 0xbd, 0x22, 0x33, 0xcb, 0x35,
	0xbc, 0xce, 0xcc, 0x72, 0x3b, 0x9e, 0x19, 0xfc, 0x34, 0x80, 0x53, 0x77, 0x83, 0x07, 0xc0, 0x59,
	0xdf, 0x55, 0xd9, 0x3f, 0xbb, 0xdb, 0x3e, 0xfc, 0x54, 0xa8, 0x55, 0x05, 0x6a, 0x64, 0xc1, 0x3e,
	0x70, 0x6a, 0x0c, 0x3e, 0x00, 0x5d, 0x4e, 0x44, 0x99, 0x49, 0x3d, 0x81, 0x8b, 0xea, 0xea, 0xba,
	0xf9, 0xcc, 0x72, 0x2d, 0xcf, 0x9e, 0x59, 0xae, 0xed, 0x75, 0x5b, 0xdc, 0xf4, 0xac, 0x16, 0xe9,
	0x7a, 0xce, 0xe8, 0xb7, 0x71, 0x3b, 0xaf, 0xa2, 0x80, 0x01, 0x70, 0x71, 0x92, 0x90, 0x42, 0x92,
	0x54, 0xff, 0x31, 0x2e, 0x6a, 0x6b, 0xf8, 0x1c, 0xf4, 0xaa, 0x94, 0xd2, 0x54, 0xd4, 0x09, 0x1d,
	0xac, 0xf3, 0x3c, 0x3d, 0x42, 0xae, 0x7e, 0x9e, 0xa6, 0x02, 0x46, 0x60, 0xd0, 0x66, 0x4a, 0xb1,
	0xcd, 0x0d, 0xec, 0x7e, 0xc3, 0x50, 0x82, 0xdd, 0x6b, 0x31, 0x26, 0x9c, 0x33, 0xae, 0x23, 0xd8,
	0x5b, 0x67, 0xf5, 0x58, 0x81, 0x70, 0xb7, 0xdd, 0xbe, 0x0a, 0xd9, 0x50, 0x75, 0x5c, 0x2f, 0x50,
	0x3f, 0xc2, 0x27, 0xa0, 0x2f, 0x16, 0xac, 0xcc, 0xd2, 0x78, 0x81, 0x33, 0xa9, 0xb3, 0xe5, 0x22,
	0x50, 0x41, 0x1f, 0x70, 0x26, 0xbf, 0x76, 0xf5, 0xff, 0xfc, 0xf2, 0x4f, 0x00, 0x00, 0x00, 0xff,
	0xff, 0x72, 0xdc, 0x5f, 0x1e, 0x2f, 0x05, 0x00, 0x00,
}
