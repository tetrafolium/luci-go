// Code generated by protoc-gen-go. DO NOT EDIT.
// source: go.chromium.org/luci/luci_notify/api/config/notify.proto

package config

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	proto1 "go.chromium.org/luci/buildbucket/proto"
	_ "go.chromium.org/luci/common/proto"
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

// ProjectConfig is a luci-notify configuration for a particular project.
type ProjectConfig struct {
	// Notifiers is a list of Notifiers which watch builders and send
	// notifications for this project.
	Notifiers            []*Notifier `protobuf:"bytes,1,rep,name=notifiers,proto3" json:"notifiers,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *ProjectConfig) Reset()         { *m = ProjectConfig{} }
func (m *ProjectConfig) String() string { return proto.CompactTextString(m) }
func (*ProjectConfig) ProtoMessage()    {}
func (*ProjectConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{0}
}

func (m *ProjectConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProjectConfig.Unmarshal(m, b)
}
func (m *ProjectConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProjectConfig.Marshal(b, m, deterministic)
}
func (m *ProjectConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProjectConfig.Merge(m, src)
}
func (m *ProjectConfig) XXX_Size() int {
	return xxx_messageInfo_ProjectConfig.Size(m)
}
func (m *ProjectConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_ProjectConfig.DiscardUnknown(m)
}

var xxx_messageInfo_ProjectConfig proto.InternalMessageInfo

func (m *ProjectConfig) GetNotifiers() []*Notifier {
	if m != nil {
		return m.Notifiers
	}
	return nil
}

// Notifier contains a set of notification configurations (which specify
// triggers to send notifications on) and a set of builders that will be
// watched for these triggers.
type Notifier struct {
	// Name is an identifier for the notifier which must be unique within a
	// project.
	//
	// Name must additionally match ^[a-z\-]+$, meaning it must only
	// use an alphabet of lowercase characters and hyphens.
	//
	// Required.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Notifications is a list of notification configurations.
	Notifications []*Notification `protobuf:"bytes,2,rep,name=notifications,proto3" json:"notifications,omitempty"`
	// Builders is a list of buildbucket builders this Notifier should watch.
	Builders             []*Builder `protobuf:"bytes,3,rep,name=builders,proto3" json:"builders,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *Notifier) Reset()         { *m = Notifier{} }
func (m *Notifier) String() string { return proto.CompactTextString(m) }
func (*Notifier) ProtoMessage()    {}
func (*Notifier) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{1}
}

func (m *Notifier) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Notifier.Unmarshal(m, b)
}
func (m *Notifier) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Notifier.Marshal(b, m, deterministic)
}
func (m *Notifier) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Notifier.Merge(m, src)
}
func (m *Notifier) XXX_Size() int {
	return xxx_messageInfo_Notifier.Size(m)
}
func (m *Notifier) XXX_DiscardUnknown() {
	xxx_messageInfo_Notifier.DiscardUnknown(m)
}

var xxx_messageInfo_Notifier proto.InternalMessageInfo

func (m *Notifier) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Notifier) GetNotifications() []*Notification {
	if m != nil {
		return m.Notifications
	}
	return nil
}

func (m *Notifier) GetBuilders() []*Builder {
	if m != nil {
		return m.Builders
	}
	return nil
}

