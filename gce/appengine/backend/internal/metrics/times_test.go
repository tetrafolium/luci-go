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

package metrics

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTimes(t *testing.T) {
	t.Parallel()

	Convey("ReportCreationTime", t, func() {
		c, _ := tsmon.WithDummyInMemory(context.Background())
		s := tsmon.Store(c)

		fields := []interface{}{"prefix", "project", "zone"}

		ReportCreationTime(c, 60.0, "prefix", "project", "zone")
		d := s.Get(c, creationTime, time.Time{}, fields).(*distribution.Distribution)
		So(d.Count(), ShouldEqual, 1)
		So(d.Sum(), ShouldEqual, 60.0)

		ReportCreationTime(c, 120.0, "prefix", "project", "zone")
		d = s.Get(c, creationTime, time.Time{}, fields).(*distribution.Distribution)
		So(d.Count(), ShouldEqual, 2)
		So(d.Sum(), ShouldEqual, 180.0)

		ReportCreationTime(c, math.Inf(1), "prefix", "project", "zone")
		d = s.Get(c, creationTime, time.Time{}, fields).(*distribution.Distribution)
		So(d.Count(), ShouldEqual, 3)
		So(d.Sum(), ShouldEqual, math.Inf(1))
	})

	Convey("ReportConnectionTime", t, func() {
		c, _ := tsmon.WithDummyInMemory(context.Background())
		s := tsmon.Store(c)

		fields := []interface{}{"prefix", "project", "server", "zone"}

		ReportConnectionTime(c, 120.0, "prefix", "project", "server", "zone")
		d := s.Get(c, connectionTime, time.Time{}, fields).(*distribution.Distribution)
		So(d.Count(), ShouldEqual, 1)
		So(d.Sum(), ShouldEqual, 120.0)

		ReportConnectionTime(c, 180.0, "prefix", "project", "server", "zone")
		d = s.Get(c, connectionTime, time.Time{}, fields).(*distribution.Distribution)
		So(d.Count(), ShouldEqual, 2)
		So(d.Sum(), ShouldEqual, 300.0)

		ReportConnectionTime(c, math.Inf(1), "prefix", "project", "server", "zone")
		d = s.Get(c, connectionTime, time.Time{}, fields).(*distribution.Distribution)
		So(d.Count(), ShouldEqual, 3)
		So(d.Sum(), ShouldEqual, math.Inf(1))
	})
}
