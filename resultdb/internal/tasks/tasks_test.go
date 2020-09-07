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

package tasks

import (
	"sort"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestTasks(t *testing.T) {
	Convey(`TestTasks`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		start := clock.Now(ctx).UTC()

		Convey(`Peek`, func() {
			testutil.MustApply(ctx,
				Enqueue(BQExport, "task1", "inv", "payload", start.Add(-time.Hour)),
				Enqueue(BQExport, "task2", "inv", "payload", start.Add(-time.Hour)),
				Enqueue(BQExport, "task3", "inv", "payload", start.Add(-time.Hour)),
				Enqueue(BQExport, "task4", "inv", "payload", start.Add(time.Hour)),
			)

			var ids []string
			err := Peek(ctx, BQExport, func(id string) error {
				ids = append(ids, id)
				return nil
			})
			So(err, ShouldBeNil)
			sort.Strings(ids)
			So(ids, ShouldResemble, []string{"task1", "task2", "task3"})
		})

		Convey(`Lease`, func() {
			test := func(processAfter time.Time, expectedProcessAfter time.Time, expectSuccess bool) {
				testutil.MustApply(ctx,
					Enqueue(BQExport, "task", "inv", []byte("payload"), processAfter),
				)

				invID, payload, err := Lease(ctx, BQExport, "task", time.Second)
				if !expectSuccess {
					So(err, ShouldEqual, ErrConflict)
					return
				}

				So(err, ShouldBeNil)
				So(invID, ShouldEqual, invocations.ID("inv"))
				So(payload, ShouldResemble, []byte("payload"))

				// Check the task's ProcessAfter is updated.
				var newProcessAfter time.Time
				err = spanutil.ReadRow(span.Single(ctx), "InvocationTasks", BQExport.Key("task"), map[string]interface{}{
					"ProcessAfter": &newProcessAfter,
				})
				So(err, ShouldBeNil)
				So(newProcessAfter, ShouldHappenWithin, time.Second, expectedProcessAfter)
			}

			Convey(`succeeded`, func() {
				test(start.Add(-time.Hour), start.Add(time.Second), true)
			})

			Convey(`skipped`, func() {
				test(start.Add(time.Hour), start.Add(time.Hour), false)
			})

			Convey(`Non-existing task`, func() {
				_, _, err := Lease(ctx, BQExport, "task", time.Second)
				So(err, ShouldEqual, ErrConflict)
			})
		})

		Convey(`Delete`, func() {
			testutil.MustApply(ctx,
				Enqueue(BQExport, "task", "inv", "payload", start.Add(-time.Hour)),
			)

			err := Delete(ctx, BQExport, "task")
			So(err, ShouldBeNil)

			var taskID string
			err = spanutil.ReadRow(span.Single(ctx), "InvocationTasks", BQExport.Key("task"), map[string]interface{}{
				"TaskId": &taskID,
			})
			So(err, ShouldErrLike, "row not found")
		})
	})
}
