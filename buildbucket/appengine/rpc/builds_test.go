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
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestBuilds(t *testing.T) {
	t.Parallel()

	Convey("Builds", t, func() {
		srv := NewBuilds()
		ctx := context.Background()

		Convey("ensure unimplemented", func() {
			srv = NewBuilds()

			Convey("Batch", func() {
				rsp, err := srv.Batch(ctx, nil)
				So(err, ShouldErrLike, "method not implemented")
				So(rsp, ShouldBeNil)
			})

			Convey("CancelBuild", func() {
				rsp, err := srv.CancelBuild(ctx, nil)
				So(err, ShouldErrLike, "method not implemented")
				So(rsp, ShouldBeNil)
			})

			Convey("ScheduleBuild", func() {
				rsp, err := srv.ScheduleBuild(ctx, nil)
				So(err, ShouldErrLike, "method not implemented")
				So(rsp, ShouldBeNil)
			})
		})
	})
}
