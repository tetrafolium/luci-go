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

package integrationtests

import (
	"context"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/server/experiments"
	"github.com/tetrafolium/luci-go/server/tq"

	"github.com/tetrafolium/luci-go/resultdb/internal/tasks"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInvocationFinalization(t *testing.T) {
	// https://crbug.com/1116284
	SkipConvey(`ShouldFinalize`, t, func() {
		ctx := testutil.SpannerTestContext(t)

		// Note: testing only new TQ-based code path. The old one will be removed
		// soon, it's fine not to test it. We "know" it works.
		ctx = experiments.Enable(ctx, tasks.UseFinalizationTQ)

		// Cancel the test after 20 sec.
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		// Setup Cloud Tasks fake to pump messages between servers.
		ctx, sched := tq.TestingContext(ctx, nil)
		go sched.Run(ctx)

		app, err := startTestApp(ctx)
		So(err, ShouldBeNil)
		defer app.Shutdown()
		c := testClient{app: app}

		// Create invocations A, B, C.
		// A includes B. B includes C.
		c.CreateInvocation(ctx, "u-a")
		c.CreateInvocation(ctx, "u-b")
		c.CreateInvocation(ctx, "u-c")
		c.Include(ctx, "invocations/u-a", "invocations/u-b")
		c.Include(ctx, "invocations/u-b", "invocations/u-c")

		// Finalize A, B and C.
		c.FinalizeInvocation(ctx, "invocations/u-a")
		So(c.GetState(ctx, "invocations/u-a"), ShouldEqual, pb.Invocation_FINALIZING)
		c.FinalizeInvocation(ctx, "invocations/u-b")
		So(c.GetState(ctx, "invocations/u-b"), ShouldEqual, pb.Invocation_FINALIZING)
		c.FinalizeInvocation(ctx, "invocations/u-c")

		// Assert that all three invocations are finalized within 10 seconds.
		for {
			time.Sleep(100 * time.Millisecond)
			if c.GetState(ctx, "invocations/u-a") != pb.Invocation_FINALIZED {
				continue
			}
			if c.GetState(ctx, "invocations/u-b") != pb.Invocation_FINALIZED {
				continue
			}
			if c.GetState(ctx, "invocations/u-c") != pb.Invocation_FINALIZED {
				continue
			}

			break
		}
	})
}
