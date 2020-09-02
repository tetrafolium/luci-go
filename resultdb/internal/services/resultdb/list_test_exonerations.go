// Copyright 2019 The LUCI Authors.
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

package resultdb

import (
	"context"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/exonerations"
	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/pagination"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

func validateListTestExonerationsRequest(req *pb.ListTestExonerationsRequest) error {
	if err := pbutil.ValidateInvocationName(req.GetInvocation()); err != nil {
		return errors.Annotate(err, "invocation").Err()
	}

	if err := pagination.ValidatePageSize(req.GetPageSize()); err != nil {
		return errors.Annotate(err, "page_size").Err()
	}

	return nil
}

// ListTestExonerations implements pb.ResultDBServer.
func (s *resultDBServer) ListTestExonerations(ctx context.Context, in *pb.ListTestExonerationsRequest) (*pb.ListTestExonerationsResponse, error) {
	if err := verifyPermissionInvNames(ctx, permListTestExonerations, in.Invocation); err != nil {
		return nil, err
	}

	if err := validateListTestExonerationsRequest(in); err != nil {
		return nil, appstatus.BadRequest(err)
	}

	q := exonerations.Query{
		InvocationIDs: invocations.NewIDSet(invocations.MustParseName(in.Invocation)),
		PageSize:      pagination.AdjustPageSize(in.PageSize),
		PageToken:     in.GetPageToken(),
	}

	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()
	tes, tok, err := q.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.ListTestExonerationsResponse{
		TestExonerations: tes,
		NextPageToken:    tok,
	}, nil
}
