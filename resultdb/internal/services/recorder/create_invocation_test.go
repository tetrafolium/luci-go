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

package recorder

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/testing/prpctest"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateInvocationDeadline(t *testing.T) {
	Convey(`ValidateInvocationDeadline`, t, func() {
		now := testclock.TestRecentTimeUTC

		Convey(`deadline in the past`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(-time.Hour))
			err := validateInvocationDeadline(deadline, now)
			So(err, ShouldErrLike, `must be at least 10 seconds in the future`)
		})

		Convey(`deadline 5s in the future`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(5 * time.Second))
			err := validateInvocationDeadline(deadline, now)
			So(err, ShouldErrLike, `must be at least 10 seconds in the future`)
		})

		Convey(`deadline in the future`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(1e3 * time.Hour))
			err := validateInvocationDeadline(deadline, now)
			So(err, ShouldErrLike, `must be before 48h in the future`)
		})
	})
}

func TestVerifyCreateInvocationPermissions(t *testing.T) {
	t.Parallel()
	Convey(`TestVerifyCreateInvocationPermissions`, t, func() {
		ctx := auth.WithState(context.Background(), &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "chromium:ci", Permission: permCreateInvocation},
			},
		})
		Convey(`reserved prefix`, func() {
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "build:8765432100",
				Invocation: &pb.Invocation{
					Realm: "chromium:ci",
				},
			})
			So(err, ShouldErrLike, `only invocations created by trusted systems may have id not starting with "u-"`)
		})

		Convey(`reserved prefix, allowed`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{
					{Realm: "chromium:ci", Permission: permCreateInvocation},
					{Realm: "chromium:ci", Permission: permCreateWithReservedID},
				},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "build:8765432100",
				Invocation: &pb.Invocation{
					Realm: "chromium:ci",
				},
			})
			So(err, ShouldBeNil)
		})
		Convey(`producer_resource disallowed`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{
					{Realm: "chromium:ci", Permission: permCreateInvocation},
				},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "u-0",
				Invocation: &pb.Invocation{
					Realm:            "chromium:ci",
					ProducerResource: "//builds.example.com/builds/1",
				},
			})
			So(err, ShouldErrLike, `only invocations created by trusted system may have a populated producer_resource field`)
		})

		Convey(`producer_resource allowed`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{
					{Realm: "chromium:ci", Permission: permCreateInvocation},
					{Realm: "chromium:ci", Permission: permSetProducerResource},
				},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "u-0",
				Invocation: &pb.Invocation{
					Realm:            "chromium:ci",
					ProducerResource: "//builds.example.com/builds/1",
				},
			})
			So(err, ShouldBeNil)
		})
		Convey(`bigquery_exports allowed`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{
					{Realm: "chromium:ci", Permission: permCreateInvocation},
					{Realm: "chromium:ci", Permission: permExportToBigQuery},
				},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Realm: "chromium:ci",
					BigqueryExports: []*pb.BigQueryExport{
						{
							Project:     "project",
							Dataset:     "dataset",
							Table:       "table",
							TestResults: &pb.BigQueryExport_TestResults{},
						},
					},
				},
			})
			So(err, ShouldBeNil)
		})
		Convey(`bigquery_exports disallowed`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity: "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{
					{Realm: "chromium:ci", Permission: permCreateInvocation},
				},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Realm: "chromium:ci",
					BigqueryExports: []*pb.BigQueryExport{
						{
							Project:     "project",
							Dataset:     "dataset",
							Table:       "table",
							TestResults: &pb.BigQueryExport_TestResults{},
						},
					},
				},
			})
			So(err, ShouldErrLike, `does not have permission to set bigquery exports`)
		})
		Convey(`creation disallowed`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity:            "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "build:8765432100",
				Invocation: &pb.Invocation{
					Realm: "chromium:ci",
				},
			})
			So(err, ShouldErrLike, `does not have permission to create invocations`)
		})
		Convey(`invalid realm`, func() {
			ctx = auth.WithState(context.Background(), &authtest.FakeState{
				Identity:            "user:someone@example.com",
				IdentityPermissions: []authtest.RealmPermission{},
			})
			err := verifyCreateInvocationPermissions(ctx, &pb.CreateInvocationRequest{
				InvocationId: "build:8765432100",
				Invocation: &pb.Invocation{
					Realm: "invalid:",
				},
			})
			So(err, ShouldHaveAppStatus, codes.InvalidArgument, `invocation.realm: bad global realm name`)
		})
	})

}
func TestValidateCreateInvocationRequest(t *testing.T) {
	t.Parallel()
	now := testclock.TestRecentTimeUTC
	Convey(`TestValidateCreateInvocationRequest`, t, func() {
		Convey(`empty`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{}, now)
			So(err, ShouldErrLike, `invocation_id: unspecified`)
		})

		Convey(`invalid id`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "1",
			}, now)
			So(err, ShouldErrLike, `invocation_id: does not match`)
		})

		Convey(`invalid request id`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u-a",
				RequestId:    "😃",
			}, now)
			So(err, ShouldErrLike, "request_id: does not match")
		})

		Convey(`invalid tags`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Realm: "chromium:ci",
					Tags:  pbutil.StringPairs("1", "a"),
				},
			}, now)
			So(err, ShouldErrLike, `invocation.tags: "1":"a": key: does not match`)
		})

		Convey(`invalid deadline`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(-time.Hour))
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Realm:    "chromium:ci",
					Deadline: deadline,
				},
			}, now)
			So(err, ShouldErrLike, `invocation: deadline: must be at least 10 seconds in the future`)
		})

		Convey(`invalid realm`, func() {
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Realm: "B@d/f::rm@t",
				},
			}, now)
			So(err, ShouldErrLike, `invocation.realm: bad global realm name`)
		})

		Convey(`invalid bigqueryExports`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(time.Hour))
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Deadline: deadline,
					Tags:     pbutil.StringPairs("a", "b", "a", "c", "d", "e"),
					Realm:    "chromium:ci",
					BigqueryExports: []*pb.BigQueryExport{
						{
							Project: "project",
						},
					},
				},
			}, now)
			So(err, ShouldErrLike, `bigquery_export[0]: dataset: unspecified`)
		})

		Convey(`valid`, func() {
			deadline := pbutil.MustTimestampProto(now.Add(time.Hour))
			err := validateCreateInvocationRequest(&pb.CreateInvocationRequest{
				InvocationId: "u-abc",
				Invocation: &pb.Invocation{
					Deadline: deadline,
					Tags:     pbutil.StringPairs("a", "b", "a", "c", "d", "e"),
					Realm:    "chromium:ci",
				},
			}, now)
			So(err, ShouldBeNil)
		})

	})
}

