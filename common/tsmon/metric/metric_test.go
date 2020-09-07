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

package metric

import (
	"context"
	"reflect"
	"testing"

	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/registry"
	"github.com/tetrafolium/luci-go/common/tsmon/target"
	"github.com/tetrafolium/luci-go/common/tsmon/types"

	. "github.com/smartystreets/goconvey/convey"
)

func makeContext() context.Context {
	ret, _ := tsmon.WithDummyInMemory(context.Background())
	return ret
}

func TestMetrics(t *testing.T) {
	t.Parallel()
	tt := target.TaskType

	Convey("Int", t, func() {
		c := makeContext()
		m := NewIntWithTargetType("int", tt, "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewIntWithTargetType("int", tt, "description", nil) },
			ShouldPanic,
		)

		So(m.Get(c), ShouldEqual, 0)
		m.Set(c, 42)
		So(m.Get(c), ShouldEqual, 42)

		So(func() { m.Set(c, 42, "field") }, ShouldPanic)
		So(func() { m.Get(c, "field") }, ShouldPanic)
	})

	Convey("Counter", t, func() {
		c := makeContext()
		m := NewCounterWithTargetType("counter", tt, "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewCounterWithTargetType("counter", tt, "description", nil) },
			ShouldPanic,
		)

		So(m.Get(c), ShouldEqual, 0)

		m.Add(c, 3)
		So(m.Get(c), ShouldEqual, 3)

		m.Add(c, 2)
		So(m.Get(c), ShouldEqual, 5)
	})

	Convey("Float", t, func() {
		c := makeContext()
		m := NewFloatWithTargetType("float", tt, "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewFloatWithTargetType("float", tt, "description", nil) },
			ShouldPanic,
		)

		So(m.Get(c), ShouldAlmostEqual, 0.0)

		m.Set(c, 42.3)
		So(m.Get(c), ShouldAlmostEqual, 42.3)

		So(func() { m.Set(c, 42.3, "field") }, ShouldPanic)
		So(func() { m.Get(c, "field") }, ShouldPanic)
	})

	Convey("FloatCounter", t, func() {
		c := makeContext()
		m := NewFloatCounterWithTargetType("float_counter", tt, "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewFloatCounterWithTargetType("float_counter", tt, "description", nil) },
			ShouldPanic,
		)

		So(m.Get(c), ShouldAlmostEqual, 0.0)

		m.Add(c, 3.1)
		So(m.Get(c), ShouldAlmostEqual, 3.1)

		m.Add(c, 2.2)
		So(m.Get(c), ShouldAlmostEqual, 5.3)
	})

	Convey("String", t, func() {
		c := makeContext()
		m := NewStringWithTargetType("string", tt, "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(func() { NewStringWithTargetType("string", tt, "description", nil) }, ShouldPanic)

		So(m.Get(c), ShouldEqual, "")

		m.Set(c, "hello")
		So(m.Get(c), ShouldEqual, "hello")

		So(func() { m.Set(c, "hello", "field") }, ShouldPanic)
		So(func() { m.Get(c, "field") }, ShouldPanic)
	})

	Convey("Bool", t, func() {
		c := makeContext()
		m := NewBoolWithTargetType("bool", tt, "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewBoolWithTargetType("bool", tt, "description", nil) },
			ShouldPanic,
		)

		So(m.Get(c), ShouldEqual, false)

		m.Set(c, true)
		So(m.Get(c), ShouldBeTrue)

		So(func() { m.Set(c, true, "field") }, ShouldPanic)
		So(func() { m.Get(c, "field") }, ShouldPanic)
	})

	Convey("CumulativeDistribution", t, func() {
		c := makeContext()
		m := NewCumulativeDistributionWithTargetType("cumul_dist", tt, "description", nil, distribution.FixedWidthBucketer(10, 20))
		So(m.Info().TargetType, ShouldResemble, tt)
		So(func() { NewCumulativeDistributionWithTargetType("cumul_dist", tt, "description", nil, m.Bucketer()) }, ShouldPanic)

		So(m.Bucketer().GrowthFactor(), ShouldEqual, 0)
		So(m.Bucketer().Width(), ShouldEqual, 10)
		So(m.Bucketer().NumFiniteBuckets(), ShouldEqual, 20)

		So(m.Get(c), ShouldBeNil)

		m.Add(c, 5)

		v := m.Get(c)
		So(v.Bucketer().GrowthFactor(), ShouldEqual, 0)
		So(v.Bucketer().Width(), ShouldEqual, 10)
		So(v.Bucketer().NumFiniteBuckets(), ShouldEqual, 20)
		So(v.Sum(), ShouldEqual, 5)
		So(v.Count(), ShouldEqual, 1)

		So(func() { m.Add(c, 5, "field") }, ShouldPanic)
		So(func() { m.Get(c, "field") }, ShouldPanic)
	})

	Convey("NonCumulativeDistribution", t, func() {
		c := makeContext()
		m := NewNonCumulativeDistributionWithTargetType("noncumul_dist", tt, "description", nil, distribution.FixedWidthBucketer(10, 20))
		So(m.Info().TargetType, ShouldResemble, tt)
		So(func() {
			NewNonCumulativeDistributionWithTargetType("noncumul_dist", tt, "description", nil, m.Bucketer())
		}, ShouldPanic)

		So(m.Bucketer().GrowthFactor(), ShouldEqual, 0)
		So(m.Bucketer().Width(), ShouldEqual, 10)
		So(m.Bucketer().NumFiniteBuckets(), ShouldEqual, 20)

		So(m.Get(c), ShouldBeNil)

		d := distribution.New(m.Bucketer())
		d.Add(15)
		m.Set(c, d)

		v := m.Get(c)
		So(v.Bucketer().GrowthFactor(), ShouldEqual, 0)
		So(v.Bucketer().Width(), ShouldEqual, 10)
		So(v.Bucketer().NumFiniteBuckets(), ShouldEqual, 20)
		So(v.Sum(), ShouldEqual, 15)
		So(v.Count(), ShouldEqual, 1)

		So(func() { m.Set(c, d, "field") }, ShouldPanic)
		So(func() { m.Get(c, "field") }, ShouldPanic)
	})
}

func TestMetricsDefaultTargetType(t *testing.T) {
	t.Parallel()

	// These tests ensure that metrics are given target.NilType, if created
	// without a target type specified.
	tt := target.NilType

	Convey("Int", t, func() {
		m := NewInt("int", "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewIntWithTargetType("int", tt, "description", nil) },
			ShouldPanic,
		)
	})

	Convey("Counter", t, func() {
		m := NewCounter("counter", "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewCounterWithTargetType("counter", tt, "description", nil) },
			ShouldPanic,
		)
	})

	Convey("Float", t, func() {
		m := NewFloat("float", "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewFloatWithTargetType("float", tt, "description", nil) },
			ShouldPanic,
		)
	})

	Convey("FloatCounter", t, func() {
		m := NewFloatCounter("float_counter", "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewFloatCounterWithTargetType("float_counter", tt, "description", nil) },
			ShouldPanic,
		)
	})

	Convey("String", t, func() {
		m := NewString("string", "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(func() { NewStringWithTargetType("string", tt, "description", nil) }, ShouldPanic)
	})

	Convey("Bool", t, func() {
		m := NewBool("bool", "description", nil)
		So(m.Info().TargetType, ShouldResemble, tt)
		So(
			func() { NewBoolWithTargetType("bool", tt, "description", nil) },
			ShouldPanic,
		)
	})

	Convey("CumulativeDistribution", t, func() {
		m := NewCumulativeDistribution("cumul_dist", "description", nil, distribution.FixedWidthBucketer(10, 20))
		So(m.Info().TargetType, ShouldResemble, tt)
		So(func() { NewCumulativeDistributionWithTargetType("cumul_dist", tt, "description", nil, m.Bucketer()) }, ShouldPanic)

	})

	Convey("NonCumulativeDistribution", t, func() {
		m := NewNonCumulativeDistribution("noncumul_dist", "description", nil, distribution.FixedWidthBucketer(10, 20))
		So(m.Info().TargetType, ShouldResemble, tt)
		So(func() {
			NewNonCumulativeDistributionWithTargetType("noncumul_dist", tt, "description", nil, m.Bucketer())
		}, ShouldPanic)
	})
}

func TestMetricsWithMultipleTargets(t *testing.T) {
	t.Parallel()
	testTaskTargets := []target.Task{{TaskNum: 0}, {TaskNum: 1}}
	testDeviceTargets := []target.NetworkDevice{{Hostname: "a"}}

	Convey("with a single TargetType", t, func() {
		c := makeContext()

		Convey("with a single target in context", func() {
			m := NewIntWithTargetType("m_with_s_s", target.TaskType, "desc", nil)
			tctx := target.Set(c, &testTaskTargets[0])
			So(m.Get(tctx), ShouldEqual, 0)
			m.Set(tctx, 42)
			So(m.Get(tctx), ShouldEqual, 42)
		})

		Convey("with multiple targets in context", func() {
			m := NewIntWithTargetType("m_with_s_m", target.TaskType, "desc", nil)
			tctx0 := target.Set(c, &testTaskTargets[0])
			tctx1 := target.Set(tctx0, &testTaskTargets[1])
			So(m.Get(tctx0), ShouldEqual, 0)
			So(m.Get(tctx1), ShouldEqual, 0)
			m.Set(tctx0, 41)
			m.Set(tctx1, 42)
			So(m.Get(tctx0), ShouldEqual, 41)
			So(m.Get(tctx1), ShouldEqual, 42)
		})
	})

	Convey("with multiple TargetTypes", t, func() {
		c := makeContext()

		Convey("with a single target in context for each type", func() {
			tctx := target.Set(
				target.Set(c, &testTaskTargets[0]), &testDeviceTargets[0],
			)

			// two metrics with the same name, but different types.
			mDevice := NewIntWithTargetType("m_with_m_s", target.DeviceType, "desc", nil)
			mTask := NewIntWithTargetType("m_with_m_s", target.TaskType, "desc", nil)

			So(mTask.Get(tctx), ShouldEqual, 0)
			So(mDevice.Get(tctx), ShouldEqual, 0)
			mTask.Set(tctx, 41)
			mDevice.Set(tctx, 42)
			So(mTask.Get(tctx), ShouldEqual, 41)
			So(mDevice.Get(tctx), ShouldEqual, 42)
		})
	})
}

// To avoid import cycle, unit tests for Registry with metrics are implemented
// here.
func TestMetricWithRegistry(t *testing.T) {
	t.Parallel()

	Convey("A single metric", t, func() {
		Convey("with TargetType", func() {
			metric := NewIntWithTargetType("registry/test/1", target.TaskType, "desc", nil)
			var registered types.Metric
			registry.Iter(func(m types.Metric) {
				if reflect.DeepEqual(m.Info(), metric.Info()) {
					registered = m
				}
			})
			So(registered, ShouldNotBeNil)
		})
		Convey("without TargetType", func() {
			metric := NewInt("registry/test/1", "desc", nil)
			var registered types.Metric
			registry.Iter(func(m types.Metric) {
				if reflect.DeepEqual(m.Info(), metric.Info()) {
					registered = m
				}
			})
			So(registered, ShouldNotBeNil)
		})
	})

	Convey("Multiple metrics", t, func() {
		Convey("with the same metric name and targe type", func() {
			NewIntWithTargetType("registry/test/2", target.TaskType, "desc", nil)
			So(func() {
				NewIntWithTargetType("registry/test/2", target.TaskType, "desc", nil)
			}, ShouldPanic)
		})

		Convey("with the same metric name, but different target type", func() {
			mTask := NewIntWithTargetType("registry/test/3", target.TaskType, "desc", nil)
			mDevice := NewIntWithTargetType("registry/test/3", target.DeviceType, "desc", nil)
			mNil := NewInt("registry/test/3", "desc", nil)

			var rTask, rDevice, rNil types.Metric
			registry.Iter(func(m types.Metric) {
				if reflect.DeepEqual(m.Info(), mTask.Info()) {
					rTask = m
				} else if reflect.DeepEqual(m.Info(), mDevice.Info()) {
					rDevice = m
				} else if reflect.DeepEqual(m.Info(), mNil.Info()) {
					rNil = m
				}
			})

			So(rTask, ShouldNotBeNil)
			So(rDevice, ShouldNotBeNil)
			So(rNil, ShouldNotBeNil)
		})
	})
}
