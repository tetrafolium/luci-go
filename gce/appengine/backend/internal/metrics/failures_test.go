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
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/gce/api/config/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/model"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFailures(t *testing.T) {
	t.Parallel()

	Convey("UpdateFailures", t, func() {
		c, _ := tsmon.WithDummyInMemory(memory.Use(context.Background()))
		datastore.GetTestable(c).AutoIndex(true)
		datastore.GetTestable(c).Consistent(true)
		s := tsmon.Store(c)

		fields := []interface{}{400, "prefix", "project", "zone"}

		UpdateFailures(c, 400, &model.VM{
			Attributes: config.VM{
				Project: "project",
				Zone:    "zone",
			},
			Hostname: "name-1",
			Prefix:   "prefix",
		})
		So(s.Get(c, creationFailures, time.Time{}, fields).(int64), ShouldEqual, 1)

		UpdateFailures(c, 400, &model.VM{
			Attributes: config.VM{
				Project: "project",
				Zone:    "zone",
			},
			Hostname: "name-1",
			Prefix:   "prefix",
		})
		So(s.Get(c, creationFailures, time.Time{}, fields).(int64), ShouldEqual, 2)
	})
}
