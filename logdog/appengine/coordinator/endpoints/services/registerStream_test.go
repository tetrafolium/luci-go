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

package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/gae/filter/featureBreaker"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/gae/service/taskqueue"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/logdog/api/config/svcconfig"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"
	ct "github.com/tetrafolium/luci-go/logdog/appengine/coordinator/coordinatorTest"
	"github.com/tetrafolium/luci-go/logdog/common/types"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestRegisterStream(t *testing.T) {
	t.Parallel()

	Convey(`With a testing configuration`, t, func() {
		c, env := ct.Install(true)

		// Set our archival delays. The project delay is smaller than the service
		// delay, so it should be used.
		env.ModServiceConfig(c, func(cfg *svcconfig.Config) {
			cfg.Coordinator.ArchiveSettleDelay = google.NewDuration(10 * time.Minute)
			cfg.Coordinator.ArchiveDelayMax = google.NewDuration(24 * time.Hour)
		})

		env.ModProjectConfig(c, "proj-foo", func(pcfg *svcconfig.ProjectConfig) {
			pcfg.MaxStreamAge = google.NewDuration(time.Hour)
		})

		// By default, the testing user is a service.
		env.JoinGroup("services")

		svr := New(ServerSettings{NumQueues: 2})

		// The testable TQ object.
		ts := taskqueue.GetTestable(c)
		ts.CreatePullQueue(RawArchiveQueueName(0))
		ts.CreatePullQueue(RawArchiveQueueName(1))

		Convey(`Returns Forbidden error if not a service.`, func() {
			env.LeaveAllGroups()

			_, err := svr.RegisterStream(c, &logdog.RegisterStreamRequest{})
			So(err, ShouldBeRPCPermissionDenied)
		})

		Convey(`When registering a testing log sream, "testing/+/foo/bar"`, func() {
			tls := ct.MakeStream(c, "proj-foo", "testing/+/foo/bar")

			req := logdog.RegisterStreamRequest{
				Project:       string(tls.Project),
				Secret:        tls.Prefix.Secret,
				ProtoVersion:  logpb.Version,
				Desc:          tls.DescBytes(),
				TerminalIndex: -1,
			}

			Convey(`Returns FailedPrecondition when the Prefix is not registered.`, func() {
				_, err := svr.RegisterStream(c, &req)
				So(err, ShouldBeRPCFailedPrecondition)
			})

			Convey(`When the Prefix is registered`, func() {
				tls.WithProjectNamespace(c, func(c context.Context) {
					if err := ds.Put(c, tls.Prefix); err != nil {
						panic(err)
					}
				})

				expResp := &logdog.RegisterStreamResponse{
					Id: string(tls.Stream.ID),
					State: &logdog.LogStreamState{
						Secret:        tls.Prefix.Secret,
						ProtoVersion:  logpb.Version,
						TerminalIndex: -1,
					},
				}

				Convey(`Can register the stream.`, func() {
					created := ds.RoundTime(env.Clock.Now())

					resp, err := svr.RegisterStream(c, &req)
					So(err, ShouldBeRPCOK)
					So(resp, ShouldResemble, expResp)
					ds.GetTestable(c).CatchupIndexes()

					So(tls.Get(c), ShouldBeNil)

					// Registers the log stream.
					So(tls.Stream.Created, ShouldResemble, created)

					// Registers the log stream state.
					So(tls.State.Created, ShouldResemble, created)
					So(tls.State.Updated, ShouldResemble, created)
					So(tls.State.Secret, ShouldResemble, req.Secret)
					So(tls.State.TerminalIndex, ShouldEqual, -1)
					So(tls.State.Terminated(), ShouldBeFalse)
					// Pessimistic archival is scheduled.
					So(tls.State.ArchivalState(), ShouldEqual, coordinator.ArchiveTasked)

					// Should also register the log stream Prefix.
					So(tls.Prefix.Created, ShouldResemble, created)
					So(tls.Prefix.Secret, ShouldResemble, req.Secret)

					Convey(`Can register the stream again (idempotent).`, func() {
						env.Clock.Set(created.Add(10 * time.Minute))

						resp, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCOK)
						So(resp, ShouldResemble, expResp)

						tls.WithProjectNamespace(c, func(c context.Context) {
							So(ds.Get(c, tls.Stream, tls.State), ShouldBeNil)
						})
						So(tls.State.Created, ShouldResemble, created)
						So(tls.Stream.Created, ShouldResemble, created)

						Convey(`Skips archival completely after 3 weeks`, func() {
							// Three weeks and an hour later
							threeWeeks := (time.Hour * 24 * 7 * 3) + time.Hour
							env.Clock.Set(created.Add(threeWeeks))
							// Make it so that any 2s sleep timers progress.
							env.Clock.SetTimerCallback(func(d time.Duration, tmr clock.Timer) {
								env.Clock.Add(3 * time.Second)
							})

							tls.WithProjectNamespace(c, func(c context.Context) {
								So(ds.Get(c, tls.State), ShouldBeNil)
							})
							SkipSo(tls.State.ArchivedTime.After(created.Add(threeWeeks)), ShouldBeTrue)
						})
					})

					Convey(`Will not re-register if secrets don't match.`, func() {
						req.Secret[0] = 0xAB
						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCInvalidArgument, "invalid secret")
					})
				})

				Convey(`Can register a terminal stream.`, func() {
					// Make it so that any 2s sleep timers progress.
					env.Clock.SetTimerCallback(func(d time.Duration, tmr clock.Timer) {
						env.Clock.Add(3 * time.Second)
					})
					prefixCreated := ds.RoundTime(env.Clock.Now())
					streamCreated := prefixCreated // same time as the prefix
					req.TerminalIndex = 1337
					expResp.State.TerminalIndex = 1337

					resp, err := svr.RegisterStream(c, &req)
					So(err, ShouldBeRPCOK)
					So(resp, ShouldResemble, expResp)
					ds.GetTestable(c).CatchupIndexes()

					So(tls.Get(c), ShouldBeNil)

					// Registers the log stream.
					So(tls.Stream.Created, ShouldResemble, streamCreated)

					// Registers the log stream state.
					So(tls.State.Created, ShouldResemble, streamCreated)
					// Tasking for archival should happen after creation.
					So(tls.State.Updated.After(streamCreated), ShouldBeTrue)
					So(tls.State.Secret, ShouldResemble, req.Secret)
					So(tls.State.TerminalIndex, ShouldEqual, 1337)
					So(tls.State.TerminatedTime, ShouldResemble, streamCreated)
					So(tls.State.Terminated(), ShouldBeTrue)
					So(tls.State.ArchivalState(), ShouldEqual, coordinator.ArchiveTasked)

					// Should also register the log stream Prefix.
					So(tls.Prefix.Created, ShouldResemble, prefixCreated)
					So(tls.Prefix.Secret, ShouldResemble, req.Secret)

					// When we advance to our settle delay, an archival task is scheduled.
					env.Clock.Add(10 * time.Minute)
				})

				Convey(`Will schedule the correct archival expiration delay`, func() {
					Convey(`When there is no project config delay.`, func() {
						// Make it so that any 2s sleep timers progress.
						env.Clock.SetTimerCallback(func(d time.Duration, tmr clock.Timer) {
							env.Clock.Add(3 * time.Second)
						})
						env.ModProjectConfig(c, "proj-foo", func(pcfg *svcconfig.ProjectConfig) {
							pcfg.MaxStreamAge = nil
						})

						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCOK)
						ds.GetTestable(c).CatchupIndexes()
					})

					Convey(`When there is no service or project config delay.`, func() {
						// Make it so that any 2s sleep timers progress.
						env.Clock.SetTimerCallback(func(d time.Duration, tmr clock.Timer) {
							env.Clock.Add(3 * time.Second)
						})
						env.ModServiceConfig(c, func(cfg *svcconfig.Config) {
							cfg.Coordinator.ArchiveDelayMax = nil
						})
						env.ModProjectConfig(c, "proj-foo", func(pcfg *svcconfig.ProjectConfig) {
							pcfg.MaxStreamAge = nil
						})

						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCOK)
						ds.GetTestable(c).CatchupIndexes()
					})
				})

				Convey(`Returns internal server error if the datastore Get() fails.`, func() {
					c, fb := featureBreaker.FilterRDS(c, nil)
					fb.BreakFeatures(errors.New("test error"), "GetMulti")

					_, err := svr.RegisterStream(c, &req)
					So(err, ShouldBeRPCInternal)
				})

				Convey(`Returns internal server error if the Prefix Put() fails.`, func() {
					c, fb := featureBreaker.FilterRDS(c, nil)
					fb.BreakFeatures(errors.New("test error"), "PutMulti")

					_, err := svr.RegisterStream(c, &req)
					So(err, ShouldBeRPCInternal)
				})

				Convey(`Registration failure cases`, func() {
					Convey(`Will not register a stream if its prefix has expired.`, func() {
						env.Clock.Set(tls.Prefix.Expiration)

						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCFailedPrecondition, "prefix has expired")
					})

					Convey(`Will not register a stream without a protobuf version.`, func() {
						req.ProtoVersion = ""
						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCInvalidArgument, "Unrecognized protobuf version")
					})

					Convey(`Will not register a stream with an unknown protobuf version.`, func() {
						req.ProtoVersion = "unknown"
						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCInvalidArgument, "Unrecognized protobuf version")
					})

					Convey(`Will not register with an empty descriptor.`, func() {
						req.Desc = nil

						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCInvalidArgument, "Invalid log stream descriptor")
					})

					Convey(`Will not register if the descriptor doesn't validate.`, func() {
						tls.Desc.ContentType = ""
						So(tls.Desc.Validate(true), ShouldNotBeNil)
						req.Desc = tls.DescBytes()

						_, err := svr.RegisterStream(c, &req)
						So(err, ShouldBeRPCInvalidArgument, "Invalid log stream descriptor")
					})
				})
			})
		})
	})
}

