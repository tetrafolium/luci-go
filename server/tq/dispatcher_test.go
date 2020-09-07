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

package tq

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/server/router"

	"github.com/tetrafolium/luci-go/server/tq/tqtesting"

	"github.com/tetrafolium/luci-go/server/tq/internal/reminder"
	"github.com/tetrafolium/luci-go/server/tq/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestAddTask(t *testing.T) {
	t.Parallel()

	Convey("With dispatcher", t, func() {
		var now = time.Unix(1442540000, 0)

		ctx, _ := testclock.UseTime(context.Background(), now)
		submitter := &submitter{}
		ctx = UseSubmitter(ctx, submitter)

		d := Dispatcher{
			CloudProject:      "proj",
			CloudRegion:       "reg",
			DefaultTargetHost: "example.com",
			PushAs:            "push-as@example.com",
		}

		d.RegisterTaskClass(TaskClass{
			ID:        "test-dur",
			Prototype: &durationpb.Duration{}, // just some proto type
			Queue:     "queue-1",
		})

		task := &Task{
			Payload: durationpb.New(10 * time.Second),
			Title:   "hi",
			Delay:   123 * time.Second,
		}
		expectedPayload := []byte(`{
	"class": "test-dur",
	"type": "google.protobuf.Duration",
	"body": "10s"
}`)

		Convey("Nameless HTTP task", func() {
			So(d.AddTask(ctx, task), ShouldBeNil)

			So(submitter.reqs, ShouldHaveLength, 1)
			So(submitter.reqs[0].CreateTaskRequest, ShouldResembleProto, &taskspb.CreateTaskRequest{
				Parent: "projects/proj/locations/reg/queues/queue-1",
				Task: &taskspb.Task{
					ScheduleTime: timestamppb.New(now.Add(123 * time.Second)),
					MessageType: &taskspb.Task_HttpRequest{
						HttpRequest: &taskspb.HttpRequest{
							HttpMethod: taskspb.HttpMethod_POST,
							Url:        "https://example.com/internal/tasks/t/test-dur/hi",
							Headers:    defaultHeaders(),
							Body:       expectedPayload,
							AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
								OidcToken: &taskspb.OidcToken{
									ServiceAccountEmail: "push-as@example.com",
								},
							},
						},
					},
				},
			})
		})

		Convey("Nameless GAE task", func() {
			d.GAE = true
			d.DefaultTargetHost = ""
			So(d.AddTask(ctx, task), ShouldBeNil)

			So(submitter.reqs, ShouldHaveLength, 1)
			So(submitter.reqs[0].CreateTaskRequest, ShouldResembleProto, &taskspb.CreateTaskRequest{
				Parent: "projects/proj/locations/reg/queues/queue-1",
				Task: &taskspb.Task{
					ScheduleTime: timestamppb.New(now.Add(123 * time.Second)),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_POST,
							RelativeUri: "/internal/tasks/t/test-dur/hi",
							Headers:     defaultHeaders(),
							Body:        expectedPayload,
						},
					},
				},
			})
		})

		Convey("Named task", func() {
			task.DeduplicationKey = "key"

			So(d.AddTask(ctx, task), ShouldBeNil)

			So(submitter.reqs, ShouldHaveLength, 1)
			So(submitter.reqs[0].CreateTaskRequest.Task.Name, ShouldEqual,
				"projects/proj/locations/reg/queues/queue-1/tasks/"+
					"ca0a124846df4b453ae63e3ad7c63073b0d25941c6e63e5708fd590c016edcef")
		})

		Convey("Titleless task", func() {
			task.Title = ""

			So(d.AddTask(ctx, task), ShouldBeNil)

			So(submitter.reqs, ShouldHaveLength, 1)
			So(
				submitter.reqs[0].CreateTaskRequest.Task.MessageType.(*taskspb.Task_HttpRequest).HttpRequest.Url,
				ShouldEqual,
				"https://example.com/internal/tasks/t/test-dur",
			)
		})

		Convey("Transient err", func() {
			submitter.err = func(title string) error {
				return status.Errorf(codes.Internal, "boo, go away")
			}
			err := d.AddTask(ctx, task)
			So(transient.Tag.In(err), ShouldBeTrue)
		})

		Convey("Fatal err", func() {
			submitter.err = func(title string) error {
				return status.Errorf(codes.PermissionDenied, "boo, go away")
			}
			err := d.AddTask(ctx, task)
			So(err, ShouldNotBeNil)
			So(transient.Tag.In(err), ShouldBeFalse)
		})

		Convey("Unknown payload type", func() {
			err := d.AddTask(ctx, &Task{
				Payload: &timestamppb.Timestamp{},
			})
			So(err, ShouldErrLike, "no task class matching type")
			So(submitter.reqs, ShouldHaveLength, 0)
		})

		Convey("Custom task payload on GAE", func() {
			d.GAE = true
			d.DefaultTargetHost = ""
			d.RegisterTaskClass(TaskClass{
				ID:        "test-ts",
				Prototype: &timestamppb.Timestamp{}, // just some proto type
				Queue:     "queue-1",
				Custom: func(ctx context.Context, m proto.Message) (*CustomPayload, error) {
					ts := m.(*timestamppb.Timestamp)
					return &CustomPayload{
						Method:      "GET",
						Meta:        map[string]string{"k": "v"},
						RelativeURI: "/zzz",
						Body:        []byte(fmt.Sprintf("%d", ts.Seconds)),
					}, nil
				},
			})

			So(d.AddTask(ctx, &Task{
				Payload: &timestamppb.Timestamp{Seconds: 123},
				Delay:   444 * time.Second,
			}), ShouldBeNil)

			So(submitter.reqs, ShouldHaveLength, 1)
			So(submitter.reqs[0].CreateTaskRequest, ShouldResembleProto, &taskspb.CreateTaskRequest{
				Parent: "projects/proj/locations/reg/queues/queue-1",
				Task: &taskspb.Task{
					ScheduleTime: timestamppb.New(now.Add(444 * time.Second)),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: "/zzz",
							Headers:     map[string]string{"k": "v"},
							Body:        []byte("123"),
						},
					},
				},
			})
		})
	})
}

