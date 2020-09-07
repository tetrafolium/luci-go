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

package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"google.golang.org/api/pubsub/v1"

	"github.com/tetrafolium/luci-go/gae/filter/featureBreaker"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/gae/service/taskqueue"

	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/appengine/tq/tqtesting"
	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	api "github.com/tetrafolium/luci-go/scheduler/api/scheduler/v1"
	"github.com/tetrafolium/luci-go/scheduler/appengine/acl"
	"github.com/tetrafolium/luci-go/scheduler/appengine/catalog"
	"github.com/tetrafolium/luci-go/scheduler/appengine/engine/cron"
	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
	"github.com/tetrafolium/luci-go/scheduler/appengine/messages"
	"github.com/tetrafolium/luci-go/scheduler/appengine/schedule"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task/noop"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

var (
	aclPublic = acl.GrantsByRole{Readers: []string{"group:all"}, Owners: []string{"group:administrators"}}
	aclSome   = acl.GrantsByRole{Readers: []string{"group:some"}}
	aclOne    = acl.GrantsByRole{Owners: []string{"one@example.com"}}
	aclAdmin  = acl.GrantsByRole{Readers: []string{"group:administrators"}, Owners: []string{"group:administrators"}}

	asUserOne = &authtest.FakeState{
		Identity:       "user:one@example.com",
		IdentityGroups: []string{"all"},
	}
)

func TestGetAllProjects(t *testing.T) {
	t.Parallel()

	Convey("works", t, func() {
		c := newTestContext(epoch)
		e, _ := newTestEngine()

		// Empty.
		projects, err := e.GetAllProjects(c)
		So(err, ShouldBeNil)
		So(len(projects), ShouldEqual, 0)

		// Non empty.
		So(datastore.Put(c,
			&Job{JobID: "abc/1", ProjectID: "abc", Enabled: true},
			&Job{JobID: "abc/2", ProjectID: "abc", Enabled: true},
			&Job{JobID: "def/1", ProjectID: "def", Enabled: true},
			&Job{JobID: "xyz/1", ProjectID: "xyz", Enabled: false},
		), ShouldBeNil)
		datastore.GetTestable(c).CatchupIndexes()
		projects, err = e.GetAllProjects(c)
		So(err, ShouldBeNil)
		So(projects, ShouldResemble, []string{"abc", "def"})
	})
}

func TestUpdateProjectJobs(t *testing.T) {
	t.Parallel()

	Convey("with test context", t, func() {
		c := newTestContext(epoch)
		e, _ := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		jobDef := catalog.Definition{
			JobID:            "proj/1",
			RealmID:          "proj:testing",
			Revision:         "rev1",
			Schedule:         "*/5 * * * * * *",
			Acls:             acl.GrantsByRole{Readers: []string{"group:r"}, Owners: []string{"groups:o"}},
			Task:             []uint8{1, 2, 3}, // note: this is actually gibberish, but we don't care here
			TriggeringPolicy: []uint8{4, 5, 6}, // same
		}

		Convey("noop", func() {
			So(e.UpdateProjectJobs(c, "proj", nil), ShouldBeNil)
			So(allJobs(c), ShouldResemble, []Job{})
		})

		Convey("adding a job", func() {
			// Adding a job that ticks each 5 sec.
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{jobDef}), ShouldBeNil)

			// Added.
			So(allJobs(c), ShouldResemble, []Job{
				{
					JobID:     "proj/1",
					ProjectID: "proj",
					RealmID:   "proj:testing",
					Revision:  "rev1",
					Enabled:   true,
					Acls:      acl.GrantsByRole{Readers: []string{"group:r"}, Owners: []string{"groups:o"}},
					Schedule:  "*/5 * * * * * *",
					Cron: cron.State{
						Enabled:    true,
						Generation: 2,
						LastRewind: epoch,
						LastTick: cron.TickLaterAction{
							When:      epoch.Add(5 * time.Second),
							TickNonce: 6278013164014963328,
						},
					},
					Task:                jobDef.Task,
					TriggeringPolicyRaw: jobDef.TriggeringPolicy,
				},
			})

			// The first tick is scheduled.
			So(tq.GetScheduledTasks().Payloads(), ShouldResembleProto, []proto.Message{
				&internal.CronTickTask{JobId: "proj/1", TickNonce: 6278013164014963328},
			})

			// Again, should be noop.
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{jobDef}), ShouldBeNil)
			So(len(tq.GetScheduledTasks()), ShouldEqual, 1) // no new tasks
		})

		Convey("adding a job with txn retry", func() {
			// Simulate the transaction retry.
			datastore.GetTestable(c).SetTransactionRetryCount(2)
			// Add a job.
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{jobDef}), ShouldBeNil)
			// Added only one task, even though we had 2 retries.
			So(len(tq.GetScheduledTasks()), ShouldEqual, 1)
		})

		Convey("adding a job with txn collision", func() {
			// Simulate the transaction refusing to land even after many tries.
			datastore.GetTestable(c).SetTransactionRetryCount(15)
			// Attempt to add a job.
			err := e.UpdateProjectJobs(c, "proj", []catalog.Definition{jobDef})
			// Failed transiently, nothing in the datastore or in TQ.
			So(transient.Tag.In(err), ShouldBeTrue)
			So(allJobs(c), ShouldResemble, []Job{})
			So(len(tq.GetScheduledTasks()), ShouldEqual, 0)
		})

		Convey("updating job's schedule", func() {
			// Adding a job that ticks every 5 sec. Make sure its tick is scheduled.
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{jobDef}), ShouldBeNil)
			So(tq.GetScheduledTasks().Payloads(), ShouldResembleProto, []proto.Message{
				&internal.CronTickTask{JobId: "proj/1", TickNonce: 6278013164014963328},
			})

			// Changing it to tick every 30 sec.
			newDef := jobDef
			newDef.Schedule = "*/30 * * * * * *"
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{newDef}), ShouldBeNil)

			// The job is updated now.
			So(allJobs(c), ShouldResemble, []Job{
				{
					JobID:     "proj/1",
					ProjectID: "proj",
					RealmID:   "proj:testing",
					Revision:  "rev1",
					Enabled:   true,
					Acls:      acl.GrantsByRole{Readers: []string{"group:r"}, Owners: []string{"groups:o"}},
					Schedule:  "*/30 * * * * * *",
					Cron: cron.State{
						Enabled:    true,
						Generation: 3,
						LastRewind: epoch,
						LastTick: cron.TickLaterAction{
							When:      epoch.Add(30 * time.Second), // new tick time
							TickNonce: 2673062197574995716,
						},
					},
					Task:                jobDef.Task,
					TriggeringPolicyRaw: jobDef.TriggeringPolicy,
				},
			})

			// The new tick is scheduled now too.
			So(tq.GetScheduledTasks().Payloads(), ShouldResembleProto, []proto.Message{
				&internal.CronTickTask{JobId: "proj/1", TickNonce: 6278013164014963328},
				&internal.CronTickTask{JobId: "proj/1", TickNonce: 2673062197574995716},
			})
		})

		Convey("updating job's triggering policy", func() {
			// Adding a job that is triggered externally (doesn't tick). Schedules no
			// tasks.
			job := jobDef
			job.Schedule = "triggered"
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{job}), ShouldBeNil)
			So(tq.GetScheduledTasks().Payloads(), ShouldHaveLength, 0)

			// Update its triggering policy. It should emit a triage to evaluate
			// the state of the job using this new policy.
			job.TriggeringPolicy = []uint8{1, 1, 1, 1}
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{job}), ShouldBeNil)

			// The job is updated now.
			So(allJobs(c), ShouldResemble, []Job{
				{
					JobID:     "proj/1",
					ProjectID: "proj",
					RealmID:   "proj:testing",
					Revision:  "rev1",
					Enabled:   true,
					Acls:      acl.GrantsByRole{Readers: []string{"group:r"}, Owners: []string{"groups:o"}},
					Schedule:  "triggered",
					Cron: cron.State{
						Enabled:    true,
						Generation: 2,
						LastRewind: epoch,
						LastTick: cron.TickLaterAction{
							When:      schedule.DistantFuture,
							TickNonce: 6278013164014963328,
						},
					},
					Task:                jobDef.Task,
					TriggeringPolicyRaw: job.TriggeringPolicy,
				},
			})

			// Kicked the triage indeed.
			So(tq.GetScheduledTasks().Payloads(), ShouldResembleProto, []proto.Message{
				&internal.KickTriageTask{JobId: "proj/1"},
			})
		})

		Convey("removing a job", func() {
			// Adding a job first.
			So(e.UpdateProjectJobs(c, "proj", []catalog.Definition{jobDef}), ShouldBeNil)
			datastore.GetTestable(c).CatchupIndexes()

			// And now removing it.
			So(e.UpdateProjectJobs(c, "proj", nil), ShouldBeNil)

			// Switched to disabled state.
			So(allJobs(c), ShouldResemble, []Job{
				{
					JobID:     "proj/1",
					ProjectID: "proj",
					RealmID:   "proj:testing",
					Revision:  "rev1",
					Enabled:   false,
					Acls:      acl.GrantsByRole{Readers: []string{"group:r"}, Owners: []string{"groups:o"}},
					Schedule:  "*/5 * * * * * *",
					Cron: cron.State{
						Enabled:    false,
						Generation: 3,
					},
					Task:                jobDef.Task,
					TriggeringPolicyRaw: jobDef.TriggeringPolicy,
				},
			})
		})
	})
}

