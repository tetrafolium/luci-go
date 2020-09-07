// Copyright 2017 The LUCI Authors.
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

package noop

import (
	"context"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"

	api "github.com/tetrafolium/luci-go/scheduler/api/scheduler/v1"
	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
	"github.com/tetrafolium/luci-go/scheduler/appengine/messages"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task/utils/tasktest"

	. "github.com/smartystreets/goconvey/convey"
)

var _ task.Manager = (*TaskManager)(nil)

func TestFullFlow(t *testing.T) {
	t.Parallel()

	Convey("Noop", t, func() {
		c, tc := testclock.UseTime(context.Background(), testclock.TestTimeUTC)
		tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
			tc.Add(d)
		})

		mgr := TaskManager{}
		ctl := &tasktest.TestController{
			TaskMessage: &messages.NoopTask{
				SleepMs:       100,
				TriggersCount: 2,
			},
			SaveCallback: func() error { return nil },
		}

		So(mgr.LaunchTask(c, ctl), ShouldBeNil)
		So(ctl.TaskState, ShouldResemble, task.State{
			Status: task.StatusSucceeded,
		})
		So(ctl.Triggers, ShouldResemble, []*internal.Trigger{
			{Id: "noop:1:0", Payload: &internal.Trigger_Noop{Noop: &api.NoopTrigger{}}},
			{Id: "noop:1:1", Payload: &internal.Trigger_Noop{Noop: &api.NoopTrigger{}}},
		})
	})
}
