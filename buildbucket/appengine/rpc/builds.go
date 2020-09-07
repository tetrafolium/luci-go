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
	"regexp"

	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/protobuf/field_mask"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/mask"
	"github.com/tetrafolium/luci-go/grpc/appstatus"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
)

var sha1Regex = regexp.MustCompile(`^[a-f0-9]{40}$`)

// defMask is the default field mask to use for GetBuild requests.
var defMask = mask.MustFromReadMask(&pb.Build{},
	"builder",
	"canary",
	"create_time",
	"created_by",
	"critical",
	"end_time",
	"id",
	"input.experimental",
	"input.gerrit_changes",
	"input.gitiles_commit",
	"number",
	"start_time",
	"status",
	"status_details",
	"update_time",
)

// TODO(crbug/1042991): Move to a common location.
func getFieldMask(fields *field_mask.FieldMask) (mask.Mask, error) {
	if len(fields.GetPaths()) == 0 {
		return defMask, nil
	}
	return mask.FromFieldMask(fields, &pb.Build{}, false, false)
}

// getBuildsSubMask returns the sub mask for "builds.*"
func getBuildsSubMask(fields *field_mask.FieldMask) (mask.Mask, error) {
	if len(fields.GetPaths()) == 0 {
		return defMask, nil
	}
	m, err := mask.FromFieldMask(fields, &pb.SearchBuildsResponse{}, false, false)
	if err != nil {
		return mask.Mask{}, err
	}
	return m.Submask("builds.*")
}

// buildsServicePostlude logs the method called, the proto response, and any
// error, but returns that the called method was unimplemented. Used to aid in
// development. Users of this function must ensure called methods do not have
// any side-effects. When removing this function, remember to ensure all methods
// have correct ACLs checks.
// TODO(crbug/1042991): Remove once methods are implemented.
func buildsServicePostlude(ctx context.Context, methodName string, rsp proto.Message, err error) error {
	err = commonPostlude(ctx, methodName, rsp, err)
	if methodName == "GetBuild" {
		return err
	}
	logging.Debugf(ctx, "%q would have returned %q with response %s", methodName, err, proto.MarshalTextString(rsp))
	return status.Errorf(codes.Unimplemented, "method not implemented")
}

// Builds implements pb.BuildsServer.
type Builds struct {
}

// Ensure Builds implements projects.ProjectsServer.
var _ pb.BuildsServer = &Builds{}

// Batch handles a batch request. Implements pb.BuildsServer.
func (*Builds) Batch(ctx context.Context, req *pb.BatchRequest) (*pb.BatchResponse, error) {
	return nil, appstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// ScheduleBuild handles a request to schedule a build. Implements pb.BuildsServer.
func (*Builds) ScheduleBuild(ctx context.Context, req *pb.ScheduleBuildRequest) (*pb.Build, error) {
	return nil, appstatus.Errorf(codes.Unimplemented, "method not implemented")
}

// NewBuilds returns a new pb.BuildsServer.
func NewBuilds() pb.BuildsServer {
	return &pb.DecoratedBuilds{
		Prelude:  logDetails,
		Service:  &Builds{},
		Postlude: buildsServicePostlude,
	}
}