func TestGenerateInvocationID(t *testing.T) {
	t.Parallel()

	Convey("generateInvocationID does not collide", t, func() {
		c := newTestContext(epoch)

		// Bunch of ids generated at the exact same moment in time do not collide.
		ids := map[int64]struct{}{}
		for i := 0; i < 20; i++ {
			id, err := generateInvocationID(c)
			So(err, ShouldBeNil)
			ids[id] = struct{}{}
		}
		So(len(ids), ShouldEqual, 20)
	})

	Convey("generateInvocationID gen IDs with most recent first", t, func() {
		c := newTestContext(epoch)

		older, err := generateInvocationID(c)
		So(err, ShouldBeNil)

		clock.Get(c).(testclock.TestClock).Add(5 * time.Second)

		newer, err := generateInvocationID(c)
		So(err, ShouldBeNil)

		So(newer, ShouldBeLessThan, older)
	})
}

func TestQueries(t *testing.T) {
	t.Parallel()

	Convey("with mock data", t, func() {
		c := newTestContext(epoch)
		e, _ := newTestEngine()

		// TODO(vadimsh): Simplify this test when old ACLs are gone.
		db := authtest.NewFakeDB(
			authtest.MockMembership("user:admin@example.com", adminGroup),
		)

		mockRealm := func(realm string, readers ...string) {
			db.AddMocks(mockEnforceRealmACL(realm))
			for _, r := range readers {
				db.AddMocks(authtest.MockPermission(identity.Identity(r), realm, permJobsGet))
			}
		}

		// Mock jobs.get permission.
		mockRealm("abc:one", "user:one@example.com")
		mockRealm("abc:some", "user:some@example.com")
		mockRealm("abc:public",
			"anonymous:anonymous",
			"user:one@example.com",
			"user:some@example.com",
			"user:admin@example.com",
		)
		mockRealm("abc:admin")

		ctxAnon := auth.WithState(c, &authtest.FakeState{
			Identity: "anonymous:anonymous",
			FakeDB:   db,
		})
		ctxOne := auth.WithState(c, &authtest.FakeState{
			Identity: "user:one@example.com",
			FakeDB:   db,
		})
		ctxSome := auth.WithState(c, &authtest.FakeState{
			Identity: "user:some@example.com",
			FakeDB:   db,
		})
		ctxAdmin := auth.WithState(c, &authtest.FakeState{
			Identity: "user:admin@example.com",
			FakeDB:   db,
		})

		job1 := "abc/1"
		job2 := "abc/2"
		job3 := "abc/3"

		So(datastore.Put(c,
			&Job{JobID: job1, ProjectID: "abc", Enabled: true, RealmID: "abc:one"},
			&Job{JobID: job2, ProjectID: "abc", Enabled: true, RealmID: "abc:some"},
			&Job{JobID: job3, ProjectID: "abc", Enabled: true, RealmID: "abc:public"},
			&Job{JobID: "def/1", ProjectID: "def", Enabled: true, RealmID: "abc:public"},
			&Job{JobID: "def/2", ProjectID: "def", Enabled: false, RealmID: "abc:public"},
			&Job{JobID: "secret/1", ProjectID: "secret", Enabled: true, RealmID: "abc:admin"},
		), ShouldBeNil)

		// Mocked invocations, all in finished state (IndexedJobID set).
		So(datastore.Put(c,
			&Invocation{ID: 1, JobID: job1, IndexedJobID: job1},
			&Invocation{ID: 2, JobID: job1, IndexedJobID: job1},
			&Invocation{ID: 3, JobID: job1, IndexedJobID: job1},
			&Invocation{ID: 4, JobID: job2, IndexedJobID: job2},
			&Invocation{ID: 5, JobID: job2, IndexedJobID: job2},
			&Invocation{ID: 6, JobID: job2, IndexedJobID: job2},
			&Invocation{ID: 7, JobID: job3, IndexedJobID: job3},
		), ShouldBeNil)

		datastore.GetTestable(c).CatchupIndexes()

		Convey("GetAllProjects ignores ACLs and CurrentIdentity", func() {
			test := func(ctx context.Context) {
				r, err := e.GetAllProjects(c)
				So(err, ShouldBeNil)
				So(r, ShouldResemble, []string{"abc", "def", "secret"})
			}
			test(c)
			test(ctxAnon)
			test(ctxAdmin)
		})

		Convey("GetVisibleJobs works", func() {
			get := func(ctx context.Context) []string {
				jobs, err := e.GetVisibleJobs(ctx)
				So(err, ShouldBeNil)
				return sortedJobIds(jobs)
			}

			Convey("Anonymous users see only public jobs", func() {
				// Only 3 jobs with default ACLs granting READER access to everyone, but
				// def/2 is disabled and so shouldn't be returned.
				So(get(ctxAnon), ShouldResemble, []string{"abc/3", "def/1"})
			})
			Convey("Owners can see their own jobs + public jobs", func() {
				// abc/1 is owned by one@example.com.
				So(get(ctxOne), ShouldResemble, []string{"abc/1", "abc/3", "def/1"})
			})
			Convey("Explicit readers", func() {
				So(get(ctxSome), ShouldResemble, []string{"abc/2", "abc/3", "def/1"})
			})
			Convey("Admins have implicit READER access to all jobs", func() {
				So(get(ctxAdmin), ShouldResemble, []string{"abc/1", "abc/2", "abc/3", "def/1", "secret/1"})
			})
		})

		Convey("GetProjectJobsRA works", func() {
			get := func(ctx context.Context, project string) []string {
				jobs, err := e.GetVisibleProjectJobs(ctx, project)
				So(err, ShouldBeNil)
				return sortedJobIds(jobs)
			}
			Convey("Anonymous can still see public jobs", func() {
				So(get(ctxAnon, "def"), ShouldResemble, []string{"def/1"})
			})
			Convey("Admin have implicit READER access to all jobs", func() {
				So(get(ctxAdmin, "abc"), ShouldResemble, []string{"abc/1", "abc/2", "abc/3"})
			})
			Convey("Owners can still see their jobs", func() {
				So(get(ctxOne, "abc"), ShouldResemble, []string{"abc/1", "abc/3"})
			})
			Convey("Readers can see their jobs", func() {
				So(get(ctxSome, "abc"), ShouldResemble, []string{"abc/2", "abc/3"})
			})
		})

		Convey("GetVisibleJob works", func() {
			_, err := e.GetVisibleJob(ctxAdmin, "missing/job")
			So(err, ShouldEqual, ErrNoSuchJob)

			_, err = e.GetVisibleJob(ctxAnon, "abc/1") // no READER permission.
			So(err, ShouldEqual, ErrNoSuchJob)

			_, err = e.GetVisibleJob(ctxAnon, "def/2") // not enabled, hence not visible.
			So(err, ShouldEqual, ErrNoSuchJob)

			job, err := e.GetVisibleJob(ctxAnon, "def/1") // OK.
			So(job, ShouldNotBeNil)
			So(err, ShouldBeNil)
		})

		Convey("ListInvocations works", func() {
			job, err := e.GetVisibleJob(ctxOne, "abc/1")
			So(err, ShouldBeNil)

			Convey("With paging", func() {
				invs, cursor, err := e.ListInvocations(ctxOne, job, ListInvocationsOpts{
					PageSize: 2,
				})
				So(err, ShouldBeNil)
				So(len(invs), ShouldEqual, 2)
				So(invs[0].ID, ShouldEqual, 1)
				So(invs[1].ID, ShouldEqual, 2)
				So(cursor, ShouldNotEqual, "")

				invs, cursor, err = e.ListInvocations(ctxOne, job, ListInvocationsOpts{
					PageSize: 2,
					Cursor:   cursor,
				})
				So(err, ShouldBeNil)
				So(len(invs), ShouldEqual, 1)
				So(invs[0].ID, ShouldEqual, 3)
				So(cursor, ShouldEqual, "")
			})
		})

		Convey("GetInvocation works", func() {
			job, err := e.GetVisibleJob(ctxOne, "abc/1")
			So(err, ShouldBeNil)

			Convey("NoSuchInvocation", func() {
				_, err = e.GetInvocation(ctxOne, job, 666) // Missing invocation.
				So(err, ShouldResemble, ErrNoSuchInvocation)
			})

			Convey("Existing invocation", func() {
				inv, err := e.GetInvocation(ctxOne, job, 1)
				So(inv, ShouldNotBeNil)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestPrepareTopic(t *testing.T) {
	t.Parallel()

	Convey("PrepareTopic works", t, func(ctx C) {
		c := newTestContext(epoch)

		e, _ := newTestEngine()

		pubSubCalls := 0
		e.configureTopic = func(c context.Context, topic, sub, pushURL, publisher string) error {
			pubSubCalls++
			ctx.So(topic, ShouldEqual, fmt.Sprintf("projects/%s/topics/dev-scheduler.noop.some~publisher.com", fakeAppID))
			ctx.So(sub, ShouldEqual, fmt.Sprintf("projects/%s/subscriptions/dev-scheduler.noop.some~publisher.com", fakeAppID))
			ctx.So(pushURL, ShouldEqual, "") // pull on dev server
			ctx.So(publisher, ShouldEqual, "some@publisher.com")
			return nil
		}

		ctl := &taskController{
			ctx:     c,
			eng:     e,
			manager: &noop.TaskManager{},
			saved:   Invocation{ID: 123456},
		}
		ctl.populateState()

		// Once.
		topic, token, err := ctl.PrepareTopic(c, "some@publisher.com")
		So(err, ShouldBeNil)
		So(topic, ShouldEqual, fmt.Sprintf("projects/%s/topics/dev-scheduler.noop.some~publisher.com", fakeAppID))
		So(token, ShouldNotEqual, "")
		So(pubSubCalls, ShouldEqual, 1)

		// Again. 'configureTopic' should not be called anymore.
		_, _, err = ctl.PrepareTopic(c, "some@publisher.com")
		So(err, ShouldBeNil)
		So(pubSubCalls, ShouldEqual, 1)
	})
}

func TestProcessPubSubPush(t *testing.T) {
	t.Parallel()

	Convey("with mock invocation", t, func() {
		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		tc := clock.Get(c).(testclock.TestClock)
		tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
			tc.Add(d)
		})

		So(datastore.Put(c, &Job{
			JobID:     "abc/1",
			ProjectID: "abc",
			Enabled:   true,
		}), ShouldBeNil)

		task, err := proto.Marshal(&messages.TaskDefWrapper{
			Noop: &messages.NoopTask{},
		})
		So(err, ShouldBeNil)

		inv := Invocation{
			ID:    1,
			JobID: "abc/1",
			Task:  task,
		}
		So(datastore.Put(c, &inv), ShouldBeNil)

		// Skip talking to PubSub for real.
		e.configureTopic = func(c context.Context, topic, sub, pushURL, publisher string) error {
			return nil
		}

		ctl, err := controllerForInvocation(c, e, &inv)
		So(err, ShouldBeNil)

		// Grab the working auth token.
		_, token, err := ctl.PrepareTopic(c, "some@publisher.com")
		So(err, ShouldBeNil)
		So(token, ShouldNotEqual, "")

		Convey("ProcessPubSubPush works and retries tq.Retry errors", func() {
			msg := struct {
				Message pubsub.PubsubMessage `json:"message"`
			}{
				Message: pubsub.PubsubMessage{
					Attributes: map[string]string{"auth_token": token},
					Data:       "blah",
				},
			}
			blob, err := json.Marshal(&msg)
			So(err, ShouldBeNil)

			calls := 0
			mgr.handleNotification = func(ctx context.Context, msg *pubsub.PubsubMessage) error {
				So(msg.Data, ShouldEqual, "blah")
				calls++
				if calls == 1 {
					return errors.New("should be retried", tq.Retry)
				}
				return nil
			}
			So(e.ProcessPubSubPush(c, blob), ShouldBeNil)
			So(calls, ShouldEqual, 2) // executed the retry
		})

		Convey("ProcessPubSubPush handles bad token", func() {
			msg := struct {
				Message pubsub.PubsubMessage `json:"message"`
			}{
				Message: pubsub.PubsubMessage{
					Attributes: map[string]string{"auth_token": token + "blah"},
					Data:       "blah",
				},
			}
			blob, err := json.Marshal(&msg)
			So(err, ShouldBeNil)
			So(e.ProcessPubSubPush(c, blob), ShouldErrLike, "bad token")
		})

		Convey("ProcessPubSubPush handles missing invocation", func() {
			datastore.Delete(c, datastore.KeyForObj(c, &inv))
			msg := pubsub.PubsubMessage{
				Attributes: map[string]string{"auth_token": token},
			}
			blob, err := json.Marshal(&msg)
			So(err, ShouldBeNil)
			So(transient.Tag.In(e.ProcessPubSubPush(c, blob)), ShouldBeFalse)
		})
	})
}

func TestTrimDebugLog(t *testing.T) {
	t.Parallel()

	ctx := clock.Set(context.Background(), testclock.New(epoch))
	junk := strings.Repeat("a", 1000)

	genLines := func(start, end int) string {
		inv := Invocation{}
		for i := start; i < end; i++ {
			inv.debugLog(ctx, "Line %d - %s", i, junk)
		}
		return inv.DebugLog
	}

	Convey("small log is not trimmed", t, func() {
		inv := Invocation{
			DebugLog: genLines(0, 100),
		}
		inv.trimDebugLog()
		So(inv.DebugLog, ShouldEqual, genLines(0, 100))
	})

	Convey("huge log is trimmed", t, func() {
		inv := Invocation{
			DebugLog: genLines(0, 500),
		}
		inv.trimDebugLog()
		So(inv.DebugLog, ShouldEqual,
			genLines(0, 94)+"--- the log has been cut here ---\n"+genLines(400, 500))
	})

	Convey("writing lines to huge log and trimming", t, func() {
		inv := Invocation{
			DebugLog: genLines(0, 500),
		}
		inv.trimDebugLog()
		for i := 0; i < 10; i++ {
			inv.debugLog(ctx, "Line %d - %s", i, junk)
			inv.trimDebugLog()
		}
		// Still single cut only. New 10 lines are at the end.
		So(inv.DebugLog, ShouldEqual,
			genLines(0, 94)+"--- the log has been cut here ---\n"+genLines(410, 500)+genLines(0, 10))
	})

	Convey("one huge line", t, func() {
		inv := Invocation{
			DebugLog: strings.Repeat("z", 300000),
		}
		inv.trimDebugLog()
		const msg = "\n--- the log has been cut here ---\n"
		So(inv.DebugLog, ShouldEqual, strings.Repeat("z", debugLogSizeLimit-len(msg))+msg)
	})
}

func TestEnqueueInvocations(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		c := newTestContext(epoch)
		e, _ := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		job := Job{JobID: "project/job"}
		So(datastore.Put(c, &job), ShouldBeNil)

		var invs []*Invocation
		err := runTxn(c, func(c context.Context) error {
			var err error
			invs, err = e.enqueueInvocations(c, &job, []task.Request{
				{TriggeredBy: "user:a@example.com"},
				{TriggeredBy: "user:b@example.com"},
			})
			datastore.Put(c, &job)
			return err
		})
		So(err, ShouldBeNil)

		// The order of new invocations is undefined (including IDs assigned to
		// them), so convert them to map and clear IDs.
		invsByTrigger := map[identity.Identity]Invocation{}
		invIDs := map[int64]bool{}
		for _, inv := range invs {
			invIDs[inv.ID] = true
			cpy := *inv
			cpy.ID = 0
			invsByTrigger[inv.TriggeredBy] = cpy
		}
		So(invsByTrigger, ShouldResemble, map[identity.Identity]Invocation{
			"user:a@example.com": {
				JobID:       "project/job",
				Started:     epoch,
				TriggeredBy: "user:a@example.com",
				Status:      task.StatusStarting,
				DebugLog: "[22:42:00.000] New invocation is queued and will start shortly\n" +
					"[22:42:00.000] Triggered by user:a@example.com\n",
			},
			"user:b@example.com": {
				JobID:       "project/job",
				Started:     epoch,
				TriggeredBy: "user:b@example.com",
				Status:      task.StatusStarting,
				DebugLog: "[22:42:00.000] New invocation is queued and will start shortly\n" +
					"[22:42:00.000] Triggered by user:b@example.com\n",
			},
		})

		// Both invocations are in ActiveInvocations list of the job.
		So(len(job.ActiveInvocations), ShouldEqual, 2)
		for _, invID := range job.ActiveInvocations {
			So(invIDs[invID], ShouldBeTrue)
		}

		// And we've emitted the launch task.
		tasks := tq.GetScheduledTasks()
		So(tasks[0].Payload, ShouldHaveSameTypeAs, &internal.LaunchInvocationsBatchTask{})
		batch := tasks[0].Payload.(*internal.LaunchInvocationsBatchTask)
		So(len(batch.Tasks), ShouldEqual, 2)
		for _, subtask := range batch.Tasks {
			So(subtask.JobId, ShouldEqual, "project/job")
			So(invIDs[subtask.InvId], ShouldBeTrue)
		}
	})
}

func TestTriageTaskDedup(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		c := newTestContext(epoch)
		e, _ := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		Convey("single task", func() {
			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			tasks := tq.GetScheduledTasks()
			So(len(tasks), ShouldEqual, 1)
			So(tasks[0].Task.ETA.Equal(epoch.Add(2*time.Second)), ShouldBeTrue)
			So(tasks[0].Payload, ShouldResemble, &internal.TriageJobStateTask{JobId: "fake/job"})
		})

		Convey("a bunch of tasks, deduplicated by hitting memcache", func() {
			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			clock.Get(c).(testclock.TestClock).Add(time.Second)
			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			clock.Get(c).(testclock.TestClock).Add(900 * time.Millisecond)
			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			tasks := tq.GetScheduledTasks()
			So(len(tasks), ShouldEqual, 1)
			So(tasks[0].Task.ETA.Equal(epoch.Add(2*time.Second)), ShouldBeTrue)
			So(tasks[0].Payload, ShouldResemble, &internal.TriageJobStateTask{JobId: "fake/job"})
		})

		Convey("a bunch of tasks, deduplicated by hitting task queue", func() {
			c, fb := featureBreaker.FilterMC(c, fmt.Errorf("omg, memcache error"))
			fb.BreakFeatures(nil, "GetMulti", "SetMulti")

			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			clock.Get(c).(testclock.TestClock).Add(time.Second)
			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			clock.Get(c).(testclock.TestClock).Add(900 * time.Millisecond)
			So(e.kickTriageNow(c, "fake/job"), ShouldBeNil)

			tasks := tq.GetScheduledTasks()
			So(len(tasks), ShouldEqual, 1)
			So(tasks[0].Task.ETA.Equal(epoch.Add(2*time.Second)), ShouldBeTrue)
			So(tasks[0].Payload, ShouldResemble, &internal.TriageJobStateTask{JobId: "fake/job"})
		})
	})
}

func TestLaunchInvocationTask(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		// Add the job.
		So(e.UpdateProjectJobs(c, "project", []catalog.Definition{
			{
				JobID:    "project/job",
				RealmID:  "project:testing",
				Revision: "rev1",
				Schedule: "triggered",
				Task:     noopTaskBytes(),
				Acls:     aclOne,
			},
		}), ShouldBeNil)

		// Prepare Invocation in Starting state.
		job := Job{JobID: "project/job"}
		So(datastore.Get(c, &job), ShouldBeNil)
		inv, err := e.allocateInvocation(c, &job, task.Request{
			IncomingTriggers: []*internal.Trigger{{Id: "a"}},
		})
		So(err, ShouldBeNil)

		callLaunchInvocation := func(c context.Context, execCount int64) error {
			return tq.ExecuteTask(c, tqtesting.Task{
				Task: &taskqueue.Task{},
				Payload: &internal.LaunchInvocationTask{
					JobId: job.JobID,
					InvId: inv.ID,
				},
			}, &taskqueue.RequestHeaders{TaskExecutionCount: execCount})
		}

		fetchInvocation := func(c context.Context) *Invocation {
			toFetch := Invocation{ID: inv.ID}
			So(datastore.Get(c, &toFetch), ShouldBeNil)
			return &toFetch
		}

		Convey("happy path", func() {
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				So(ctl.InvocationID(), ShouldEqual, inv.ID)
				ctl.DebugLog("Succeeded!")
				ctl.State().Status = task.StatusSucceeded
				return nil
			}
			So(callLaunchInvocation(c, 0), ShouldBeNil)

			updated := fetchInvocation(c)
			triggers, err := updated.IncomingTriggers()
			updated.IncomingTriggersRaw = nil

			So(err, ShouldBeNil)
			So(triggers, ShouldResemble, []*internal.Trigger{{Id: "a"}})
			So(updated, ShouldResemble, &Invocation{
				ID:             inv.ID,
				JobID:          "project/job",
				IndexedJobID:   "project/job",
				Started:        epoch,
				Finished:       epoch,
				Revision:       job.Revision,
				Task:           job.Task,
				Status:         task.StatusSucceeded,
				MutationsCount: 2,
				DebugLog: "[22:42:00.000] New invocation is queued and will start shortly\n" +
					"[22:42:00.000] Starting the invocation (attempt 1)\n" +
					"[22:42:00.000] Succeeded!\n" +
					"[22:42:00.000] Invocation finished in 0s with status SUCCEEDED\n",
			})
		})

		Convey("already aborted", func() {
			inv.Status = task.StatusAborted
			So(datastore.Put(c, inv), ShouldBeNil)
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				return fmt.Errorf("must not be called")
			}
			So(callLaunchInvocation(c, 0), ShouldBeNil)
			So(fetchInvocation(c).Status, ShouldEqual, task.StatusAborted)
		})

		Convey("successful retry", func() {
			// Attempt #1.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				return transient.Tag.Apply(fmt.Errorf("oops, failed to start"))
			}
			So(callLaunchInvocation(c, 0), ShouldEqual, errRetryingLaunch)
			So(fetchInvocation(c).Status, ShouldEqual, task.StatusRetrying)

			// Attempt #2.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				ctl.DebugLog("Succeeded!")
				ctl.State().Status = task.StatusSucceeded
				return nil
			}
			So(callLaunchInvocation(c, 1), ShouldBeNil)

			updated := fetchInvocation(c)
			So(updated.Status, ShouldEqual, task.StatusSucceeded)
			So(updated.RetryCount, ShouldEqual, 1)
			So(updated.DebugLog, ShouldEqual, "[22:42:00.000] New invocation is queued and will start shortly\n"+
				"[22:42:00.000] Starting the invocation (attempt 1)\n"+
				"[22:42:00.000] The invocation will be retried\n"+
				"[22:42:00.000] Starting the invocation (attempt 2)\n"+
				"[22:42:00.000] Succeeded!\n"+
				"[22:42:00.000] Invocation finished in 0s with status SUCCEEDED\n")
		})

		Convey("failed retry", func() {
			// Attempt #1.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				return transient.Tag.Apply(fmt.Errorf("oops, failed to start"))
			}
			So(callLaunchInvocation(c, 0), ShouldEqual, errRetryingLaunch)
			So(fetchInvocation(c).Status, ShouldEqual, task.StatusRetrying)

			// Attempt #2.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				return fmt.Errorf("boom, fatal shot")
			}
			So(callLaunchInvocation(c, 1), ShouldBeNil) // didn't ask for a retry

			updated := fetchInvocation(c)
			So(updated.Status, ShouldEqual, task.StatusFailed)
			So(updated.RetryCount, ShouldEqual, 1)
			So(updated.DebugLog, ShouldEqual, "[22:42:00.000] New invocation is queued and will start shortly\n"+
				"[22:42:00.000] Starting the invocation (attempt 1)\n"+
				"[22:42:00.000] The invocation will be retried\n"+
				"[22:42:00.000] Starting the invocation (attempt 2)\n"+
				"[22:42:00.000] Invocation finished in 0s with status FAILED\n")
		})
	})
}

