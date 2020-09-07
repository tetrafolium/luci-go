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

package deps

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/golang/protobuf/proto"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	dm "github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/api/template"

	"github.com/tetrafolium/luci-go/tumble"

	"github.com/tetrafolium/luci-go/dm/appengine/distributor"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	"github.com/tetrafolium/luci-go/dm/appengine/mutate"
)

func (d *deps) runEnsureGraphDepsWalk(c context.Context, req *dm.EnsureGraphDataReq, newAttempts *dm.AttemptList) (*dm.GraphData, error) {
	// first lets run a query to load all of the proposed attempts.
	wgreq := &dm.WalkGraphReq{
		Query: dm.AttemptListQuery(newAttempts),
		Limit: &dm.WalkGraphReq_Limit{
			MaxDepth:    1,
			MaxDataSize: req.Limit.MaxDataSize,
		},
		Include: &dm.WalkGraphReq_Include{
			Quest: &dm.WalkGraphReq_Include_Options{Data: true},
			Attempt: &dm.WalkGraphReq_Include_Options{
				Abnormal: true,
				Expired:  true,
				Data:     true,
				Result:   req.Include.Attempt.Result,
			},
		},
	}
	if err := wgreq.Normalize(); err != nil {
		panic(err)
	}
	qryRsp, err := doGraphWalk(c, wgreq)
	if err != nil {
		return nil, err
	}
	if qryRsp.HadErrors {
		return nil, grpcutil.Internal
	}
	return qryRsp, nil
}

func allFinished(gd *dm.GraphData) bool {
	for _, qst := range gd.Quests {
		if qst.DNE {
			return false
		}
		for _, atmpt := range qst.Attempts {
			if atmpt.DNE || atmpt.Data.GetFinished() == nil {
				return false
			}
		}
	}
	return true
}

func filterQuestsByNewTemplateData(g *dm.GraphData, newQuests []*model.Quest) (ret []*model.Quest, quests stringset.Set) {
	quests = stringset.New(0)
	for _, q := range newQuests {
		curQuest := g.Quests[q.ID]
		if curQuest.DNE || !q.BuiltBy.EqualsData(curQuest.Data.BuiltBy) {
			ret = append(ret, q)
			quests.Add(q.ID)
		}
	}
	return
}

func filterAttemptsByDNE(gd *dm.GraphData, newAttempts *dm.AttemptList, newQuestSet stringset.Set) (ret *dm.AttemptList, count int, err error) {
	ret = dm.NewAttemptList(nil)
	tmpAID := &dm.Attempt_ID{}
	nums := &dm.AttemptList_Nums{}
	for tmpAID.Quest, nums = range newAttempts.To {
		if gd.Quests[tmpAID.Quest].DNE && !newQuestSet.Has(tmpAID.Quest) {
			err = fmt.Errorf("cannot create attempts for absent quest %q", tmpAID.Quest)
			return
		}
		for _, tmpAID.Id = range nums.Nums {
			if gd.Quests[tmpAID.Quest].Attempts[tmpAID.Id].DNE {
				count++
				ret.AddAIDs(tmpAID)
			}
		}
	}
	return
}

func depsFromMissing(c context.Context, fwdDepKeys []*ds.Key, exists ds.BoolList) (ret *dm.AttemptList) {
	ret = dm.NewAttemptList(nil)
	tmpAID := &dm.Attempt_ID{}
	for i, fkey := range fwdDepKeys {
		if !exists[i] {
			if err := tmpAID.SetDMEncoded(fkey.StringID()); err != nil {
				logging.Fields{
					logging.ErrorKey:  err,
					"FwdDep.StringID": fkey.StringID(),
				}.Errorf(c, "impossible parsing error")
				panic(err)
			}
			ret.AddAIDs(tmpAID)
		}
	}
	return
}

func journalQuestAttempts(c context.Context, newQuests []*model.Quest, newAttempts *dm.AttemptList) error {
	if len(newQuests) == 0 && len(newAttempts.To) == 0 {
		return nil
	}
	newAttempts = newAttempts.Dup()
	muts := make([]tumble.Mutation, 0, len(newQuests)+len(newAttempts.To))
	for _, q := range newQuests {
		mut := &mutate.EnsureQuestAttempts{Quest: q}
		if nums, ok := newAttempts.To[q.ID]; ok {
			delete(newAttempts.To, q.ID)
			mut.AIDs = nums.Nums
		}
		muts = append(muts, mut)
	}
	for qid, nums := range newAttempts.To {
		muts = append(muts, &mutate.EnsureQuestAttempts{
			Quest:           &model.Quest{ID: qid},
			AIDs:            nums.Nums,
			DoNotMergeQuest: true,
		})
	}
	return grpcAnnotate(tumble.AddToJournal(c, muts...), codes.Internal, "attempting to journal").Err()
}

