// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/tetrafolium/luci-go/common/data/text/templateproto/template.proto

package templateproto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
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

// Value defines a specific value for a parameter, and is used at Template
// expansion time.
type Value struct {
	// Types that are valid to be assigned to Value:
	//	*Value_Int
	//	*Value_Uint
	//	*Value_Float
	//	*Value_Bool
	//	*Value_Str
	//	*Value_Bytes
	//	*Value_Object
	//	*Value_Array
	//	*Value_Null
	Value                isValue_Value `protobuf_oneof:"value"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *Value) Reset()         { *m = Value{} }
func (m *Value) String() string { return proto.CompactTextString(m) }
func (*Value) ProtoMessage()    {}
func (*Value) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{0}
}

func (m *Value) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Value.Unmarshal(m, b)
}
func (m *Value) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Value.Marshal(b, m, deterministic)
}
func (m *Value) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Value.Merge(m, src)
}
func (m *Value) XXX_Size() int {
	return xxx_messageInfo_Value.Size(m)
}
func (m *Value) XXX_DiscardUnknown() {
	xxx_messageInfo_Value.DiscardUnknown(m)
}

var xxx_messageInfo_Value proto.InternalMessageInfo

type isValue_Value interface {
	isValue_Value()
}

type Value_Int struct {
	Int int64 `protobuf:"varint,1,opt,name=int,proto3,oneof"`
}

type Value_Uint struct {
	Uint uint64 `protobuf:"varint,2,opt,name=uint,proto3,oneof"`
}

type Value_Float struct {
	Float float64 `protobuf:"fixed64,3,opt,name=float,proto3,oneof"`
}

type Value_Bool struct {
	Bool bool `protobuf:"varint,4,opt,name=bool,proto3,oneof"`
}

type Value_Str struct {
	Str string `protobuf:"bytes,5,opt,name=str,proto3,oneof"`
}

type Value_Bytes struct {
	Bytes []byte `protobuf:"bytes,6,opt,name=bytes,proto3,oneof"`
}

type Value_Object struct {
	Object string `protobuf:"bytes,7,opt,name=object,proto3,oneof"`
}

type Value_Array struct {
	Array string `protobuf:"bytes,8,opt,name=array,proto3,oneof"`
}

type Value_Null struct {
	Null *empty.Empty `protobuf:"bytes,9,opt,name=null,proto3,oneof"`
}

func (*Value_Int) isValue_Value() {}

func (*Value_Uint) isValue_Value() {}

func (*Value_Float) isValue_Value() {}

func (*Value_Bool) isValue_Value() {}

func (*Value_Str) isValue_Value() {}

func (*Value_Bytes) isValue_Value() {}

func (*Value_Object) isValue_Value() {}

func (*Value_Array) isValue_Value() {}

func (*Value_Null) isValue_Value() {}

func (m *Value) GetValue() isValue_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Value) GetInt() int64 {
	if x, ok := m.GetValue().(*Value_Int); ok {
		return x.Int
	}
	return 0
}

func (m *Value) GetUint() uint64 {
	if x, ok := m.GetValue().(*Value_Uint); ok {
		return x.Uint
	}
	return 0
}

func (m *Value) GetFloat() float64 {
	if x, ok := m.GetValue().(*Value_Float); ok {
		return x.Float
	}
	return 0
}

func (m *Value) GetBool() bool {
	if x, ok := m.GetValue().(*Value_Bool); ok {
		return x.Bool
	}
	return false
}

func (m *Value) GetStr() string {
	if x, ok := m.GetValue().(*Value_Str); ok {
		return x.Str
	}
	return ""
}

func (m *Value) GetBytes() []byte {
	if x, ok := m.GetValue().(*Value_Bytes); ok {
		return x.Bytes
	}
	return nil
}

func (m *Value) GetObject() string {
	if x, ok := m.GetValue().(*Value_Object); ok {
		return x.Object
	}
	return ""
}

func (m *Value) GetArray() string {
	if x, ok := m.GetValue().(*Value_Array); ok {
		return x.Array
	}
	return ""
}

func (m *Value) GetNull() *empty.Empty {
	if x, ok := m.GetValue().(*Value_Null); ok {
		return x.Null
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Value) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Value_Int)(nil),
		(*Value_Uint)(nil),
		(*Value_Float)(nil),
		(*Value_Bool)(nil),
		(*Value_Str)(nil),
		(*Value_Bytes)(nil),
		(*Value_Object)(nil),
		(*Value_Array)(nil),
		(*Value_Null)(nil),
	}
}

type Schema struct {
	// Types that are valid to be assigned to Schema:
	//	*Schema_Int
	//	*Schema_Uint
	//	*Schema_Float
	//	*Schema_Bool
	//	*Schema_Str
	//	*Schema_Bytes
	//	*Schema_Enum
	//	*Schema_Object
	//	*Schema_Array
	Schema               isSchema_Schema `protobuf_oneof:"schema"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *Schema) Reset()         { *m = Schema{} }
