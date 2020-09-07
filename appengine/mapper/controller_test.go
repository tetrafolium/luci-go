// Copyright 2018 The LUCI Authors.
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

package mapper

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/retry/transient"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/appengine/tq/tqtesting"

	"github.com/tetrafolium/luci-go/appengine/mapper/internal/tasks"
	"github.com/tetrafolium/luci-go/appengine/mapper/splitter"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

var (
	testTime        = testclock.TestRecentTimeUTC.Round(time.Millisecond)
	testTimeAsProto = google.NewTimestamp(testTime)
)

type intEnt struct {
	ID int64 `gae:"$id"`
}

func TestController(t *testing.T) {
	t.Parallel()

	Convey("With controller", t, func() {
		ctx := gaetesting.TestingContext()
		ctx, _ = testclock.UseTime(ctx, testTime)

		dispatcher := &tq.Dispatcher{}

		ctl := Controller{
			MapperQueue:  "mapper-queue",
			ControlQueue: "control-queue",
		}
		ctl.Install(dispatcher)

		// mapperFunc is set by test cases.
		var mapperFunc func(params []byte, shardIdx int, keys []*datastore.Key) error

		const testMapperID ID = "test-mapper"
		ctl.RegisterFactory(testMapperID, func(_ context.Context, j *Job, idx int) (Mapper, error) {
			return func(_ context.Context, keys []*datastore.Key) error {
				if mapperFunc == nil {
					return nil
				}
				return mapperFunc(j.Config.Params, idx, keys)
			}, nil
		})

		tqt := tqtesting.GetTestable(ctx, dispatcher)
		tqt.CreateQueues()

		spinUntilDone := func(expectErrors bool) (executed []proto.Message) {
			for {
				tasks, _, err := tqt.RunSimulation(ctx, nil)
				executed = append(executed, tasks.Payloads()...)
				if err == nil {
					return
				}
				if !expectErrors {
					So(err, ShouldBeNil)
				}
			}
		}

		// Create a bunch of entities to run the mapper over.
		entities := make([]intEnt, 512)
		So(datastore.Put(ctx, entities), ShouldBeNil)
		datastore.GetTestable(ctx).CatchupIndexes()

		Convey("LaunchJob works", func() {
			cfg := JobConfig{
				Query: Query{
					Kind: "intEnt",
				},
				Mapper:        testMapperID,
				Params:        []byte("zzz"),
				ShardCount:    4,
				PageSize:      33, // make it weird to trigger "incomplete" pages
				PagesPerTask:  2,  // to trigger multiple mapping tasks in a chain
				TrackProgress: true,
			}

			// Before we start, there' no job with ID 1.
			j, err := ctl.GetJob(ctx, 1)
			So(err, ShouldEqual, ErrNoSuchJob)
			So(j, ShouldBeNil)

			jobID, err := ctl.LaunchJob(ctx, &cfg)
			So(err, ShouldBeNil)
			So(jobID, ShouldEqual, 1)

			// In "starting" state.
			job, err := ctl.GetJob(ctx, jobID)
			So(err, ShouldBeNil)
			So(job, ShouldResemble, &Job{
				ID:      jobID,
				Config:  cfg,
				State:   State_STARTING,
				Created: testTime,
				Updated: testTime,
			})

			// No shards in the info yet.
			info, err := job.FetchInfo(ctx)
			So(err, ShouldBeNil)
			So(info, ShouldResemble, &JobInfo{
				Id:            int64(jobID),
				State:         State_STARTING,
				Created:       testTimeAsProto,
				Updated:       testTimeAsProto,
				TotalEntities: -1,
			})

			// Roll TQ forward.
			_, _, err = tqt.RunSimulation(ctx, &tqtesting.SimulationParams{
				ShouldStopBefore: func(t tqtesting.Task) bool {
					_, yep := t.Payload.(*tasks.FanOutShards)
					return yep
				},
			})
			So(err, ShouldBeNil)

			// Switched into "running" state.
			job, err = ctl.GetJob(ctx, jobID)
			So(err, ShouldBeNil)
			So(job.State, ShouldEqual, State_RUNNING)

			expectedShard := func(id int64, idx int, l, r, expected int64) shard {
				rng := splitter.Range{}
				if l != -1 {
					rng.Start = datastore.KeyForObj(ctx, &intEnt{ID: l})
				}
				if r != -1 {
					rng.End = datastore.KeyForObj(ctx, &intEnt{ID: r})
				}
				return shard{
					ID:            id,
					JobID:         jobID,
					Index:         idx,
					State:         State_STARTING,
					Range:         rng,
					ExpectedCount: expected,
					Created:       testTime,
					Updated:       testTime,
				}
			}

			// Created the shard entities.
			shards, err := job.fetchShards(ctx)
			So(err, ShouldBeNil)
			So(shards, ShouldResemble, []shard{
				expectedShard(1, 0, -1, 136, 136),
				expectedShard(2, 1, 136, 268, 132),
				expectedShard(3, 2, 268, 399, 131),
				expectedShard(4, 3, 399, -1, 113),
			})

			// Shards also appear in the info now.
			info, err = job.FetchInfo(ctx)
			So(err, ShouldBeNil)

			expectedShardInfo := func(idx, total int) *ShardInfo {
				return &ShardInfo{
					Index:         int32(idx),
					State:         State_STARTING,
					Created:       testTimeAsProto,
					Updated:       testTimeAsProto,
					TotalEntities: int64(total),
				}
			}
			So(info, ShouldResemble, &JobInfo{
				Id:            int64(jobID),
				State:         State_RUNNING,
				Created:       testTimeAsProto,
				Updated:       testTimeAsProto,
				TotalEntities: 512,
				Shards: []*ShardInfo{
					expectedShardInfo(0, 136),
					expectedShardInfo(1, 132),
					expectedShardInfo(2, 131),
					expectedShardInfo(3, 113),
				},
			})

			visitShards := func(cb func(s shard)) {
				visitedShards, err := job.fetchShards(ctx)
				So(err, ShouldBeNil)
				So(visitedShards, ShouldHaveLength, cfg.ShardCount)
				for _, s := range visitedShards {
					cb(s)
				}
			}

			seen := make(map[int64]struct{}, len(entities))

			updateSeen := func(keys []*datastore.Key) {
				for _, k := range keys {
					_, ok := seen[k.IntID()]
					So(ok, ShouldBeFalse)
					seen[k.IntID()] = struct{}{}
				}
			}

			assertAllSeen := func() {
				So(len(seen), ShouldEqual, len(entities))
				for _, e := range entities {
					_, ok := seen[e.ID]
					So(ok, ShouldBeTrue)
				}
			}

			Convey("No errors when processing shards", func() {
				mapperFunc = func(params []byte, shardIdx int, keys []*datastore.Key) error {
					So(len(keys), ShouldBeLessThanOrEqualTo, cfg.PageSize)
					So(params, ShouldResemble, cfg.Params)
					updateSeen(keys)
					return nil
				}

				spinUntilDone(false)

				visitShards(func(s shard) {
					So(s.State, ShouldEqual, State_SUCCESS)
					So(s.ProcessTaskNum, ShouldEqual, 2)
					So(s.ProcessedCount, ShouldEqual, []int64{
						136, 132, 131, 113,
					}[s.Index])
				})

				assertAllSeen()

				job, err = ctl.GetJob(ctx, jobID)
				So(err, ShouldBeNil)
				So(job.State, ShouldEqual, State_SUCCESS)

				info, err := job.FetchInfo(ctx)
				So(err, ShouldBeNil)

				expectedShardInfo := func(idx, total int) *ShardInfo {
					return &ShardInfo{
						Index:             int32(idx),
						State:             State_SUCCESS,
						Created:           testTimeAsProto,
						Updated:           testTimeAsProto,
						TotalEntities:     int64(total),
						ProcessedEntities: int64(total),
					}
				}
				So(info, ShouldResemble, &JobInfo{
					Id:      int64(jobID),
					State:   State_SUCCESS,
					Created: testTimeAsProto,
					// There's 2 sec delay before UpdateJobState task.
					Updated:           google.NewTimestamp(testTime.Add(2 * time.Second)),
					TotalEntities:     512,
					ProcessedEntities: 512,
					EntitiesPerSec:    256,
					Shards: []*ShardInfo{
						expectedShardInfo(0, 136),
						expectedShardInfo(1, 132),
						expectedShardInfo(2, 131),
						expectedShardInfo(3, 113),
					},
				})
			})

			Convey("One shard fails", func() {
				page := 0
				processed := 0

				mapperFunc = func(_ []byte, shardIdx int, keys []*datastore.Key) error {
					if shardIdx == 1 {
						page++
						if page == 2 {
							return errors.New("boom")
						}
					}
					processed += len(keys)
					return nil
				}

				spinUntilDone(true)

				visitShards(func(s shard) {
					if s.Index == 1 {
						So(s.State, ShouldEqual, State_FAIL)
						So(s.Error, ShouldEqual, `while mapping 33 keys: boom`)
					} else {
						So(s.State, ShouldEqual, State_SUCCESS)
						So(s.ProcessTaskNum, ShouldEqual, 2)
					}
					So(s.ProcessedCount, ShouldEqual, []int64{
						136, 33, 131, 113, // the failed shard is incomplete
					}[s.Index])
				})

				// There are 5 pages per shard. We aborted on second. So 3 are skipped.
				So(processed, ShouldEqual, len(entities)-3*cfg.PageSize)

				job, err = ctl.GetJob(ctx, jobID)
				So(err, ShouldBeNil)
				So(job.State, ShouldEqual, State_FAIL)
			})

			Convey("Job aborted midway", func() {
				processed := 0

				mapperFunc = func(_ []byte, shardIdx int, keys []*datastore.Key) error {
					processed += len(keys)

					job, err = ctl.AbortJob(ctx, jobID)
					So(err, ShouldBeNil)
					So(job.State, ShouldEqual, State_ABORTING)

					return nil
				}

				spinUntilDone(false)

				// All shards eventually discovered that the job was aborted.
				visitShards(func(s shard) {
					So(s.State, ShouldEqual, State_ABORTED)
					if s.Index == 0 {
						// Zeroth shard did manage to run for a bit.
						So(s.ProcessedCount, ShouldEqual, 66)
					} else {
						So(s.ProcessedCount, ShouldEqual, 0)
					}
				})

				// And the job itself eventually switched into ABORTED state.
				job, err = ctl.GetJob(ctx, jobID)
				So(err, ShouldBeNil)
				So(job.State, ShouldEqual, State_ABORTED)

				// Processed 2 pages (instead of 1), since processShardHandler doesn't
				// check job state inside the processing loop (only at the beginning).
				So(processed, ShouldEqual, 2*cfg.PageSize)
			})

			Convey("processShardHandler saves state on transient errors", func() {
				pages := 0

				mapperFunc = func(_ []byte, shardIdx int, keys []*datastore.Key) error {
					pages++
					if pages == 2 {
						return errors.New("boom", transient.Tag)
					}
					return nil
				}

				err := ctl.processShardHandler(ctx, &tasks.ProcessShard{
					JobId:   int64(job.ID),
					ShardId: shards[0].ID,
				})
				So(transient.Tag.In(err), ShouldBeTrue)

				// Shard's resume point is updated. Its taskNum is left unchanged, since
				// we are going to retry the task.
				sh, err := getActiveShard(ctx, shards[0].ID, shards[0].ProcessTaskNum)
				So(err, ShouldBeNil)
				So(sh.ResumeFrom, ShouldNotBeNil)
				So(sh.ProcessedCount, ShouldEqual, 33)
			})
		})

		Convey("With simple starting job", func() {
			cfg := JobConfig{
				Query:      Query{Kind: "intEnt"},
				Mapper:     testMapperID,
				ShardCount: 4,
				PageSize:   64,
			}

			jobID, err := ctl.LaunchJob(ctx, &cfg)
			So(err, ShouldBeNil)
			So(jobID, ShouldEqual, 1)

			// In "starting" state initially.
			job, err := ctl.GetJob(ctx, jobID)
			So(err, ShouldBeNil)
			So(job.State, ShouldEqual, State_STARTING)

			Convey("Abort right after start", func() {
				job, err := ctl.AbortJob(ctx, jobID)
				So(err, ShouldBeNil)
				So(job.State, ShouldEqual, State_ABORTED) // aborted right away

				// Didn't actually launch any shards.
				So(spinUntilDone(false), ShouldResembleProto, []proto.Message{
					&tasks.SplitAndLaunch{JobId: int64(jobID)},
				})
			})

			Convey("Abort after shards are created", func() {
				// Stop right after we created the shards, before we launch them.
				_, _, err = tqt.RunSimulation(ctx, &tqtesting.SimulationParams{
					ShouldStopBefore: func(t tqtesting.Task) bool {
						_, yep := t.Payload.(*tasks.FanOutShards)
						return yep
					},
				})
				So(err, ShouldBeNil)

				job, err := ctl.AbortJob(ctx, jobID)
				So(err, ShouldBeNil)
				So(job.State, ShouldEqual, State_ABORTING) // waits for shards to die

				spinUntilDone(false)

				job, err = ctl.AbortJob(ctx, jobID)
				So(err, ShouldBeNil)
				So(job.State, ShouldEqual, State_ABORTED) // all shards are dead now

				// Dead indeed.
				info, err := job.FetchInfo(ctx)
				So(err, ShouldBeNil)
				So(info.Shards, ShouldHaveLength, 4)
				for _, s := range info.Shards {
					So(s.State, ShouldEqual, State_ABORTED)
				}
			})
		})
	})
}