// Notification specifies the triggers to watch for and send
// notifications on. It also specifies email recipients.
//
// Next ID: 11.
type Notification struct {
	// Deprecated. Notify on each build success.
	OnSuccess bool `protobuf:"varint,1,opt,name=on_success,json=onSuccess,proto3" json:"on_success,omitempty"`
	// Deprecated. Notify on each build failure.
	OnFailure bool `protobuf:"varint,2,opt,name=on_failure,json=onFailure,proto3" json:"on_failure,omitempty"`
	// Deprecated. Notify on each build status different than the previous one.
	OnChange bool `protobuf:"varint,3,opt,name=on_change,json=onChange,proto3" json:"on_change,omitempty"`
	// Deprecated. Notify on each build failure unless the previous build was a
	// failure.
	OnNewFailure bool `protobuf:"varint,7,opt,name=on_new_failure,json=onNewFailure,proto3" json:"on_new_failure,omitempty"`
	// Notify on each build with a specified status.
	OnOccurrence []proto1.Status `protobuf:"varint,9,rep,packed,name=on_occurrence,json=onOccurrence,proto3,enum=buildbucket.v2.Status" json:"on_occurrence,omitempty"`
	// Notify on each build with a specified status different than the previous
	// one.
	OnNewStatus []proto1.Status `protobuf:"varint,10,rep,packed,name=on_new_status,json=onNewStatus,proto3,enum=buildbucket.v2.Status" json:"on_new_status,omitempty"`
	// Email is the set of email addresses to notify.
	//
	// Optional.
	Email *Notification_Email `protobuf:"bytes,4,opt,name=email,proto3" json:"email,omitempty"`
	// Refers to which project template name to use to format this email.
	// If not present, "default" will be used.
	//
	// Optional.
	Template string `protobuf:"bytes,5,opt,name=template,proto3" json:"template,omitempty"`
	// NotifyBlamelist specifies whether to notify the computed blamelist for a
	// given build.
	//
	// If set, this notification will be sent to the blamelist of a build. Note
	// that if this is set in multiple notifications pertaining to the same
	// builder, the blamelist may receive multiple emails.
	//
	// Optional.
	NotifyBlamelist      *Notification_Blamelist `protobuf:"bytes,6,opt,name=notify_blamelist,json=notifyBlamelist,proto3" json:"notify_blamelist,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *Notification) Reset()         { *m = Notification{} }
func (m *Notification) String() string { return proto.CompactTextString(m) }
func (*Notification) ProtoMessage()    {}
func (*Notification) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{2}
}

func (m *Notification) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Notification.Unmarshal(m, b)
}
func (m *Notification) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Notification.Marshal(b, m, deterministic)
}
func (m *Notification) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Notification.Merge(m, src)
}
func (m *Notification) XXX_Size() int {
	return xxx_messageInfo_Notification.Size(m)
}
func (m *Notification) XXX_DiscardUnknown() {
	xxx_messageInfo_Notification.DiscardUnknown(m)
}

var xxx_messageInfo_Notification proto.InternalMessageInfo

func (m *Notification) GetOnSuccess() bool {
	if m != nil {
		return m.OnSuccess
	}
	return false
}

func (m *Notification) GetOnFailure() bool {
	if m != nil {
		return m.OnFailure
	}
	return false
}

func (m *Notification) GetOnChange() bool {
	if m != nil {
		return m.OnChange
	}
	return false
}

func (m *Notification) GetOnNewFailure() bool {
	if m != nil {
		return m.OnNewFailure
	}
	return false
}

func (m *Notification) GetOnOccurrence() []proto1.Status {
	if m != nil {
		return m.OnOccurrence
	}
	return nil
}

func (m *Notification) GetOnNewStatus() []proto1.Status {
	if m != nil {
		return m.OnNewStatus
	}
	return nil
}

func (m *Notification) GetEmail() *Notification_Email {
	if m != nil {
		return m.Email
	}
	return nil
}

func (m *Notification) GetTemplate() string {
	if m != nil {
		return m.Template
	}
	return ""
}

func (m *Notification) GetNotifyBlamelist() *Notification_Blamelist {
	if m != nil {
		return m.NotifyBlamelist
	}
	return nil
}

// Email is a message representing a set of mail recipients (email
// addresses).
type Notification_Email struct {
	// Recipients is a list of email addresses to notify.
	Recipients           []string `protobuf:"bytes,1,rep,name=recipients,proto3" json:"recipients,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Notification_Email) Reset()         { *m = Notification_Email{} }
func (m *Notification_Email) String() string { return proto.CompactTextString(m) }
func (*Notification_Email) ProtoMessage()    {}
func (*Notification_Email) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{2, 0}
}

