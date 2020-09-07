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

package certconfig

import (
	"testing"
	"time"

	ds "github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/server/caching"

	. "github.com/smartystreets/goconvey/convey"
)

func TestListCAs(t *testing.T) {
	Convey("ListCAs works", t, func() {
		ctx := gaetesting.TestingContext()

		// Empty.
		cas, err := ListCAs(ctx)
		So(err, ShouldBeNil)
		So(len(cas), ShouldEqual, 0)

		// Add some.
		err = ds.Put(ctx, &CA{CN: "abc", Removed: true}, &CA{CN: "def"})
		So(err, ShouldBeNil)
		ds.GetTestable(ctx).CatchupIndexes()

		cas, err = ListCAs(ctx)
		So(err, ShouldBeNil)
		So(cas, ShouldResemble, []string{"def"})
	})
}

func TestCAUniqueIDToCNMapLoadStore(t *testing.T) {
	Convey("CAUniqueIDToCNMap Load and Store works", t, func() {
		ctx := gaetesting.TestingContext()

		// Empty.
		mapping, err := LoadCAUniqueIDToCNMap(ctx)
		So(err, ShouldEqual, nil)
		So(len(mapping), ShouldEqual, 0)

		// Store some.
		toStore := map[int64]string{
			1: "abc",
			2: "def",
		}
		err = StoreCAUniqueIDToCNMap(ctx, toStore)
		So(err, ShouldBeNil)

		// Not empty now.
		mapping, err = LoadCAUniqueIDToCNMap(ctx)
		So(err, ShouldEqual, nil)
		So(mapping, ShouldResemble, toStore)
	})
}

func TestGetCAByUniqueID(t *testing.T) {
	Convey("GetCAByUniqueID works", t, func() {
		ctx := gaetesting.TestingContext()
		ctx = caching.WithEmptyProcessCache(ctx)
		ctx, clk := testclock.UseTime(ctx, testclock.TestTimeUTC)

		// Empty now.
		val, err := GetCAByUniqueID(ctx, 1)
		So(err, ShouldBeNil)
		So(val, ShouldEqual, "")

		// Add some.
		err = StoreCAUniqueIDToCNMap(ctx, map[int64]string{
			1: "abc",
			2: "def",
		})
		So(err, ShouldBeNil)

		// Still empty (cached old value).
		val, err = GetCAByUniqueID(ctx, 1)
		So(err, ShouldBeNil)
		So(val, ShouldEqual, "")

		// Updated after cache expires.
		clk.Add(2 * time.Minute)
		val, err = GetCAByUniqueID(ctx, 1)
		So(err, ShouldBeNil)
		So(val, ShouldEqual, "abc")
	})
}
