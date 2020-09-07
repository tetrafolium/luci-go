// Copyright 2016 The LUCI Authors.
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

package tsmon

import (
	"testing"

	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/monitor"
	"github.com/tetrafolium/luci-go/common/tsmon/store"
	"github.com/tetrafolium/luci-go/common/tsmon/target"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGlobalMetrics(t *testing.T) {
	t.Parallel()

	Convey("Default version", t, func() {
		c, _ := buildGAETestContext()
		tsmon.GetState(c).SetStore(store.NewInMemory(&target.Task{ServiceName: "default target"}))
		collectGlobalMetrics(c)
		tsmon.Flush(c)

		monitor := tsmon.GetState(c).Monitor().(*monitor.Fake)
		So(len(monitor.Cells), ShouldEqual, 1)
		So(monitor.Cells[0][0].Name, ShouldEqual, "appengine/default_version")
		So(monitor.Cells[0][0].Value, ShouldEqual, "testVersion1")
	})
}
