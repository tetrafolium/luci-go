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

package cli

import (
	"io/ioutil"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestPropertiesFlag(t *testing.T) {
	t.Parallel()

	Convey("PropertieFlag", t, func() {
		props := &structpb.Struct{}
		f := PropertiesFlag(props)

		Convey("File", func() {
			file, err := ioutil.TempFile("", "")
			So(err, ShouldBeNil)
			defer file.Close()

			_, err = file.WriteString(`{
				"in-file-1": "orig",
				"in-file-2": "orig"
			}`)
			So(err, ShouldBeNil)

			So(f.Set("@"+file.Name()), ShouldBeNil)

			So(props, ShouldResembleProtoJSON, `{
				"in-file-1": "orig",
				"in-file-2": "orig"
			}`)

			Convey("Override", func() {
				So(f.Set("in-file-2=override"), ShouldBeNil)
				So(props, ShouldResembleProtoJSON, `{
					"in-file-1": "orig",
					"in-file-2": "override"
				}`)

				So(f.Set("a=b"), ShouldBeNil)
				So(props, ShouldResembleProtoJSON, `{
					"in-file-1": "orig",
					"in-file-2": "override",
					"a": "b"
				}`)
			})
		})

		Convey("Name=Value", func() {
			So(f.Set("foo=bar"), ShouldBeNil)
			So(props, ShouldResembleProtoJSON, `{"foo": "bar"}`)

			Convey("JSON", func() {
				So(f.Set("array=[1]"), ShouldBeNil)
				So(props, ShouldResembleProtoJSON, `{
					"foo": "bar",
					"array": [1]}
				`)
			})

			Convey("Trims spaces", func() {
				So(f.Set("array = [1]"), ShouldBeNil)
				So(props, ShouldResembleProtoJSON, `{
					"foo": "bar",
					"array": [1]}
				`)
			})

			Convey("Dup", func() {
				So(f.Set("foo=bar"), ShouldErrLike, `duplicate property "foo`)
			})
		})
	})
}
