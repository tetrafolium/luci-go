// Copyright 2019 The LUCI Authors.
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

package exe

import (
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"

	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

type testStruct struct {
	Field string `json:"field"`
}

func TestProperties(t *testing.T) {
	Convey(`test property helpers`, t, func() {
		props := &structpb.Struct{}

		expectedStruct := &testStruct{Field: "hi"}
		expectedProto := &bbpb.Build{SummaryMarkdown: "there"}
		expectedStrings := []string{"not", "a", "struct"}
		So(WriteProperties(props, map[string]interface{}{
			"struct":  expectedStruct,
			"proto":   expectedProto,
			"strings": expectedStrings,
			"null":    Null,
		}), ShouldBeNil)
		So(props, ShouldResembleProto, &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"struct": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"field": {Kind: &structpb.Value_StringValue{
							StringValue: "hi",
						}},
					}},
				}},
				"proto": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"summary_markdown": {Kind: &structpb.Value_StringValue{
							StringValue: "there",
						}},
					}},
				}},
				"strings": {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
					Values: []*structpb.Value{
						{Kind: &structpb.Value_StringValue{StringValue: "not"}},
						{Kind: &structpb.Value_StringValue{StringValue: "a"}},
						{Kind: &structpb.Value_StringValue{StringValue: "struct"}},
					},
				}}},
				"null": {Kind: &structpb.Value_NullValue{NullValue: 0}},
			},
		})

		readStruct := &testStruct{}
		extraStruct := &testStruct{}
		readProto := &bbpb.Build{}
		var readStrings []string
		readNil := interface{}(100) // not currently nil
		So(ParseProperties(props, map[string]interface{}{
			"struct":       readStruct,
			"extra_struct": extraStruct,
			"proto":        readProto,
			"strings":      &readStrings,
			"null":         &readNil,
		}), ShouldBeNil)
		So(readStruct, ShouldResemble, expectedStruct)
		So(extraStruct, ShouldResemble, &testStruct{})
		So(readStrings, ShouldResemble, expectedStrings)
		So(readNil, ShouldResemble, nil)
		So(readProto, ShouldResembleProto, expectedProto)

		// now, delete some keys
		So(WriteProperties(props, map[string]interface{}{
			"struct":         nil,
			"proto":          nil,
			"does_not_exist": nil,
		}), ShouldBeNil)
		So(props, ShouldResembleProto, &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"strings": {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
					Values: []*structpb.Value{
						{Kind: &structpb.Value_StringValue{StringValue: "not"}},
						{Kind: &structpb.Value_StringValue{StringValue: "a"}},
						{Kind: &structpb.Value_StringValue{StringValue: "struct"}},
					},
				}}},
				"null": {Kind: &structpb.Value_NullValue{NullValue: 0}},
			},
		})
	})
}