func (m *Schema) String() string { return proto.CompactTextString(m) }
func (*Schema) ProtoMessage()    {}
func (*Schema) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{1}
}

func (m *Schema) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Schema.Unmarshal(m, b)
}
func (m *Schema) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Schema.Marshal(b, m, deterministic)
}
func (m *Schema) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Schema.Merge(m, src)
}
func (m *Schema) XXX_Size() int {
	return xxx_messageInfo_Schema.Size(m)
}
func (m *Schema) XXX_DiscardUnknown() {
	xxx_messageInfo_Schema.DiscardUnknown(m)
}

var xxx_messageInfo_Schema proto.InternalMessageInfo

type isSchema_Schema interface {
	isSchema_Schema()
}

type Schema_Int struct {
	Int *Schema_Atom `protobuf:"bytes,1,opt,name=int,proto3,oneof"`
}

type Schema_Uint struct {
	Uint *Schema_Atom `protobuf:"bytes,2,opt,name=uint,proto3,oneof"`
}

type Schema_Float struct {
	Float *Schema_Atom `protobuf:"bytes,3,opt,name=float,proto3,oneof"`
}

type Schema_Bool struct {
	Bool *Schema_Atom `protobuf:"bytes,4,opt,name=bool,proto3,oneof"`
}

type Schema_Str struct {
	Str *Schema_Sequence `protobuf:"bytes,5,opt,name=str,proto3,oneof"`
}

type Schema_Bytes struct {
	Bytes *Schema_Sequence `protobuf:"bytes,6,opt,name=bytes,proto3,oneof"`
}

type Schema_Enum struct {
	Enum *Schema_Set `protobuf:"bytes,7,opt,name=enum,proto3,oneof"`
}

type Schema_Object struct {
	Object *Schema_JSON `protobuf:"bytes,8,opt,name=object,proto3,oneof"`
}

type Schema_Array struct {
	Array *Schema_JSON `protobuf:"bytes,9,opt,name=array,proto3,oneof"`
}

func (*Schema_Int) isSchema_Schema() {}

func (*Schema_Uint) isSchema_Schema() {}

func (*Schema_Float) isSchema_Schema() {}

func (*Schema_Bool) isSchema_Schema() {}

func (*Schema_Str) isSchema_Schema() {}

func (*Schema_Bytes) isSchema_Schema() {}

func (*Schema_Enum) isSchema_Schema() {}

func (*Schema_Object) isSchema_Schema() {}

func (*Schema_Array) isSchema_Schema() {}

func (m *Schema) GetSchema() isSchema_Schema {
	if m != nil {
		return m.Schema
	}
	return nil
}

func (m *Schema) GetInt() *Schema_Atom {
	if x, ok := m.GetSchema().(*Schema_Int); ok {
		return x.Int
	}
	return nil
}

func (m *Schema) GetUint() *Schema_Atom {
	if x, ok := m.GetSchema().(*Schema_Uint); ok {
		return x.Uint
	}
	return nil
}

func (m *Schema) GetFloat() *Schema_Atom {
	if x, ok := m.GetSchema().(*Schema_Float); ok {
		return x.Float
	}
	return nil
}

func (m *Schema) GetBool() *Schema_Atom {
	if x, ok := m.GetSchema().(*Schema_Bool); ok {
		return x.Bool
	}
	return nil
}

func (m *Schema) GetStr() *Schema_Sequence {
	if x, ok := m.GetSchema().(*Schema_Str); ok {
		return x.Str
	}
	return nil
}