func TestAbortJob(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		const jobID = "project/job"
		const expectedInvID int64 = 9200093523825193008

		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		asOne := auth.WithState(c, asUserOne)

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		So(e.UpdateProjectJobs(c, "project", []catalog.Definition{
			{
				JobID:    jobID,
				Revision: "rev1",
				Schedule: "triggered",
				Task:     noopTaskBytes(),
				Acls:     aclOne,
			},
		}), ShouldBeNil)

		// Launch a new invocation.
		job, err := e.getJob(c, jobID)
		So(err, ShouldBeNil)
		inv := forceInvocation(c, e, jobID)
		invID := inv.ID
		So(invID, ShouldEqual, expectedInvID)

		Convey("inv aborted before it starts", func() {
			// Kill it right away before it had a chance to start.
			So(e.AbortJob(asOne, job), ShouldBeNil)

			// It is dead right away.
			inv, err := e.getInvocation(c, jobID, invID)
			So(err, ShouldBeNil)
			So(inv, ShouldResemble, &Invocation{
				ID:             expectedInvID,
				JobID:          jobID,
				IndexedJobID:   jobID,
				Started:        epoch,
				Finished:       epoch,
				Revision:       "rev1",
				Task:           noopTaskBytes(),
				Status:         task.StatusAborted,
				MutationsCount: 1,
				DebugLog: "[22:42:00.000] New invocation is queued and will start shortly\n" +
					"[22:42:00.000] Invocation is manually aborted by user:one@example.com\n" +
					"[22:42:00.000] Invocation finished in 0s with status ABORTED\n",
			})

			// Unpause the task queue to confirm the new invocation doesn't actually
			// start.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				panic("must not be called")
				return nil
			}
			tasks, _, err := tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)

			// The sequence of tasks we've just performed.
			So(tasks.Payloads(), ShouldResembleProto, []proto.Message{
				// The delayed triage directly from AbortJob.
				&internal.KickTriageTask{JobId: jobID},
				// The invocation finalization from AbortInvocation.
				&internal.InvocationFinishedTask{JobId: jobID, InvId: expectedInvID},

				// Noop launch. We can't undo the already posted tasks. Note that the
				// actual launch didn't happen, as checked by mgr.launchTask.
				&internal.LaunchInvocationsBatchTask{
					Tasks: []*internal.LaunchInvocationTask{{JobId: jobID, InvId: expectedInvID}},
				},
				&internal.LaunchInvocationTask{JobId: jobID, InvId: expectedInvID},

				// The triage from KickTriageTask and from InvocationFinishedTask
				// finally arrives.
				&internal.TriageJobStateTask{JobId: jobID},
			})

			// The job state is updated (the invocation is no longer active).
			job, err = e.getJob(c, jobID)
			So(err, ShouldBeNil)
			So(job.ActiveInvocations, ShouldBeNil)

			// The invocation is now in the list of finished invocations.
			datastore.GetTestable(c).CatchupIndexes()
			invs, _, _ := e.ListInvocations(asOne, job, ListInvocationsOpts{
				PageSize:     100,
				FinishedOnly: true,
			})
			So(invs, ShouldResemble, []*Invocation{inv})
		})

		Convey("inv aborted while it is running", func() {
			// Let the invocation start and set a timer. Abort the simulation before
			// the timer ticks.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				ctl.State().Status = task.StatusRunning
				ctl.AddTimer(ctx, time.Minute, "1 min", nil)
				return nil
			}
			tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
				ShouldStopBefore: func(t tqtesting.Task) bool {
					_, ok := t.Payload.(*internal.TimerTask)
					return ok
				},
			})
			So(err, ShouldBeNil)

			// The sequence of tasks we've just performed.
			So(tasks.Payloads(), ShouldResembleProto, []proto.Message{
				&internal.LaunchInvocationsBatchTask{
					Tasks: []*internal.LaunchInvocationTask{{JobId: jobID, InvId: expectedInvID}},
				},
				&internal.LaunchInvocationTask{
					JobId: jobID, InvId: expectedInvID,
				},
			})

			// At this point the timer tick is scheduled to happen 1 min from now, but
			// we abort the job.
			mgr.abortTask = func(ctx context.Context, ctl task.Controller) error {
				ctl.DebugLog("Really aborted!")
				return nil
			}
			So(e.AbortJob(asOne, job), ShouldBeNil)

			// It is dead right away.
			inv, err := e.getInvocation(c, jobID, invID)
			So(inv.Status, ShouldEqual, task.StatusAborted)

			// And AbortTask callback was really called.
			So(inv.DebugLog, ShouldContainSubstring, "Really aborted!")

			// Run all processes to completion.
			mgr.handleTimer = func(ctx context.Context, ctl task.Controller, name string, payload []byte) error {
				panic("must not be called")
				return nil
			}
			tasks, _, err = tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)

			// The sequence of tasks we've just performed.
			So(tasks.Payloads(), ShouldResembleProto, []proto.Message{
				// The delayed triage directly from AbortJob.
				&internal.KickTriageTask{JobId: jobID},
				// The invocation finalization from AbortInvocation.
				&internal.InvocationFinishedTask{JobId: jobID, InvId: expectedInvID},

				// The triage from KickTriageTask and from InvocationFinishedTask
				// finally arrives.
				&internal.TriageJobStateTask{JobId: jobID},

				// And delayed TimerTask arrives and gets skipped, as confirmed by
				// mgr.handleTimer.
				&internal.TimerTask{
					JobId: jobID,
					InvId: expectedInvID,
					Timer: &internal.Timer{
						Id:      "project/job:9200093523825193008:1:0",
						Created: google.NewTimestamp(epoch.Add(time.Second)),
						Eta:     google.NewTimestamp(epoch.Add(time.Minute + time.Second)),
						Title:   "1 min",
					},
				},
			})
		})
	})
}

