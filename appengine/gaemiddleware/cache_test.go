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

package gaemiddleware

import (
	"context"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/server/caching"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGlobalCache(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		ctx := context.Background()
		ctx, _ = testclock.UseTime(ctx, testclock.TestRecentTimeUTC)
		ctx = memory.Use(ctx)
		ctx = caching.WithGlobalCache(ctx, blobCacheProvider)

		cache := caching.GlobalCache(ctx, "namespace")

		// Cache miss.
		val, err := cache.Get(ctx, "key")
		So(err, ShouldEqual, caching.ErrCacheMiss)
		So(val, ShouldBeNil)

		So(cache.Set(ctx, "key_permanent", []byte("1"), 0), ShouldBeNil)
		So(cache.Set(ctx, "key_temp", []byte("2"), time.Minute), ShouldBeNil)

		// Cache hit.
		val, err = cache.Get(ctx, "key_permanent")
		So(err, ShouldBeNil)
		So(val, ShouldResemble, []byte("1"))

		val, err = cache.Get(ctx, "key_temp")
		So(err, ShouldBeNil)
		So(val, ShouldResemble, []byte("2"))

		// Expire one item.
		clock.Get(ctx).(testclock.TestClock).Add(2 * time.Minute)

		val, err = cache.Get(ctx, "key_permanent")
		So(err, ShouldBeNil)
		So(val, ShouldResemble, []byte("1"))

		// Expired!
		val, err = cache.Get(ctx, "key_temp")
		So(err, ShouldEqual, caching.ErrCacheMiss)
		So(val, ShouldBeNil)
	})
}
