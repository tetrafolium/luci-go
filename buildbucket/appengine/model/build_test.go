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

package model

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/proto/mask"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	Convey("Build", t, func() {
		ctx := memory.Use(context.Background())
		datastore.GetTestable(ctx).AutoIndex(true)
		datastore.GetTestable(ctx).Consistent(true)
		m := mask.All(&pb.Build{})

		Convey("read/write", func() {
			So(datastore.Put(ctx, &Build{
				ID: 1,
				Proto: pb.Build{
					Id: 1,
					Builder: &pb.BuilderID{
						Project: "project",
						Bucket:  "bucket",
						Builder: "builder",
					},
					Status: pb.Status_SUCCESS,
				},
				CreateTime: testclock.TestRecentTimeUTC,
			}), ShouldBeNil)

			b := &Build{
				ID: 1,
			}
			So(datastore.Get(ctx, b), ShouldBeNil)
			So(b, ShouldResemble, &Build{
				ID:           1,
				Proto:        b.Proto, // assert protobufs separately
				BucketID:     "project/bucket",
				BuilderID:    "project/bucket/builder",
				Canary:       false,
				CreateTime:   datastore.RoundTime(testclock.TestRecentTimeUTC),
				Experimental: false,
				Incomplete:   false,
				Status:       pb.Status_SUCCESS,
				Project:      "project",
			})
			So(&b.Proto, ShouldResembleProto, &pb.Build{
				Id: 1,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder",
				},
				Status: pb.Status_SUCCESS,
			})
		})

		Convey("ToProto", func() {
			b := &Build{
				ID: 1,
				Proto: pb.Build{
					Id: 1,
				},
				Tags: []string{
					"key1:value1",
					"builder:hidden",
					"key2:value2",
				},
			}
			key := datastore.KeyForObj(ctx, b)
			So(datastore.Put(ctx, &BuildInfra{
				ID:    1,
				Build: key,
				Proto: DSBuildInfra{
					pb.BuildInfra{
						Buildbucket: &pb.BuildInfra_Buildbucket{
							Hostname: "example.com",
						},
					},
				},
			}), ShouldBeNil)
			So(datastore.Put(ctx, &BuildInputProperties{
				ID:    1,
				Build: key,
				Proto: DSStruct{
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

			Convey("mask", func() {
				Convey("include", func() {
					m := mask.MustFromReadMask(&pb.Build{}, "id")
					p, err := b.ToProto(ctx, m)
					So(err, ShouldBeNil)
					So(p.Id, ShouldEqual, 1)
				})

				Convey("exclude", func() {
					m := mask.MustFromReadMask(&pb.Build{}, "builder")
					p, err := b.ToProto(ctx, m)
					So(err, ShouldBeNil)
					So(p.Id, ShouldEqual, 0)
				})
			})

			Convey("tags", func() {
				p, err := b.ToProto(ctx, m)
				So(err, ShouldBeNil)
				So(p.Tags, ShouldResembleProto, []*pb.StringPair{
					{
						Key:   "key1",
						Value: "value1",
					},
					{
						Key:   "key2",
						Value: "value2",
					},
				})
				So(b.Proto.Tags, ShouldBeEmpty)
			})

			Convey("infra", func() {
				p, err := b.ToProto(ctx, m)
				So(err, ShouldBeNil)
				So(p.Infra, ShouldResembleProto, &pb.BuildInfra{
					Buildbucket: &pb.BuildInfra_Buildbucket{
						Hostname: "example.com",
					},
				})
				So(b.Proto.Infra, ShouldBeNil)
			})

			Convey("input properties", func() {
				p, err := b.ToProto(ctx, m)
				So(err, ShouldBeNil)
				So(p.Input.Properties, ShouldResembleProtoJSON, `{"input": "input value"}`)
				So(b.Proto.Input, ShouldBeNil)
			})

			Convey("output properties", func() {
				So(datastore.Put(ctx, &BuildOutputProperties{
					ID:    1,
					Build: key,
					Proto: DSStruct{
						Struct: structpb.Struct{
							Fields: map[string]*structpb.Value{
								"output": {
									Kind: &structpb.Value_StringValue{
										StringValue: "output value",
									},
								},
							},
						},
					},
				}), ShouldBeNil)
				p, err := b.ToProto(ctx, m)
				So(err, ShouldBeNil)
				So(p.Output.Properties, ShouldResembleProtoJSON, `{"output": "output value"}`)
				So(b.Proto.Output, ShouldBeNil)
			})

			Convey("steps", func() {
				s, err := proto.Marshal(&pb.Build{
					Steps: []*pb.Step{
						{
							Name: "step",
						},
					},
				})
				So(err, ShouldBeNil)
				So(datastore.Put(ctx, &BuildSteps{
					ID:       1,
					Build:    key,
					Bytes:    s,
					IsZipped: false,
				}), ShouldBeNil)
				p, err := b.ToProto(ctx, m)
				So(err, ShouldBeNil)
				So(p.Steps, ShouldResembleProto, []*pb.Step{
					{
						Name: "step",
					},
				})
				So(b.Proto.Steps, ShouldBeEmpty)
			})
		})

		Convey("ToSimpleBuildProto", func() {
			b := &Build{
				ID: 1,
				Proto: pb.Build{
					Id: 1,
					Builder: &pb.BuilderID{
						Project: "project",
						Bucket:  "bucket",
						Builder: "builder",
					},
				},
				Project:   "project",
				BucketID:  "project/bucket",
				BuilderID: "project/bucket/builder",
				Tags: []string{
					"k1:v1",
				},
			}

			actual := b.ToSimpleBuildProto(ctx)
			So(actual, ShouldResembleProto, &pb.Build{
				Id: 1,
				Builder: &pb.BuilderID{
					Project: "project",
					Bucket:  "bucket",
					Builder: "builder",
				},
				Tags: []*pb.StringPair{
					{
						Key:   "k1",
						Value: "v1",
					},
				},
			})
		})
	})
}