func TestEmitTriggers(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		const testingJob = "project/job"

		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		So(e.UpdateProjectJobs(c, "project", []catalog.Definition{
			{
				JobID:    testingJob,
				Revision: "rev1",
				Schedule: "triggered",
				Task:     noopTaskBytes(),
				Acls:     aclOne, // owned by one@example.com
			},
		}), ShouldBeNil)

		Convey("happy path", func() {
			var incomingTriggers []*internal.Trigger
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				incomingTriggers = ctl.Request().IncomingTriggers
				ctl.State().Status = task.StatusSucceeded
				return nil
			}

			job, err := e.getJob(c, testingJob)
			So(err, ShouldBeNil)

			// Simulate EmitTriggers call from an owner.
			emittedTriggers := []*internal.Trigger{
				{
					Id:           "t1",
					OrderInBatch: 1,
					Payload: &internal.Trigger_Buildbucket{
						Buildbucket: &api.BuildbucketTrigger{
							Tags: []string{"a:b"},
						},
					},
				},
				{
					Id:           "t2",
					OrderInBatch: 2,
				},
			}
			asOne := auth.WithState(c, &authtest.FakeState{Identity: "user:one@example.com"})
			err = e.EmitTriggers(asOne, map[*Job][]*internal.Trigger{job: emittedTriggers})
			So(err, ShouldBeNil)

			// Run TQ until all activities stop.
			tasks, _, err := tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)

			// We expect a triage invoked by EmitTrigger, and one full invocation.
			expect := expectedTasks{JobID: testingJob, Epoch: epoch}
			expect.triage()
			expect.invocationSequence(9200093521727759856)
			expect.triage()
			So(tasks.Payloads(), ShouldResembleProto, expect.Tasks)

			// The task manager received all triggers (though maybe out of order, it
			// is not defined).
			sort.Slice(incomingTriggers, func(i, j int) bool {
				return incomingTriggers[i].Id < incomingTriggers[j].Id
			})
			So(incomingTriggers, ShouldResembleProto, emittedTriggers)
		})

		Convey("no TRIGGERER permission", func() {
			job, err := e.getJob(c, testingJob)
			So(err, ShouldBeNil)

			asTwo := auth.WithState(c, &authtest.FakeState{Identity: "user:two@example.com"})
			err = e.EmitTriggers(asTwo, map[*Job][]*internal.Trigger{
				job: {
					{Id: "t1"},
				},
			})
			So(err, ShouldEqual, ErrNoPermission)
		})

		Convey("paused job ignores triggers", func() {
			job, err := e.getJob(c, testingJob)
			So(err, ShouldBeNil)
			So(e.setJobPausedFlag(c, job, true, ""), ShouldBeNil)

			// The pause emits a triage, get over it now.
			tasks, _, err := tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)
			expect := expectedTasks{JobID: testingJob, Epoch: epoch}
			expect.kickTriage()
			expect.triage()
			So(tasks.Payloads(), ShouldResembleProto, expect.Tasks)

			// Make the RPC, which succeeds.
			asOne := auth.WithState(c, &authtest.FakeState{Identity: "user:one@example.com"})
			err = e.EmitTriggers(asOne, map[*Job][]*internal.Trigger{
				job: {
					{Id: "t1"},
				},
			})
			So(err, ShouldBeNil)

			// But nothing really happens.
			tasks, _, err = tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)
			So(tasks.Payloads(), ShouldHaveLength, 0)
		})
	})
}