func TestCreateInvocation(t *testing.T) {
	Convey(`TestCreateInvocation`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "testproject:testrealm", Permission: permCreateInvocation},
				{Realm: "testproject:testrealm", Permission: permCreateWithReservedID},
				{Realm: "testproject:testrealm", Permission: permExportToBigQuery},
				{Realm: "testproject:testrealm", Permission: permSetProducerResource},
			},
		})

		start := clock.Now(ctx).UTC()

		// Setup a full HTTP server in order to retrieve response headers.
		server := &prpctest.Server{}
		server.UnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			res, err := handler(ctx, req)
			err = appstatus.GRPCifyAndLog(ctx, err)
			return res, err
		}
		pb.RegisterRecorderServer(server, newTestRecorderServer())
		server.Start(ctx)
		defer server.Close()
		client, err := server.NewClient()
		So(err, ShouldBeNil)
		recorder := pb.NewRecorderPRPCClient(client)

		Convey(`empty request`, func() {
			_, err := recorder.CreateInvocation(ctx, &pb.CreateInvocationRequest{})
			So(err, ShouldHaveGRPCStatus, codes.InvalidArgument, `invocation: unspecified`)
		})
		Convey(`invalid realm`, func() {
			req := &pb.CreateInvocationRequest{
				InvocationId: "u-inv",
				Invocation: &pb.Invocation{
					Realm: "testproject:",
				},
				RequestId: "request id",
			}
			_, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldHaveGRPCStatus, codes.InvalidArgument, `invocation.realm`)
		})
		Convey(`missing invocation id`, func() {
			_, err := recorder.CreateInvocation(ctx, &pb.CreateInvocationRequest{
				Invocation: &pb.Invocation{
					Realm: "testproject:testrealm",
				},
			})
			So(err, ShouldHaveGRPCStatus, codes.InvalidArgument, `invocation_id: unspecified`)
		})

		req := &pb.CreateInvocationRequest{
			InvocationId: "u-inv",
			Invocation: &pb.Invocation{
				Realm: "testproject:testrealm",
			},
		}

		Convey(`already exists`, func() {
			_, err := span.Apply(ctx, []*spanner.Mutation{
				insert.Invocation("u-inv", 1, nil),
			})
			So(err, ShouldBeNil)

			_, err = recorder.CreateInvocation(ctx, req)
			So(err, ShouldHaveGRPCStatus, codes.AlreadyExists)
		})

		Convey(`unsorted tags`, func() {
			req.Invocation.Tags = pbutil.StringPairs("b", "2", "a", "1")
			inv, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldBeNil)
			So(inv.Tags, ShouldResemble, pbutil.StringPairs("a", "1", "b", "2"))
		})

		Convey(`no invocation in request`, func() {
			_, err := recorder.CreateInvocation(ctx, &pb.CreateInvocationRequest{InvocationId: "u-inv"})
			So(err, ShouldErrLike, "invocation: unspecified")
		})

		Convey(`idempotent`, func() {
			req := &pb.CreateInvocationRequest{
				InvocationId: "u-inv",
				Invocation: &pb.Invocation{
					Realm: "testproject:testrealm",
				},
				RequestId: "request id",
			}
			res, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldBeNil)

			res2, err := recorder.CreateInvocation(ctx, req)
			So(err, ShouldBeNil)
			So(res2, ShouldResembleProto, res)
		})

		Convey(`end to end`, func() {
			deadline := pbutil.MustTimestampProto(start.Add(time.Hour))
			headers := &metadata.MD{}
			bqExport := &pb.BigQueryExport{
				Project:     "project",
				Dataset:     "dataset",
				Table:       "table",
				TestResults: &pb.BigQueryExport_TestResults{},
			}
			req := &pb.CreateInvocationRequest{
				InvocationId: "u-inv",
				Invocation: &pb.Invocation{
					Deadline: deadline,
					Tags:     pbutil.StringPairs("a", "1", "b", "2"),
					BigqueryExports: []*pb.BigQueryExport{
						bqExport,
					},
					ProducerResource: "//builds.example.com/builds/1",
					Realm:            "testproject:testrealm",
				},
			}
			inv, err := recorder.CreateInvocation(ctx, req, prpc.Header(headers))
			So(err, ShouldBeNil)

			expected := proto.Clone(req.Invocation).(*pb.Invocation)
			proto.Merge(expected, &pb.Invocation{
				Name:      "invocations/u-inv",
				State:     pb.Invocation_ACTIVE,
				CreatedBy: "user:someone@example.com",

				// we use Spanner commit time, so skip the check
				CreateTime: inv.CreateTime,
			})
			So(inv, ShouldResembleProto, expected)

			So(headers.Get(UpdateTokenMetadataKey), ShouldHaveLength, 1)

			ctx, cancel := span.ReadOnlyTransaction(ctx)
			defer cancel()

			inv, err = invocations.Read(ctx, "u-inv")
			So(err, ShouldBeNil)
			So(inv, ShouldResembleProto, expected)

			// Check fields not present in the proto.
			var invExpirationTime, expectedResultsExpirationTime time.Time
			err = invocations.ReadColumns(ctx, "u-inv", map[string]interface{}{
				"InvocationExpirationTime":          &invExpirationTime,
				"ExpectedTestResultsExpirationTime": &expectedResultsExpirationTime,
			})
			So(err, ShouldBeNil)
			So(expectedResultsExpirationTime, ShouldHappenWithin, time.Second, start.Add(expectedResultExpiration))
			So(invExpirationTime, ShouldHappenWithin, time.Second, start.Add(invocationExpirationDuration))
		})
	})
}
