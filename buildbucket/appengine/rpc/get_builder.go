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

	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"

	"github.com/tetrafolium/luci-go/buildbucket/appengine/internal/perm"
	"github.com/tetrafolium/luci-go/buildbucket/appengine/model"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
)

// validateGetBuilder validates the given request.
func validateGetBuilder(req *pb.GetBuilderRequest) error {
	if err := protoutil.ValidateBuilderID(req.Id); err != nil {
		return errors.Annotate(err, "id").Err()
	}

	return nil
}

// GetBuilder handles a request to retrieve a builder. Implements pb.BuildersServer.
func (*Builders) GetBuilder(ctx context.Context, req *pb.GetBuilderRequest) (*pb.BuilderItem, error) {
	if err := validateGetBuilder(req); err != nil {
		return nil, appstatus.BadRequest(err)
	}

	if err := perm.HasInBuilder(ctx, perm.BuildersGet, req.Id); err != nil {
		return nil, err
	}

	builder := &model.Builder{
		Parent: model.BucketKey(ctx, req.Id.Project, req.Id.Bucket),
		ID:     req.Id.Builder,
	}
	switch err := datastore.Get(ctx, builder); {
	case err == datastore.ErrNoSuchEntity:
		return nil, perm.NotFoundErr(ctx)
	case err != nil:
		return nil, err
	}

	return &pb.BuilderItem{
		Id:     req.Id,
		Config: &builder.Config,
	}, nil
}
