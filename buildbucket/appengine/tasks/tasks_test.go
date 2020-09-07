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

package tasks

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/gae/filter/txndefer"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/tq"

	taskdef "github.com/tetrafolium/luci-go/buildbucket/appengine/tasks/defs"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestTasks(t *testing.T) {
	t.Parallel()

	Convey("tasks", t, func() {
		ctx := txndefer.FilterRDS(memory.Use(context.Background()))
		datastore.GetTestable(ctx).AutoIndex(true)
		datastore.GetTestable(ctx).Consistent(true)

		ctx, sch := tq.TestingContext(ctx, nil)

		Convey("CancelSwarmingTask", func() {
			Convey("invalid", func() {
				Convey("nil", func() {
					So(CancelSwarmingTask(ctx, nil), ShouldErrLike, "hostname is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})

				Convey("empty", func() {
					task := &taskdef.CancelSwarmingTask{}
					So(CancelSwarmingTask(ctx, task), ShouldErrLike, "hostname is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})

				Convey("hostname", func() {
					task := &taskdef.CancelSwarmingTask{
						TaskId: "id",
					}
					So(CancelSwarmingTask(ctx, task), ShouldErrLike, "hostname is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})

				Convey("task id", func() {
					task := &taskdef.CancelSwarmingTask{
						Hostname: "example.com",
					}
					So(CancelSwarmingTask(ctx, task), ShouldErrLike, "task_id is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})
			})

			Convey("valid", func() {
				Convey("empty realm", func() {
					task := &taskdef.CancelSwarmingTask{
						Hostname: "example.com",
						TaskId:   "id",
					}
					So(datastore.RunInTransaction(ctx, func(ctx context.Context) error {
						return CancelSwarmingTask(ctx, task)
					}, nil), ShouldBeNil)
					So(sch.Tasks(), ShouldHaveLength, 1)
				})

				Convey("non-empty realm", func() {
					task := &taskdef.CancelSwarmingTask{
						Hostname: "example.com",
						TaskId:   "id",
						Realm:    "realm",
					}
					So(datastore.RunInTransaction(ctx, func(ctx context.Context) error {
						return CancelSwarmingTask(ctx, task)
					}, nil), ShouldBeNil)
					So(sch.Tasks(), ShouldHaveLength, 1)
				})
			})
		})

		Convey("NotifyPubSub", func() {
			Convey("invalid", func() {
				Convey("nil", func() {
					So(NotifyPubSub(ctx, nil), ShouldErrLike, "build_id is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})

				Convey("empty", func() {
					task := &taskdef.NotifyPubSub{}
					So(NotifyPubSub(ctx, task), ShouldErrLike, "build_id is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})

				Convey("zero", func() {
					task := &taskdef.NotifyPubSub{
						BuildId: 0,
					}
					So(NotifyPubSub(ctx, task), ShouldErrLike, "build_id is required")
					So(sch.Tasks(), ShouldBeEmpty)
				})
			})

			Convey("valid", func() {
				task := &taskdef.NotifyPubSub{
					BuildId:  1,
					Callback: true,
				}
				So(datastore.RunInTransaction(ctx, func(ctx context.Context) error {
					return NotifyPubSub(ctx, task)
				}, nil), ShouldBeNil)
				So(sch.Tasks(), ShouldHaveLength, 1)
			})
		})
	})
}
