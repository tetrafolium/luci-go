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

package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/pmezard/go-difflib/difflib"

	"github.com/tetrafolium/luci-go/common/data/text/indented"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/proto/google/descutil"
)

// sourceCodeInfoMap maps descriptor proto messages to source code info,
// if available.
// See also descutil.IndexSourceCodeInfo.
type sourceCodeInfoMap map[interface{}]*descriptor.SourceCodeInfo_Location

type schemaConverter struct {
	desc           *descriptor.FileDescriptorSet
	sourceCodeInfo map[*descriptor.FileDescriptorProto]sourceCodeInfoMap
}

// schema constructs a bigquery.Schema from a named message.
func (c *schemaConverter) schema(messageName string) (schema bigquery.Schema, description string, err error) {
	file, obj, _ := descutil.Resolve(c.desc, messageName)
	if obj == nil {
		return nil, "", fmt.Errorf("message %q is not found", messageName)
	}
	msg, isMsg := obj.(*descriptor.DescriptorProto)
	if !isMsg {
		return nil, "", fmt.Errorf("expected %q to be a message, but it is %T", messageName, obj)
	}

	schema = make(bigquery.Schema, 0, len(msg.Field))
	for _, field := range msg.Field {
		switch s, err := c.field(file, field); {
		case err != nil:
			return nil, "", errors.Annotate(err, "failed to derive schema for field %q in message %q", field.GetName(), msg.GetName()).Err()
		case s != nil:
			schema = append(schema, s)
		}
	}
	return schema, c.description(file, msg), nil
}

// field constructs bigquery.FieldSchema from proto field descriptor.
func (c *schemaConverter) field(file *descriptor.FileDescriptorProto, field *descriptor.FieldDescriptorProto) (*bigquery.FieldSchema, error) {
	schema := &bigquery.FieldSchema{
		Name:        field.GetName(),
		Description: c.description(file, field),
		Repeated:    descutil.Repeated(field),
		Required:    descutil.Required(field),
	}

	typeName := strings.TrimPrefix(field.GetTypeName(), ".")
	switch field.GetType() {
	case
		descriptor.FieldDescriptorProto_TYPE_DOUBLE,
		descriptor.FieldDescriptorProto_TYPE_FLOAT:

		schema.Type = bigquery.FloatFieldType

	case
		descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_SINT32,
		descriptor.FieldDescriptorProto_TYPE_SINT64:

		schema.Type = bigquery.IntegerFieldType

	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		schema.Type = bigquery.BooleanFieldType

	case descriptor.FieldDescriptorProto_TYPE_STRING:
		schema.Type = bigquery.StringFieldType

	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		schema.Type = bigquery.BytesFieldType

	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		schema.Type = bigquery.StringFieldType

	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		switch typeName {
		case "google.protobuf.Duration":
			schema.Type = bigquery.FloatFieldType
		case "google.protobuf.Timestamp":
			schema.Type = bigquery.TimestampFieldType
		case "google.protobuf.Struct":
			// google.protobuf.Struct is persisted as JSONPB string.
			// See also https://bit.ly/chromium-bq-struct
			schema.Type = bigquery.StringFieldType
		default:
			switch s, _, err := c.schema(typeName); {
			case err != nil:
				return nil, err
			case len(s) == 0:
				// BigQuery does not like empty record fields.
				return nil, nil
			default:
				schema.Type = bigquery.RecordFieldType
				schema.Schema = s
			}
		}
	default:
		return nil, fmt.Errorf("not supported field type %q", field.GetType())
	}
	return schema, nil
}

// description returns a string description of the descriptor proto that
// ptr points to.
// If ptr is a field of an enum type, appends
// "\nValid values: <comma-separated enum member names>".
func (c *schemaConverter) description(file *descriptor.FileDescriptorProto, ptr interface{}) string {
	description := c.sourceCodeInfo[file][ptr].GetLeadingComments()

	// Trim leading whitespace.
	lines := strings.Split(description, "\n")
	trimSize := -1
	for _, l := range lines {
		if len(strings.TrimSpace(l)) == 0 {
			// skip empty lines
			continue
		}
		space := 0
		for _, r := range l {
			if unicode.IsSpace(r) {
				space++
			} else {
				break
			}
		}
		if trimSize == -1 || space < trimSize {
			trimSize = space
		}
	}
	if trimSize > 0 {
		for i := range lines {
			if len(lines[i]) >= trimSize {
				lines[i] = lines[i][trimSize:]
			}
		}
		description = strings.Join(lines, "\n")
	}
	description = strings.TrimSpace(description)

	// Append valid enum values.
	if field, ok := ptr.(*descriptor.FieldDescriptorProto); ok && field.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
		_, obj, _ := descutil.Resolve(c.desc, strings.TrimPrefix(field.GetTypeName(), "."))
		if enum, ok := obj.(*descriptor.EnumDescriptorProto); ok {
			names := make([]string, len(enum.Value))
			for i, v := range enum.Value {
				names[i] = v.GetName()
			}
			if description != "" {
				description += "\n"
			}
			description += fmt.Sprintf("Valid values: %s.", strings.Join(names, ", "))
		}
	}
	return description
}

func printSchema(w *indented.Writer, s bigquery.Schema) {
	// Field order does not matter.
	// A new field is always added to the end of the field list in a live table.
	// Sort fields by name to make the result deterministic.
	// Schema diffing relies on it.

	s = append(bigquery.Schema(nil), s...)
	sort.Slice(s, func(i, j int) bool {
		return s[i].Name < s[j].Name
	})

	for i, f := range s {
		if i > 0 {
			fmt.Fprintln(w)
		}

		if f.Description != "" {
			for _, line := range strings.Split(f.Description, "\n") {
				fmt.Fprintln(w, "//", line)
			}
		}

		switch {
		case f.Repeated:
			fmt.Fprint(w, "repeated ")
		case f.Required:
			fmt.Fprint(w, "required ")
		}

		fmt.Fprintf(w, "%s %s", f.Type, f.Name)

		if f.Type == bigquery.RecordFieldType {
			fmt.Fprintln(w, " {")
			w.Level++
			printSchema(w, f.Schema)
			w.Level--
			fmt.Fprint(w, "}")
		}

		fmt.Fprintln(w)
	}
}

func schemaString(s bigquery.Schema) string {
	var buf bytes.Buffer
	printSchema(&indented.Writer{Writer: &buf}, s)
	return buf.String()
}

// schemaDiff returns unified diff of two schemas.
// Returns "" if there is no difference.
func schemaDiff(before, after bigquery.Schema) string {
	ret, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(schemaString(before)),
		B:        difflib.SplitLines(schemaString(after)),
		FromFile: "Current",
		ToFile:   "New",
		Context:  3,
		Eol:      "\n",
	})
	if err != nil {
		// GetUnifiedDiffString returns an error only if it fails
		// to write to a bytes.Buffer, which either cannot happen or we better
		// panic.
		panic(err)
	}
	return ret
}

// addMissingFields copies fields from src to dest if they are not present in
// dest.
func addMissingFields(dest *bigquery.Schema, src bigquery.Schema) {
	destFields := indexFields(*dest)
	for _, sf := range src {
		switch df := destFields[sf.Name]; {
		case df == nil:
			*dest = append(*dest, sf)
		default:
			addMissingFields(&df.Schema, sf.Schema)
		}
	}
}

func indexFields(s bigquery.Schema) map[string]*bigquery.FieldSchema {
	ret := make(map[string]*bigquery.FieldSchema, len(s))
	for _, f := range s {
		ret[f.Name] = f
	}
	return ret
}
