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

package perm

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/auth/realms"

	"github.com/tetrafolium/luci-go/buildbucket/appengine/model"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHasInBucket(t *testing.T) {
	t.Parallel()

	Convey("With mocked auth DB", t, func() {
		const (
			anon           = identity.AnonymousIdentity
			admin          = identity.Identity("user:admin@example.com")
			reader         = identity.Identity("user:reader@example.com")
			writer         = identity.Identity("user:writer@example.com")
			sameProject    = identity.Identity("project:some-project")
			anotherProject = identity.Identity("project:another-project")
		)

		s := &authtest.FakeState{
			FakeDB: authtest.NewFakeDB(
				authtest.MockMembership(admin, "administrators"),
				authtest.MockMembership(reader, "readers"),
				authtest.MockMembership(writer, "writers"),
			),
		}
		ctx := auth.WithState(memory.Use(context.Background()), s)

		makeBucket := func(acls []*pb.Acl) {
			So(datastore.Put(ctx, &model.Bucket{
				ID:     "some-bucket",
				Parent: model.ProjectKey(ctx, "some-project"),
				Proto:  pb.Bucket{Acls: acls},
			}), ShouldBeNil)
		}

		check := func(perm realms.Permission, caller identity.Identity) codes.Code {
			s.Identity = caller
			err := HasInBucket(ctx, perm, "some-project", "some-bucket")
			if err == nil {
				return codes.OK
			}
			status, ok := appstatus.Get(err)
			if !ok {
				return codes.Internal
			}
			return status.Code()
		}

		Convey("Missing bucket", func() {
			So(check(BuildsGet, anon), ShouldEqual, codes.NotFound)
			So(check(BuildsGet, admin), ShouldEqual, codes.NotFound)
			So(check(BuildsGet, sameProject), ShouldEqual, codes.NotFound)
			So(check(BuildsGet, reader), ShouldEqual, codes.NotFound)
		})

		Convey("Existing bucket, no ACLs in it", func() {
			makeBucket(nil)

			So(check(BuildsGet, anon), ShouldEqual, codes.NotFound)
			So(check(BuildsGet, admin), ShouldEqual, codes.OK)
			So(check(BuildsGet, sameProject), ShouldEqual, codes.OK)
			So(check(BuildsGet, anotherProject), ShouldEqual, codes.NotFound)
			So(check(BuildsGet, reader), ShouldEqual, codes.NotFound)
		})

		Convey("Existing bucket, with ACLs", func() {
			makeBucket([]*pb.Acl{
				{Role: pb.Acl_READER, Group: "readers"},
				{Role: pb.Acl_WRITER, Group: "writers"},
			})

			Convey("Read perm", func() {
				So(check(BuildsGet, anon), ShouldEqual, codes.NotFound)
				So(check(BuildsGet, admin), ShouldEqual, codes.OK)
				So(check(BuildsGet, sameProject), ShouldEqual, codes.OK)
				So(check(BuildsGet, anotherProject), ShouldEqual, codes.NotFound)
				So(check(BuildsGet, reader), ShouldEqual, codes.OK)
				So(check(BuildsGet, writer), ShouldEqual, codes.OK)
			})

			Convey("Write perm", func() {
				So(check(BuildsCancel, anon), ShouldEqual, codes.NotFound)
				So(check(BuildsCancel, admin), ShouldEqual, codes.OK)
				So(check(BuildsCancel, sameProject), ShouldEqual, codes.OK)
				So(check(BuildsCancel, anotherProject), ShouldEqual, codes.NotFound)
				So(check(BuildsCancel, reader), ShouldEqual, codes.PermissionDenied)
				So(check(BuildsCancel, writer), ShouldEqual, codes.OK)
			})
		})
	})
}

