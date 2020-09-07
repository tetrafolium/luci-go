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

package mutate

import (
	"context"
	"fmt"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	dm "github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/distributor"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/tumble"
)

// TimeoutExecution is a named mutation which triggers on a delay. If the
// execution is in the noted state when the trigger hits, this sets the
// Execution to have an AbnormalFinish status of TIMED_OUT.
type TimeoutExecution struct {
	For   *dm.Execution_ID
	State dm.Execution_State
	// TimeoutAttempt is the number of attempts to stop a STOPPING execution,
	// since this potentially requires an RPC to the distributor to enact.
	TimeoutAttempt uint
	Deadline       time.Time
}

const maxTimeoutAttempts = 3

var _ tumble.DelayedMutation = (*TimeoutExecution)(nil)

// Root implements tumble.Mutation
func (t *TimeoutExecution) Root(c context.Context) *ds.Key {
	return model.AttemptKeyFromID(c, t.For.AttemptID())
}

// RollForward implements tumble.Mutation
func (t *TimeoutExecution) RollForward(c context.Context) (muts []tumble.Mutation, err error) {
	e := model.ExecutionFromID(c, t.For)

	if err = ds.Get(c, e); err != nil {
		return
	}
	if e.State != t.State {
		logging.Errorf(c, "EARLY EXIT: %s v %s", e.State, t.State)
		return
	}

	// will be overwritten if this execution is STOPPING and the timeout is not
	// abnormal
	rslt := &dm.Result{AbnormalFinish: &dm.AbnormalFinish{
		Reason: fmt.Sprintf("DM timeout (%s)", e.State),
		Status: dm.AbnormalFinish_TIMED_OUT}}

	if e.State == dm.Execution_STOPPING {
		// if it's supposed to be STOPPING, maybe we just missed a notification from
		// the distributor (or the distributor is not using pubsub).
		reg := distributor.GetRegistry(c)
		var dist distributor.D
		var vers string
		dist, vers, err = reg.MakeDistributor(c, e.DistributorConfigName)

		if vers != "" && vers != e.DistributorConfigVersion {
			logging.Fields{
				"cfg_name":      e.DistributorConfigName,
				"orig_cfg_vers": e.DistributorConfigVersion,
				"cur_cfg_vers":  vers,
			}.Warningf(c, "mismatched distributor config versions")
		}

		// TODO(iannucci): make this set the REJECTED state if we loaded the config,
		// but the distributor no longer exists.
		if err != nil {
			logging.Fields{
				logging.ErrorKey: err,
				"cfgName":        e.DistributorConfigName,
			}.Errorf(c, "Could not MakeDistributor")
			return
		}
		var realRslt *dm.Result
		q := model.QuestFromID(t.For.Quest)
		if err = ds.Get(ds.WithoutTransaction(c), q); err != nil {
			err = errors.Annotate(err, "loading quest").Err()
			return
		}
		realRslt, err = dist.GetStatus(&q.Desc, distributor.Token(e.DistributorToken))
		if (err != nil || realRslt == nil) && t.TimeoutAttempt < maxTimeoutAttempts {
			logging.Fields{
				logging.ErrorKey:  err,
				"task_result":     realRslt,
				"timeout_attempt": t.TimeoutAttempt,
			}.Infof(c, "GetStatus failed/nop'd while timing out STOPPING execution")
			// TODO(riannucci): do randomized exponential backoff instead of constant
			// backoff? Kinda don't really want to spend more than 1.5m waiting
			// anyway, and the actual GetStatus call does local retries already, so
			// hopefully this is fine. If this is wrong, the distributor should adjust
			// its timeToStop value to be better.
			t.Deadline = t.Deadline.Add(time.Second * 30)
			t.TimeoutAttempt++
			err = nil
			muts = append(muts, t)
			return
		}

		if err != nil {
			rslt.AbnormalFinish.Reason = fmt.Sprintf("DM timeout (%s) w/ error: %s", e.State, err)
			err = nil
		} else if realRslt != nil {
			rslt = realRslt
		}
	}

	muts = append(muts, &FinishExecution{t.For, rslt})
	return
}

// ProcessAfter implements tumble.DelayedMutation
func (t *TimeoutExecution) ProcessAfter() time.Time { return t.Deadline }

// HighPriority implements tumble.DelayedMutation
func (t *TimeoutExecution) HighPriority() bool { return false }

// ResetExecutionTimeout schedules a Timeout for this Execution. It inspects the
// Execution's State to determine which timeout should be set, if any. If no
// timeout should be active, this will cancel any existing timeouts for this
// Execution.
func ResetExecutionTimeout(c context.Context, e *model.Execution) error {
	howLong := time.Duration(0)
	switch e.State {
	case dm.Execution_SCHEDULING:
		howLong = e.TimeToStart
	case dm.Execution_RUNNING:
		howLong = e.TimeToRun
	case dm.Execution_STOPPING:
		howLong = e.TimeToStop
	}
	eid := e.GetEID()
	key := model.ExecutionKeyFromID(c, eid)
	if howLong == 0 {
		return tumble.CancelNamedMutations(c, key, "timeout")
	}
	return tumble.PutNamedMutations(c, key, map[string]tumble.Mutation{
		"timeout": &TimeoutExecution{eid, e.State, 0, clock.Now(c).UTC().Add(howLong)},
	})
}

func init() {
	tumble.Register((*TimeoutExecution)(nil))
}
