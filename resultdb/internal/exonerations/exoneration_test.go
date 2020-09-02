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

package exonerations

import (
	"testing"

	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestRead(t *testing.T) {
	Convey(`Read`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		invID := invocations.ID("inv")
		// Insert a TestExoneration.
		testutil.MustApply(ctx,
			insert.Invocation("inv", pb.Invocation_ACTIVE, nil),
			spanutil.InsertMap("TestExonerations", map[string]interface{}{
				"InvocationId":    invID,
				"TestId":          "t t",
				"ExonerationId":   "id",
				"Variant":         pbutil.Variant("k1", "v1", "k2", "v2"),
				"VariantHash":     "deadbeef",
				"ExplanationHTML": spanutil.Compressed("broken"),
			}))

		const name = "invocations/inv/tests/t%20t/exonerations/id"
		ex, err := Read(span.Single(ctx), name)
		So(err, ShouldBeNil)
		So(ex, ShouldResembleProto, &pb.TestExoneration{
			Name:            name,
			ExonerationId:   "id",
			TestId:          "t t",
			Variant:         pbutil.Variant("k1", "v1", "k2", "v2"),
			ExplanationHtml: "broken",
			VariantHash:     "deadbeef",
		})
	})
}
