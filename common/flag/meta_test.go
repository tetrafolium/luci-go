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

package flag

import (
	"flag"
	"testing"

	"google.golang.org/grpc/metadata"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestGRPCMEtadata(t *testing.T) {
	t.Parallel()

	Convey("GRPCMetadata", t, func() {
		md := metadata.MD{}
		v := GRPCMetadata(md)

		Convey("Set", func() {
			Convey("Once", func() {
				So(v.Set("a:1"), ShouldBeNil)
				So(md, ShouldResemble, metadata.Pairs("a", "1"))

				Convey("Second time", func() {
					So(v.Set("b:1"), ShouldBeNil)
					So(md, ShouldResemble, metadata.Pairs("a", "1", "b", "1"))
				})

				Convey("Same key", func() {
					So(v.Set("a:2"), ShouldBeNil)
					So(md, ShouldResemble, metadata.Pairs("a", "1", "a", "2"))
				})
			})
			Convey("No colon", func() {
				So(v.Set("a"), ShouldErrLike, "no colon")
			})
		})

		Convey("String", func() {
			md.Append("a", "1", "2")
			md.Append("b", "1")
			So(v.String(), ShouldEqual, "a:1, a:2, b:1")
		})

		Convey("Get", func() {
			So(v.(flag.Getter).Get(), ShouldEqual, md)
		})
	})
}