func (m *Schema) GetBytes() *Schema_Sequence {
	if x, ok := m.GetSchema().(*Schema_Bytes); ok {
		return x.Bytes
	}
	return nil
}

func (m *Schema) GetEnum() *Schema_Set {
	if x, ok := m.GetSchema().(*Schema_Enum); ok {
		return x.Enum
	}
	return nil
}

func (m *Schema) GetObject() *Schema_JSON {
	if x, ok := m.GetSchema().(*Schema_Object); ok {
		return x.Object
	}
	return nil
}

func (m *Schema) GetArray() *Schema_JSON {
	if x, ok := m.GetSchema().(*Schema_Array); ok {
		return x.Array
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Schema) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Schema_Int)(nil),
		(*Schema_Uint)(nil),
		(*Schema_Float)(nil),
		(*Schema_Bool)(nil),
		(*Schema_Str)(nil),
		(*Schema_Bytes)(nil),
		(*Schema_Enum)(nil),
		(*Schema_Object)(nil),
		(*Schema_Array)(nil),
	}
}

type Schema_Set struct {
	// entry lists the possible tokens that this set can have.
	Entry                []*Schema_Set_Entry `protobuf:"bytes,1,rep,name=entry,proto3" json:"entry,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *Schema_Set) Reset()         { *m = Schema_Set{} }
func (m *Schema_Set) String() string { return proto.CompactTextString(m) }
func (*Schema_Set) ProtoMessage()    {}
func (*Schema_Set) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{1, 0}
}

func (m *Schema_Set) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Schema_Set.Unmarshal(m, b)
}
func (m *Schema_Set) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Schema_Set.Marshal(b, m, deterministic)
}
func (m *Schema_Set) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Schema_Set.Merge(m, src)
}
func (m *Schema_Set) XXX_Size() int {
	return xxx_messageInfo_Schema_Set.Size(m)
}
func (m *Schema_Set) XXX_DiscardUnknown() {
	xxx_messageInfo_Schema_Set.DiscardUnknown(m)
}

var xxx_messageInfo_Schema_Set proto.InternalMessageInfo

func (m *Schema_Set) GetEntry() []*Schema_Set_Entry {
	if m != nil {
		return m.Entry
	}
	return nil
}

type Schema_Set_Entry struct {
	// Markdown-formatted documentation for this schema entry.
	Doc                  string   `protobuf:"bytes,1,opt,name=doc,proto3" json:"doc,omitempty"`
	Token                string   `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Schema_Set_Entry) Reset()         { *m = Schema_Set_Entry{} }
func (m *Schema_Set_Entry) String() string { return proto.CompactTextString(m) }
func (*Schema_Set_Entry) ProtoMessage()    {}
func (*Schema_Set_Entry) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{1, 0, 0}
}

func (m *Schema_Set_Entry) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Schema_Set_Entry.Unmarshal(m, b)
}
func (m *Schema_Set_Entry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Schema_Set_Entry.Marshal(b, m, deterministic)
}
func (m *Schema_Set_Entry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Schema_Set_Entry.Merge(m, src)
}
func (m *Schema_Set_Entry) XXX_Size() int {
	return xxx_messageInfo_Schema_Set_Entry.Size(m)
}
func (m *Schema_Set_Entry) XXX_DiscardUnknown() {
	xxx_messageInfo_Schema_Set_Entry.DiscardUnknown(m)
}

var xxx_messageInfo_Schema_Set_Entry proto.InternalMessageInfo

func (m *Schema_Set_Entry) GetDoc() string {
	if m != nil {
		return m.Doc
	}
	return ""
}

func (m *Schema_Set_Entry) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

