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
	"fmt"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"

	"github.com/tetrafolium/luci-go/buildbucket/appengine/internal/perm"
	"github.com/tetrafolium/luci-go/buildbucket/appengine/model"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
)

// validateGet validates the given request.
func validateGet(req *pb.GetBuildRequest) error {
	switch {
	case req.GetId() != 0:
		if req.Builder != nil || req.BuildNumber != 0 {
			return errors.Reason("id is mutually exclusive with (builder and build_number)").Err()
		}
	case req.GetBuilder() != nil && req.BuildNumber != 0:
		switch err := protoutil.ValidateBuilderID(req.Builder); {
		case err != nil:
			return errors.Annotate(err, "builder").Err()
		case req.Builder.Bucket == "":
			return errors.Annotate(errors.Reason("bucket is required").Err(), "builder").Err()
		case req.Builder.Builder == "":
			return errors.Annotate(errors.Reason("builder is required").Err(), "builder").Err()
		}
	default:
		return errors.Reason("one of id or (builder and build_number) is required").Err()
	}
	return nil
}

// GetBuild handles a request to retrieve a build. Implements pb.BuildsServer.
func (*Builds) GetBuild(ctx context.Context, req *pb.GetBuildRequest) (*pb.Build, error) {
	if err := validateGet(req); err != nil {
		return nil, appstatus.BadRequest(err)
	}
	m, err := getFieldMask(req.Fields)
	if err != nil {
		return nil, appstatus.BadRequest(errors.Annotate(err, "fields").Err())
	}
	if req.Id == 0 {
		addr := fmt.Sprintf("luci.%s.%s/%s/%d", req.Builder.Project, req.Builder.Bucket, req.Builder.Builder, req.BuildNumber)
		switch ents, err := model.SearchTagIndex(ctx, "build_address", addr); {
		case model.TagIndexIncomplete.In(err):
			// Shouldn't happen because build address is globally unique (exactly one entry in a complete index).
			return nil, errors.Reason("unexpected incomplete index for build address %q", addr).Err()
		case err != nil:
			return nil, err
		case len(ents) == 0:
			return nil, perm.NotFoundErr(ctx)
		case len(ents) == 1:
			req.Id = ents[0].BuildID
		default:
			// Shouldn't happen because build address is globally unique and created before the build.
			return nil, errors.Reason("unexpected number of results for build address %q: %d", addr, len(ents)).Err()
		}
	}

	bld, err := getBuild(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if err := perm.HasInBuilder(ctx, perm.BuildsGet, bld.Proto.Builder); err != nil {
		return nil, err
	}

	return bld.ToProto(ctx, m)
}
