// Copyright 2020 The LUCI Authors.
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

package rpc

import (
	"fmt"
	"strings"
	"testing"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateTags(t *testing.T) {
	t.Parallel()

	Convey("validate build set", t, func() {
		Convey("valid", func() {
			// test gitiles format
			gitiles := fmt.Sprintf("commit/gitiles/chromium.googlesource.com/chromium/src/+/%s", strings.Repeat("a", 40))
			So(validateBuildSet(gitiles), ShouldBeNil)
			// test gerrit format
			So(validateBuildSet("patch/gerrit/chromium-review.googlesource.com/123/456"), ShouldBeNil)
			// test user format
			So(validateBuildSet("myformat/x"), ShouldBeNil)
		})
		Convey("invalid", func() {
			So(validateBuildSet("commit/gitiles/chromium.googlesource.com/chromium/src.git/+/aaa"), ShouldErrLike, `does not match regex "^commit/gitiles/([^/]+)/(.+?)/\+/([a-f0-9]{40})$"`)
			gitiles := fmt.Sprintf("commit/gitiles/chromium.googlesource.com/a/chromium/src/+/%s", strings.Repeat("a", 40))
			So(validateBuildSet(gitiles), ShouldErrLike, `gitiles project must not start with "a/"`)
			gitiles = fmt.Sprintf("commit/gitiles/chromium.googlesource.com/chromium/src.git/+/%s", strings.Repeat("a", 40))
			So(validateBuildSet(gitiles), ShouldErrLike, `gitiles project must not end with ".git"`)

			So(validateBuildSet("patch/gerrit/chromium-review.googlesource.com/aa/bb"), ShouldErrLike, `does not match regex "^patch/gerrit/([^/]+)/(\d+)/(\d+)$"`)
			So(validateBuildSet(strings.Repeat("a", 2000)), ShouldErrLike, "buildset tag is too long")
		})
	})

	Convey("validate tags", t, func() {
		Convey("invalid", func() {
			// in general
			So(validateTags([]*pb.StringPair{{Key: "k:1", Value: "v"}}, TagNew), ShouldErrLike, "cannot have a colon")

			// build address
			So(validateTags([]*pb.StringPair{{Key: "build_address", Value: "v"}}, TagNew), ShouldErrLike, `tag "build_address" is reserved`)
			So(validateTags([]*pb.StringPair{{Key: "build_address", Value: "v"}}, TagAppend), ShouldErrLike, `cannot be added to an existing build`)

			// buildset
			So(validateTags([]*pb.StringPair{{Key: "buildset", Value: "patch/gerrit/foo"}}, TagNew), ShouldErrLike, `does not match regex "^patch/gerrit/([^/]+)/(\d+)/(\d+)$"`)

			gitiles1 := fmt.Sprintf("commit/gitiles/chromium.googlesource.com/chromium/src/+/%s", strings.Repeat("a", 40))
			gitiles2 := fmt.Sprintf("commit/gitiles/chromium.googlesource.com/chromium/src/+/%s", strings.Repeat("b", 40))
			So(validateTags([]*pb.StringPair{
				{Key: "buildset", Value: gitiles1},
				{Key: "buildset", Value: gitiles2},
			}, TagNew),
				ShouldErrLike,
				fmt.Sprintf(`tag "buildset:%s" conflicts with tag "buildset:%s"`, gitiles2, gitiles1))
			So(validateTags([]*pb.StringPair{
				{Key: "buildset", Value: gitiles1},
				{Key: "buildset", Value: gitiles1},
			}, TagNew),
				ShouldBeNil)

			// builder
			So(validateTags([]*pb.StringPair{
				{Key: "builder", Value: "1"},
				{Key: "builder", Value: "2"},
			}, TagNew),
				ShouldErrLike,
				`tag "builder:2" conflicts with tag "builder:1"`)
			So(validateTags([]*pb.StringPair{
				{Key: "builder", Value: "1"},
				{Key: "builder", Value: "1"},
			}, TagNew),
				ShouldBeNil)
			So(validateTags([]*pb.StringPair{{Key: "builder", Value: "v"}}, TagAppend), ShouldErrLike, "cannot be added to an existing build")
		})
	})

	Convey("validate summary_markdown", t, func() {
		Convey("valid", func() {
			So(validateSummaryMarkdown("[this](http://example.org) is a link"), ShouldBeNil)
		})

		Convey("too big", func() {
			So(validateSummaryMarkdown(strings.Repeat("☕", summaryMarkdownMaxLength)), ShouldErrLike, "too big to accept")
		})
	})

	Convey("validateCommit", t, func() {
		Convey("nil", func() {
			err := validateCommit(nil)
			So(err, ShouldErrLike, "host is required")
		})

		Convey("empty", func() {
			cm := &pb.GitilesCommit{}
			err := validateCommit(cm)
			So(err, ShouldErrLike, "host is required")
		})

		Convey("project", func() {
			cm := &pb.GitilesCommit{
				Host: "host",
			}
			err := validateCommit(cm)
			So(err, ShouldErrLike, "project is required")
		})

		Convey("id", func() {
			cm := &pb.GitilesCommit{
				Host:    "host",
				Project: "project",
				Id:      "id",
			}
			err := validateCommit(cm)
			So(err, ShouldErrLike, "id must match")
		})

		Convey("ref", func() {
			cm := &pb.GitilesCommit{
				Host:    "host",
				Project: "project",
				Ref:     "ref",
			}
			err := validateCommit(cm)
			So(err, ShouldErrLike, "ref must match")
		})

		Convey("mutual exclusion", func() {
			Convey("ref", func() {
				cm := &pb.GitilesCommit{
					Host:    "host",
					Project: "project",
					Id:      "id",
					Ref:     "ref",
				}
				err := validateCommit(cm)
				So(err, ShouldErrLike, "id is mutually exclusive with (ref and position)")
			})

			Convey("position", func() {
				cm := &pb.GitilesCommit{
					Host:     "host",
					Project:  "project",
					Id:       "id",
					Position: 1,
				}
				err := validateCommit(cm)
				So(err, ShouldErrLike, "id is mutually exclusive with (ref and position)")
			})

			Convey("neither", func() {
				cm := &pb.GitilesCommit{
					Host:    "host",
					Project: "project",
				}
				err := validateCommit(cm)
				So(err, ShouldErrLike, "one of")
			})
		})

		Convey("valid", func() {
			Convey("id", func() {
				cm := &pb.GitilesCommit{
					Host:    "host",
					Project: "project",
					Id:      "1234567890123456789012345678901234567890",
				}
				err := validateCommit(cm)
				So(err, ShouldBeNil)
			})

			Convey("ref", func() {
				cm := &pb.GitilesCommit{
					Host:     "host",
					Project:  "project",
					Ref:      "refs/ref",
					Position: 1,
				}
				err := validateCommit(cm)
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("validateCommitWithRef", t, func() {
		Convey("nil", func() {
			So(validateCommitWithRef(nil), ShouldErrLike, "ref is required")
		})

		Convey("empty", func() {
			So(validateCommitWithRef(&pb.GitilesCommit{}), ShouldErrLike, "ref is required")
		})

		Convey("with id", func() {
			cm := &pb.GitilesCommit{
				Host:    "host",
				Project: "project",
				Id:      "id",
			}
			So(validateCommitWithRef(cm), ShouldErrLike, "ref is required")
		})

		Convey("with ref", func() {
			cm := &pb.GitilesCommit{
				Host:     "host",
				Project:  "project",
				Ref:      "refs/",
				Position: 1,
			}
			So(validateCommitWithRef(cm), ShouldBeNil)
		})
	})
}