func TestOneJobTriggersAnother(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		triggeringJob := "project/triggering-job"
		triggeredJob := "project/triggered-job"

		So(e.UpdateProjectJobs(c, "project", []catalog.Definition{
			{
				JobID:           triggeringJob,
				TriggeredJobIDs: []string{triggeredJob},
				Revision:        "rev1",
				Schedule:        "triggered",
				Task:            noopTaskBytes(),
				Acls:            aclOne,
			},
			{
				JobID:    triggeredJob,
				Revision: "rev1",
				Schedule: "triggered",
				Task:     noopTaskBytes(),
				Acls:     aclOne,
			},
		}), ShouldBeNil)

		Convey("happy path", func() {
			const triggeringInvID int64 = 9200093523824911856
			const triggeredInvID int64 = 9200093521728457040

			// Force launch the triggering job.
			forceInvocation(c, e, triggeringJob)

			// Eventually it runs the task which emits a bunch of triggers, which
			// causes the triggered job triage. We stop right before it and examine
			// what we see.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				ctl.EmitTrigger(ctx, &internal.Trigger{Id: "t1"})
				So(ctl.Save(ctx), ShouldBeNil)
				ctl.EmitTrigger(ctx, &internal.Trigger{Id: "t2"})
				ctl.State().Status = task.StatusSucceeded
				return nil
			}
			tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
				ShouldStopBefore: func(t tqtesting.Task) bool {
					_, ok := t.Payload.(*internal.TriageJobStateTask)
					return ok
				},
			})
			So(err, ShouldBeNil)

			// How these triggers are seen from outside the task.
			expectedTrigger1 := &internal.Trigger{
				Id:           "t1",
				JobId:        triggeringJob,
				InvocationId: triggeringInvID,
				Created:      google.NewTimestamp(epoch.Add(1 * time.Second)),
			}
			expectedTrigger2 := &internal.Trigger{
				Id:           "t2",
				JobId:        triggeringJob,
				InvocationId: triggeringInvID,
				Created:      google.NewTimestamp(epoch.Add(1 * time.Second)),
				OrderInBatch: 1, // second call to EmitTrigger done by the invocation
			}

			// All the tasks we've just executed.
			So(tasks.Payloads(), ShouldResembleProto, []proto.Message{
				// Triggering job begins execution.
				&internal.LaunchInvocationsBatchTask{
					Tasks: []*internal.LaunchInvocationTask{{JobId: triggeringJob, InvId: triggeringInvID}},
				},
				&internal.LaunchInvocationTask{
					JobId: triggeringJob, InvId: triggeringInvID,
				},

				// It emits a trigger in the middle.
				&internal.FanOutTriggersTask{
					JobIds:   []string{triggeredJob},
					Triggers: []*internal.Trigger{expectedTrigger1},
				},
				&internal.EnqueueTriggersTask{
					JobId:    triggeredJob,
					Triggers: []*internal.Trigger{expectedTrigger1},
				},

				// Triggering job finishes execution, emitting another trigger.
				&internal.InvocationFinishedTask{
					JobId: triggeringJob,
					InvId: triggeringInvID,
					Triggers: &internal.FanOutTriggersTask{
						JobIds:   []string{triggeredJob},
						Triggers: []*internal.Trigger{expectedTrigger2},
					},
				},
				&internal.EnqueueTriggersTask{
					JobId:    triggeredJob,
					Triggers: []*internal.Trigger{expectedTrigger2},
				},
			})

			// Triggering invocation has finished (with triggers recorded).
			triggeringInv, err := e.getInvocation(c, triggeringJob, triggeringInvID)
			So(err, ShouldBeNil)
			So(triggeringInv.Status, ShouldEqual, task.StatusSucceeded)
			outgoing, err := triggeringInv.OutgoingTriggers()
			So(err, ShouldBeNil)
			So(outgoing, ShouldResembleProto, []*internal.Trigger{expectedTrigger1, expectedTrigger2})

			// At this point triggered job's triage is about to start. Before it does,
			// verify emitted trigger (sitting in the pending triggers set) is
			// discoverable through ListTriggers.
			tj, _ := e.getJob(c, triggeredJob)
			triggers, err := e.ListTriggers(c, tj)
			So(err, ShouldBeNil)
			So(triggers, ShouldResembleProto, []*internal.Trigger{expectedTrigger1, expectedTrigger2})

			// Resume the simulation to do the triages and start the triggered
			// invocation.
			var seen []*internal.Trigger
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				seen = ctl.Request().IncomingTriggers
				ctl.State().Status = task.StatusSucceeded
				return nil
			}
			tasks, _, err = tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)

			// All the tasks we've just executed.
			So(tasks.Payloads(), ShouldResembleProto, []proto.Message{
				// Triggered job is getting triaged (because pending triggers).
				&internal.TriageJobStateTask{
					JobId: triggeredJob,
				},
				// Triggering job is getting triaged (because it has just finished).
				&internal.TriageJobStateTask{
					JobId: triggeringJob,
				},
				// The triggered job begins execution.
				&internal.LaunchInvocationsBatchTask{
					Tasks: []*internal.LaunchInvocationTask{{JobId: triggeredJob, InvId: triggeredInvID}},
				},
				&internal.LaunchInvocationTask{
					JobId: triggeredJob, InvId: triggeredInvID,
				},
				// ...and finishes. Note that the triage doesn't launch new invocation.
				&internal.InvocationFinishedTask{
					JobId: triggeredJob, InvId: triggeredInvID,
				},
				&internal.TriageJobStateTask{JobId: triggeredJob},
			})

			// Verify LaunchTask callback saw the triggers.
			So(seen, ShouldResembleProto, []*internal.Trigger{expectedTrigger1, expectedTrigger2})

			// And they are recoded in IncomingTriggers set.
			triggeredInv, err := e.getInvocation(c, triggeredJob, triggeredInvID)
			So(err, ShouldBeNil)
			So(triggeredInv.Status, ShouldEqual, task.StatusSucceeded)
			incoming, err := triggeredInv.IncomingTriggers()
			So(err, ShouldBeNil)
			So(incoming, ShouldResembleProto, []*internal.Trigger{expectedTrigger1, expectedTrigger2})
		})
	})
}