type Schema_JSON struct {
	// restricts the maximum amount of bytes that a Value for this field may
	// take.
	MaxLength            uint32   `protobuf:"varint,1,opt,name=max_length,json=maxLength,proto3" json:"max_length,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Schema_JSON) Reset()         { *m = Schema_JSON{} }
func (m *Schema_JSON) String() string { return proto.CompactTextString(m) }
func (*Schema_JSON) ProtoMessage()    {}
func (*Schema_JSON) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{1, 1}
}

func (m *Schema_JSON) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Schema_JSON.Unmarshal(m, b)
}
func (m *Schema_JSON) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Schema_JSON.Marshal(b, m, deterministic)
}
func (m *Schema_JSON) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Schema_JSON.Merge(m, src)
}
func (m *Schema_JSON) XXX_Size() int {
	return xxx_messageInfo_Schema_JSON.Size(m)
}
func (m *Schema_JSON) XXX_DiscardUnknown() {
	xxx_messageInfo_Schema_JSON.DiscardUnknown(m)
}

var xxx_messageInfo_Schema_JSON proto.InternalMessageInfo

func (m *Schema_JSON) GetMaxLength() uint32 {
	if m != nil {
		return m.MaxLength
	}
	return 0
}

type Schema_Sequence struct {
	// restricts the maximum amount of bytes that a Value for this field may
	// take.
	MaxLength            uint32   `protobuf:"varint,1,opt,name=max_length,json=maxLength,proto3" json:"max_length,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Schema_Sequence) Reset()         { *m = Schema_Sequence{} }
func (m *Schema_Sequence) String() string { return proto.CompactTextString(m) }
func (*Schema_Sequence) ProtoMessage()    {}
func (*Schema_Sequence) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{1, 2}
}

func (m *Schema_Sequence) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Schema_Sequence.Unmarshal(m, b)
}
func (m *Schema_Sequence) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Schema_Sequence.Marshal(b, m, deterministic)
}
func (m *Schema_Sequence) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Schema_Sequence.Merge(m, src)
}
func (m *Schema_Sequence) XXX_Size() int {
	return xxx_messageInfo_Schema_Sequence.Size(m)
}
func (m *Schema_Sequence) XXX_DiscardUnknown() {
	xxx_messageInfo_Schema_Sequence.DiscardUnknown(m)
}

var xxx_messageInfo_Schema_Sequence proto.InternalMessageInfo

func (m *Schema_Sequence) GetMaxLength() uint32 {
	if m != nil {
		return m.MaxLength
	}
	return 0
}

type Schema_Atom struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Schema_Atom) Reset()         { *m = Schema_Atom{} }
func (m *Schema_Atom) String() string { return proto.CompactTextString(m) }
func (*Schema_Atom) ProtoMessage()    {}
func (*Schema_Atom) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{1, 3}
}

func (m *Schema_Atom) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Schema_Atom.Unmarshal(m, b)
}
func (m *Schema_Atom) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Schema_Atom.Marshal(b, m, deterministic)
}
func (m *Schema_Atom) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Schema_Atom.Merge(m, src)
}
func (m *Schema_Atom) XXX_Size() int {
	return xxx_messageInfo_Schema_Atom.Size(m)
}
func (m *Schema_Atom) XXX_DiscardUnknown() {
	xxx_messageInfo_Schema_Atom.DiscardUnknown(m)
}

var xxx_messageInfo_Schema_Atom proto.InternalMessageInfo

// File represents a file full of template definitions.
type File struct {
	Template             map[string]*File_Template `protobuf:"bytes,1,rep,name=template,proto3" json:"template,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}                  `json:"-"`
	XXX_unrecognized     []byte                    `json:"-"`
	XXX_sizecache        int32                     `json:"-"`
}

func (m *File) Reset()         { *m = File{} }
func (m *File) String() string { return proto.CompactTextString(m) }
func (*File) ProtoMessage()    {}
func (*File) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{2}
}

func (m *File) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_File.Unmarshal(m, b)
}
func (m *File) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_File.Marshal(b, m, deterministic)
}
func (m *File) XXX_Merge(src proto.Message) {
	xxx_messageInfo_File.Merge(m, src)
}
func (m *File) XXX_Size() int {
	return xxx_messageInfo_File.Size(m)
}
func (m *File) XXX_DiscardUnknown() {
	xxx_messageInfo_File.DiscardUnknown(m)
}

var xxx_messageInfo_File proto.InternalMessageInfo

func (m *File) GetTemplate() map[string]*File_Template {
	if m != nil {
		return m.Template
	}
	return nil
}

// Template defines a single template.
type File_Template struct {
	// Markdown-formatted documentation for this schema entry.
	Doc string `protobuf:"bytes,1,opt,name=doc,proto3" json:"doc,omitempty"`
	// body is the main JSON output for this template. It must have the form
	// of valid json, modulo the substitution parameters. In order for this
	// Template to be valid, body must parse as valid JSON, after all
	// substitutions have been applied.
	Body string `protobuf:"bytes,2,opt,name=body,proto3" json:"body,omitempty"`
	// param is a listing of all of the parameterized bits in the Template body.
	// The key must match the regex /\${[^}]+}/. So "${foo}" would be ok, but
	// "foo", "$foo", or "${}" would not.
	//
	// params provided here must be present in Body at least once in order
	// for the Template to be valid.
	Param                map[string]*File_Template_Parameter `protobuf:"bytes,3,rep,name=param,proto3" json:"param,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}                            `json:"-"`
	XXX_unrecognized     []byte                              `json:"-"`
	XXX_sizecache        int32                               `json:"-"`
}