func (m *Notification_Email) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Notification_Email.Unmarshal(m, b)
}
func (m *Notification_Email) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Notification_Email.Marshal(b, m, deterministic)
}
func (m *Notification_Email) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Notification_Email.Merge(m, src)
}
func (m *Notification_Email) XXX_Size() int {
	return xxx_messageInfo_Notification_Email.Size(m)
}
func (m *Notification_Email) XXX_DiscardUnknown() {
	xxx_messageInfo_Notification_Email.DiscardUnknown(m)
}

var xxx_messageInfo_Notification_Email proto.InternalMessageInfo

func (m *Notification_Email) GetRecipients() []string {
	if m != nil {
		return m.Recipients
	}
	return nil
}

// Blamelist is a message representing configuration for notifying the
// blamelist.
type Notification_Blamelist struct {
	// A list of repositories which we are allowed to be included as part of the
	// blamelist. If unset, a blamelist will be computed based on a Builder's
	// repository field. If set, however luci-notify computes the blamelist for
	// all commits related to a build (which may span multiple repositories)
	// which are part of repository in this repository whitelist.
	//
	// Repositories should be valid Gerrit/Gitiles repository URLs, such as
	// https://chromium.googlesource.com/chromium/src
	//
	// Optional.
	RepositoryWhitelist  []string `protobuf:"bytes,1,rep,name=repository_whitelist,json=repositoryWhitelist,proto3" json:"repository_whitelist,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Notification_Blamelist) Reset()         { *m = Notification_Blamelist{} }
func (m *Notification_Blamelist) String() string { return proto.CompactTextString(m) }
func (*Notification_Blamelist) ProtoMessage()    {}
func (*Notification_Blamelist) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{2, 1}
}

func (m *Notification_Blamelist) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Notification_Blamelist.Unmarshal(m, b)
}
func (m *Notification_Blamelist) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Notification_Blamelist.Marshal(b, m, deterministic)
}
func (m *Notification_Blamelist) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Notification_Blamelist.Merge(m, src)
}
func (m *Notification_Blamelist) XXX_Size() int {
	return xxx_messageInfo_Notification_Blamelist.Size(m)
}
func (m *Notification_Blamelist) XXX_DiscardUnknown() {
	xxx_messageInfo_Notification_Blamelist.DiscardUnknown(m)
}

var xxx_messageInfo_Notification_Blamelist proto.InternalMessageInfo

func (m *Notification_Blamelist) GetRepositoryWhitelist() []string {
	if m != nil {
		return m.RepositoryWhitelist
	}
	return nil
}

// Builder references a buildbucket builder in the current project.
type Builder struct {
	// Bucket is the buildbucket bucket that the builder is a part of.
	//
	// Required.
	Bucket string `protobuf:"bytes,1,opt,name=bucket,proto3" json:"bucket,omitempty"`
	// Name is the name of the buildbucket builder.
	//
	// Required.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Repository is the git repository associated with this particular builder.
	//
	// The repository should look like a URL, e.g.
	// https://chromium.googlesource.com/src
	//
	// Currently, luci-notify only supports Gerrit-like URLs since it checks
	// against gitiles commits, so the URL's path (e.g. "src" in the above
	// example) should map directly to a Gerrit project.
	//
	// Builds attached to the history of this repository will use this
	// repository's git history to determine the order between two builds for the
	// OnChange notification.
	//
	// Optional.
	//
	// If not set, OnChange notifications will derive their notion of
	// "previous" build solely from build creation time, which is potentially
	// less reliable.
	Repository           string   `protobuf:"bytes,3,opt,name=repository,proto3" json:"repository,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Builder) Reset()         { *m = Builder{} }
func (m *Builder) String() string { return proto.CompactTextString(m) }
func (*Builder) ProtoMessage()    {}
func (*Builder) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{3}
}

