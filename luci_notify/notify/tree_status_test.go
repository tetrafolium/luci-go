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

package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/logging/memlogger"
	notifypb "go.chromium.org/luci/luci_notify/api/config"
	"go.chromium.org/luci/luci_notify/config"

	. "github.com/smartystreets/goconvey/convey"
)

// fakeTreeStatusClient simulates the behaviour of a real tree status instance,
// but locally, in-memory.
type fakeTreeStatusClient struct {
	statusForHosts map[string]treeStatus
	nextKey        int64
	mtx            sync.Mutex
}

func (ts *fakeTreeStatusClient) getStatus(c context.Context, host string) (*treeStatus, error) {
	ts.mtx.Lock()
	defer ts.mtx.Unlock()

	status, exists := ts.statusForHosts[host]
	if exists {
		return &status, nil
	}
	return nil, errors.New(fmt.Sprintf("No status for host %s", host))
}

func (ts *fakeTreeStatusClient) putStatus(c context.Context, host, message string, prevKey int64) error {
	ts.mtx.Lock()
	defer ts.mtx.Unlock()

	currStatus, exists := ts.statusForHosts[host]
	if exists && currStatus.key != prevKey {
		return errors.New(fmt.Sprintf(
			"prevKey %q passed to putStatus doesn't match previously stored key %q",
			prevKey, currStatus.key))
	}

	key := ts.nextKey
	ts.nextKey++

	var status config.TreeCloserStatus
	if strings.Contains(message, "close") {
		status = config.Closed
	} else {
		status = config.Open
	}

	ts.statusForHosts[host] = treeStatus{
		"buildbot@chromium.org", message, key, status, time.Now(),
	}
	return nil
}

func TestUpdateTrees(t *testing.T) {
	Convey("Test environment", t, func() {
		c := gaetesting.TestingContextWithAppID("luci-notify-test")
		datastore.GetTestable(c).Consistent(true)
		c = memlogger.Use(c)

		project := &config.Project{Name: "chromium"}
		projectKey := datastore.KeyForObj(c, project)
		builder1 := &config.Builder{ProjectKey: projectKey, ID: "ci/builder1"}
		builder2 := &config.Builder{ProjectKey: projectKey, ID: "ci/builder2"}
		builder3 := &config.Builder{ProjectKey: projectKey, ID: "ci/builder3"}
		builder4 := &config.Builder{ProjectKey: projectKey, ID: "ci/builder4"}
		So(datastore.Put(c, project, builder1, builder2, builder3, builder4), ShouldBeNil)

		earlierTime := time.Now().AddDate(-1, 0, 0)

		cleanup := func() {
			var treeClosers []*config.TreeCloser
			So(datastore.GetAll(c, datastore.NewQuery("TreeClosers"), &treeClosers), ShouldBeNil)
			datastore.Delete(c, treeClosers)
		}

		// Helper function for basic tests. Sets an initial tree state, adds two tree closers
		// for the tree, and checks that updateTrees sets the tree to the correct state.
		testUpdateTrees := func(initialTreeStatus, builder1Status, builder2Status, expectedStatus config.TreeCloserStatus) {
			var statusMessage string
			if initialTreeStatus == config.Open {
				statusMessage = "Open for business"
			} else {
				statusMessage = "Closed up"
			}
			ts := fakeTreeStatusClient{
				statusForHosts: map[string]treeStatus{
					"chromium-status.appspot.com": treeStatus{
						username:  botUsername,
						message:   statusMessage,
						key:       -1,
						status:    initialTreeStatus,
						timestamp: earlierTime,
					},
				},
			}

			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder1),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         builder1Status,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder2),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         builder2Status,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			defer cleanup()

			So(updateTrees(c, &ts), ShouldBeNil)

			status, err := ts.getStatus(c, "chromium-status.appspot.com")
			So(err, ShouldBeNil)
			So(status.status, ShouldEqual, expectedStatus)
		}

		Convey("Open, both TCs failing, closes", func() {
			testUpdateTrees(config.Open, config.Closed, config.Closed, config.Closed)
		})

		Convey("Open, 1 failing & 1 passing TC, closes", func() {
			testUpdateTrees(config.Open, config.Closed, config.Open, config.Closed)
		})

		Convey("Open, both TCs passing, stays open", func() {
			testUpdateTrees(config.Open, config.Open, config.Open, config.Open)
		})

		Convey("Closed, both TCs failing, stays closed", func() {
			testUpdateTrees(config.Closed, config.Closed, config.Closed, config.Closed)
		})

		Convey("Closed, 1 failing & 1 passing TC, stays closed", func() {
			testUpdateTrees(config.Closed, config.Closed, config.Open, config.Closed)
		})

		Convey("Closed, both TCs, stays closed", func() {
			testUpdateTrees(config.Closed, config.Closed, config.Open, config.Closed)
		})

		Convey("Closed manually, doesn't re-open", func() {
			ts := fakeTreeStatusClient{
				statusForHosts: map[string]treeStatus{
					"chromium-status.appspot.com": treeStatus{
						username:  "somedev@chromium.org",
						message:   "Closed because of reasons",
						key:       -1,
						status:    config.Closed,
						timestamp: earlierTime,
					},
				},
			}

			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder1),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Open,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			defer cleanup()

			So(updateTrees(c, &ts), ShouldBeNil)

			status, err := ts.getStatus(c, "chromium-status.appspot.com")
			So(err, ShouldBeNil)
			So(status.status, ShouldEqual, config.Closed)
		})

		Convey("Opened manually, still closes", func() {
			ts := fakeTreeStatusClient{
				statusForHosts: map[string]treeStatus{
					"chromium-status.appspot.com": treeStatus{
						username:  "somedev@chromium.org",
						message:   "Opened, because I feel like it",
						key:       -1,
						status:    config.Open,
						timestamp: earlierTime,
					},
				},
			}

			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder1),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Closed,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			defer cleanup()

			So(updateTrees(c, &ts), ShouldBeNil)

			status, err := ts.getStatus(c, "chromium-status.appspot.com")
			So(err, ShouldBeNil)
			So(status.status, ShouldEqual, config.Closed)
		})

		Convey("Multiple trees", func() {
			ts := fakeTreeStatusClient{
				statusForHosts: map[string]treeStatus{
					"chromium-status.appspot.com": treeStatus{
						username:  botUsername,
						message:   "Closed up",
						key:       -1,
						status:    config.Closed,
						timestamp: earlierTime,
					},
					"v8-status.appspot.com": treeStatus{
						username:  botUsername,
						message:   "Open for business",
						key:       -1,
						status:    config.Open,
						timestamp: earlierTime,
					},
				},
			}

			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder1),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Open,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder2),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Open,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder3),
				TreeStatusHost: "chromium-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Open,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)

			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder2),
				TreeStatusHost: "v8-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Open,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder3),
				TreeStatusHost: "v8-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Open,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)
			So(datastore.Put(c, &config.TreeCloser{
				BuilderKey:     datastore.KeyForObj(c, builder4),
				TreeStatusHost: "v8-status.appspot.com",
				TreeCloser:     notifypb.TreeCloser{},
				Status:         config.Closed,
				Timestamp:      time.Now().UTC(),
			}), ShouldBeNil)

			defer cleanup()

			So(updateTrees(c, &ts), ShouldBeNil)

			status, err := ts.getStatus(c, "chromium-status.appspot.com")
			So(err, ShouldBeNil)
			So(status.status, ShouldEqual, config.Open)

			status, err = ts.getStatus(c, "v8-status.appspot.com")
			So(err, ShouldBeNil)
			So(status.status, ShouldEqual, config.Closed)
		})
	})
}

