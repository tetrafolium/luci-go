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

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	dm "github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/tumble"
	"github.com/tetrafolium/luci-go/tumble/bitfield"

	"google.golang.org/grpc/codes"
)

// AddDeps transactionally stops the current execution and adds one or more
// dependencies.
type AddDeps struct {
	Auth   *dm.Execution_Auth
	Quests []*model.Quest

	// Attempts is attempts we think are missing from the global graph.
	Attempts *dm.AttemptList

	// Deps are fwddeps we think are missing from the auth'd attempt.
	Deps *dm.AttemptList
}

// Root implements tumble.Mutation
func (a *AddDeps) Root(c context.Context) *ds.Key {
	return model.AttemptKeyFromID(c, a.Auth.Id.AttemptID())
}

// RollForward implements tumble.Mutation
//
// This mutation is called directly.
func (a *AddDeps) RollForward(c context.Context) (muts []tumble.Mutation, err error) {
	// Invalidate the execution key so that they can't make more API calls.
	atmpt, ex, err := model.InvalidateExecution(c, a.Auth)
	if err != nil {
		return
	}

	if err = ResetExecutionTimeout(c, ex); err != nil {
		logging.WithError(err).Errorf(c, "could not reset timeout")
		return
	}

	fwdDeps, err := filterExisting(c, model.FwdDepsFromList(c, a.Auth.Id.AttemptID(), a.Deps))
	err = errors.Annotate(err, "while filtering deps").Tag(grpcutil.Tag.With(codes.Internal)).Err()
	if err != nil || len(fwdDeps) == 0 {
		return
	}

	logging.Fields{"aid": atmpt.ID, "count": len(fwdDeps)}.Infof(c, "added deps")
	atmpt.DepMap = bitfield.Make(uint32(len(fwdDeps)))

	for i, fdp := range fwdDeps {
		fdp.BitIndex = uint32(i)
		fdp.ForExecution = atmpt.CurExecution
	}

	if err = ds.Put(c, fwdDeps, atmpt, ex); err != nil {
		err = errors.Annotate(err, "putting stuff").Tag(grpcutil.Tag.With(codes.Internal)).Err()
		return
	}

	mergeQuestMap := map[string]*MergeQuest(nil)
	if len(a.Quests) > 0 {
		mergeQuestMap = make(map[string]*MergeQuest, len(a.Quests))
		for _, q := range a.Quests {
			mergeQuestMap[q.ID] = &MergeQuest{Quest: q}
		}
	}

	muts = make([]tumble.Mutation, 0, len(fwdDeps)+len(a.Attempts.GetTo())+len(a.Quests))
	for _, dep := range fwdDeps {
		toAppend := &muts
		if mq := mergeQuestMap[dep.Dependee.Quest]; mq != nil {
			toAppend = &mq.AndThen
		}

		if nums, ok := a.Attempts.GetTo()[dep.Dependee.Quest]; ok {
			for _, n := range nums.Nums {
				if n == dep.Dependee.Id {
					*toAppend = append(*toAppend, &EnsureAttempt{ID: &dep.Dependee})
					break
				}
			}
		}
		*toAppend = append(*toAppend, &AddBackDep{
			Dep:      dep.Edge(),
			NeedsAck: true,
		})
	}

	// TODO(iannucci): This could run into datastore transaction limits. We could
	// allieviate this by only emitting a single mutation which does tail-calls to
	// decrease its own, unprocessed size by emitting new MergeQuest mutations.
	for _, mut := range mergeQuestMap {
		muts = append(muts, mut)
	}

	return
}

func init() {
	tumble.Register((*AddDeps)(nil))
}