func TestGetRole(t *testing.T) {
	t.Parallel()

	Convey("With mocked auth DB", t, func() {
		s := &authtest.FakeState{
			FakeDB: authtest.NewFakeDB(
				authtest.MockMembership("user:reader@example.com", "readers"),
				authtest.MockMembership("user:writer@example.com", "writers"),
				authtest.MockMembership("user:writer@example.com", "readers"), // also a reader
			),
		}
		ctx := auth.WithState(context.Background(), s)

		role := func(id identity.Identity, acls []*pb.Acl) pb.Acl_Role {
			r, err := getRole(ctx, id, acls)
			So(err, ShouldBeNil)
			return r
		}

		Convey("Empty", func() {
			So(role("user:someone@example.com", nil), ShouldEqual, -1)
		})

		Convey("Email", func() {
			acls := []*pb.Acl{
				{
					Identity: "some-email@example.com",
					Role:     pb.Acl_READER,
				},
			}

			So(role("user:some-email@example.com", acls), ShouldEqual, pb.Acl_READER)
			So(role("user:another-email@example.com", acls), ShouldEqual, -1)
		})

		Convey("User", func() {
			acls := []*pb.Acl{
				{
					Identity: "user:some-email@example.com",
					Role:     pb.Acl_READER,
				},
			}

			So(role("user:some-email@example.com", acls), ShouldEqual, pb.Acl_READER)
			So(role("user:another-email@example.com", acls), ShouldEqual, -1)
		})

		Convey("Group", func() {
			acls := []*pb.Acl{
				{
					Group: "readers",
					Role:  pb.Acl_READER,
				},
				{
					Group: "empty",
					Role:  pb.Acl_READER,
				},
			}

			So(role("user:reader@example.com", acls), ShouldEqual, pb.Acl_READER)
			So(role("user:unknown@example.com", acls), ShouldEqual, -1)
		})

		Convey("Highest role wins", func() {
			acls := []*pb.Acl{
				{
					Group: "readers",
					Role:  pb.Acl_READER,
				},
				{
					Group: "writers",
					Role:  pb.Acl_WRITER,
				},
			}

			So(role("user:reader@example.com", acls), ShouldEqual, pb.Acl_READER)
			So(role("user:writer@example.com", acls), ShouldEqual, pb.Acl_WRITER)
			So(role("user:unknown@example.com", acls), ShouldEqual, -1)
		})
	})
}

func TestBucketsByPerm(t *testing.T) {
	t.Parallel()

	Convey("GetAccessibleBuckets", t, func() {
		ctx := memory.Use(context.Background())
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: identity.Identity("user:user"),
		})
		datastore.GetTestable(ctx).AutoIndex(true)
		datastore.GetTestable(ctx).Consistent(true)

		So(datastore.Put(ctx, &model.Bucket{
			ID:     "bucket1",
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
		So(datastore.Put(ctx, &model.Bucket{
			ID:     "bucket1",
			Parent: model.ProjectKey(ctx, "project2"),
			Proto: pb.Bucket{
				Acls: []*pb.Acl{
					{
						Identity: "user:user",
						Role:     pb.Acl_READER,
					},
				},
			},
		}), ShouldBeNil)
		So(datastore.Put(ctx, &model.Bucket{
			ID:     "bucket2",
			Parent: model.ProjectKey(ctx, "project"),
			Proto: pb.Bucket{
				Acls: []*pb.Acl{
					{
						Identity: "user:user",
						Role:     pb.Acl_WRITER,
					},
				},
			},
		}), ShouldBeNil)

		buckets1, err := BucketsByPerm(ctx, BuildersList, "")
		So(err, ShouldBeNil)
		So(buckets1, ShouldResemble, []string{"project/bucket1", "project/bucket2", "project2/bucket1"})

		buckets2, err := BucketsByPerm(ctx, BuildsCancel, "")
		So(err, ShouldBeNil)
		So(buckets2, ShouldResemble, []string{"project/bucket2"})

		buckets3, err := BucketsByPerm(ctx, BuildersList, "project2")
		So(err, ShouldBeNil)
		So(buckets3, ShouldResemble, []string{"project2/bucket1"})

		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: identity.Identity("user:no_any_permission"),
		})
		buckets4, err := BucketsByPerm(ctx, BuildersList, "")
		So(err, ShouldBeNil)
		So(buckets4, ShouldBeNil)
	})
}
