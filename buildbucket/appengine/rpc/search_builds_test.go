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

	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/logging/memlogger"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	"github.com/tetrafolium/luci-go/buildbucket/appengine/model"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateSearchBuilds(t *testing.T) {
	t.Parallel()

	Convey("validateChange", t, func() {
		Convey("nil", func() {
			err := validateChange(nil)
			So(err, ShouldErrLike, "host is required")
		})

		Convey("empty", func() {
			ch := &pb.GerritChange{}
			err := validateChange(ch)
			So(err, ShouldErrLike, "host is required")
		})

		Convey("change", func() {
			ch := &pb.GerritChange{
				Host: "host",
			}
			err := validateChange(ch)
			So(err, ShouldErrLike, "change is required")
		})

		Convey("patchset", func() {
			ch := &pb.GerritChange{
				Host:   "host",
				Change: 1,
			}
			err := validateChange(ch)
			So(err, ShouldErrLike, "patchset is required")
		})

		Convey("valid", func() {
			ch := &pb.GerritChange{
				Host:     "host",
				Change:   1,
				Patchset: 1,
			}
			err := validateChange(ch)
			So(err, ShouldBeNil)
		})
	})

	Convey("validatePredicate", t, func() {
		Convey("nil", func() {
			err := validatePredicate(nil)
			So(err, ShouldBeNil)
		})

		Convey("empty", func() {
			pr := &pb.BuildPredicate{}
			err := validatePredicate(pr)
			So(err, ShouldBeNil)
		})

		Convey("mutual exclusion", func() {
			pr := &pb.BuildPredicate{
				Build:      &pb.BuildRange{},
				CreateTime: &pb.TimeRange{},
			}
			err := validatePredicate(pr)
			So(err, ShouldErrLike, "build is mutually exclusive with create_time")
		})
	})

	Convey("validatePageToken", t, func() {
		Convey("empty token", func() {
			err := validatePageToken("")
			So(err, ShouldBeNil)
		})

		Convey("invalid page token", func() {
			err := validatePageToken("abc")
			So(err, ShouldErrLike, "invalid page_token")
		})

		Convey("valid page token", func() {
			err := validatePageToken("id>123")
			So(err, ShouldBeNil)
		})
	})

	Convey("validateSearch", t, func() {
		Convey("nil", func() {
			err := validateSearch(nil)
			So(err, ShouldBeNil)
		})

		Convey("empty", func() {
			req := &pb.SearchBuildsRequest{}
			err := validateSearch(req)
			So(err, ShouldBeNil)
		})

		Convey("page size", func() {
			Convey("negative", func() {
				req := &pb.SearchBuildsRequest{
					PageSize: -1,
				}
				err := validateSearch(req)
				So(err, ShouldErrLike, "page_size cannot be negative")
			})

			Convey("zero", func() {
				req := &pb.SearchBuildsRequest{
					PageSize: 0,
				}
				err := validateSearch(req)
				So(err, ShouldBeNil)
			})

			Convey("positive", func() {
				req := &pb.SearchBuildsRequest{
					PageSize: 1,
				}
				err := validateSearch(req)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestSearchBuilds(t *testing.T) {
	t.Parallel()

	Convey("search builds", t, func() {
		srv := &Builds{}
		ctx := memory.Use(context.Background())
		ctx = memlogger.Use(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: identity.Identity("user:user"),
		})
		datastore.GetTestable(ctx).AutoIndex(true)
		datastore.GetTestable(ctx).Consistent(true)

		So(datastore.Put(ctx, &model.Bucket{
			ID:     "bucket",
			Parent: model.ProjectKey(ctx, "project"),
			Proto: pb.Bucket{
				Acls: []*pb.Acl{
					{
						Identity: "user:user",
						Role:     pb.Acl_READER,
					},
				},
			},
		}), ShouldBeNil)
		So(datastore.Put(ctx, &model.Build{
			Proto: pb.Build{
				Id: 1,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder",
				},
			},
			BucketID:  "project/bucket",
			BuilderID: "project/bucket/builder",
			Tags:      []string{"k1:v1", "k2:v2"},
		}), ShouldBeNil)
		So(datastore.Put(ctx, &model.Build{
			Proto: pb.Build{
				Id: 2,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder2",
				},
			},
			BucketID:  "project/bucket",
			BuilderID: "project/bucket/builder2",
		}), ShouldBeNil)
		Convey("query search on Builds", func() {
			req := &pb.SearchBuildsRequest{
				Predicate: &pb.BuildPredicate{
					Builder: &pb.BuilderID{
						Project: "project",
						Bucket:  "bucket",
						Builder: "builder",
					},
					Tags: []*pb.StringPair{
						{Key: "k1", Value: "v1"},
						{Key: "k2", Value: "v2"},
					},
				},
			}
			rsp, err := srv.SearchBuilds(ctx, req)
			So(err, ShouldBeNil)
			expectedRsp := &pb.SearchBuildsResponse{
				Builds: []*pb.Build{
					{
						Id: 1,
						Builder: &pb.BuilderID{
							Project: "project",
							Bucket:  "bucket",
							Builder: "builder",
						},
						Input: &pb.Build_Input{},
					},
				},
			}
			So(rsp, ShouldResembleProto, expectedRsp)
		})

		Convey("search builds with field masks", func() {
			b := &model.Build{
				ID: 1,
			}
			key := datastore.KeyForObj(ctx, b)
			So(datastore.Put(ctx, &model.BuildInfra{
				ID:    1,
				Build: key,
				Proto: model.DSBuildInfra{
					BuildInfra: pb.BuildInfra{
						Buildbucket: &pb.BuildInfra_Buildbucket{
							Hostname: "example.com",
						},
					},
				},
			}), ShouldBeNil)
			So(datastore.Put(ctx, &model.BuildInputProperties{
				ID:    1,
				Build: key,
				Proto: model.DSStruct{
					Struct: structpb.Struct{
						Fields: map[string]*structpb.Value{
							"input": {
								Kind: &structpb.Value_StringValue{
									StringValue: "input value",
								},
							},
						},
					},
				},
			}), ShouldBeNil)

			req := &pb.SearchBuildsRequest{
				Fields: &field_mask.FieldMask{
					Paths: []string{"builds.*.id", "builds.*.input", "builds.*.infra"},
				},
			}
			rsp, err := srv.SearchBuilds(ctx, req)
			So(err, ShouldBeNil)
			expectedRsp := &pb.SearchBuildsResponse{
				Builds: []*pb.Build{
					{
						Id: 1,
						Input: &pb.Build_Input{
							Properties: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"input": {
										Kind: &structpb.Value_StringValue{
											StringValue: "input value",
										},
									},
								},
							},
						},
						Infra: &pb.BuildInfra{
							Buildbucket: &pb.BuildInfra_Buildbucket{
								Hostname: "example.com",
							},
						},
					},
					{
						Id: 2,
						Input: &pb.Build_Input{
							Properties: &structpb.Struct{},
						},
						Infra: &pb.BuildInfra{},
					},
				},
			}
			So(rsp, ShouldResembleProto, expectedRsp)
		})
	})
}