func BenchmarkRegisterStream(b *testing.B) {
	c, env := ct.Install(true)

	// Set our archival delays. The project delay is smaller than the service
	// delay, so it should be used.
	env.ModServiceConfig(c, func(cfg *svcconfig.Config) {
		cfg.Coordinator.ArchiveSettleDelay = google.NewDuration(10 * time.Minute)
		cfg.Coordinator.ArchiveDelayMax = google.NewDuration(24 * time.Hour)
	})
	env.ModProjectConfig(c, "proj-foo", func(pcfg *svcconfig.ProjectConfig) {
		pcfg.MaxStreamAge = google.NewDuration(time.Hour)
	})

	// By default, the testing user is a service.
	env.JoinGroup("services")

	const (
		prefix  = types.StreamName("testing")
		project = "proj-foo"
	)

	tls := ct.MakeStream(c, project, prefix.Join(types.StreamName(fmt.Sprintf("foo/bar"))))
	tls.WithProjectNamespace(c, func(c context.Context) {
		if err := ds.Put(c, tls.Prefix); err != nil {
			b.Fatalf("failed to register prefix: %v", err)
		}
	})

	svr := New(ServerSettings{NumQueues: 2})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tls := ct.MakeStream(c, project, prefix.Join(types.StreamName(fmt.Sprintf("foo/bar/%d", i))))

		req := logdog.RegisterStreamRequest{
			Project:       string(tls.Project),
			Secret:        tls.Prefix.Secret,
			ProtoVersion:  logpb.Version,
			Desc:          tls.DescBytes(),
			TerminalIndex: -1,
		}

		_, err := svr.RegisterStream(c, &req)
		if err != nil {
			b.Fatalf("failed to get OK response (%s)", err)
		}
	}
}
