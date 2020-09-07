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

package mutate

import (
	"context"
	"fmt"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	dm "github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/distributor"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/tumble"
)

// ScheduleExecution is a placeholder mutation that will be an entry into the
// Distributor scheduling state-machine.
type ScheduleExecution struct {
	For *dm.Attempt_ID
}

// Root implements tumble.Mutation
func (s *ScheduleExecution) Root(c context.Context) *ds.Key {
	return model.AttemptKeyFromID(c, s.For)
}

// RollForward implements tumble.Mutation
func (s *ScheduleExecution) RollForward(c context.Context) (muts []tumble.Mutation, err error) {
	a := model.AttemptFromID(s.For)
	if err = ds.Get(c, a); err != nil {
		logging.WithError(err).Errorf(c, "loading attempt")
		return
	}

	if a.State != dm.Attempt_SCHEDULING {
		logging.Infof(c, "EARLY EXIT: already scheduling")
		return
	}

	q := model.QuestFromID(s.For.Quest)
	if err = ds.Get(ds.WithoutTransaction(c), q); err != nil {
		logging.WithError(err).Errorf(c, "loading quest")
		return
	}

	prevResult := (*dm.JsonResult)(nil)
	if a.LastSuccessfulExecution != 0 {
		prevExecution := model.ExecutionFromID(c, s.For.Execution(a.LastSuccessfulExecution))
		if err = ds.Get(c, prevExecution); err != nil {
			logging.Errorf(c, "loading previous execution: %s", err)
			return
		}
		prevResult = prevExecution.Result.Data
	}

	reg := distributor.GetRegistry(c)
	dist, ver, err := reg.MakeDistributor(c, q.Desc.DistributorConfigName)
	if err != nil {
		logging.WithError(err).Errorf(c, "making distributor %s", q.Desc.DistributorConfigName)
		return
	}

	a.CurExecution++
	if err = a.ModifyState(c, dm.Attempt_EXECUTING); err != nil {
		logging.WithError(err).Errorf(c, "modifying state")
		return
	}

	eid := dm.NewExecutionID(s.For.Quest, s.For.Id, a.CurExecution)
	e := model.MakeExecution(c, eid, q.Desc.DistributorConfigName, ver)
	e.TimeToStart = google.DurationFromProto(q.Desc.Meta.Timeouts.Start)
	e.TimeToRun = google.DurationFromProto(q.Desc.Meta.Timeouts.Run)

	exAuth := &dm.Execution_Auth{Id: eid, Token: e.Token}

	var distTok distributor.Token
	distTok, e.TimeToStop, err = dist.Run(&q.Desc, exAuth, prevResult)
	if e.TimeToStop <= 0 {
		e.TimeToStop = google.DurationFromProto(q.Desc.Meta.Timeouts.Stop)
	}
	e.DistributorToken = string(distTok)
	if err != nil {
		if transient.Tag.In(err) {
			// tumble will retry us later
			logging.WithError(err).Errorf(c, "got transient error in ScheduleExecution")
			return
		}
		logging.WithError(err).Errorf(c, "got non-transient error in ScheduleExecution")
		origErr := err

		// put a and e to the transaction buffer, so that
		// FinishExecution.RollForward can see them.
		if err = ds.Put(c, a, e); err != nil {
			logging.WithError(err).Errorf(c, "putting attempt+execution for non-transient distributor error")
			return
		}
		return NewFinishExecutionAbnormal(
			eid, dm.AbnormalFinish_REJECTED,
			fmt.Sprintf("rejected during scheduling with non-transient error: %s", origErr),
		).RollForward(c)
	}

	if err = ResetExecutionTimeout(c, e); err != nil {
		logging.WithError(err).Errorf(c, "resetting timeout")
		return
	}

	if err = ds.Put(c, a, e); err != nil {
		logging.WithError(err).Errorf(c, "putting attempt+execution")
	}

	return
}

func init() {
	tumble.Register((*ScheduleExecution)(nil))
}
