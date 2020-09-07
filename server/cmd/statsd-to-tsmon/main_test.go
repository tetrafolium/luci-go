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

package main

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"

	"github.com/tetrafolium/luci-go/server/cmd/statsd-to-tsmon/config"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEndToEnd(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		cfg, err := loadConfig(&config.Config{
			Metrics: []*config.Metric{
				{
					Metric: "e2e/counter",
					Kind:   config.Kind_COUNTER,
					Fields: []string{"f1", "f2"},
					Rules: []*config.Rule{
						{
							Pattern: "statsd.${f}.counter",
							Fields:  map[string]string{"f1": "static", "f2": "${f}"},
						},
					},
				},
				{
					Metric: "e2e/gauge",
					Kind:   config.Kind_GAUGE,
					Fields: []string{"f1", "f2"},
					Rules: []*config.Rule{
						{
							Pattern: "statsd.${f}.gauge",
							Fields:  map[string]string{"f1": "static", "f2": "${f}"},
						},
					},
				},
				{
					Metric: "e2e/timer",
					Kind:   config.Kind_CUMULATIVE_DISTRIBUTION,
					Fields: []string{"f1", "f2"},
					Rules: []*config.Rule{
						{
							Pattern: "statsd.${f}.timer",
							Fields:  map[string]string{"f1": "static", "f2": "${f}"},
						},
					},
				},
			},
		})
		So(err, ShouldBeNil)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ctx, _ = tsmon.WithDummyInMemory(ctx)
		store := tsmon.Store(ctx)

		// The listening socket.
		pc, err := net.ListenPacket("udp", "localhost:0")
		So(err, ShouldBeNil)
		defer pc.Close()

		// The socket used by the test to send packets.
		con, err := net.Dial("udp", pc.LocalAddr().String())
		So(err, ShouldBeNil)
		defer con.Close()

		// Tick is signaled after each processed UDP packet.
		tick := make(chan struct{})

		// Run mainLoop in background, make sure it is done before we exit.
		done := make(chan struct{})
		go func() {
			defer close(done)
			mainLoop(ctx, pc, cfg, tick)
		}()
		defer func() { <-done }()

		// This must be the last defer, so it is called first to trigger
		// the shutdown of everything else.
		defer cancel()

		// Sends a statsd UDP packet and waits until it is processed.
		send := func(packet string) {
			_, err := con.Write([]byte(packet))
			So(err, ShouldBeNil)
			select {
			case <-tick:
			case <-time.After(5 * time.Second):
				panic("timeout")
			}
		}

		// Send a bunch of metrics.
		send("statsd.a.counter:1|c")
		send("statsd.a.counter:1|c")
		send("statsd.b.counter:1|c")
		send("statsd.a.gauge:123|g")
		send("statsd.a.timer:123|ms")

		// Parsed successfully.
		val := store.Get(ctx, cfg.metrics["e2e/counter"], time.Time{}, []interface{}{"static", "a"})
		So(val, ShouldEqual, 2)
		val = store.Get(ctx, cfg.metrics["e2e/counter"], time.Time{}, []interface{}{"static", "b"})
		So(val, ShouldEqual, 1)
		val = store.Get(ctx, cfg.metrics["e2e/gauge"], time.Time{}, []interface{}{"static", "a"})
		So(val, ShouldEqual, 123)
		val = store.Get(ctx, cfg.metrics["e2e/timer"], time.Time{}, []interface{}{"static", "a"})
		So(val.(*distribution.Distribution).Sum(), ShouldEqual, 123)

		// Updated its own internal metric.
		So(getStatsdMetricsProcessed(ctx), ShouldResemble, map[string]int64{
			"OK": 5,
		})

		// Send a bunch of metrics in a single packet. Intermix some broken metrics.
		send(strings.Join([]string{
			"statsd.a.counter:1|c",
			"broken",
			"stats.unsupported:1|h",
			"statsd.a.counter:1|g", // wrong type
			"statsd.skipped:1|c",   // skipped
			"statsd.b.counter:1|c",
		}, "\n"))

		// Tsmon metrics are updated now.
		val = store.Get(ctx, cfg.metrics["e2e/counter"], time.Time{}, []interface{}{"static", "a"})
		So(val, ShouldEqual, 3)
		val = store.Get(ctx, cfg.metrics["e2e/counter"], time.Time{}, []interface{}{"static", "b"})
		So(val, ShouldEqual, 2)

		// Updated its own internal metric.
		So(getStatsdMetricsProcessed(ctx), ShouldResemble, map[string]int64{
			"OK":          7,
			"MALFORMED":   1,
			"UNSUPPORTED": 1,
			"UNEXPECTED":  1,
			"SKIPPED":     1,
		})
	})
}

func getStatsdMetricsProcessed(ctx context.Context) map[string]int64 {
	out := map[string]int64{}
	store := tsmon.Store(ctx)
	for _, f := range []string{
		"OK",
		"MALFORMED",
		"UNSUPPORTED",
		"UNEXPECTED",
		"SKIPPED",
		"UNKNOWN",
	} {
		val := store.Get(ctx, statsdMetricsProcessed, time.Time{}, []interface{}{f})
		if val != nil {
			out[f] = val.(int64)
		}
	}
	return out
}
