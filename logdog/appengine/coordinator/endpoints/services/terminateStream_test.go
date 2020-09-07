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
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/gae/filter/featureBreaker"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/gae/service/taskqueue"

	"github.com/tetrafolium/luci-go/logdog/api/config/svcconfig"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"
	ct "github.com/tetrafolium/luci-go/logdog/appengine/coordinator/coordinatorTest"

	"github.com/tetrafolium/luci-go/common/proto/google"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestTerminateStream(t *testing.T) {
	t.Parallel()

	Convey(`With a testing configuration`, t, func() {
		c, env := ct.Install(true)

		// Set our archival delays. The project delay is smaller than the service
		// delay, so it should be used.
		env.ModServiceConfig(c, func(cfg *svcconfig.Config) {
			coord := cfg.Coordinator
			coord.ArchiveTopic = "projects/test/topics/archive"
			coord.ArchiveSettleDelay = google.NewDuration(10 * time.Second)
			coord.ArchiveDelayMax = google.NewDuration(24 * time.Hour)
		})
		env.ModProjectConfig(c, "proj-foo", func(pcfg *svcconfig.ProjectConfig) {
			pcfg.MaxStreamAge = google.NewDuration(time.Hour)
		})

		svr := New(ServerSettings{NumQueues: 2})

		tls := ct.MakeStream(c, "proj-foo", "testing/+/foo/bar")

		req := logdog.TerminateStreamRequest{
			Project:       string(tls.Project),
			Id:            string(tls.Stream.ID),
			Secret:        tls.Prefix.Secret,
			TerminalIndex: 1337,
		}

		// The testable TQ object.
		ts := taskqueue.GetTestable(c)
		ts.CreatePullQueue(RawArchiveQueueName(0))
		ts.CreatePullQueue(RawArchiveQueueName(1))

		Convey(`Returns Forbidden error if not a service.`, func() {
			_, err := svr.TerminateStream(c, &req)
			So(err, ShouldBeRPCPermissionDenied)
		})

		Convey(`When logged in as a service`, func() {
			env.JoinGroup("services")

			Convey(`A non-terminal registered stream, "testing/+/foo/bar"`, func() {
				So(tls.Put(c), ShouldBeNil)
				ds.GetTestable(c).CatchupIndexes()

				Convey(`Can be marked terminal and schedules an archival mutation.`, func() {
					_, err := svr.TerminateStream(c, &req)
					So(err, ShouldBeRPCOK)
					ds.GetTestable(c).CatchupIndexes()

					// Reload the state and confirm.
					tls.WithProjectNamespace(c, func(c context.Context) {
						So(ds.Get(c, tls.State), ShouldBeNil)
					})
					So(tls.State.TerminalIndex, ShouldEqual, 1337)
					So(tls.State.Terminated(), ShouldBeTrue)
					So(tls.State.ArchivalState(), ShouldEqual, coordinator.ArchiveTasked)

					Convey(`Can be marked terminal again (idempotent).`, func() {
						_, err := svr.TerminateStream(c, &req)
						So(err, ShouldBeRPCOK)

						// Reload state and confirm.
						So(tls.Get(c), ShouldBeNil)

						So(tls.State.Terminated(), ShouldBeTrue)
						So(tls.State.TerminalIndex, ShouldEqual, 1337)
						So(tls.State.ArchivalState(), ShouldEqual, coordinator.ArchiveTasked)
					})

					Convey(`Will reject attempts to change the terminal index.`, func() {
						req.TerminalIndex = 1338
						_, err := svr.TerminateStream(c, &req)
						So(err, ShouldBeRPCFailedPrecondition, "Log stream is incompatibly terminated.")

						// Reload state and confirm.
						So(tls.Get(c), ShouldBeNil)

						So(tls.State.TerminalIndex, ShouldEqual, 1337)
						So(tls.State.Terminated(), ShouldBeTrue)
						So(tls.State.ArchivalState(), ShouldEqual, coordinator.ArchiveTasked)
					})

					Convey(`Will reject attempts to clear the terminal index.`, func() {
						req.TerminalIndex = -1
						_, err := svr.TerminateStream(c, &req)
						So(err, ShouldBeRPCInvalidArgument, "Negative terminal index.")

						// Reload state and confirm.
						So(tls.Get(c), ShouldBeNil)

						So(tls.State.TerminalIndex, ShouldEqual, 1337)
						So(tls.State.Terminated(), ShouldBeTrue)
						So(tls.State.ArchivalState(), ShouldEqual, coordinator.ArchiveTasked)
					})
				})

				Convey(`Will return an internal server error if Put() fails.`, func() {
					c, fb := featureBreaker.FilterRDS(c, nil)
					fb.BreakFeatures(errors.New("test error"), "PutMulti")
					_, err := svr.TerminateStream(c, &req)
					So(err, ShouldBeRPCInternal)
				})

				Convey(`Will return an internal server error if Get() fails.`, func() {
					c, fb := featureBreaker.FilterRDS(c, nil)
					fb.BreakFeatures(errors.New("test error"), "GetMulti")
					_, err := svr.TerminateStream(c, &req)
					So(err, ShouldBeRPCInternal)
				})

				Convey(`Will return a bad request error if the secret doesn't match.`, func() {
					req.Secret[0] ^= 0xFF
					_, err := svr.TerminateStream(c, &req)
					So(err, ShouldBeRPCInvalidArgument, "Request secret doesn't match the stream secret.")
				})
			})

			Convey(`Will not try and terminate a stream with an invalid path.`, func() {
				req.Id = "!!!invalid path!!!"
				_, err := svr.TerminateStream(c, &req)
				So(err, ShouldBeRPCInvalidArgument, "Invalid ID")
			})

			Convey(`Will fail if the stream is not registered.`, func() {
				_, err := svr.TerminateStream(c, &req)
				So(err, ShouldBeRPCNotFound, "is not registered")
			})
		})
	})
}
