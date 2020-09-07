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
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateQueryTestExonerationsRequest(t *testing.T) {
	t.Parallel()
	Convey(`Valid`, t, func() {
		err := validateQueryTestExonerationsRequest(&pb.QueryTestExonerationsRequest{
			Invocations:  []string{"invocations/x"},
			PageSize:     50,
			MaxStaleness: &durpb.Duration{Seconds: 60},
		})
		So(err, ShouldBeNil)
	})

	Convey(`invalid predicate`, t, func() {
		err := validateQueryTestExonerationsRequest(&pb.QueryTestExonerationsRequest{
			Invocations: []string{"x"},
		})
		So(err, ShouldErrLike, `invocations: "x": does not match`)
	})
}

func TestQueryTestExonerations(t *testing.T) {
	Convey(`QueryTestExonerations`, t, func() {
		ctx := auth.WithState(testutil.SpannerTestContext(t), &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "testproject:testrealm", Permission: permListTestExonerations},
			},
		})

		insertInv := insert.FinalizedInvocationWithInclusions
		insertEx := insert.TestExonerations
		testutil.MustApply(ctx, testutil.CombineMutations(
			insertInv("x", map[string]interface{}{"Realm": "secretproject:testrealm"}, "a"),
			insertInv("a", map[string]interface{}{"Realm": "testproject:testrealm"}, "b"),
			insertInv("b", nil, "c"),
			insertInv("c", nil),
			insertEx("a", "A", pbutil.Variant("v", "a"), 2),
			insertEx("c", "C", pbutil.Variant("v", "c"), 1),
		)...)

		srv := newTestResultDBService()

		Convey(`Permission denied`, func() {
			_, err := srv.QueryTestExonerations(ctx, &pb.QueryTestExonerationsRequest{
				Invocations: []string{"invocations/x"},
			})
			So(err, ShouldHaveAppStatus, codes.PermissionDenied)
		})

		Convey(`Valid`, func() {
			res, err := srv.QueryTestExonerations(ctx, &pb.QueryTestExonerationsRequest{
				Invocations: []string{"invocations/a"},
			})
			So(err, ShouldBeNil)
			actual := res.TestExonerations
			sort.Slice(actual, func(i, j int) bool {
				return actual[i].Name < actual[j].Name
			})
			So(actual, ShouldResembleProto, []*pb.TestExoneration{
				{
					Name:            "invocations/a/tests/A/exonerations/0",
					TestId:          "A",
					Variant:         pbutil.Variant("v", "a"),
					VariantHash:     pbutil.VariantHash(pbutil.Variant("v", "a")),
					ExonerationId:   "0",
					ExplanationHtml: "explanation 0",
				},
				{
					Name:            "invocations/a/tests/A/exonerations/1",
					TestId:          "A",
					Variant:         pbutil.Variant("v", "a"),
					VariantHash:     pbutil.VariantHash(pbutil.Variant("v", "a")),
					ExonerationId:   "1",
					ExplanationHtml: "explanation 1",
				},
				{
					Name:            "invocations/c/tests/C/exonerations/0",
					TestId:          "C",
					Variant:         pbutil.Variant("v", "c"),
					VariantHash:     pbutil.VariantHash(pbutil.Variant("v", "c")),
					ExonerationId:   "0",
					ExplanationHtml: "explanation 0",
				},
			})
		})
	})
}