func (m *Builder) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Builder.Unmarshal(m, b)
}
func (m *Builder) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Builder.Marshal(b, m, deterministic)
}
func (m *Builder) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Builder.Merge(m, src)
}
func (m *Builder) XXX_Size() int {
	return xxx_messageInfo_Builder.Size(m)
}
func (m *Builder) XXX_DiscardUnknown() {
	xxx_messageInfo_Builder.DiscardUnknown(m)
}

var xxx_messageInfo_Builder proto.InternalMessageInfo

func (m *Builder) GetBucket() string {
	if m != nil {
		return m.Bucket
	}
	return ""
}

func (m *Builder) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Builder) GetRepository() string {
	if m != nil {
		return m.Repository
	}
	return ""
}

// Notifications encapsulates a list of notifications as a proto so code for
// storing it in the datastore may be generated.
type Notifications struct {
	// Notifications is a list of notification configurations.
	Notifications        []*Notification `protobuf:"bytes,1,rep,name=notifications,proto3" json:"notifications,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *Notifications) Reset()         { *m = Notifications{} }
func (m *Notifications) String() string { return proto.CompactTextString(m) }
func (*Notifications) ProtoMessage()    {}
func (*Notifications) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{4}
}

func (m *Notifications) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Notifications.Unmarshal(m, b)
}
func (m *Notifications) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Notifications.Marshal(b, m, deterministic)
}
func (m *Notifications) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Notifications.Merge(m, src)
}
func (m *Notifications) XXX_Size() int {
	return xxx_messageInfo_Notifications.Size(m)
}
func (m *Notifications) XXX_DiscardUnknown() {
	xxx_messageInfo_Notifications.DiscardUnknown(m)
}

var xxx_messageInfo_Notifications proto.InternalMessageInfo

func (m *Notifications) GetNotifications() []*Notification {
	if m != nil {
		return m.Notifications
	}
	return nil
}

// A collection of landed Git commits hosted on Gitiles.
type GitilesCommits struct {
	// The Gitiles commits in this collection.
	Commits              []*proto1.GitilesCommit `protobuf:"bytes,1,rep,name=commits,proto3" json:"commits,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *GitilesCommits) Reset()         { *m = GitilesCommits{} }
func (m *GitilesCommits) String() string { return proto.CompactTextString(m) }
func (*GitilesCommits) ProtoMessage()    {}
func (*GitilesCommits) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{5}
}

func (m *GitilesCommits) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GitilesCommits.Unmarshal(m, b)
}
func (m *GitilesCommits) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GitilesCommits.Marshal(b, m, deterministic)
}
func (m *GitilesCommits) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GitilesCommits.Merge(m, src)
}
func (m *GitilesCommits) XXX_Size() int {
	return xxx_messageInfo_GitilesCommits.Size(m)
}
func (m *GitilesCommits) XXX_DiscardUnknown() {
	xxx_messageInfo_GitilesCommits.DiscardUnknown(m)
}

var xxx_messageInfo_GitilesCommits proto.InternalMessageInfo

func (m *GitilesCommits) GetCommits() []*proto1.GitilesCommit {
	if m != nil {
		return m.Commits
	}
	return nil
}