func (m *File_Template) Reset()         { *m = File_Template{} }
func (m *File_Template) String() string { return proto.CompactTextString(m) }
func (*File_Template) ProtoMessage()    {}
func (*File_Template) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{2, 0}
}

func (m *File_Template) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_File_Template.Unmarshal(m, b)
}
func (m *File_Template) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_File_Template.Marshal(b, m, deterministic)
}
func (m *File_Template) XXX_Merge(src proto.Message) {
	xxx_messageInfo_File_Template.Merge(m, src)
}
func (m *File_Template) XXX_Size() int {
	return xxx_messageInfo_File_Template.Size(m)
}
func (m *File_Template) XXX_DiscardUnknown() {
	xxx_messageInfo_File_Template.DiscardUnknown(m)
}

var xxx_messageInfo_File_Template proto.InternalMessageInfo

func (m *File_Template) GetDoc() string {
	if m != nil {
		return m.Doc
	}
	return ""
}

func (m *File_Template) GetBody() string {
	if m != nil {
		return m.Body
	}
	return ""
}

func (m *File_Template) GetParam() map[string]*File_Template_Parameter {
	if m != nil {
		return m.Param
	}
	return nil
}

type File_Template_Parameter struct {
	// Markdown-formatted documentation for this schema entry.
	Doc     string `protobuf:"bytes,1,opt,name=doc,proto3" json:"doc,omitempty"`
	Default *Value `protobuf:"bytes,2,opt,name=default,proto3" json:"default,omitempty"`
	// nullable indicates if 'null' is a valid value for this parameter. This
	// can be used to distinguish e.g. "" from not-supplied. If default is
	// Value{null: {}}, this must be true.
	Nullable             bool     `protobuf:"varint,3,opt,name=nullable,proto3" json:"nullable,omitempty"`
	Schema               *Schema  `protobuf:"bytes,4,opt,name=schema,proto3" json:"schema,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *File_Template_Parameter) Reset()         { *m = File_Template_Parameter{} }
func (m *File_Template_Parameter) String() string { return proto.CompactTextString(m) }
func (*File_Template_Parameter) ProtoMessage()    {}
func (*File_Template_Parameter) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{2, 0, 0}
}

func (m *File_Template_Parameter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_File_Template_Parameter.Unmarshal(m, b)
}
func (m *File_Template_Parameter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_File_Template_Parameter.Marshal(b, m, deterministic)
}
func (m *File_Template_Parameter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_File_Template_Parameter.Merge(m, src)
}
func (m *File_Template_Parameter) XXX_Size() int {
	return xxx_messageInfo_File_Template_Parameter.Size(m)
}
func (m *File_Template_Parameter) XXX_DiscardUnknown() {
	xxx_messageInfo_File_Template_Parameter.DiscardUnknown(m)
}

var xxx_messageInfo_File_Template_Parameter proto.InternalMessageInfo

func (m *File_Template_Parameter) GetDoc() string {
	if m != nil {
		return m.Doc
	}
	return ""
}

func (m *File_Template_Parameter) GetDefault() *Value {
	if m != nil {
		return m.Default
	}
	return nil
}

func (m *File_Template_Parameter) GetNullable() bool {
	if m != nil {
		return m.Nullable
	}
	return false
}

func (m *File_Template_Parameter) GetSchema() *Schema {
	if m != nil {
		return m.Schema
	}
	return nil
}

type Specifier struct {
	TemplateName         string            `protobuf:"bytes,1,opt,name=template_name,json=templateName,proto3" json:"template_name,omitempty"`
	Params               map[string]*Value `protobuf:"bytes,2,rep,name=params,proto3" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Specifier) Reset()         { *m = Specifier{} }
