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
	"testing"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestParseCommit(t *testing.T) {
	Convey("ParseCommit", t, func() {

		Convey("https://chromium.googlesource.com/infra/luci/luci-go/+/7a63166bfab5de38ddb2cb8e29aca756bdc2a28d", func() {
			actual, confirm, err := parseCommit("https://chromium.googlesource.com/infra/luci/luci-go/+/7a63166bfab5de38ddb2cb8e29aca756bdc2a28d")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeFalse)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "",
				Id:      "7a63166bfab5de38ddb2cb8e29aca756bdc2a28d",
			})
		})

		Convey("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/x", func() {
			actual, confirm, err := parseCommit("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/x")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeFalse)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "refs/heads/x",
				Id:      "",
			})
		})

		Convey("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/x/y", func() {
			actual, confirm, err := parseCommit("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/heads/x/y")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeTrue)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "refs/heads/x",
				Id:      "",
			})
		})

		Convey("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/branch-heads/x", func() {
			actual, confirm, err := parseCommit("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/branch-heads/x")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeTrue)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "refs/branch-heads/x",
				Id:      "",
			})
		})

		Convey("https://chromium.googlesource.com/infra/luci/luci-go/+/master", func() {
			actual, confirm, err := parseCommit("https://chromium.googlesource.com/infra/luci/luci-go/+/master")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeTrue)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "refs/heads/master",
				Id:      "",
			})
		})

		Convey("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/x", func() {
			actual, confirm, err := parseCommit("https://chromium.googlesource.com/infra/luci/luci-go/+/refs/x")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeTrue)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "refs/x",
				Id:      "",
			})
		})

		Convey("https://chromium-foo.googlesource.com/infra/luci/luci-go/+/refs/x", func() {
			actual, confirm, err := parseCommit("https://chromium-foo.googlesource.com/infra/luci/luci-go/+/refs/x")
			So(err, ShouldBeNil)
			So(confirm, ShouldBeTrue)
			So(actual, ShouldResembleProto, &pb.GitilesCommit{
				Host:    "chromium-foo.googlesource.com",
				Project: "infra/luci/luci-go",
				Ref:     "refs/x",
				Id:      "",
			})
		})
	})
}
