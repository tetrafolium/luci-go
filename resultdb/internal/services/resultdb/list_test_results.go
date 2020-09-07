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

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/pagination"
	"github.com/tetrafolium/luci-go/resultdb/internal/testresults"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

func validateListTestResultsRequest(req *pb.ListTestResultsRequest) error {
	if err := pbutil.ValidateInvocationName(req.GetInvocation()); err != nil {
		return errors.Annotate(err, "invocation").Err()
	}

	if err := pagination.ValidatePageSize(req.GetPageSize()); err != nil {
		return errors.Annotate(err, "page_size").Err()
	}

	return nil
}

// ListTestResults implements pb.ResultDBServer.
func (s *resultDBServer) ListTestResults(ctx context.Context, in *pb.ListTestResultsRequest) (*pb.ListTestResultsResponse, error) {
	if err := verifyPermissionInvNames(ctx, permListTestResults, in.Invocation); err != nil {
		return nil, err
	}

	if err := validateListTestResultsRequest(in); err != nil {
		return nil, appstatus.BadRequest(err)
	}

	readMask, err := testresults.ListMask(in.GetReadMask())
	if err != nil {
		return nil, appstatus.BadRequest(err)
	}

	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	q := testresults.Query{
		PageSize:      pagination.AdjustPageSize(in.PageSize),
		PageToken:     in.PageToken,
		InvocationIDs: invocations.NewIDSet(invocations.MustParseName(in.Invocation)),
		Mask:          readMask,
	}
	trs, tok, err := q.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.ListTestResultsResponse{
		TestResults:   trs,
		NextPageToken: tok,
	}, nil
}
