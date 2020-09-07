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

package bqlog

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	bqapi "google.golang.org/api/bigquery/v2"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/gae/filter/featureBreaker"
	"github.com/tetrafolium/luci-go/gae/service/taskqueue"

	. "github.com/smartystreets/goconvey/convey"
)

var testingLog = Log{
	QueueName:           "pull-queue",
	ProjectID:           "projectID",
	DatasetID:           "datasetID",
	TableID:             "tableID",
	DumpEntriesToLogger: true,
}

type testEntry struct {
	InsertID string
	Data     map[string]bigquery.Value
}

func (e testEntry) Save() (map[string]bigquery.Value, string, error) {
	return e.Data, e.InsertID, nil
}

func TestInsert(t *testing.T) {
	Convey("With mock context", t, func() {
		ctx := gaetesting.TestingContext()
		ctx = mathrand.Set(ctx, rand.New(rand.NewSource(12345)))
		tq := taskqueue.GetTestable(ctx)

		tq.CreatePullQueue("pull-queue")

		Convey("simple insert works", func() {
			err := testingLog.Insert(ctx,
				testEntry{
					InsertID: "abc",
					Data: map[string]bigquery.Value{
						"a": map[string]bigquery.Value{"b": "c"},
					},
				},
				testEntry{
					InsertID: "def",
				})
			So(err, ShouldBeNil)

			tasks := tq.GetScheduledTasks()["pull-queue"]
			So(len(tasks), ShouldEqual, 1)
			var task *taskqueue.Task
			for _, t := range tasks {
				task = t
				break
			}

			decoded := []rawEntry{}
			So(gob.NewDecoder(bytes.NewReader(task.Payload)).Decode(&decoded), ShouldBeNil)
			So(decoded, ShouldResemble, []rawEntry{
				{
					InsertID: "abc",
					// Note that outer bigquery.Value deserializes into bqapi.JsonValue,
					// but the inner one don't. This is fine, since at the end bqapi just
					// encodes the whole thing to JSON and it doesn't matter what alias
					// of interface{} is used for that.
					Data: map[string]bqapi.JsonValue{
						"a": json.RawMessage(`{"b":"c"}`),
					},
				},
				{
					InsertID: "def",
				},
			})
		})

		Convey("null insert works", func() {
			err := testingLog.Insert(ctx)
			So(err, ShouldBeNil)
			tasks := tq.GetScheduledTasks()["pull-queue"]
			So(len(tasks), ShouldEqual, 0)
		})
	})
}

