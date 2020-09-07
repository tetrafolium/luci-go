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

package rpc

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/appstatus"

	"github.com/tetrafolium/luci-go/buildbucket/appengine/model"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
)

var (
	// updateBuildStatuses is a set of build statuses supported by UpdateBuild RPC.
	updateBuildStatuses = map[pb.Status]struct{}{
		pb.Status_STARTED: {},
		// kitchen does not actually use SUCCESS. It relies on swarming pubsub
		// handler in Buildbucket because a task may fail after recipe succeeded.
		pb.Status_SUCCESS:       {},
		pb.Status_FAILURE:       {},
		pb.Status_INFRA_FAILURE: {},
	}

	// statusesWithStartTime is a set of step statuses that requires the step to have
	// a start_time field set.
	statusesWithStartTime = map[pb.Status]struct{}{
		pb.Status_STARTED: {},
		pb.Status_SUCCESS: {},
		pb.Status_FAILURE: {},
	}
)

// validateUpdate validates the given request.
func validateUpdate(req *pb.UpdateBuildRequest, bs *model.BuildSteps) error {
	if req.GetBuild().GetId() == 0 {
		return errors.Reason("build.id: required").Err()
	}

	for _, p := range req.UpdateMask.GetPaths() {
		// TODO(1110990): validate gitiles_commit, summary_markdown, and steps
		switch p {
		case "build.output":
		case "build.output.properties":
		case "build.output.gitiles_commit":
			if err := validateCommitWithRef(req.Build.Output.GetGitilesCommit()); err != nil {
				return errors.Annotate(err, "build.output.gitiles_commit").Err()
			}
		case "build.status":
			if _, ok := updateBuildStatuses[req.Build.Status]; !ok {
				return errors.Reason("build.status: invalid status %s for UpdateBuild", req.Build.Status).Err()
			}
		case "build.status_details":
		case "build.steps":
			if err := validateSteps(bs, req.Build.Steps); err != nil {
				return errors.Annotate(err, "build.steps").Err()
			}
		case "build.summary_markdown":
			if err := validateSummaryMarkdown(req.Build.SummaryMarkdown); err != nil {
				return errors.Annotate(err, "build.summary_markdown").Err()
			}
		case "build.tags":
			if err := validateTags(req.Build.Tags, TagAppend); err != nil {
				return errors.Annotate(err, "build.tags").Err()
			}
		default:
			return errors.Reason("unsupported path %q", p).Err()
		}
	}
	return nil
}

// validateSteps validates the steps of the Build.
func validateSteps(bs *model.BuildSteps, steps []*pb.Step) error {
	if err := bs.FromProto(steps); err != nil {
		return err
	}
	if len(bs.Bytes) > model.BuildStepsMaxBytes {
		return errors.Reason("too big to accept").Err()
	}

	seen := stringset.New(len(steps))
	for i, step := range steps {
		if err := protoutil.ValidateStepName(step.Name); err != nil {
			return errors.Annotate(err, "step[%d].name", i).Err()
		}
		if !seen.Add(step.Name) {
			return errors.Reason("step[%d]: duplicate: %q", i, step.Name).Err()
		}
		if pn := protoutil.ParentStepName(step.Name); pn != "" && !seen.Has(pn) {
			return errors.Reason("step[%d]: parent of %q must precede", i, step.Name).Err()
		}

		if err := validateStep(step); err != nil {
			return errors.Annotate(err, "step[%d]", i).Err()
		}
	}
	return nil
}

func validateStep(step *pb.Step) error {
	var st, et time.Time
	var err error
	if step.StartTime != nil {
		if st, err = ptypes.Timestamp(step.StartTime); err != nil {
			return errors.Annotate(err, "start_time").Err()
		}
	}
	if step.EndTime != nil {
		if et, err = ptypes.Timestamp(step.EndTime); err != nil {
			return errors.Annotate(err, "end_time").Err()
		}
	}

	_, stRequired := statusesWithStartTime[step.Status]
	_, isValidStatus := pb.Status_name[int32(step.Status)]
	switch {
	// This is the case, where the status field is set with an invalid int directly
	// inside the server code.
	case !isValidStatus:
		return errors.Reason("status: invalid status %d", int32(step.Status)).Err()
	// If a client sends a request with an invalid status num, then it is coerced to
	// the zero value. Thus, if the status is STATUS_UNSPECIFIED, it's either
	// unspecified or specified with an invalid value.
	case step.Status == pb.Status_STATUS_UNSPECIFIED:
		return errors.Reason("status: is unspecified or unknown").Err()
	case step.Status == pb.Status_ENDED_MASK:
		return errors.Reason("status: must not be ENDED_MASK").Err()
	case stRequired && st.IsZero():
		return errors.Reason("start_time: required by status %q", step.Status).Err()
	case step.Status < pb.Status_STARTED && !st.IsZero():
		return errors.Reason("start_time: must not be specified for status %q", step.Status).Err()
	case protoutil.IsEnded(step.Status) == et.IsZero():
		return errors.Reason("end_time: must have both or neither end_time and a terminal status").Err()
	case !et.IsZero() && et.Before(st):
		return errors.Reason("start_time: is after the end_time (%d > %d)", st.Unix(), et.Unix()).Err()
	}

	seen := stringset.New(len(step.Logs))
	for i, log := range step.Logs {
		switch {
		case log.GetName() == "":
			return errors.Reason("logs[%d].name: required", i).Err()
		case log.Url == "":
			return errors.Reason("logs[%d].url: required", i).Err()
		case log.ViewUrl == "":
			return errors.Reason("logs[%d].view_url: required", i).Err()
		case !seen.Add(log.Name):
			return errors.Reason("logs[%d].name: duplicate: %q", i, log.Name).Err()
		}
	}

	// TODO(ddoman): validate status with parent.
	return nil
}

// UpdateBuild handles a request to update a build. Implements pb.UpdateBuild.
func (*Builds) UpdateBuild(ctx context.Context, req *pb.UpdateBuildRequest) (*pb.Build, error) {
	var bs model.BuildSteps
	if err := validateUpdate(req, &bs); err != nil {
		return nil, appstatus.BadRequest(err)
	}
	bs.Build = datastore.KeyForObj(ctx, &model.Build{ID: req.Build.Id})

	return nil, appstatus.Errorf(codes.Unimplemented, "method not implemented")
}
