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

package sweep

import (
	"testing"

	"github.com/tetrafolium/luci-go/server/tq/internal/partition"
	"github.com/tetrafolium/luci-go/server/tq/internal/reminder"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHelpers(t *testing.T) {
	t.Parallel()

	const keySpaceBytes = 16

	Convey("OnlyLeased", t, func() {
		reminders := []*reminder.Reminder{
			// Each key be exactly 2*keySpaceBytes chars long.
			{ID: "00000000000000000000000000000001"},
			{ID: "00000000000000000000000000000005"},
			{ID: "00000000000000000000000000000009"},
			{ID: "0000000000000000000000000000000f"}, // ie 15
		}
		leased := partition.SortedPartitions{partition.FromInts(5, 9)}
		So(onlyLeased(reminders, leased, keySpaceBytes), ShouldResemble, []*reminder.Reminder{
			{ID: "00000000000000000000000000000005"},
		})
	})
}
