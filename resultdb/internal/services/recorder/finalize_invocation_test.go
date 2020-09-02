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

	"google.golang.org/grpc/metadata"

	"github.com/tetrafolium/luci-go/server/experiments"
	"github.com/tetrafolium/luci-go/server/span"
	"github.com/tetrafolium/luci-go/server/tq"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/tasks"
	"github.com/tetrafolium/luci-go/resultdb/internal/tasks/taskspb"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil/insert"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateFinalizeInvocationRequest(t *testing.T) {
	Convey(`TestValidateFinalizeInvocationRequest`, t, func() {
		Convey(`Valid`, func() {
			err := validateFinalizeInvocationRequest(&pb.FinalizeInvocationRequest{
				Name: "invocations/a",
			})
			So(err, ShouldBeNil)
		})

		Convey(`Invalid name`, func() {
			err := validateFinalizeInvocationRequest(&pb.FinalizeInvocationRequest{
				Name: "x",
			})
			So(err, ShouldErrLike, `name: does not match`)
		})
	})
}

func TestFinalizeInvocation(t *testing.T) {
	Convey(`TestFinalizeInvocation`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, sched := tq.TestingContext(ctx, nil)
		recorder := newTestRecorderServer()

		// Note: testing only new TQ-based code path. The old one will be removed
		// soon, it's fine not to test it. We "know" it works.
		ctx = experiments.Enable(ctx, tasks.UseFinalizationTQ)

		token, err := generateInvocationToken(ctx, "inv")
		So(err, ShouldBeNil)
		ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(UpdateTokenMetadataKey, token))

		Convey(`Idempotent`, func() {
			testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_ACTIVE, nil))

			inv, err := recorder.FinalizeInvocation(ctx, &pb.FinalizeInvocationRequest{Name: "invocations/inv"})
			So(err, ShouldBeNil)
			So(inv.State, ShouldEqual, pb.Invocation_FINALIZING)

			inv, err = recorder.FinalizeInvocation(ctx, &pb.FinalizeInvocationRequest{Name: "invocations/inv"})
			So(err, ShouldBeNil)
			So(inv.State, ShouldEqual, pb.Invocation_FINALIZING)

			So(sched.Tasks(), ShouldHaveLength, 1)
		})

		Convey(`Success`, func() {
			testutil.MustApply(ctx, insert.Invocation("inv", pb.Invocation_ACTIVE, nil))
			inv, err := recorder.FinalizeInvocation(ctx, &pb.FinalizeInvocationRequest{Name: "invocations/inv"})
			So(err, ShouldBeNil)
			So(inv.State, ShouldEqual, pb.Invocation_FINALIZING)

			// Read the invocation from Spanner to confirm it's really FINALIZING.
			inv, err = invocations.Read(span.Single(ctx), "inv")
			So(err, ShouldBeNil)
			So(inv.State, ShouldEqual, pb.Invocation_FINALIZING)

			// Enqueued the finalization task.
			So(sched.Tasks().Payloads(), ShouldResembleProto, []*taskspb.TryFinalizeInvocation{
				{InvocationId: "inv"},
			})
		})
	})
}