func TestPushHandler(t *testing.T) {
	t.Parallel()

	Convey("With dispatcher", t, func() {
		var handlerErr error

		d := Dispatcher{NoAuth: true}
		ref := d.RegisterTaskClass(TaskClass{
			ID:        "test-1",
			Prototype: &emptypb.Empty{},
			Queue:     "queue",
			Handler: func(ctx context.Context, payload proto.Message) error {
				return handlerErr
			},
		})

		srv := router.New()
		d.InstallTasksRoutes(srv, "/pfx")

		call := func(body string) int {
			req := httptest.NewRequest("POST", "/pfx/ignored/part", strings.NewReader(body))
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			return rec.Result().StatusCode
		}

		Convey("Using class ID", func() {
			Convey("Success", func() {
				So(call(`{"class": "test-1", "body": {}}`), ShouldEqual, 200)
			})
			Convey("Unknown", func() {
				So(call(`{"class": "unknown", "body": {}}`), ShouldEqual, 202)
			})
		})

		Convey("Using type name", func() {
			Convey("Success", func() {
				So(call(`{"type": "google.protobuf.Empty", "body": {}}`), ShouldEqual, 200)
			})
			Convey("Totally unknown", func() {
				So(call(`{"type": "unknown", "body": {}}`), ShouldEqual, 202)
			})
			Convey("Not a registered task", func() {
				So(call(`{"type": "google.protobuf.Duration", "body": {}}`), ShouldEqual, 202)
			})
		})

		Convey("Not a JSON body", func() {
			So(call(`blarg`), ShouldEqual, 202)
		})

		Convey("Bad envelope", func() {
			So(call(`{}`), ShouldEqual, 202)
		})

		Convey("Missing message body", func() {
			So(call(`{"class": "test-1"}`), ShouldEqual, 202)
		})

		Convey("Bad message body", func() {
			So(call(`{"class": "test-1", "body": "huh"}`), ShouldEqual, 202)
		})

		Convey("Handler asks for retry", func() {
			handlerErr = errors.New("boo", Retry)
			So(call(`{"class": "test-1", "body": {}}`), ShouldEqual, 429)
		})

		Convey("Handler transient error", func() {
			handlerErr = errors.New("boo", transient.Tag)
			So(call(`{"class": "test-1", "body": {}}`), ShouldEqual, 500)
		})

		Convey("Handler fatal error", func() {
			handlerErr = errors.New("boo")
			So(call(`{"class": "test-1", "body": {}}`), ShouldEqual, 202)
		})

		Convey("No handler", func() {
			ref.(*taskClassImpl).Handler = nil
			So(call(`{"class": "test-1", "body": {}}`), ShouldEqual, 202)
		})
	})
}

