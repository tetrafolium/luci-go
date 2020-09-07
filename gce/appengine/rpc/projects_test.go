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

package rpc

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/gce/api/projects/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/model"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestProjects(t *testing.T) {
	t.Parallel()

	Convey("Projects", t, func() {
		srv := &Projects{}
		c := memory.Use(context.Background())
		datastore.GetTestable(c).AutoIndex(true)
		datastore.GetTestable(c).Consistent(true)

		Convey("List", func() {
			Convey("invalid", func() {
				Convey("page token", func() {
					req := &projects.ListRequest{
						PageToken: "token",
					}
					_, err := srv.List(c, req)
					So(err, ShouldErrLike, "invalid page token")
				})
			})

			Convey("valid", func() {
				Convey("nil", func() {
					Convey("none", func() {
						rsp, err := srv.List(c, nil)
						So(err, ShouldBeNil)
						So(rsp.Projects, ShouldBeEmpty)
					})

					Convey("one", func() {
						p := &model.Project{
							ID: "id",
						}
						So(datastore.Put(c, p), ShouldBeNil)

						rsp, err := srv.List(c, nil)
						So(err, ShouldBeNil)
						So(rsp.Projects, ShouldHaveLength, 1)
					})
				})

				Convey("empty", func() {
					Convey("none", func() {
						req := &projects.ListRequest{}
						rsp, err := srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.Projects, ShouldBeEmpty)
					})

					Convey("one", func() {
						p := &model.Project{
							ID: "id",
						}
						So(datastore.Put(c, p), ShouldBeNil)

						req := &projects.ListRequest{}
						rsp, err := srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.Projects, ShouldHaveLength, 1)
					})
				})

				Convey("pages", func() {
					So(datastore.Put(c, &model.Project{ID: "id1"}), ShouldBeNil)
					So(datastore.Put(c, &model.Project{ID: "id2"}), ShouldBeNil)
					So(datastore.Put(c, &model.Project{ID: "id3"}), ShouldBeNil)

					Convey("default", func() {
						req := &projects.ListRequest{}
						rsp, err := srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.Projects, ShouldNotBeEmpty)
					})

					Convey("one", func() {
						req := &projects.ListRequest{
							PageSize: 1,
						}
						rsp, err := srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.NextPageToken, ShouldNotBeEmpty)
						So(rsp.Projects, ShouldHaveLength, 1)

						req.PageToken = rsp.NextPageToken
						rsp, err = srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.NextPageToken, ShouldNotBeEmpty)
						So(rsp.Projects, ShouldHaveLength, 1)

						req.PageToken = rsp.NextPageToken
						rsp, err = srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.NextPageToken, ShouldBeEmpty)
						So(rsp.Projects, ShouldHaveLength, 1)
					})

					Convey("two", func() {
						req := &projects.ListRequest{
							PageSize: 2,
						}
						rsp, err := srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.NextPageToken, ShouldNotBeEmpty)
						So(rsp.Projects, ShouldHaveLength, 2)

						req.PageToken = rsp.NextPageToken
						rsp, err = srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.NextPageToken, ShouldBeEmpty)
						So(rsp.Projects, ShouldHaveLength, 1)
					})

					Convey("many", func() {
						req := &projects.ListRequest{
							PageSize: 200,
						}
						rsp, err := srv.List(c, req)
						So(err, ShouldBeNil)
						So(rsp.NextPageToken, ShouldBeEmpty)
						So(rsp.Projects, ShouldHaveLength, 3)
					})
				})
			})
		})

		Convey("Ensure", func() {
			Convey("Binary", func() {
				req := &projects.EnsureRequest{
					Id: "id",
					Project: &projects.Config{
						Project: "project",
						Region: []string{
							"region1",
							"region2",
						},
						Revision: "revision-1",
					},
				}
				cfg, err := srv.Ensure(c, req)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeEmpty)
				So(cfg, ShouldResembleProto, &projects.Config{
					Project: "project",
					Region: []string{
						"region1",
						"region2",
					},
					Revision: "revision-1",
				})
			})
		})
	})
}