func TestReadOnlyTreeStatusClient(t *testing.T) {
	Convey("Test environment for readOnlyTreeStatusClient", t, func() {
		c := gaetesting.TestingContextWithAppID("luci-notify-test")

		// Real responses, with usernames redacted and readable formatting applied.
		responses := map[string]string{
			"https://chromium-status.appspot.com/current?format=json": `{
				"username": "someone@google.com",
				"can_commit_freely": false,
				"general_state": "throttled",
				"key": 5656890264518656,
				"date": "2020-03-31 05:33:52.682351",
				"message": "Tree is throttled (win rel 32 appears to be a goma flake. the other builds seem to be charging ahead OK. will fully open / fully close if win32 does/doesn't improve)"
			}`,
			"https://v8-status.appspot.com/current?format=json": `{
				"username": "someone-else@google.com",
				"can_commit_freely": true,
				"general_state": "open",
				"key": 5739466035560448,
				"date": "2020-04-02 15:21:39.981072",
				"message": "open (flake?)"
			}`,
		}

		fetch := func(_ context.Context, url string) ([]byte, error) {
			if s, e := responses[url]; e {
				return []byte(s), nil
			} else {
				return nil, fmt.Errorf("Key not present: %q", url)
			}
		}
		ts := readOnlyTreeStatusClient{fetch}

		Convey("Open tree", func() {
			status, err := ts.getStatus(c, "chromium-status.appspot.com")
			So(err, ShouldBeNil)

			expectedTime := time.Date(2020, time.March, 31, 5, 33, 52, 682351000, time.UTC)
			So(status, ShouldResemble, &treeStatus{
				username:  "someone@google.com",
				message:   "Tree is throttled (win rel 32 appears to be a goma flake. the other builds seem to be charging ahead OK. will fully open / fully close if win32 does/doesn't improve)",
				key:       5656890264518656,
				status:    config.Closed,
				timestamp: expectedTime,
			})
		})

		Convey("Closed tree", func() {
			status, err := ts.getStatus(c, "v8-status.appspot.com")
			So(err, ShouldBeNil)

			expectedTime := time.Date(2020, time.April, 2, 15, 21, 39, 981072000, time.UTC)
			So(status, ShouldResemble, &treeStatus{
				username:  "someone-else@google.com",
				message:   "open (flake?)",
				key:       5739466035560448,
				status:    config.Open,
				timestamp: expectedTime,
			})
		})
	})
}