func (d *deps) ensureGraphData(c context.Context, req *dm.EnsureGraphDataReq, newQuests []*model.Quest, newAttempts *dm.AttemptList, rsp *dm.EnsureGraphDataRsp) (err error) {
	var (
		fwdDepExists *ds.ExistsResult
		fwdDepKeys   []*ds.Key
	)
	if req.ForExecution != nil {
		fwdDepKeys = model.FwdDepKeysFromList(c, req.ForExecution.Id.AttemptID(), newAttempts)
	}
	// Do a graph walk for all of the newAttempts, and also check the existence
	// of all proposed deps (if needed).
	err = parallel.FanOutIn(func(gen chan<- func() error) {
		gen <- func() error {
			rsp.Result, err = d.runEnsureGraphDepsWalk(c, req, newAttempts)
			if err != nil {
				err = fmt.Errorf("while walking graph: %s", err)
			}
			return err
		}
		if req.ForExecution != nil {
			gen <- func() (err error) {
				fwdDepExists, err = ds.Exists(c, fwdDepKeys)
				if err != nil {
					err = fmt.Errorf("while finding FwdDeps: %s", err)
				}
				return err
			}
		}
	})
	if err != nil {
		return grpcAnnotate(err, codes.Internal, "failed to gather prerequisites").Err()
	}

	// Now that we've walked the graph, prune the lists of new Quest and Attempts
	// by the information retrieved in the graph walk. newQuest and newAttempts
	// will be reduced to contain only the missing information.
	newQuests, newQuestSet := filterQuestsByNewTemplateData(rsp.Result, newQuests)
	newAttempts, newAttemptsLen, err := filterAttemptsByDNE(rsp.Result, newAttempts, newQuestSet)
	if err != nil {
		return grpcAnnotate(err, codes.InvalidArgument, "filterAttemptsByDNE").Err()
	}

	// we're just asserting nodes, no edges, so journal whatever's left
	if req.ForExecution == nil {
		logging.Fields{"qs": len(newQuests), "atmpts": newAttemptsLen}.Infof(c,
			"journaling without deps")
		err := journalQuestAttempts(c, newQuests, newAttempts)
		rsp.Accepted = err == nil
		return err
	}

	// we're asserting nodes+edges
	missingDeps := depsFromMissing(c, fwdDepKeys, fwdDepExists.List(0))

	// we have no missing deps, or all the attempts we want are finished already
	if len(missingDeps.To) == 0 || allFinished(rsp.Result) {
		// if we have new quests to journal, or deps to add, journal them.
		if len(newQuests) > 0 || len(missingDeps.To) > 0 {
			err = tumbleNow(c, &mutate.AddFinishedDeps{
				Auth:             req.ForExecution,
				MergeQuests:      newQuests,
				FinishedAttempts: missingDeps,
			})
			rsp.Accepted = err == nil
			return
		}

		// otherwise we're done already
		rsp.Accepted = true
		return nil
	}

	// not all of the attemps exist/are finished, we have to block.
	rsp.Result = nil
	err = tumbleNow(c, &mutate.AddDeps{
		Auth:   req.ForExecution,
		Quests: newQuests,
		// Attempts we think are missing
		Attempts: newAttempts,
		// Deps we think are missing (>= newAttempts)
		Deps: missingDeps,
	})
	if err == nil {
		rsp.Accepted = true
		rsp.ShouldHalt = true
	}
	return err
}

type templateFileKey struct {
	project, ref string
}

type templateFile struct {
	file    *dmTemplate.File
	version string
}

type templateFileCache map[templateFileKey]templateFile

func (cache templateFileCache) render(c context.Context, inst *dm.TemplateInstantiation) (desc *dm.Quest_Desc, vers string, err error) {
	key := templateFileKey{inst.Project, inst.Ref}
	f, ok := cache[key]
	if !ok {
		f.file, f.version, err = dmTemplate.LoadFile(c, inst.Project, inst.Ref)
		if err != nil {
			err = fmt.Errorf("failed to load templates %#v: %s", key, err)
			return
		}
		cache[key] = f
	}
	vers = f.version
	desc, err = f.file.Render(inst.Specifier)
	return
}