func TestTransactionalEnqueue(t *testing.T) {
	t.Parallel()

	Convey("With mocks", t, func() {
		var now = time.Unix(1442540000, 0)

		submitter := &submitter{}
		db := testutil.FakeDB{}
		d := Dispatcher{
			CloudProject:      "proj",
			CloudRegion:       "reg",
			DefaultTargetHost: "example.com",
			PushAs:            "push-as@example.com",
		}
		d.RegisterTaskClass(TaskClass{
			ID:        "test-dur",
			Prototype: &durationpb.Duration{}, // just some proto type
			Kind:      Transactional,
			Queue:     "queue-1",
		})

		ctx, tc := testclock.UseTime(context.Background(), now)
		ctx = UseSubmitter(ctx, submitter)
		txn := db.Inject(ctx)

		Convey("Happy path", func() {
			task := &Task{
				Payload: durationpb.New(5 * time.Second),
				Delay:   10 * time.Second,
			}
			err := d.AddTask(txn, task)
			So(err, ShouldBeNil)

			// Created the reminder.
			So(db.AllReminders(), ShouldHaveLength, 1)
			rem := db.AllReminders()[0]

			// But didn't submitted the task yet.
			So(submitter.reqs, ShouldBeEmpty)

			// The defer will submit the task and wipe the reminder.
			db.ExecDefers(ctx)
			So(db.AllReminders(), ShouldBeEmpty)
			So(submitter.reqs, ShouldHaveLength, 1)
			req := submitter.reqs[0]

			// Make sure the reminder and the task look as expected.
			So(rem.ID, ShouldHaveLength, reminderKeySpaceBytes*2)
			So(rem.FreshUntil.Equal(now.Add(happyPathMaxDuration)), ShouldBeTrue)
			So(req.TaskClass, ShouldEqual, "test-dur")
			So(req.Created.Equal(now), ShouldBeTrue)
			So(req.Raw, ShouldEqual, task.Payload) // the exact same pointer
			So(req.CreateTaskRequest.Task.Name, ShouldEqual, "projects/proj/locations/reg/queues/queue-1/tasks/"+rem.ID)

			// The task request inside the reminder's raw payload is correct.
			remPayload, err := rem.DropPayload().Payload()
			So(err, ShouldBeNil)
			So(req.CreateTaskRequest, ShouldResembleProto, remPayload.CreateTaskRequest)
		})

		Convey("Fatal Submit error", func() {
			submitter.err = func(string) error { return status.Errorf(codes.PermissionDenied, "boom") }

			err := d.AddTask(txn, &Task{
				Payload: durationpb.New(5 * time.Second),
				Delay:   10 * time.Second,
			})
			So(err, ShouldBeNil)

			So(db.AllReminders(), ShouldHaveLength, 1)
			db.ExecDefers(ctx)
			So(db.AllReminders(), ShouldBeEmpty)
		})

		Convey("Transient Submit error", func() {
			submitter.err = func(string) error { return status.Errorf(codes.Internal, "boom") }

			err := d.AddTask(txn, &Task{
				Payload: durationpb.New(5 * time.Second),
				Delay:   10 * time.Second,
			})
			So(err, ShouldBeNil)

			So(db.AllReminders(), ShouldHaveLength, 1)
			db.ExecDefers(ctx)
			So(db.AllReminders(), ShouldHaveLength, 1)
		})

		Convey("Slow", func() {
			err := d.AddTask(txn, &Task{
				Payload: durationpb.New(5 * time.Second),
				Delay:   10 * time.Second,
			})
			So(err, ShouldBeNil)

			tc.Add(happyPathMaxDuration + 1*time.Second)

			So(db.AllReminders(), ShouldHaveLength, 1)
			db.ExecDefers(ctx)
			So(db.AllReminders(), ShouldHaveLength, 1)
			So(submitter.reqs, ShouldBeEmpty)
		})
	})
}

