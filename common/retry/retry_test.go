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

package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
)

// testIterator is an Iterator implementation used for testing.
type testIterator struct {
	total int
	count int
}

func (i *testIterator) Next(_ context.Context, _ error) time.Duration {
	defer func() { i.count++ }()
	if i.count >= i.total {
		return Stop
	}
	return time.Second
}

func TestRetry(t *testing.T) {
	t.Parallel()

	// Generic test failure.
	failure := errors.New("retry: test error")

	Convey(`A testing function`, t, func() {
		ctx, c := testclock.UseTime(context.Background(), time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC))

		// Every time we sleep, update time by one second and count.
		sleeps := 0
		c.SetTimerCallback(func(time.Duration, clock.Timer) {
			c.Add(1 * time.Second)
			sleeps++
		})

		Convey(`A test Iterator with three retries`, func() {
			g := func() Iterator {
				return &testIterator{total: 3}
			}

			Convey(`Executes a successful function once.`, func() {
				var count, callbacks int
				err := Retry(ctx, g, func() error {
					count++
					return nil
				}, func(error, time.Duration) {
					callbacks++
				})
				So(err, ShouldBeNil)
				So(count, ShouldEqual, 1)
				So(callbacks, ShouldEqual, 0)
				So(sleeps, ShouldEqual, 0)
			})

			Convey(`Executes a failing function three times.`, func() {
				var count, callbacks int
				err := Retry(ctx, g, func() error {
					count++
					return failure
				}, func(error, time.Duration) {
					callbacks++
				})
				So(err, ShouldEqual, failure)
				So(count, ShouldEqual, 4)
				So(callbacks, ShouldEqual, 3)
				So(sleeps, ShouldEqual, 3)
			})

			Convey(`Executes a function that fails once, then succeeds once.`, func() {
				var count, callbacks int
				err := Retry(ctx, g, func() error {
					defer func() { count++ }()
					if count == 0 {
						return errors.New("retry: test error")
					}
					return nil
				}, func(error, time.Duration) {
					callbacks++
				})
				So(err, ShouldEqual, nil)
				So(count, ShouldEqual, 2)
				So(callbacks, ShouldEqual, 1)
				So(sleeps, ShouldEqual, 1)
			})

			Convey(`Does not retry if context is done.`, func() {
				ctx, cancel := context.WithCancel(ctx)
				var count, callbacks int
				err := Retry(ctx, g, func() error {
					cancel()
					count++
					return failure
				}, func(error, time.Duration) {
					callbacks++
				})
				So(err, ShouldEqual, failure)
				So(count, ShouldEqual, 1)
				So(callbacks, ShouldEqual, 0)
				So(sleeps, ShouldEqual, 0)
			})
		})

		Convey(`Does not retry if callback is not set.`, func() {
			So(Retry(ctx, nil, func() error {
				return failure
			}, nil), ShouldEqual, failure)
			So(sleeps, ShouldEqual, 0)
		})
	})
}