func TestInvocationTimers(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		const testJobID = "project/job"
		So(e.UpdateProjectJobs(c, "project", []catalog.Definition{
			{
				JobID:    testJobID,
				Revision: "rev1",
				Schedule: "triggered",
				Task:     noopTaskBytes(),
				Acls:     aclOne,
			},
		}), ShouldBeNil)

		Convey("happy path", func() {
			const testInvID int64 = 9200093523825193008

			// Force launch the job.
			forceInvocation(c, e, testJobID)

			// See handelTimer. Name of the timer => time since epoch.
			callTimes := map[string]time.Duration{}

			// Eventually it runs the task which emits a bunch of timers and then
			// some more, and then stops.
			mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
				ctl.AddTimer(ctx, time.Minute, "1 min", []byte{1})
				ctl.AddTimer(ctx, 2*time.Minute, "2 min", []byte{2})
				ctl.State().Status = task.StatusRunning
				return nil
			}
			mgr.handleTimer = func(ctx context.Context, ctl task.Controller, name string, payload []byte) error {
				callTimes[name] = clock.Now(ctx).Sub(epoch)
				switch name {
				case "1 min": // ignore
				case "2 min":
					// Call us again later.
					ctl.AddTimer(ctx, time.Minute, "stop", []byte{3})
				case "stop":
					ctl.AddTimer(ctx, time.Minute, "ignored-timer", nil)
					ctl.State().Status = task.StatusSucceeded
				}
				return nil
			}
			tasks, _, err := tq.RunSimulation(c, nil)
			So(err, ShouldBeNil)

			timerMsg := func(idSuffix string, created, eta time.Duration, title string, payload []byte) *internal.Timer {
				return &internal.Timer{
					Id:      fmt.Sprintf("%s:%d:%s", testJobID, testInvID, idSuffix),
					Created: google.NewTimestamp(epoch.Add(created)),
					Eta:     google.NewTimestamp(epoch.Add(eta)),
					Title:   title,
					Payload: payload,
				}
			}

			// Individual timers emitted by the test. Note that 1 extra sec comes from
			// the delay added by kickLaunchInvocationsBatchTask.
			timer1 := timerMsg("1:0", time.Second, time.Second+time.Minute, "1 min", []byte{1})
			timer2 := timerMsg("1:1", time.Second, time.Second+2*time.Minute, "2 min", []byte{2})
			timer3 := timerMsg("3:0", time.Second+2*time.Minute, time.Second+3*time.Minute, "stop", []byte{3})

			// All 'handleTimer' ticks happened at expected moments in time.
			So(callTimes, ShouldResemble, map[string]time.Duration{
				"1 min": time.Second + time.Minute,
				"2 min": time.Second + 2*time.Minute,
				"stop":  time.Second + 3*time.Minute,
			})

			// All the tasks we've just executed.
			So(tasks.Payloads(), ShouldResembleProto, []proto.Message{
				// Triggering job begins execution.
				&internal.LaunchInvocationsBatchTask{
					Tasks: []*internal.LaunchInvocationTask{{JobId: testJobID, InvId: testInvID}},
				},
				&internal.LaunchInvocationTask{
					JobId: testJobID, InvId: testInvID,
				},

				// Request to schedule a bunch of timers.
				&internal.ScheduleTimersTask{
					JobId:  testJobID,
					InvId:  testInvID,
					Timers: []*internal.Timer{timer1, timer2},
				},

				// Actual individual timers.
				&internal.TimerTask{
					JobId: testJobID,
					InvId: testInvID,
					Timer: timer1,
				},
				&internal.TimerTask{
					JobId: testJobID,
					InvId: testInvID,
					Timer: timer2,
				},

				// One more, scheduled from handleTimer.
				&internal.TimerTask{
					JobId: testJobID,
					InvId: testInvID,
					Timer: timer3,
				},

				// End of the invocation.
				&internal.InvocationFinishedTask{
					JobId: testJobID, InvId: testInvID,
				},
				&internal.TriageJobStateTask{JobId: testJobID},
			})
		})
	})
}

