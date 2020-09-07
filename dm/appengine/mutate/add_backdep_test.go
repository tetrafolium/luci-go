// Copyright 2015 The LUCI Authors.
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

package mutate

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	"github.com/tetrafolium/luci-go/gae/filter/featureBreaker"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/tumble"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestAddBackDep(t *testing.T) {
	t.Parallel()

	Convey("AddBackDep", t, func() {
		c := memory.Use(context.Background())

		abd := &AddBackDep{
			Dep: &model.FwdEdge{
				From: dm.NewAttemptID("quest", 1),
				To:   dm.NewAttemptID("to", 1),
			},
		}

		Convey("Root", func() {
			So(abd.Root(c).String(), ShouldEqual, `dev~app::/BackDepGroup,"to|fffffffe"`)
		})

		Convey("RollForward", func() {
			bdg, bd := abd.Dep.Back(c)
			So(bd.Propagated, ShouldBeFalse)

			Convey("attempt finished", func() {
				bdg.AttemptFinished = true
				So(ds.Put(c, bdg), ShouldBeNil)

				Convey("no need completion", func() {
					muts, err := abd.RollForward(c)
					So(err, ShouldBeNil)
					So(muts, ShouldBeNil)

					So(ds.Get(c, bdg, bd), ShouldBeNil)
					So(bd.Edge(), ShouldResemble, abd.Dep)
					So(bd.Propagated, ShouldBeTrue)
				})

				Convey("need completion", func() {
					abd.NeedsAck = true
					muts, err := abd.RollForward(c)
					So(err, ShouldBeNil)
					So(muts, ShouldResemble, []tumble.Mutation{&AckFwdDep{abd.Dep}})

					So(ds.Get(c, bdg, bd), ShouldBeNil)
					So(bd.Edge(), ShouldResemble, abd.Dep)
					So(bd.Propagated, ShouldBeTrue)
				})
			})

			Convey("attempt not finished, need completion", func() {
				ex, err := ds.Exists(c, ds.KeyForObj(c, bdg))
				So(err, ShouldBeNil)
				So(ex.Any(), ShouldBeFalse)

				abd.NeedsAck = true
				muts, err := abd.RollForward(c)
				So(err, ShouldBeNil)
				So(muts, ShouldBeNil)

				// Note that bdg was created as a side effect.
				So(ds.Get(c, bdg, bd), ShouldBeNil)
				So(bd.Edge(), ShouldResemble, abd.Dep)
				So(bd.Propagated, ShouldBeFalse)
				So(bdg.AttemptFinished, ShouldBeFalse)
			})

			Convey("failure", func() {
				c, fb := featureBreaker.FilterRDS(c, nil)
				fb.BreakFeatures(nil, "PutMulti")

				_, err := abd.RollForward(c)
				So(err, ShouldErrLike, `feature "PutMulti" is broken`)
			})
		})
	})
}