func TestTesting(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		var epoch = testclock.TestRecentTimeUTC

		ctx, tc := testclock.UseTime(context.Background(), epoch)
		tc.SetTimerCallback(func(d time.Duration, t clock.Timer) {
			if testclock.HasTags(t, tqtesting.ClockTag) {
				tc.Add(d)
			}
		})

		disp := Dispatcher{}
		ctx, sched := TestingContext(ctx, &disp)

		var success tqtesting.TaskList
		sched.TaskSucceeded = tqtesting.TasksCollector(&success)

		m := sync.Mutex{}
		etas := []time.Duration{}

		disp.RegisterTaskClass(TaskClass{
			ID:        "test-dur",
			Prototype: &durationpb.Duration{}, // just some proto type
			Queue:     "queue-1",
			Handler: func(ctx context.Context, msg proto.Message) error {
				m.Lock()
				etas = append(etas, clock.Now(ctx).Sub(epoch))
				m.Unlock()
				if clock.Now(ctx).Sub(epoch) < 3*time.Second {
					disp.AddTask(ctx, &Task{
						Payload: &durationpb.Duration{
							Seconds: msg.(*durationpb.Duration).Seconds + 1,
						},
						Delay: time.Second,
					})
				}
				return nil
			},
		})

		So(disp.AddTask(ctx, &Task{Payload: &durationpb.Duration{Seconds: 1}}), ShouldBeNil)
		sched.Run(ctx, tqtesting.StopWhenDrained())
		So(etas, ShouldResemble, []time.Duration{
			0, 1 * time.Second, 2 * time.Second, 3 * time.Second,
		})

		So(success, ShouldHaveLength, 4)
		So(success.Payloads(), ShouldResembleProto, []*durationpb.Duration{
			{Seconds: 1},
			{Seconds: 2},
			{Seconds: 3},
			{Seconds: 4},
		})
	})
}

func TestPubSubEnqueue(t *testing.T) {
	t.Parallel()

	Convey("With dispatcher", t, func() {
		var epoch = testclock.TestRecentTimeUTC

		ctx, tc := testclock.UseTime(context.Background(), epoch)
		db := testutil.FakeDB{}

		disp := Dispatcher{Sweeper: NewInProcSweeper(InProcSweeperOptions{})}
		ctx, sched := TestingContext(ctx, &disp)

		disp.RegisterTaskClass(TaskClass{
			ID:        "test-dur",
			Prototype: &durationpb.Duration{}, // just some proto type
			Topic:     "topic-1",
			Kind:      Transactional,
			Custom: func(_ context.Context, msg proto.Message) (*CustomPayload, error) {
				return &CustomPayload{
					Meta: map[string]string{"a": "b"},
					Body: []byte(fmt.Sprintf("%d", msg.(*durationpb.Duration).Seconds)),
				}, nil
			},
		})

		So(disp.AddTask(db.Inject(ctx), &Task{Payload: &durationpb.Duration{Seconds: 1}}), ShouldBeNil)

		Convey("Happy path", func() {
			db.ExecDefers(ctx) // actually enqueue

			So(sched.Tasks(), ShouldHaveLength, 1)

			task := sched.Tasks()[0]
			So(task.Payload, ShouldResembleProto, &durationpb.Duration{Seconds: 1})
			So(task.Message, ShouldResembleProto, &pubsubpb.PubsubMessage{
				Data: []byte("1"),
				Attributes: map[string]string{
					"a":                     "b",
					"X-Luci-Tq-Reminder-Id": task.Message.Attributes["X-Luci-Tq-Reminder-Id"],
				},
			})
		})

		Convey("Unhappy path", func() {
			// Not enqueued, but have a reminder.
			So(sched.Tasks(), ShouldHaveLength, 0)
			So(db.AllReminders(), ShouldHaveLength, 1)

			// Make reminder sufficiently stale to be eligible for sweeping.
			tc.Add(5 * time.Minute)

			// Run the sweeper to enqueue from the reminder.
			So(disp.Sweep(db.Inject(ctx)), ShouldBeNil)

			// Have the task now!
			So(sched.Tasks(), ShouldHaveLength, 1)

			task := sched.Tasks()[0]
			So(task.Payload, ShouldBeNil) // not available on non-happy path
			So(task.Message, ShouldResembleProto, &pubsubpb.PubsubMessage{
				Data: []byte("1"),
				Attributes: map[string]string{
					"a":                     "b",
					"X-Luci-Tq-Reminder-Id": task.Message.Attributes["X-Luci-Tq-Reminder-Id"],
				},
			})
		})
	})
}

type submitter struct {
	err  func(title string) error
	m    sync.Mutex
	reqs []*reminder.Payload
}

func (s *submitter) Submit(ctx context.Context, req *reminder.Payload) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.reqs = append(s.reqs, req)
	if s.err == nil {
		return nil
	}
	return s.err(title(req))
}

func (s *submitter) titles() []string {
	var t []string
	for _, r := range s.reqs {
		t = append(t, title(r))
	}
	sort.Strings(t)
	return t
}

func title(req *reminder.Payload) string {
	url := ""
	switch mt := req.CreateTaskRequest.Task.MessageType.(type) {
	case *taskspb.Task_HttpRequest:
		url = mt.HttpRequest.Url
	case *taskspb.Task_AppEngineHttpRequest:
		url = mt.AppEngineHttpRequest.RelativeUri
	}
	idx := strings.LastIndex(url, "/")
	return url[idx+1:]
}
