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

package tsmon

import (
	"context"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/target"
	"github.com/tetrafolium/luci-go/common/tsmon/types"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFlush(t *testing.T) {
	t.Parallel()

	defaultTarget := &target.Task{ServiceName: "test"}

	Convey("With a testing State", t, func() {
		c := WithState(context.Background(), NewState())

		Convey("Sends a metric", func() {
			c, s, m := WithFakes(c)
			s.Cells = []types.Cell{
				{
					types.MetricInfo{
						Name:      "foo",
						Fields:    []field.Field{},
						ValueType: types.StringType,
					},
					types.MetricMetadata{},
					types.CellData{
						FieldVals: []interface{}{},
						ResetTime: time.Unix(1234, 1000),
						Value:     "bar",
					},
				},
			}
			s.DT = defaultTarget
			m.CS = 42

			So(Flush(c), ShouldBeNil)

			So(len(m.Cells), ShouldEqual, 1)
			So(len(m.Cells[0]), ShouldEqual, 1)
			So(m.Cells[0][0], ShouldResemble, types.Cell{
				types.MetricInfo{
					Name:      "foo",
					Fields:    []field.Field{},
					ValueType: types.StringType,
				},
				types.MetricMetadata{},
				types.CellData{
					FieldVals: []interface{}{},
					ResetTime: time.Unix(1234, 1000),
					Value:     "bar",
				},
			})
		})

		Convey("Splits up ChunkSize metrics", func() {
			c, s, m := WithFakes(c)
			s.Cells = make([]types.Cell, 43)
			s.DT = defaultTarget
			m.CS = 42

			for i := 0; i < 43; i++ {
				s.Cells[i] = types.Cell{
					types.MetricInfo{
						Name:      "foo",
						Fields:    []field.Field{},
						ValueType: types.StringType,
					},
					types.MetricMetadata{},
					types.CellData{
						FieldVals: []interface{}{},
						ResetTime: time.Unix(1234, 1000),
						Value:     "bar",
					},
				}
			}

			So(Flush(c), ShouldBeNil)

			So(len(m.Cells), ShouldEqual, 2)
			So(len(m.Cells[0]), ShouldEqual, 42)
			So(len(m.Cells[1]), ShouldEqual, 1)
		})

		Convey("Doesn't split metrics when ChunkSize is 0", func() {
			c, s, m := WithFakes(c)
			s.Cells = make([]types.Cell, 43)
			s.DT = defaultTarget
			m.CS = 0

			for i := 0; i < 43; i++ {
				s.Cells[i] = types.Cell{
					types.MetricInfo{
						Name:      "foo",
						Fields:    []field.Field{},
						ValueType: types.StringType,
					},
					types.MetricMetadata{},
					types.CellData{
						FieldVals: []interface{}{},
						ResetTime: time.Unix(1234, 1000),
						Value:     "bar",
					},
				}
			}

			So(Flush(c), ShouldBeNil)

			So(len(m.Cells), ShouldEqual, 1)
			So(len(m.Cells[0]), ShouldEqual, 43)
		})

		Convey("No Monitor configured", func() {
			c, _, _ := WithFakes(c)
			state := GetState(c)
			state.SetMonitor(nil)

			So(Flush(c), ShouldNotBeNil)
		})

		Convey("Auto flush works", func() {
			start := time.Unix(1454561232, 0)
			c, tc := testclock.UseTime(c, start)
			tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
				tc.Add(d)
			})

			moments := make(chan int)
			flusher := autoFlusher{
				flush: func(ctx context.Context) error {
					select {
					case <-ctx.Done():
					case moments <- int(clock.Now(ctx).Sub(start).Seconds()):
					}
					return nil
				},
			}

			flusher.start(c, time.Second)

			// Each 'flush' gets blocked on sending into 'moments'. Once unblocked, it
			// advances timer by 'interval' sec (1 sec in the test).
			So(<-moments, ShouldEqual, 1)
			So(<-moments, ShouldEqual, 2)
			// and so on ...

			// Doesn't timeout => works.
			flusher.stop()
		})
	})
}
