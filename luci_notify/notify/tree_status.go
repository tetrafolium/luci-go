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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/luci_notify/config"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
)

const botUsername = "luci-notify@chromium.org"
const legacyBotUsername = "buildbot@chromium.org"

type treeStatus struct {
	username  string
	message   string
	key       int64
	status    config.TreeCloserStatus
	timestamp time.Time
}

type treeStatusClient interface {
	getStatus(c context.Context, host string) (*treeStatus, error)
	putStatus(c context.Context, host, message string, prevKey int64) error
}

type readOnlyTreeStatusClient struct {
	fetchFunc func(context.Context, string) ([]byte, error)
}

func (ts *readOnlyTreeStatusClient) getStatus(c context.Context, host string) (*treeStatus, error) {
	respJSON, err := ts.fetchFunc(c, fmt.Sprintf("https://%s/current?format=json", host))
	if err != nil {
		return nil, err
	}

	var r struct {
		Username        string
		CanCommitFreely bool `json:"can_commit_freely"`
		Key             int64
		Date            string
		Message         string
	}
	if err = json.Unmarshal(respJSON, &r); err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal JSON").Err()
	}

	var status config.TreeCloserStatus = config.Closed
	if r.CanCommitFreely {
		status = config.Open
	}

	// Similar to RFC3339, but not quite the same. No time zone is specified,
	// so this will default to UTC, which is correct here.
	const dateFormat = "2006-01-02 15:04:05.999999"
	t, err := time.Parse(dateFormat, r.Date)
	if err != nil {
		return nil, errors.Annotate(err, "failed to parse date from tree status").Err()
	}

	return &treeStatus{
		username:  r.Username,
		message:   r.Message,
		key:       r.Key,
		status:    status,
		timestamp: t,
	}, nil
}

// TODO: Make this actually update the tree status, once we're confident it's
// doing the right thing.
func (ts *readOnlyTreeStatusClient) putStatus(c context.Context, host, message string, prevKey int64) error {
	logging.Infof(c, "Updating status for %s: %s", host, message)
	return nil
}

func fetchHttp(c context.Context, url string) ([]byte, error) {
	transport, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(c)

	response, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get data from %q", url).Err()
	}

	defer response.Body.Close()
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Annotate(err, "failed to read response body from %q", url).Err()
	}

	return bytes, nil
}

// UpdateTreeStatus is the HTTP handler triggered by cron when it's time to
// check tree closers and update tree status if necessary.
func UpdateTreeStatus(c *router.Context) {
	ctx, w := c.Context, c.Writer
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	if err := updateTrees(ctx, &readOnlyTreeStatusClient{fetchHttp}); err != nil {
		logging.WithError(err).Errorf(ctx, "error while updating tree status")
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// updateTrees fetches all TreeClosers from datastore, uses this to determine if
// any trees should be opened or closed, and makes the necessary updates.
func updateTrees(c context.Context, ts treeStatusClient) error {
	var treeClosers []*config.TreeCloser
	if err := datastore.GetAll(c, datastore.NewQuery("TreeCloser"), &treeClosers); err != nil {
		return err
	}

	return parallel.WorkPool(32, func(ch chan<- func() error) {
		for host, treeClosers := range groupTreeClosers(treeClosers) {
			host, treeClosers := host, treeClosers
			ch <- func() error { return updateHost(c, ts, host, treeClosers) }
		}
	})
}

func groupTreeClosers(treeClosers []*config.TreeCloser) map[string][]*config.TreeCloser {
	byHost := map[string][]*config.TreeCloser{}
	for _, tc := range treeClosers {
		byHost[tc.TreeStatusHost] = append(byHost[tc.TreeStatusHost], tc)
	}

	return byHost
}

func updateHost(c context.Context, ts treeStatusClient, host string, treeClosers []*config.TreeCloser) error {
	var oldestClosed *config.TreeCloser
	for _, tc := range treeClosers {
		if tc.Status == config.Closed && (oldestClosed == nil || tc.Timestamp.Before(oldestClosed.Timestamp)) {
			oldestClosed = tc
		}
	}

	var overallStatus config.TreeCloserStatus
	if oldestClosed == nil {
		overallStatus = config.Open
	} else {
		overallStatus = config.Closed
	}

	treeStatus, err := ts.getStatus(c, host)
	switch {
	case err != nil:
		return err
	case treeStatus.status == overallStatus:
		// Nothing to do, status is already correct.
		return nil
	case treeStatus.status == config.Closed && treeStatus.username != botUsername && treeStatus.username != legacyBotUsername:
		// Don't reopen the tree if it wasn't automatically closed.
		return nil
	}

	var message string
	if overallStatus == config.Open {
		message = fmt.Sprintf("Tree is open (Automatic: %s)", randomEmoji())
	} else {
		// TODO: We actually want to render the template from oldestClosed.
		// We can't do this yet as we don't store the necessary info in the
		// TreeCloser config struct.
		message = fmt.Sprintf("Tree is closed (Automatic: %s)", "TODO")
	}
	// TODO: We also need to compare the TreeCloser timestamps against the
	// existing status timestamp, to ensure we're only acting on new builds.

	return ts.putStatus(c, host, message, treeStatus.key)
}

func randomEmoji() string {
	// TODO: Import the emojis from Gatekeeper.
	return "Yes!"
}
