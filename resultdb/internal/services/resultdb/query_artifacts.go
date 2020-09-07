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

package resultdb

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/ptypes"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/artifacts"
	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/pagination"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// validateQueryArtifactsRequest returns a non-nil error if req is determined
// to be invalid.
func validateQueryArtifactsRequest(req *pb.QueryArtifactsRequest) error {
	if err := pbutil.ValidateTestResultPredicate(req.TestResultPredicate); err != nil {
		return errors.Annotate(err, "test_result_predicate").Err()
	}
	return validateQueryRequest(req)
}

// QueryArtifacts implements pb.ResultDBServer.
func (s *resultDBServer) QueryArtifacts(ctx context.Context, in *pb.QueryArtifactsRequest) (*pb.QueryArtifactsResponse, error) {
	if err := verifyPermissionInvNames(ctx, permListArtifacts, in.Invocations...); err != nil {
		return nil, err
	}

	if err := validateQueryArtifactsRequest(in); err != nil {
		return nil, appstatus.BadRequest(err)
	}

	// Open a transaction.
	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()
	if in.MaxStaleness != nil {
		st, _ := ptypes.Duration(in.MaxStaleness)
		span.RO(ctx).WithTimestampBound(spanner.MaxStaleness(st))
	}

	// Get the transitive closure.
	invs, err := invocations.Reachable(ctx, invocations.MustParseNames(in.Invocations))
	if err != nil {
		return nil, err
	}

	// Query artifacts.
	q := artifacts.Query{
		InvocationIDs:       invs,
		TestResultPredicate: in.TestResultPredicate,
		PageSize:            pagination.AdjustPageSize(in.PageSize),
		PageToken:           in.PageToken,
		FollowEdges:         in.FollowEdges,
	}
	arts, token, err := q.Fetch(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.populateFetchURLs(ctx, arts...); err != nil {
		return nil, err
	}

	return &pb.QueryArtifactsResponse{
		Artifacts:     arts,
		NextPageToken: token,
	}, nil
}