func TestCron(t *testing.T) {
	t.Parallel()

	Convey("with fake env", t, func() {
		const testJobID = "project/job"

		c := newTestContext(epoch)
		e, mgr := newTestEngine()

		updateJob := func(schedule string) {
			So(e.UpdateProjectJobs(c, "project", []catalog.Definition{
				{
					JobID:    testJobID,
					Revision: "rev1",
					Schedule: schedule,
					Task:     noopTaskBytes(),
					Acls:     aclOne,
				},
			}), ShouldBeNil)
			datastore.GetTestable(c).CatchupIndexes()
		}

		mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
			ctl.State().Status = task.StatusSucceeded
			return nil
		}

		tq := tqtesting.GetTestable(c, e.cfg.Dispatcher)
		tq.CreateQueues()

		Convey("relative schedule", func() {
			updateJob("with 10s interval")

			Convey("happy path", func() {
				// Let the TQ spin for two full cycles.
				tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(30 * time.Second),
				})
				So(err, ShouldBeNil)

				// Collect the list of TQ tasks we expect to be executed.
				expect := expectedTasks{JobID: testJobID, Epoch: epoch}
				// 10 sec after the job is created, a tick comes and emits a trigger.
				expect.cronTickSequence(6278013164014963328, 3, 10*time.Second)
				// It causes an invocation.
				expect.invocationSequence(9200093511241999856)
				expect.triage()
				// 10 sec after it finishes, new tick comes (4 extra seconds are from
				// 2 sec delays induced by 2 triages).
				expect.cronTickSequence(928953616732700780, 6, 24*time.Second)
				// It causes an invocation.
				expect.invocationSequence(9200093496562633040)
				expect.triage()
				// ... and so on

				So(tasks.Payloads(), ShouldResembleProto, expect.Tasks)
			})

			Convey("schedule changes", func() {
				// Let the TQ spin until it hits the task execution.
				tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					ShouldStopBefore: func(t tqtesting.Task) bool {
						_, ok := t.Payload.(*internal.LaunchInvocationTask)
						return ok
					},
				})
				So(err, ShouldBeNil)

				// At this point the job's schedule changes.
				updateJob("with 30s interval")

				// We let the TQ spin some more.
				moreTasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(50 * time.Second),
				})
				So(err, ShouldBeNil)

				// Here's what we expect to execute.
				expect := expectedTasks{JobID: testJobID, Epoch: epoch}
				// 10 sec after the job is created, a tick comes and emits a trigger.
				expect.cronTickSequence(6278013164014963328, 3, 10*time.Second)
				// It causes an invocation. We changed the schedule to 30s after that.
				expect.invocationSequence(9200093511241999856)
				expect.triage()
				// 30 sec after it finishes, new tick comes (4 extra seconds are from
				// 2 sec delays induced by 2 triages).
				expect.cronTickSequence(928953616732700780, 6, 44*time.Second)
				// It causes an invocation.
				expect.invocationSequence(9200093475591113040)
				expect.triage()

				// Got it?
				So(append(tasks, moreTasks...).Payloads(), ShouldResembleProto, expect.Tasks)
			})

			Convey("pause/unpause", func() {
				// Let the TQ spin until it hits the end of task execution.
				tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					ShouldStopAfter: func(t tqtesting.Task) bool {
						_, ok := t.Payload.(*internal.InvocationFinishedTask)
						return ok
					},
				})
				So(err, ShouldBeNil)

				// At this point we pause the job.
				j, err := e.getJob(c, testJobID)
				So(err, ShouldBeNil)
				So(e.setJobPausedFlag(c, j, true, ""), ShouldBeNil)

				// We let the TQ spin some more.
				moreTasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(time.Hour),
				})
				So(err, ShouldBeNil)

				// Here's what we expect to execute.
				expect := expectedTasks{JobID: testJobID, Epoch: epoch}
				// 10 sec after the job is created, a tick comes and emits a trigger.
				expect.cronTickSequence(6278013164014963328, 3, 10*time.Second)
				// It causes an invocation. We pause the job after that.
				expect.invocationSequence(9200093511241999856)
				// The pause kicks the triage, which later executes.
				expect.kickTriage()
				expect.triage()
				// and nothing else happens ...

				// Got it?
				So(append(tasks, moreTasks...).Payloads(), ShouldResembleProto, expect.Tasks)

				// Some time later we unpause the job, it starts again immediately.
				clock.Get(c).(testclock.TestClock).Set(epoch.Add(time.Hour))
				So(e.setJobPausedFlag(c, j, false, ""), ShouldBeNil)

				tasks, _, err = tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(time.Hour + 10*time.Second),
				})
				So(err, ShouldBeNil)

				// Did it?
				expect.clear()
				expect.cronTickSequence(325298467681248558, 7, time.Hour)
				expect.invocationSequence(9200089746854060480)
				expect.triage()
				So(tasks.Payloads(), ShouldResembleProto, expect.Tasks)
			})

			Convey("disabling", func() {
				// Let the TQ spin until it hits the end of task execution.
				tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					ShouldStopAfter: func(t tqtesting.Task) bool {
						_, ok := t.Payload.(*internal.InvocationFinishedTask)
						return ok
					},
				})
				So(err, ShouldBeNil)

				// At this point we disable the job.
				So(e.UpdateProjectJobs(c, "project", nil), ShouldBeNil)

				// We let the TQ spin some more...
				moreTasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(time.Hour),
				})
				So(err, ShouldBeNil)

				// Here's what we expect to execute.
				expect := expectedTasks{JobID: testJobID, Epoch: epoch}
				// 10 sec after the job is created, a tick comes and emits a trigger.
				expect.cronTickSequence(6278013164014963328, 3, 10*time.Second)
				// It causes an invocation. We disable the job after that.
				expect.invocationSequence(9200093511241999856)
				// Disabling kicks the triage, which later executes.
				expect.kickTriage()
				expect.triage()
				// and nothing else happens ...

				// Got it?
				So(append(tasks, moreTasks...).Payloads(), ShouldResembleProto, expect.Tasks)
			})
		})

		Convey("absolute schedule", func() {
			updateJob("5,10 * * * * * *") // on 5th and 10th sec

			Convey("happy path", func() {
				// Let the TQ spin for two full cycles.
				tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(14 * time.Second),
				})
				So(err, ShouldBeNil)

				// Collect the list of TQ tasks we expect to be executed.
				expect := expectedTasks{JobID: testJobID, Epoch: epoch}
				// 5 sec after the job is created, a tick comes and emits a trigger.
				expect.cronTickSequence(6278013164014963328, 3, 5*time.Second)
				// It causes an invocation.
				expect.invocationSequence(9200093517533908688)
				expect.triage()
				// Next tick comes right on schedule, 10 sec after the start.
				expect.cronTickSequence(2673062197574995716, 6, 10*time.Second)
				// It causes an invocation.
				expect.invocationSequence(9200093511241900480)
				expect.triage()
				// ... and so on

				So(tasks.Payloads(), ShouldResembleProto, expect.Tasks)
			})

			Convey("overrun", func() {
				// Currently cron just keeps submitting triggers that ends up in the
				// pending triggers queue if there's some invocation currently running.
				// Once the invocation finishes, the next one start right away (just
				// like with any other kind of trigger). Overruns are not recorded.

				// Simulate "long" task with timers.
				mgr.launchTask = func(ctx context.Context, ctl task.Controller) error {
					ctl.AddTimer(ctx, 6*time.Second, "6 sec", nil)
					ctl.State().Status = task.StatusRunning
					return nil
				}
				mgr.handleTimer = func(ctx context.Context, ctl task.Controller, name string, payload []byte) error {
					ctl.State().Status = task.StatusSucceeded
					return nil
				}

				// Let the TQ spin for two full cycles.
				tasks, _, err := tq.RunSimulation(c, &tqtesting.SimulationParams{
					Deadline: epoch.Add(15 * time.Second),
				})
				So(err, ShouldBeNil)

				// Collect the list of TQ tasks we expect to be executed.
				expect := expectedTasks{JobID: testJobID, Epoch: epoch}
				// 5 sec after the job is created, a tick comes and emits a trigger.
				expect.cronTickSequence(6278013164014963328, 3, 5*time.Second)
				// It causes an invocation to start (and keep running).
				expect.invocationStart(9200093517533908688)
				// The next tick comes right on the schedule, but it doesn't start an
				// invocation yet, just submits a trigger.
				expect.cronTickSequence(2673062197574995716, 6, 10*time.Second)
				// A scheduled invocation timer arrives and finishes the invocation.
				expect.invocationTimer(9200093517533908688, 1, "6 sec", 7*time.Second, 13*time.Second)
				expect.invocationEnd(9200093517533908688)
				expect.triage()
				// The triage detects pending cron trigger and launches a new invocation
				// right away.
				expect.invocationStart(9200093509144748480)

				So(tasks.Payloads(), ShouldResembleProto, expect.Tasks)
			})
		})
	})
}

