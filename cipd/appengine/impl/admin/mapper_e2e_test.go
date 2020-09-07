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

package admin

import (
	"context"

	"github.com/tetrafolium/luci-go/appengine/mapper"
	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/appengine/tq/tqtesting"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	api "github.com/tetrafolium/luci-go/cipd/api/admin/v1"
	"github.com/tetrafolium/luci-go/cipd/appengine/impl/testutil"
)

// SetupTest prepares a test environment for running mappers.
//
// Puts datastore mock into always consistent mode.
func SetupTest() (context.Context, *adminImpl) {
	ctx, _, _ := testutil.TestingContext()
	datastore.GetTestable(ctx).Consistent(true)

	admin := &adminImpl{
		tq: &tq.Dispatcher{BaseURL: "/internal/tq/"},
	}
	admin.init()

	tq := tqtesting.GetTestable(ctx, admin.tq)
	tq.CreateQueues()

	return ctx, admin
}

// RunMapper launches a mapper and runs it till successful completion.
func RunMapper(ctx context.Context, admin *adminImpl, cfg *api.JobConfig) (mapper.JobID, error) {
	// Launching the job creates an initial tq task.
	jobID, err := admin.LaunchJob(ctx, cfg)
	if err != nil {
		return 0, err
	}

	// Run the tq loop until there are no more pending tasks.
	tq := tqtesting.GetTestable(ctx, admin.tq)
	_, _, err = tq.RunSimulation(ctx, nil)
	if err != nil {
		return 0, err
	}

	// Collect the result. Should be successful, otherwise RunSimulation would
	// have returned an error (it aborts on a first error from a tq task).
	state, err := admin.GetJobState(ctx, jobID)
	if err != nil {
		return 0, err
	}
	if state.Info.State != mapper.State_SUCCESS {
		return 0, errors.Reason("expecting SUCCESS state, got %s", state.Info.State).Err()
	}

	return mapper.JobID(jobID.JobId), nil
}
