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

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateGetTestExonerationRequest(t *testing.T) {
	t.Parallel()
	Convey(`ValidateGetTestExonerationRequest`, t, func() {
		Convey(`Valid`, func() {
			req := &pb.GetTestExonerationRequest{Name: "invocations/a/tests/ninja:%2F%2Fchrome%2Ftest:foo_tests%2FBarTest.DoBaz/exonerations/id"}
			So(validateGetTestExonerationRequest(req), ShouldBeNil)
		})

		Convey(`Invalid name`, func() {
			req := &pb.GetTestExonerationRequest{}
			So(validateGetTestExonerationRequest(req), ShouldErrLike, "unspecified")
		})
	})
}

func TestGetTestExoneration(t *testing.T) {
	Convey(`GetTestExoneration`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		srv := newTestResultDBService()

		invID := invocations.ID("inv_0")
		// Insert a TestExoneration.
		testutil.MustApply(ctx,
			insert.Invocation("inv_0", pb.Invocation_ACTIVE, nil),
			spanutil.InsertMap("TestExonerations", map[string]interface{}{
				"InvocationId":    invID,
				"TestId":          "ninja://chrome/test:foo_tests/BarTest.DoBaz",
				"ExonerationId":   "id",
				"Variant":         pbutil.Variant("k1", "v1", "k2", "v2"),
				"VariantHash":     "deadbeef",
				"ExplanationHTML": spanutil.Compressed("broken"),
			}))

		req := &pb.GetTestExonerationRequest{Name: "invocations/inv_0/tests/ninja:%2F%2Fchrome%2Ftest:foo_tests%2FBarTest.DoBaz/exonerations/id"}
		tr, err := srv.GetTestExoneration(ctx, req)
		So(err, ShouldBeNil)
		So(tr, ShouldResembleProto, &pb.TestExoneration{
			Name:            "invocations/inv_0/tests/ninja:%2F%2Fchrome%2Ftest:foo_tests%2FBarTest.DoBaz/exonerations/id",
			ExonerationId:   "id",
			TestId:          "ninja://chrome/test:foo_tests/BarTest.DoBaz",
			Variant:         pbutil.Variant("k1", "v1", "k2", "v2"),
			VariantHash:     "deadbeef",
			ExplanationHtml: "broken",
		})
	})
}