func noopTaskBytes() []byte {
	buf, _ := proto.Marshal(&messages.TaskDefWrapper{Noop: &messages.NoopTask{}})
	return buf
}

////////////////////////////////////////////////////////////////////////////////

func forceInvocation(c context.Context, e *engineImpl, jobID string) (inv *Invocation) {
	err := e.jobTxn(c, jobID, func(c context.Context, job *Job, isNew bool) (err error) {
		invs, err := e.enqueueInvocations(c, job, []task.Request{
			{},
		})
		if err != nil {
			return err
		}
		inv = invs[0]
		return nil
	})
	if err != nil {
		panic(err)
	}
	return
}

////////////////////////////////////////////////////////////////////////////////

type expectedTasks struct {
	JobID string
	Epoch time.Time
	Tasks []proto.Message
}

func (e *expectedTasks) clear() {
	e.Tasks = nil
}

func (e *expectedTasks) triage() {
	e.Tasks = append(e.Tasks, &internal.TriageJobStateTask{JobId: e.JobID})
}

func (e *expectedTasks) kickTriage() {
	e.Tasks = append(e.Tasks, &internal.KickTriageTask{JobId: e.JobID})
}

func (e *expectedTasks) cronTickSequence(nonce, gen int64, when time.Duration) {
	e.Tasks = append(e.Tasks,
		&internal.CronTickTask{JobId: e.JobID, TickNonce: nonce},
		&internal.EnqueueTriggersTask{
			JobId: e.JobID,
			Triggers: []*internal.Trigger{
				{
					Id:      fmt.Sprintf("cron:v1:%d", gen),
					Created: google.NewTimestamp(e.Epoch.Add(when)),
					Payload: &internal.Trigger_Cron{
						Cron: &api.CronTrigger{Generation: gen},
					},
				},
			},
		},
	)
	e.triage()
}

func (e *expectedTasks) invocationSequence(invID int64) {
	e.invocationStart(invID)
	e.invocationEnd(invID)
}

func (e *expectedTasks) invocationStart(invID int64) {
	e.Tasks = append(e.Tasks,
		&internal.LaunchInvocationsBatchTask{
			Tasks: []*internal.LaunchInvocationTask{
				{JobId: e.JobID, InvId: invID},
			},
		},
		&internal.LaunchInvocationTask{JobId: e.JobID, InvId: invID},
	)
}

func (e *expectedTasks) invocationEnd(invID int64) {
	e.Tasks = append(e.Tasks, &internal.InvocationFinishedTask{JobId: e.JobID, InvId: invID})
}

func (e *expectedTasks) invocationTimer(invID, seq int64, title string, created, eta time.Duration) {
	e.Tasks = append(e.Tasks, &internal.TimerTask{
		JobId: e.JobID,
		InvId: invID,
		Timer: &internal.Timer{
			Id:      fmt.Sprintf("%s:%d:%d:0", e.JobID, invID, seq),
			Title:   title,
			Created: google.NewTimestamp(e.Epoch.Add(created)),
			Eta:     google.NewTimestamp(e.Epoch.Add(eta)),
		},
	})
}