func renderRequest(c context.Context, req *dm.EnsureGraphDataReq) (rsp *dm.EnsureGraphDataRsp, newQuests map[string]*model.Quest, newAttempts *dm.AttemptList, err error) {
	rsp = &dm.EnsureGraphDataRsp{}

	setTemplateErr := func(i int, err error) bool {
		if err == nil {
			return false
		}
		if rsp.TemplateError == nil {
			rsp.TemplateError = make([]string, len(req.TemplateQuest))
		}
		rsp.TemplateError[i] = err.Error()
		return true
	}

	dists := map[string]distributor.D{}

	newQuests = make(map[string]*model.Quest, len(req.Quest)+len(req.TemplateQuest))
	newAttempts = proto.Clone(req.RawAttempts).(*dm.AttemptList)
	if newAttempts == nil {
		newAttempts = dm.NewAttemptList(nil)
	}

	reg := distributor.GetRegistry(c)

	// render all quest descriptions
	for i, qDesc := range req.Quest {
		q := model.NewQuest(c, qDesc)

		d, ok := dists[qDesc.DistributorConfigName]
		if !ok {
			if d, _, err = reg.MakeDistributor(c, qDesc.DistributorConfigName); err != nil {
				return
			}
			dists[qDesc.DistributorConfigName] = d
		}

		if err = d.Validate(qDesc.DistributorParameters); err != nil {
			err = grpcAnnotate(err, codes.InvalidArgument,
				"JSON distributor parameters are invalid for this distributor configuration.").Err()
			return
		}

		if _, ok := newQuests[q.ID]; !ok {
			newQuests[q.ID] = q
		}
		rsp.QuestIds = append(rsp.QuestIds, dm.NewQuestID(q.ID))
		anums := newAttempts.To[q.ID]
		if anums == nil {
			anums = &dm.AttemptList_Nums{}
			newAttempts.To[q.ID] = anums
		}
		anums.Nums = append(anums.Nums, req.QuestAttempt[i].Nums...)
		if err := anums.Normalize(); err != nil {
			logging.Errorf(c, "impossible: these inputs were already validated: %s", err)
			panic(err)
		}
	}

	// render all templates and template attempts into newQuests
	templateFiles := templateFileCache{}
	for i := 0; i < len(req.TemplateQuest); i++ {
		inst := req.TemplateQuest[i]

		var vers string
		var desc *dm.Quest_Desc
		if desc, vers, err = templateFiles.render(c, inst); setTemplateErr(i, err) {
			continue
		}
		if setTemplateErr(i, desc.Normalize()) {
			continue
		}

		q := model.NewQuest(c, desc)

		rsp.TemplateIds = append(rsp.TemplateIds, dm.NewQuestID(q.ID))

		// if we have any errors going on, might as well skip the rest
		if len(rsp.TemplateError) > 0 {
			continue
		}

		anums := newAttempts.To[q.ID]
		anums.Nums = append(anums.Nums, req.TemplateAttempt[i].Nums...)
		if err := anums.Normalize(); err != nil {
			logging.WithError(err).Errorf(c, "impossible: these inputs were already validated")
			panic(err)
		}

		toAddTemplateInfo, ok := newQuests[q.ID]
		if !ok {
			toAddTemplateInfo = q
			newQuests[q.ID] = q
		}
		toAddTemplateInfo.BuiltBy.Add(dm.Quest_TemplateSpec{
			Project: inst.Project, Ref: inst.Ref, Version: vers,
			Name: inst.Specifier.TemplateName})
	}

	return
}

func (d *deps) EnsureGraphData(c context.Context, req *dm.EnsureGraphDataReq) (rsp *dm.EnsureGraphDataRsp, err error) {
	if req.ForExecution != nil {
		logging.Fields{"execution": req.ForExecution.Id}.Infof(c, "on behalf of")
		_, _, err := model.AuthenticateExecution(c, req.ForExecution)
		if err != nil {
			return nil, grpcAnnotate(err, codes.Unauthenticated, "bad execution auth").Err()
		}
	} else {
		if err = canWrite(c); err != nil {
			return
		}
	}

	// render any quest descriptions, templates and template attempts into
	// a single merged set of new quests and new attempts
	rsp, newQuests, newAttempts, err := renderRequest(c, req)
	if err != nil || len(rsp.TemplateError) > 0 {
		return
	}

	newQuestList := make([]*model.Quest, 0, len(newQuests))
	for _, q := range newQuests {
		newQuestList = append(newQuestList, q)
	}

	err = d.ensureGraphData(c, req, newQuestList, newAttempts, rsp)

	return
}
