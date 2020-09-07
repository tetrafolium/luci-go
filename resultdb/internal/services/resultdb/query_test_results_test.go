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

package resultdb

import (
	"sort"
	"testing"

	durpb "github.com/golang/protobuf/ptypes/duration"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateQueryRequest(t *testing.T) {
	t.Parallel()
	Convey(`TestValidateQueryRequest`, t, func() {
		Convey(`no invocations`, func() {
			err := validateQueryRequest(&pb.QueryTestResultsRequest{})
			So(err, ShouldErrLike, `invocations: unspecified`)
		})

		Convey(`invalid invocation`, func() {
			err := validateQueryRequest(&pb.QueryTestResultsRequest{
				Invocations: []string{"x"},
			})
			So(err, ShouldErrLike, `invocations: "x": does not match`)
		})

		Convey(`invalid max staleness`, func() {
			err := validateQueryRequest(&pb.QueryTestResultsRequest{
				Invocations:  []string{"invocations/x"},
				MaxStaleness: &durpb.Duration{Seconds: -1},
			})
			So(err, ShouldErrLike, `max_staleness: must between 0 and 30m, inclusive`)
		})

		Convey(`invalid page size`, func() {
			err := validateQueryRequest(&pb.QueryTestResultsRequest{
				Invocations: []string{"invocations/x"},
				PageSize:    -1,
			})
			So(err, ShouldErrLike, `page_size: negative`)
		})
	})
}

func TestValidateQueryTestResultsRequest(t *testing.T) {
	t.Parallel()
	Convey(`Valid`, t, func() {
		err := validateQueryTestResultsRequest(&pb.QueryTestResultsRequest{
			Invocations:  []string{"invocations/x"},
			PageSize:     50,
			MaxStaleness: &durpb.Duration{Seconds: 60},
		})
		So(err, ShouldBeNil)
	})

	Convey(`invalid invocation`, t, func() {
		err := validateQueryTestResultsRequest(&pb.QueryTestResultsRequest{
			Invocations: []string{"x"},
		})
		So(err, ShouldErrLike, `invocations: "x": does not match`)
	})
}

func TestQueryTestResults(t *testing.T) {
	Convey(`QueryTestResults`, t, func() {
		ctx := auth.WithState(testutil.SpannerTestContext(t), &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "testproject:testrealm", Permission: permListTestResults},
			},
		})

		insertInv := insert.FinalizedInvocationWithInclusions
		insertTRs := insert.TestResults
		testutil.MustApply(ctx, testutil.CombineMutations(
			insertInv("x", map[string]interface{}{"Realm": "secretproject:testrealm"}, "a"),
			insertInv("a", map[string]interface{}{"Realm": "testproject:testrealm"}, "b"),
			insertInv("b", map[string]interface{}{"Realm": "otherproject:testrealm"}, "c"),
			insertInv("c", map[string]interface{}{"Realm": "otherproject:testrealm"}),
			insertTRs("a", "A", nil, pb.TestStatus_FAIL, pb.TestStatus_PASS),
			insertTRs("b", "B", nil, pb.TestStatus_CRASH, pb.TestStatus_PASS),
			insertTRs("c", "C", nil, pb.TestStatus_PASS),
		)...)

		srv := newTestResultDBService()

		Convey(`Without readMask`, func() {
			res, err := srv.QueryTestResults(ctx, &pb.QueryTestResultsRequest{
				Invocations: []string{"invocations/a"},
			})
			So(err, ShouldBeNil)
			So(res.TestResults, ShouldHaveLength, 5)
		})

		Convey(`Permission denied`, func() {
			_, err := srv.QueryTestResults(ctx, &pb.QueryTestResultsRequest{
				Invocations: []string{"invocations/x"},
			})

			So(err, ShouldHaveAppStatus, codes.PermissionDenied)
		})

		Convey(`Valid`, func() {
			res, err := srv.QueryTestResults(ctx, &pb.QueryTestResultsRequest{
				Invocations: []string{"invocations/a"},
			})
			So(err, ShouldBeNil)
			So(res.TestResults, ShouldHaveLength, 5)

			sort.Slice(res.TestResults, func(i, j int) bool {
				return res.TestResults[i].Name < res.TestResults[j].Name
			})

			So(res.TestResults[0].Name, ShouldEqual, "invocations/a/tests/A/results/0")
			So(res.TestResults[0].Status, ShouldEqual, pb.TestStatus_FAIL)
			So(res.TestResults[0].SummaryHtml, ShouldEqual, "")

			So(res.TestResults[1].Name, ShouldEqual, "invocations/a/tests/A/results/1")
			So(res.TestResults[1].Status, ShouldEqual, pb.TestStatus_PASS)

			So(res.TestResults[2].Name, ShouldEqual, "invocations/b/tests/B/results/0")
			So(res.TestResults[2].Status, ShouldEqual, pb.TestStatus_CRASH)

			So(res.TestResults[3].Name, ShouldEqual, "invocations/b/tests/B/results/1")
			So(res.TestResults[3].Status, ShouldEqual, pb.TestStatus_PASS)

			So(res.TestResults[4].Name, ShouldEqual, "invocations/c/tests/C/results/0")
			So(res.TestResults[4].Status, ShouldEqual, pb.TestStatus_PASS)
		})

		Convey(`With readMask`, func() {
			res, err := srv.QueryTestResults(ctx, &pb.QueryTestResultsRequest{
				Invocations: []string{"invocations/a"},
				PageSize:    1,
				ReadMask: &field_mask.FieldMask{
					Paths: []string{
						"status",
						"summary_html",
					}},
			})
			So(err, ShouldBeNil)
			So(res.TestResults[0].Name, ShouldEqual, "invocations/c/tests/C/results/0")
			So(res.TestResults[0].TestId, ShouldEqual, "")
			So(res.TestResults[0].SummaryHtml, ShouldEqual, "SummaryHtml")
		})
	})
}