func (m *Specifier) String() string { return proto.CompactTextString(m) }
func (*Specifier) ProtoMessage()    {}
func (*Specifier) Descriptor() ([]byte, []int) {
	return fileDescriptor_41b1a7e1a454e1ec, []int{3}
}

func (m *Specifier) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Specifier.Unmarshal(m, b)
}
func (m *Specifier) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Specifier.Marshal(b, m, deterministic)
}
func (m *Specifier) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Specifier.Merge(m, src)
}
func (m *Specifier) XXX_Size() int {
	return xxx_messageInfo_Specifier.Size(m)
}
func (m *Specifier) XXX_DiscardUnknown() {
	xxx_messageInfo_Specifier.DiscardUnknown(m)
}

var xxx_messageInfo_Specifier proto.InternalMessageInfo

func (m *Specifier) GetTemplateName() string {
	if m != nil {
		return m.TemplateName
	}
	return ""
}

func (m *Specifier) GetParams() map[string]*Value {
	if m != nil {
		return m.Params
	}
	return nil
}

func init() {
	proto.RegisterType((*Value)(nil), "templateproto.Value")
	proto.RegisterType((*Schema)(nil), "templateproto.Schema")
	proto.RegisterType((*Schema_Set)(nil), "templateproto.Schema.Set")
	proto.RegisterType((*Schema_Set_Entry)(nil), "templateproto.Schema.Set.Entry")
	proto.RegisterType((*Schema_JSON)(nil), "templateproto.Schema.JSON")
	proto.RegisterType((*Schema_Sequence)(nil), "templateproto.Schema.Sequence")
	proto.RegisterType((*Schema_Atom)(nil), "templateproto.Schema.Atom")
	proto.RegisterType((*File)(nil), "templateproto.File")
	proto.RegisterMapType((map[string]*File_Template)(nil), "templateproto.File.TemplateEntry")
	proto.RegisterType((*File_Template)(nil), "templateproto.File.Template")
	proto.RegisterMapType((map[string]*File_Template_Parameter)(nil), "templateproto.File.Template.ParamEntry")
	proto.RegisterType((*File_Template_Parameter)(nil), "templateproto.File.Template.Parameter")
	proto.RegisterType((*Specifier)(nil), "templateproto.Specifier")
	proto.RegisterMapType((map[string]*Value)(nil), "templateproto.Specifier.ParamsEntry")
}

func init() {
	proto.RegisterFile("github.com/tetrafolium/luci-go/common/data/text/templateproto/template.proto", fileDescriptor_41b1a7e1a454e1ec)
}

