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
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateGetInvocationRequest(t *testing.T) {
	t.Parallel()
	Convey(`ValidateGetInvocationRequest`, t, func() {
		Convey(`Valid`, func() {
			req := &pb.GetInvocationRequest{Name: "invocations/valid_id_0"}
			So(validateGetInvocationRequest(req), ShouldBeNil)
		})

		Convey(`Invalid name`, func() {
			Convey(`, missing`, func() {
				req := &pb.GetInvocationRequest{}
				So(validateGetInvocationRequest(req), ShouldErrLike, "name missing")
			})

			Convey(`, invalid format`, func() {
				req := &pb.GetInvocationRequest{Name: "bad_name"}
				So(validateGetInvocationRequest(req), ShouldErrLike, "does not match")
			})
		})
	})
}

func TestGetInvocation(t *testing.T) {
	Convey(`GetInvocation`, t, func() {
		ctx := auth.WithState(testutil.SpannerTestContext(t), &authtest.FakeState{
			Identity: "user:someone@example.com",
			IdentityPermissions: []authtest.RealmPermission{
				{Realm: "testproject:testrealm", Permission: permGetInvocation},
			},
		})
		ct := testclock.TestRecentTimeUTC
		deadline := ct.Add(time.Hour)
		srv := newTestResultDBService()

		Convey(`Valid`, func() {

			// Insert some Invocations.
			testutil.MustApply(ctx,
				insert.Invocation("including", pb.Invocation_ACTIVE, map[string]interface{}{
					"CreateTime": ct,
					"Deadline":   deadline,
					"Realm":      "testproject:testrealm",
				}),
				insert.Invocation("included0", pb.Invocation_FINALIZED, nil),
				insert.Invocation("included1", pb.Invocation_FINALIZED, nil),
				insert.Inclusion("including", "included0"),
				insert.Inclusion("including", "included1"),
			)

			// Fetch back the top-level Invocation.
			req := &pb.GetInvocationRequest{Name: "invocations/including"}
			inv, err := srv.GetInvocation(ctx, req)
			So(err, ShouldBeNil)
			So(inv, ShouldResembleProto, &pb.Invocation{
				Name:                "invocations/including",
				State:               pb.Invocation_ACTIVE,
				CreateTime:          pbutil.MustTimestampProto(ct),
				Deadline:            pbutil.MustTimestampProto(deadline),
				IncludedInvocations: []string{"invocations/included0", "invocations/included1"},
				Realm:               "testproject:testrealm",
			})
		})

		Convey(`Permission denied`, func() {
			testutil.MustApply(ctx,
				insert.Invocation("secret", pb.Invocation_ACTIVE, map[string]interface{}{
					"Realm": "secretproject:testrealm",
				}),
			)
			req := &pb.GetInvocationRequest{Name: "invocations/secret"}
			_, err := srv.GetInvocation(ctx, req)
			So(err, ShouldHaveAppStatus, codes.PermissionDenied)
		})
	})
}
