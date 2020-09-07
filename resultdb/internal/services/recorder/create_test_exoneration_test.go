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
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/exonerations"
	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateCreateTestExonerationRequest(t *testing.T) {
	t.Parallel()
	Convey(`TestValidateCreateTestExonerationRequest`, t, func() {
		Convey(`Empty`, func() {
			err := validateCreateTestExonerationRequest(&pb.CreateTestExonerationRequest{}, true)
			So(err, ShouldErrLike, `invocation: unspecified`)
		})

		Convey(`NUL in test id`, func() {
			err := validateCreateTestExonerationRequest(&pb.CreateTestExonerationRequest{
				Invocation: "invocations/inv",
				TestExoneration: &pb.TestExoneration{
					TestId: "\x01",
				},
			}, true)
			So(err, ShouldErrLike, "test_id: does not match")
		})

		Convey(`Invalid variant`, func() {
			err := validateCreateTestExonerationRequest(&pb.CreateTestExonerationRequest{
				Invocation: "invocations/inv",
				TestExoneration: &pb.TestExoneration{
					TestId:  "a",
					Variant: pbutil.Variant("", ""),
				},
			}, true)
			So(err, ShouldErrLike, `variant: "":"": key: unspecified`)
		})

		Convey(`Valid`, func() {
			err := validateCreateTestExonerationRequest(&pb.CreateTestExonerationRequest{
				Invocation: "invocations/inv",
				TestExoneration: &pb.TestExoneration{
					TestId: "ninja://ab/cd.ef",
					Variant: pbutil.Variant(
						"a/b", "1",
						"c", "2",
					),
				},
			}, true)
			So(err, ShouldBeNil)
		})
	})
}

func TestCreateTestExoneration(t *testing.T) {
	Convey(`TestCreateTestExoneration`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		recorder := newTestRecorderServer()

		token, err := generateInvocationToken(ctx, "inv")
		So(err, ShouldBeNil)
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(UpdateTokenMetadataKey, token))

		Convey(`Invalid request`, func() {
			req := &pb.CreateTestExonerationRequest{
				Invocation: "invocations/inv",
				TestExoneration: &pb.TestExoneration{
					TestId: "\x01",
				},
			}
			_, err := recorder.CreateTestExoneration(ctx, req)
			So(err, ShouldHaveAppStatus, codes.InvalidArgument, `bad request: test_exoneration: test_id: does not match`)
		})

		Convey(`No invocation`, func() {
			req := &pb.CreateTestExonerationRequest{
				Invocation: "invocations/inv",
				TestExoneration: &pb.TestExoneration{
					TestId: "a",
				},
			}
			_, err := recorder.CreateTestExoneration(ctx, req)
			So(err, ShouldHaveAppStatus, codes.NotFound, `invocations/inv not found`)
		})

		// Insert the invocation.
		testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_ACTIVE, nil))

		e2eTest := func(withRequestID bool) {
			req := &pb.CreateTestExonerationRequest{
				Invocation: "invocations/inv",
				TestExoneration: &pb.TestExoneration{
					TestId:  "a",
					Variant: pbutil.Variant("a", "1", "b", "2"),
				},
			}

			if withRequestID {
				req.RequestId = "request id"
			}

			res, err := recorder.CreateTestExoneration(ctx, req)
			So(err, ShouldBeNil)

			So(res.ExonerationId, ShouldStartWith, "6408fdc5c36df5df:") // hash of the variant
			if withRequestID {
				So(res.ExonerationId, ShouldEqual, "6408fdc5c36df5df:d:2960f0231ce23039cdf7d4a62e31939ecd897bbf465e0fb2d35bf425ae1c5ae14eb0714d6dd0a0c244eaa66ae2b645b0637f58e91ed1b820bb1f01d8d4a72e67")
			}

			expected := proto.Clone(req.TestExoneration).(*pb.TestExoneration)
			proto.Merge(expected, &pb.TestExoneration{
				Name:          pbutil.TestExonerationName("inv", "a", res.ExonerationId),
				ExonerationId: res.ExonerationId,
			})
			So(res, ShouldResembleProto, expected)

			// Now check the database.
			row, err := exonerations.Read(span.Single(ctx), res.Name)
			So(err, ShouldBeNil)
			So(row.Variant, ShouldResembleProto, expected.Variant)
			So(row.ExplanationHtml, ShouldEqual, expected.ExplanationHtml)

			// Check variant hash.
			key := invocations.ID("inv").Key(res.TestId, res.ExonerationId)
			var variantHash string
			testutil.MustReadRow(ctx, "TestExonerations", key, map[string]interface{}{
				"VariantHash": &variantHash,
			})
			So(variantHash, ShouldEqual, pbutil.VariantHash(res.Variant))

			if withRequestID {
				// Test idempotency.
				res2, err := recorder.CreateTestExoneration(ctx, req)
				So(err, ShouldBeNil)
				So(res2, ShouldResembleProto, res)
			}
		}

		Convey(`Without request id, e2e`, func() {
			e2eTest(false)
		})
		Convey(`With request id, e2e`, func() {
			e2eTest(true)
		})
	})
}