// Input to an email template.
type TemplateInput struct {
	// Buildbucket hostname, e.g. "cr-buildbucket.appspot.com".
	BuildbucketHostname string `protobuf:"bytes,1,opt,name=buildbucket_hostname,json=buildbucketHostname,proto3" json:"buildbucket_hostname,omitempty"`
	// The completed build.
	Build *proto1.Build `protobuf:"bytes,2,opt,name=build,proto3" json:"build,omitempty"`
	// State of the previous build in this builder.
	OldStatus            proto1.Status `protobuf:"varint,3,opt,name=old_status,json=oldStatus,proto3,enum=buildbucket.v2.Status" json:"old_status,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *TemplateInput) Reset()         { *m = TemplateInput{} }
func (m *TemplateInput) String() string { return proto.CompactTextString(m) }
func (*TemplateInput) ProtoMessage()    {}
func (*TemplateInput) Descriptor() ([]byte, []int) {
	return fileDescriptor_9a6945a7af0ec43b, []int{6}
}

func (m *TemplateInput) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TemplateInput.Unmarshal(m, b)
}
func (m *TemplateInput) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TemplateInput.Marshal(b, m, deterministic)
}
func (m *TemplateInput) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TemplateInput.Merge(m, src)
}
func (m *TemplateInput) XXX_Size() int {
	return xxx_messageInfo_TemplateInput.Size(m)
}
func (m *TemplateInput) XXX_DiscardUnknown() {
	xxx_messageInfo_TemplateInput.DiscardUnknown(m)
}

var xxx_messageInfo_TemplateInput proto.InternalMessageInfo

func (m *TemplateInput) GetBuildbucketHostname() string {
	if m != nil {
		return m.BuildbucketHostname
	}
	return ""
}

func (m *TemplateInput) GetBuild() *proto1.Build {
	if m != nil {
		return m.Build
	}
	return nil
}

func (m *TemplateInput) GetOldStatus() proto1.Status {
	if m != nil {
		return m.OldStatus
	}
	return proto1.Status_STATUS_UNSPECIFIED
}

func init() {
	proto.RegisterType((*ProjectConfig)(nil), "notify.ProjectConfig")
	proto.RegisterType((*Notifier)(nil), "notify.Notifier")
	proto.RegisterType((*Notification)(nil), "notify.Notification")
	proto.RegisterType((*Notification_Email)(nil), "notify.Notification.Email")
	proto.RegisterType((*Notification_Blamelist)(nil), "notify.Notification.Blamelist")
	proto.RegisterType((*Builder)(nil), "notify.Builder")
	proto.RegisterType((*Notifications)(nil), "notify.Notifications")
	proto.RegisterType((*GitilesCommits)(nil), "notify.GitilesCommits")
	proto.RegisterType((*TemplateInput)(nil), "notify.TemplateInput")
}

func init() {
	proto.RegisterFile("go.chromium.org/luci/luci_notify/api/config/notify.proto", fileDescriptor_9a6945a7af0ec43b)
}

var fileDescriptor_9a6945a7af0ec43b = []byte{
	// 676 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x54, 0x5d, 0x6b, 0x13, 0x4d,
	0x14, 0x26, 0xcd, 0x47, 0xb3, 0x27, 0x4d, 0x5b, 0xa6, 0x7d, 0xcb, 0x92, 0x97, 0x96, 0x10, 0x05,
	0x03, 0xc5, 0x5d, 0x4d, 0x11, 0xa5, 0x82, 0x4a, 0x8a, 0x1f, 0x55, 0x88, 0xb2, 0x55, 0x04, 0x6f,
	0x96, 0xcd, 0x74, 0x92, 0x8c, 0xee, 0xce, 0x2c, 0x3b, 0xb3, 0x96, 0xfe, 0x02, 0x6f, 0xfd, 0x0d,
	0x5e, 0xf8, 0x33, 0x45, 0xf6, 0xcc, 0xee, 0x76, 0x53, 0xa3, 0xf4, 0x26, 0xcc, 0x3c, 0x1f, 0x67,
	0xce, 0x9e, 0x79, 0x32, 0xf0, 0x68, 0x2e, 0x1d, 0xba, 0x48, 0x64, 0xc4, 0xd3, 0xc8, 0x91, 0xc9,
	0xdc, 0x0d, 0x53, 0xca, 0xf1, 0xc7, 0x17, 0x52, 0xf3, 0xd9, 0xa5, 0x1b, 0xc4, 0xdc, 0xa5, 0x52,
	0xcc, 0xf8, 0xdc, 0x35, 0x88, 0x13, 0x27, 0x52, 0x4b, 0xd2, 0x32, 0xbb, 0xde, 0x68, 0x65, 0x85,
	0x69, 0xca, 0xc3, 0xf3, 0x69, 0x4a, 0xbf, 0x30, 0xed, 0xa2, 0xde, 0x20, 0xc6, 0xdb, 0x3b, 0xba,
	0xa1, 0x87, 0xca, 0x28, 0x92, 0x22, 0x37, 0xb9, 0x2b, 0x4d, 0x46, 0x92, 0xeb, 0x65, 0xac, 0xb9,
	0x14, 0xca, 0x18, 0x06, 0x4f, 0xa1, 0xfb, 0x2e, 0x91, 0x9f, 0x19, 0xd5, 0x27, 0xd8, 0x3f, 0x71,
	0xc0, 0xc2, 0xa6, 0x39, 0x4b, 0x94, 0x5d, 0xeb, 0xd7, 0x87, 0x9d, 0xd1, 0xb6, 0x93, 0x7f, 0xd4,
	0x24, 0x27, 0xbc, 0x2b, 0xc9, 0xe0, 0x5b, 0x0d, 0xda, 0x05, 0x4e, 0x08, 0x34, 0x44, 0x10, 0x31,
	0xbb, 0xd6, 0xaf, 0x0d, 0x2d, 0x0f, 0xd7, 0xe4, 0x18, 0xba, 0x46, 0x4d, 0x03, 0x3c, 0xd8, 0x5e,
	0xc3, 0xa2, 0xbb, 0xcb, 0x45, 0x0d, 0xe9, 0x2d, 0x4b, 0xc9, 0x21, 0xb4, 0xf1, 0x83, 0xb3, 0x5e,
	0xea, 0x68, 0xdb, 0x2a, 0x6c, 0x63, 0x83, 0x7b, 0xa5, 0x60, 0xf0, 0xbd, 0x01, 0x1b, 0xd5, 0x62,
	0x64, 0x1f, 0x40, 0x0a, 0x5f, 0xa5, 0x94, 0x32, 0xa5, 0xb0, 0xa7, 0xb6, 0x67, 0x49, 0x71, 0x66,
	0x80, 0x9c, 0x9e, 0x05, 0x3c, 0x4c, 0x13, 0x66, 0xaf, 0x15, 0xf4, 0x0b, 0x03, 0x90, 0xff, 0xc1,
	0x92, 0xc2, 0xa7, 0x8b, 0x40, 0xcc, 0x99, 0x5d, 0x47, 0xb6, 0x2d, 0xc5, 0x09, 0xee, 0xc9, 0x6d,
	0xd8, 0x94, 0xc2, 0x17, 0xec, 0xa2, 0xf4, 0xaf, 0xa3, 0x62, 0x43, 0x8a, 0x09, 0xbb, 0x28, 0x4a,
	0x3c, 0x86, 0xae, 0x14, 0xbe, 0xa4, 0x34, 0x4d, 0x12, 0x26, 0x28, 0xb3, 0xad, 0x7e, 0x7d, 0xb8,
	0x39, 0xda, 0x73, 0x2a, 0xb7, 0xe8, 0x7c, 0x1d, 0x39, 0x67, 0x3a, 0xd0, 0xa9, 0xca, 0xcc, 0x6f,
	0x4b, 0x6d, 0x36, 0xb7, 0xfc, 0x08, 0x85, 0xb4, 0x0d, 0xff, 0x34, 0x77, 0xf0, 0x64, 0xb3, 0x21,
	0xf7, 0xa0, 0xc9, 0xa2, 0x80, 0x87, 0x76, 0xa3, 0x5f, 0x1b, 0x76, 0x46, 0xbd, 0x55, 0xb3, 0x76,
	0x9e, 0x67, 0x0a, 0xcf, 0x08, 0x49, 0x0f, 0xda, 0x9a, 0x45, 0x71, 0x18, 0x68, 0x66, 0x37, 0xf1,
	0xf6, 0xca, 0x3d, 0x39, 0x85, 0x6d, 0xe3, 0xf7, 0xa7, 0x61, 0x10, 0xb1, 0x90, 0x2b, 0x6d, 0xb7,
	0xb0, 0xf0, 0xc1, 0xca, 0xc2, 0xe3, 0x42, 0xe5, 0x6d, 0x19, 0xba, 0x04, 0x7a, 0x77, 0xa0, 0x89,
	0xc7, 0x92, 0x03, 0x80, 0x84, 0x51, 0x1e, 0x73, 0x26, 0xb4, 0xc9, 0x99, 0xe5, 0x55, 0x90, 0xde,
	0x13, 0xb0, 0x4a, 0x17, 0xb9, 0x0f, 0xbb, 0x09, 0x8b, 0xa5, 0xe2, 0x5a, 0x26, 0x97, 0xfe, 0xc5,
	0x82, 0x6b, 0xd3, 0x84, 0xb1, 0xed, 0x5c, 0x71, 0x1f, 0x0b, 0xea, 0x75, 0xa3, 0xdd, 0xde, 0xb6,
	0x06, 0x1f, 0x60, 0x3d, 0xcf, 0x09, 0xd9, 0x83, 0x96, 0x99, 0x59, 0x1e, 0xce, 0x7c, 0x57, 0x46,
	0x76, 0xad, 0x12, 0x59, 0x6c, 0xae, 0xa8, 0x89, 0x77, 0x8f, 0xcd, 0x15, 0xc8, 0xe0, 0x0d, 0x74,
	0x27, 0x4b, 0x39, 0xfd, 0x23, 0xe3, 0xb5, 0x1b, 0x67, 0x7c, 0x70, 0x0a, 0x9b, 0x2f, 0xb9, 0xe6,
	0x21, 0x53, 0x27, 0x32, 0x8a, 0xb8, 0x56, 0xe4, 0x21, 0xac, 0x53, 0xb3, 0xcc, 0xeb, 0xec, 0x5f,
	0xbf, 0xf3, 0x25, 0x83, 0x57, 0xa8, 0x07, 0x3f, 0x6b, 0xd0, 0x7d, 0x9f, 0xdf, 0xda, 0xa9, 0x88,
	0x53, 0x9c, 0x5c, 0xc5, 0xea, 0x2f, 0xa4, 0xd2, 0x95, 0x3f, 0xe8, 0x4e, 0x85, 0x7b, 0x95, 0x53,
	0xe4, 0x10, 0x9a, 0x08, 0xe3, 0x44, 0x3a, 0xa3, 0xff, 0xae, 0x9f, 0x8d, 0x03, 0xf5, 0x8c, 0x86,
	0x3c, 0x00, 0x90, 0xe1, 0x79, 0x91, 0xd0, 0x6c, 0x52, 0x7f, 0x4f, 0xa8, 0x25, 0xc3, 0x73, 0xb3,
	0x1c, 0x4f, 0x7e, 0xfc, 0xba, 0x35, 0x86, 0x67, 0x0b, 0xad, 0x63, 0x75, 0xec, 0xe2, 0x23, 0x75,
	0xd7, 0xbc, 0x9f, 0x4e, 0x10, 0xc7, 0x2a, 0x96, 0xda, 0xa1, 0x32, 0x72, 0x15, 0x5d, 0xb0, 0x28,
	0x50, 0xd9, 0xc3, 0x95, 0xbd, 0x50, 0xea, 0x18, 0x85, 0xf9, 0x4c, 0xe9, 0x6c, 0xfe, 0xa9, 0x65,
	0x4c, 0xd3, 0x16, 0x3e, 0x66, 0x47, 0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0x23, 0x2a, 0x29, 0xdb,
	0xaa, 0x05, 0x00, 0x00,
}
