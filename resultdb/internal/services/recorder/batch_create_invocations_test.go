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

package recorder

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/testing/prpctest"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateBatchCreateInvocationsRequest(t *testing.T) {
	t.Parallel()
	now := testclock.TestRecentTimeUTC

	Convey(`TestValidateBatchCreateInvocationsRequest`, t, func() {
		Convey(`invalid request id - Batch`, func() {
			_, err := validateBatchCreateInvocationsRequest(
				now,
				[]*pb.CreateInvocationRequest{{
					InvocationId: "u-a",
					Invocation: &pb.Invocation{
						Realm: "testproject:testrealm",
					},
				}},
				"😃",
			)
			So(err, ShouldErrLike, "request_id: does not match")
		})
		Convey(`non-matching request id - Batch`, func() {
			_, err := validateBatchCreateInvocationsRequest(
				now,
				[]*pb.CreateInvocationRequest{{
					InvocationId: "u-a",
					Invocation: &pb.Invocation{
						Realm: "testproject:testrealm",
					},
					RequestId: "valid, but different"}},
				"valid",
			)
			So(err, ShouldErrLike, `request_id: "valid" does not match`)
		})
		Convey(`Too many requests`, func() {
			_, err := validateBatchCreateInvocationsRequest(
				now,
				make([]*pb.CreateInvocationRequest, 1000),
				"valid",
			)
			So(err, ShouldErrLike, `the number of requests in the batch exceeds 500`)
		})
		Convey(`valid`, func() {
			ids, err := validateBatchCreateInvocationsRequest(
				now,
				[]*pb.CreateInvocationRequest{{
					InvocationId: "u-a",
					RequestId:    "valid",
					Invocation: &pb.Invocation{
						Realm: "testproject:testrealm",
					},
				}},
				"valid",
			)
			So(err, ShouldBeNil)
			So(ids.Has("u-a"), ShouldBeTrue)
			So(len(ids), ShouldEqual, 1)
		})
	})
}

func TestBatchCreateInvocations(t *testing.T) {
	Convey(`TestBatchCreateInvocations`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		// Configure mock authentication to allow creation of custom invocation ids.
		authState := &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "testproject:testrealm", Permission: permCreateInvocation},
				{Realm: "testproject:testrealm", Permission: permExportToBigQuery},
				{Realm: "testproject:testrealm", Permission: permSetProducerResource},
			},
		}
		ctx = auth.WithState(ctx, authState)

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

		Convey(`idempotent`, func() {
			req := &pb.BatchCreateInvocationsRequest{
				Requests: []*pb.CreateInvocationRequest{{
					InvocationId: "u-batchinv",
					Invocation:   &pb.Invocation{Realm: "testproject:testrealm"},
				}, {
					InvocationId: "u-batchinv2",
					Invocation:   &pb.Invocation{Realm: "testproject:testrealm"},
				}},
				RequestId: "request id",
			}
			res, err := recorder.BatchCreateInvocations(ctx, req)
			So(err, ShouldBeNil)

			res2, err := recorder.BatchCreateInvocations(ctx, req)
			So(err, ShouldBeNil)
			// Update tokens are regenerated the second time, but they are both valid.
			res2.UpdateTokens = res.UpdateTokens
			// Otherwise, the responses must be identical.
			So(res2, ShouldResembleProto, res)
		})

		Convey(`Same request ID, different identity`, func() {
			req := &pb.BatchCreateInvocationsRequest{
				Requests: []*pb.CreateInvocationRequest{{
					InvocationId: "u-inv",
					Invocation:   &pb.Invocation{Realm: "testproject:testrealm"},
				}},
				RequestId: "request id",
			}
			_, err := recorder.BatchCreateInvocations(ctx, req)
			So(err, ShouldBeNil)

			authState.Identity = "user:someone-else@example.com"
			_, err = recorder.BatchCreateInvocations(ctx, req)
			So(status.Code(err), ShouldEqual, codes.AlreadyExists)
		})

		Convey(`end to end`, func() {
			deadline := pbutil.MustTimestampProto(start.Add(time.Hour))
			bqExport := &pb.BigQueryExport{
				Project:     "project",
				Dataset:     "dataset",
				Table:       "table",
				TestResults: &pb.BigQueryExport_TestResults{},
			}
			req := &pb.BatchCreateInvocationsRequest{
				Requests: []*pb.CreateInvocationRequest{
					{
						InvocationId: "u-batch-inv",
						Invocation: &pb.Invocation{
							Deadline: deadline,
							Tags:     pbutil.StringPairs("a", "1", "b", "2"),
							BigqueryExports: []*pb.BigQueryExport{
								bqExport,
							},
							ProducerResource: "//builds.example.com/builds/1",
							Realm:            "testproject:testrealm",
						},
					},
					{
						InvocationId: "u-batch-inv2",
						Invocation: &pb.Invocation{
							Deadline: deadline,
							Tags:     pbutil.StringPairs("a", "1", "b", "2"),
							BigqueryExports: []*pb.BigQueryExport{
								bqExport,
							},
							ProducerResource: "//builds.example.com/builds/2",
							Realm:            "testproject:testrealm",
						},
					},
				},
			}

			resp, err := recorder.BatchCreateInvocations(ctx, req)
			So(err, ShouldBeNil)

			expected := proto.Clone(req.Requests[0].Invocation).(*pb.Invocation)
			proto.Merge(expected, &pb.Invocation{
				Name:      "invocations/u-batch-inv",
				State:     pb.Invocation_ACTIVE,
				CreatedBy: "user:someone@example.com",

				// we use Spanner commit time, so skip the check
				CreateTime: resp.Invocations[0].CreateTime,
			})
			expected2 := proto.Clone(req.Requests[1].Invocation).(*pb.Invocation)
			proto.Merge(expected2, &pb.Invocation{
				Name:      "invocations/u-batch-inv2",
				State:     pb.Invocation_ACTIVE,
				CreatedBy: "user:someone@example.com",

				// we use Spanner commit time, so skip the check
				CreateTime: resp.Invocations[1].CreateTime,
			})
			So(resp.Invocations[0], ShouldResembleProto, expected)
			So(resp.Invocations[1], ShouldResembleProto, expected2)
			So(resp.UpdateTokens, ShouldHaveLength, 2)

			ctx, cancel := span.ReadOnlyTransaction(ctx)
			defer cancel()

			inv, err := invocations.Read(ctx, "u-batch-inv")
			So(err, ShouldBeNil)
			So(inv, ShouldResembleProto, expected)

			inv2, err := invocations.Read(ctx, "u-batch-inv2")
			So(err, ShouldBeNil)
			So(inv2, ShouldResembleProto, expected2)

			// Check fields not present in the proto.
			var invExpirationTime, expectedResultsExpirationTime time.Time
			err = invocations.ReadColumns(ctx, "u-batch-inv", map[string]interface{}{
				"InvocationExpirationTime":          &invExpirationTime,
				"ExpectedTestResultsExpirationTime": &expectedResultsExpirationTime,
			})
			So(err, ShouldBeNil)
			So(expectedResultsExpirationTime, ShouldHappenWithin, time.Second, start.Add(expectedResultExpiration))
			So(invExpirationTime, ShouldHappenWithin, time.Second, start.Add(invocationExpirationDuration))
		})
	})
}