func TestFlush(t *testing.T) {
	Convey("With mock context", t, func() {
		ctx := gaetesting.TestingContext()
		ctx = mathrand.Set(ctx, rand.New(rand.NewSource(12345)))
		ctx, tc := testclock.UseTime(ctx, time.Time{})
		tq := taskqueue.GetTestable(ctx)

		tq.CreatePullQueue("pull-queue")

		Convey("No concurrency, no batches", func() {
			testingLog := testingLog
			testingLog.MaxParallelUploads = 1
			testingLog.BatchesPerRequest = 20

			for i := 0; i < 3; i++ {
				err := testingLog.Insert(ctx, testEntry{
					Data: map[string]bigquery.Value{"i": i},
				})
				So(err, ShouldBeNil)
				tc.Add(time.Millisecond) // emulate passage of time to sort entries
			}

			reqs := []*bqapi.TableDataInsertAllRequest{}
			mockInsertAll(&testingLog, &reqs)

			count, err := testingLog.Flush(ctx)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 3)

			So(len(reqs), ShouldEqual, 1)

			blob, _ := json.MarshalIndent(reqs[0], "", "\t")
			So(string(blob), ShouldEqual, `{
	"rows": [
		{
			"insertId": "bqlog:5119905750835961307:0",
			"json": {
				"i": 0
			}
		},
		{
			"insertId": "bqlog:5119905750835961308:0",
			"json": {
				"i": 1
			}
		},
		{
			"insertId": "bqlog:5119905750835961309:0",
			"json": {
				"i": 2
			}
		}
	],
	"skipInvalidRows": true
}`)

			// Bump time to make sure all pull queue leases (if any) expire.
			tc.Add(time.Hour)
			So(len(tq.GetScheduledTasks()["pull-queue"]), ShouldEqual, 0)

			// Nothing to flush.
			count, err = testingLog.Flush(ctx)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 0)
			So(len(reqs), ShouldEqual, 1)
		})

		Convey("Concurrency and batches", func() {
			testingLog := testingLog
			testingLog.MaxParallelUploads = 5
			testingLog.BatchesPerRequest = 2

			for i := 0; i < 20; i++ {
				err := testingLog.Insert(ctx, testEntry{
					Data: map[string]bigquery.Value{"i": i},
				})
				So(err, ShouldBeNil)
				tc.Add(time.Millisecond) // emulate passage of time to sort entries
			}

			reqs := []*bqapi.TableDataInsertAllRequest{}
			mockInsertAll(&testingLog, &reqs)

			count, err := testingLog.Flush(ctx)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 20)

			So(len(reqs), ShouldEqual, 10)

			// Make sure all data has been sent and insertIDs are all different.
			ints := stringset.New(0)
			ids := stringset.New(0)
			for _, req := range reqs {
				for _, row := range req.Rows {
					ids.Add(row.InsertId)
					ints.Add(string(row.Json["i"].(json.RawMessage)))
				}
			}
			So(ints.Len(), ShouldEqual, 20)
			So(ids.Len(), ShouldEqual, 20)

			// Bump time to make sure all pull queue leases (if any) expire.
			tc.Add(time.Hour)
			So(len(tq.GetScheduledTasks()["pull-queue"]), ShouldEqual, 0)
		})

		Convey("Stops enumerating by timeout", func() {
			testingLog := testingLog
			testingLog.MaxParallelUploads = 1
			testingLog.BatchesPerRequest = 1
			testingLog.FlushTimeout = 5 * time.Second

			for i := 0; i < 10; i++ {
				err := testingLog.Insert(ctx, testEntry{
					Data: map[string]bigquery.Value{"i": i},
				})
				So(err, ShouldBeNil)
				tc.Add(time.Millisecond) // emulate passage of time to sort entries
			}

			// Roll time in same goroutine that fetches tasks from the queue.
			testingLog.beforeSendChunk = func(context.Context, []*taskqueue.Task) {
				tc.Add(time.Second)
			}

			reqs := []*bqapi.TableDataInsertAllRequest{}
			mockInsertAll(&testingLog, &reqs)

			// First batch (until timeout).
			count, err := testingLog.Flush(ctx)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 5)

			// The rest.
			count, err = testingLog.Flush(ctx)
			So(err, ShouldBeNil)
			So(count, ShouldEqual, 5)

			// Total number of requests.
			So(len(reqs), ShouldEqual, 10)

			// Bump time to make sure all pull queue leases (if any) expire.
			tc.Add(time.Hour)
			So(len(tq.GetScheduledTasks()["pull-queue"]), ShouldEqual, 0)
		})

		Convey("Handles fatal bq failure", func() {
			testingLog := testingLog
			testingLog.MaxParallelUploads = 5
			testingLog.BatchesPerRequest = 2

			for i := 0; i < 20; i++ {
				err := testingLog.Insert(ctx, testEntry{
					Data: map[string]bigquery.Value{"i": i},
				})
				So(err, ShouldBeNil)
				tc.Add(time.Millisecond) // emulate passage of time to sort entries
			}

			testingLog.insertMock = func(_ context.Context, r *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error) {
				return nil, fmt.Errorf("omg, error")
			}

			count, err := testingLog.Flush(ctx)
			So(err.Error(), ShouldEqual, "omg, error (and 9 other errors)")
			So(count, ShouldEqual, 0)

			// Bump time to make sure all pull queue leases (if any) expire. On fatal
			// errors, we drop the data.
			tc.Add(time.Hour)
			So(len(tq.GetScheduledTasks()["pull-queue"]), ShouldEqual, 0)
		})

		// TODO(vadimsh): This test is flaky.
		SkipConvey("Handles transient bq failure", func() {
			testingLog := testingLog
			testingLog.MaxParallelUploads = 1
			testingLog.BatchesPerRequest = 2

			for i := 0; i < 20; i++ {
				err := testingLog.Insert(ctx, testEntry{
					Data: map[string]bigquery.Value{"i": i},
				})
				So(err, ShouldBeNil)
				tc.Add(time.Millisecond) // emulate passage of time to sort entries
			}

			testingLog.insertMock = func(_ context.Context, r *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error) {
				return nil, errors.New("omg, transient error", transient.Tag)
			}

			tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
				if testclock.HasTags(t, "insert-retry") {
					tc.Add(d)
				}
			})

			count, err := testingLog.Flush(ctx)
			So(err.Error(), ShouldEqual, "omg, transient error (and 2 other errors)")
			So(count, ShouldEqual, 0)

			// Bump time to make sure all pull queue leases (if any) expire. On
			// transient error we keep the data.
			tc.Add(time.Hour)
			So(len(tq.GetScheduledTasks()["pull-queue"]), ShouldEqual, 20)
		})

		Convey("Handles Lease failure", func() {
			testingLog := testingLog
			testingLog.MaxParallelUploads = 5
			testingLog.BatchesPerRequest = 2

			for i := 0; i < 20; i++ {
				err := testingLog.Insert(ctx, testEntry{
					Data: map[string]bigquery.Value{"i": i},
				})
				So(err, ShouldBeNil)
				tc.Add(time.Millisecond) // emulate passage of time to sort entries
			}

			testingLog.insertMock = func(_ context.Context, r *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error) {
				panic("must not be called")
			}

			ctx, fb := featureBreaker.FilterTQ(ctx, nil)
			fb.BreakFeatures(fmt.Errorf("lease error"), "Lease")
			tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
				if testclock.HasTags(t, "lease-retry") {
					tc.Add(d)
				}
			})

			count, err := testingLog.Flush(ctx)
			So(err.Error(), ShouldEqual, "lease error")
			So(count, ShouldEqual, 0)

			// Bump time to make sure all pull queue leases (if any) expire.
			tc.Add(time.Hour)
			So(len(tq.GetScheduledTasks()["pull-queue"]), ShouldEqual, 20)
		})
	})
}

func mockInsertAll(l *Log, reqs *[]*bqapi.TableDataInsertAllRequest) {
	lock := sync.Mutex{}
	l.insertMock = func(ctx context.Context, r *bqapi.TableDataInsertAllRequest) (*bqapi.TableDataInsertAllResponse, error) {
		lock.Lock()
		defer lock.Unlock()
		*reqs = append(*reqs, r)
		return &bqapi.TableDataInsertAllResponse{}, nil
	}
}