var fileDescriptor_41b1a7e1a454e1ec = []byte{
	// 733 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x54, 0xdb, 0x6e, 0xd3, 0x4a,
	0x14, 0xad, 0x13, 0x3b, 0xb5, 0x77, 0x5a, 0xe9, 0x68, 0xd4, 0x53, 0xf9, 0xf8, 0x9c, 0x03, 0x26,
	0xdc, 0x0c, 0x82, 0x71, 0x65, 0x2e, 0x42, 0xa8, 0x7d, 0xa0, 0x52, 0x51, 0x85, 0x50, 0x8b, 0x1c,
	0x84, 0xc4, 0x53, 0x99, 0x38, 0x93, 0x34, 0xd4, 0xe3, 0x09, 0xce, 0x18, 0x35, 0x5f, 0xc1, 0x2b,
	0xdf, 0xc0, 0x4f, 0xf0, 0x43, 0x3c, 0xf2, 0x01, 0x68, 0xc6, 0xe3, 0x5c, 0x2a, 0xb7, 0xe9, 0x9b,
	0xd7, 0xf6, 0xda, 0xb3, 0xf7, 0x5a, 0xcb, 0x63, 0xd8, 0x1f, 0x72, 0x9c, 0x9c, 0xe6, 0x9c, 0x8d,
	0x0a, 0x86, 0x79, 0x3e, 0x0c, 0xd3, 0x22, 0x19, 0x85, 0x09, 0x67, 0x8c, 0x67, 0x61, 0x9f, 0x08,
	0x12, 0x0a, 0x7a, 0x2e, 0x42, 0x41, 0xd9, 0x38, 0x25, 0x82, 0x8e, 0x73, 0x2e, 0xf8, 0x0c, 0x61,
	0x05, 0xd1, 0xe6, 0xd2, 0x5b, 0xef, 0xdf, 0x21, 0xe7, 0xc3, 0x94, 0x86, 0x0a, 0xf5, 0x8a, 0x41,
	0x48, 0xd9, 0x58, 0x4c, 0x4b, 0x6e, 0xe7, 0xb7, 0x01, 0xd6, 0x07, 0x92, 0x16, 0x14, 0x21, 0x68,
	0x8e, 0x32, 0xe1, 0x1a, 0xbe, 0x11, 0x34, 0x0f, 0xd7, 0x62, 0x09, 0xd0, 0x16, 0x98, 0x85, 0x2c,
	0x36, 0x7c, 0x23, 0x30, 0x0f, 0xd7, 0x62, 0x85, 0xd0, 0x36, 0x58, 0x83, 0x94, 0x13, 0xe1, 0x36,
	0x7d, 0x23, 0x30, 0x0e, 0xd7, 0xe2, 0x12, 0x4a, 0x76, 0x8f, 0xf3, 0xd4, 0x35, 0x7d, 0x23, 0xb0,
	0x25, 0x5b, 0x22, 0x79, 0xee, 0x44, 0xe4, 0xae, 0xe5, 0x1b, 0x81, 0x23, 0xcf, 0x9d, 0x88, 0x5c,
	0x9e, 0xd0, 0x9b, 0x0a, 0x3a, 0x71, 0x5b, 0xbe, 0x11, 0x6c, 0xc8, 0x13, 0x14, 0x44, 0x2e, 0xb4,
	0x78, 0xef, 0x33, 0x4d, 0x84, 0xbb, 0xae, 0xe9, 0x1a, 0xcb, 0x0e, 0x92, 0xe7, 0x64, 0xea, 0xda,
	0xfa, 0x45, 0x09, 0xd1, 0x23, 0x30, 0xb3, 0x22, 0x4d, 0x5d, 0xc7, 0x37, 0x82, 0x76, 0xb4, 0x8d,
	0x4b, 0xad, 0xb8, 0xd2, 0x8a, 0x0f, 0xa4, 0x56, 0xb9, 0x8b, 0x64, 0xed, 0xaf, 0x83, 0xf5, 0x55,
	0x8a, 0xed, 0xfc, 0xb0, 0xa0, 0xd5, 0x4d, 0x4e, 0x29, 0x23, 0x08, 0xcf, 0x75, 0xb7, 0x23, 0x0f,
	0x2f, 0x79, 0x87, 0x4b, 0x0e, 0x7e, 0x25, 0x38, 0xab, 0x3c, 0xd9, 0x59, 0xf0, 0x64, 0x55, 0x43,
	0xe9, 0x57, 0xb4, 0xe8, 0xd7, 0xaa, 0x16, 0xed, 0xe5, 0xce, 0x82, 0x97, 0x2b, 0xa7, 0x28, 0x9f,
	0xa3, 0xb9, 0xcf, 0xed, 0xe8, 0x46, 0x7d, 0x43, 0x97, 0x7e, 0x29, 0x68, 0x96, 0xd0, 0x2a, 0x87,
	0xe7, 0x8b, 0x39, 0x5c, 0xa7, 0x4b, 0xe7, 0x14, 0x82, 0x49, 0xb3, 0x82, 0xa9, 0x94, 0xda, 0xd1,
	0x3f, 0x97, 0xb5, 0x09, 0xb9, 0x9c, 0x24, 0xa2, 0xa7, 0xb3, 0x60, 0xed, 0xab, 0x04, 0xbd, 0xe9,
	0x1e, 0x1f, 0x2d, 0x84, 0x1e, 0x55, 0xa1, 0x3b, 0xd7, 0x68, 0x2a, 0xa9, 0x1e, 0x83, 0x66, 0x97,
	0x0a, 0xf4, 0x0c, 0x2c, 0x9a, 0x89, 0x7c, 0xea, 0x1a, 0x7e, 0x33, 0x68, 0x47, 0x37, 0x2f, 0x5d,
	0x11, 0x1f, 0x48, 0x5a, 0x5c, 0xb2, 0xbd, 0x10, 0x2c, 0x85, 0xd1, 0x5f, 0xd0, 0xec, 0xf3, 0x44,
	0x7d, 0x15, 0x4e, 0x2c, 0x1f, 0xd1, 0x16, 0x58, 0x82, 0x9f, 0xd1, 0x4c, 0x05, 0xef, 0xc4, 0x25,
	0xf0, 0xee, 0x82, 0x29, 0xe7, 0xa3, 0xff, 0x01, 0x18, 0x39, 0x3f, 0x49, 0x69, 0x36, 0x14, 0xa7,
	0xaa, 0x6d, 0x33, 0x76, 0x18, 0x39, 0x7f, 0xab, 0x0a, 0xde, 0x03, 0xb0, 0x2b, 0x17, 0x57, 0x51,
	0x5b, 0x60, 0xca, 0x5c, 0xf7, 0x6d, 0x68, 0x4d, 0xd4, 0x96, 0x9d, 0x6f, 0x26, 0x98, 0xaf, 0x47,
	0x29, 0x45, 0x7b, 0x60, 0x57, 0x32, 0xb4, 0xae, 0x5b, 0x17, 0x74, 0x49, 0x1a, 0x7e, 0xaf, 0x4b,
	0xa5, 0xb2, 0x59, 0x8b, 0xf7, 0xab, 0x01, 0x76, 0xf5, 0xae, 0x46, 0x20, 0x92, 0x9f, 0x5c, 0x7f,
	0xaa, 0xf5, 0xa9, 0x67, 0xb4, 0x07, 0xd6, 0x98, 0xe4, 0x84, 0xb9, 0x4d, 0x35, 0xee, 0xfe, 0x55,
	0xe3, 0xf0, 0x3b, 0xc9, 0xd4, 0x76, 0xaa, 0x2e, 0xef, 0xbb, 0x01, 0x8e, 0xaa, 0x52, 0x41, 0xf3,
	0x9a, 0x91, 0x18, 0xd6, 0xfb, 0x74, 0x40, 0x8a, 0xb4, 0xba, 0x4e, 0x5b, 0x17, 0x06, 0xa8, 0x5f,
	0x53, 0x5c, 0x91, 0x90, 0x07, 0xb6, 0xbc, 0xc7, 0xa4, 0x97, 0x52, 0x75, 0x99, 0xec, 0x78, 0x86,
	0xd1, 0xe3, 0xca, 0x2f, 0x7d, 0x67, 0xfe, 0xae, 0x8d, 0x3c, 0xd6, 0x24, 0xef, 0x13, 0xc0, 0x7c,
	0x5f, 0xb9, 0xda, 0x19, 0x9d, 0x56, 0xab, 0x9d, 0xd1, 0x29, 0xda, 0xd5, 0xbf, 0x0a, 0xbd, 0xd8,
	0xbd, 0xd5, 0xca, 0xa5, 0xc6, 0xb8, 0x6c, 0x7a, 0xd9, 0x78, 0x61, 0x78, 0x1f, 0x61, 0x73, 0x29,
	0x89, 0x9a, 0x21, 0xd1, 0xf2, 0x90, 0xff, 0xae, 0x1a, 0xb2, 0x70, 0x74, 0xe7, 0xa7, 0x01, 0x4e,
	0x77, 0x4c, 0x93, 0xd1, 0x60, 0x44, 0x73, 0x74, 0x1b, 0x66, 0x7f, 0xfc, 0x93, 0x8c, 0x30, 0xaa,
	0x27, 0x6c, 0x54, 0xc5, 0x23, 0xc2, 0x28, 0xda, 0x85, 0x96, 0xca, 0x64, 0xe2, 0x36, 0x54, 0x94,
	0x77, 0x2e, 0xda, 0x53, 0x1d, 0x57, 0x8a, 0x99, 0x94, 0x39, 0xea, 0x1e, 0xef, 0x18, 0xda, 0x0b,
	0xe5, 0x1a, 0x25, 0x0f, 0x97, 0x95, 0xd4, 0xe7, 0x38, 0x57, 0xd0, 0x6b, 0xa9, 0xfa, 0x93, 0x3f,
	0x01, 0x00, 0x00, 0xff, 0xff, 0x0f, 0x21, 0xa3, 0x84, 0xf0, 0x06, 0x00, 0x00,
}
