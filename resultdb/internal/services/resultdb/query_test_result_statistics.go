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

	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// QueryTestResultStatistics implements pb.ResultDBServer.
func (s *resultDBServer) QueryTestResultStatistics(ctx context.Context, in *pb.QueryTestResultStatisticsRequest) (*pb.QueryTestResultStatisticsResponse, error) {
	if err := verifyPermissionInvNames(ctx, permListTestResults, in.Invocations...); err != nil {
		return nil, err
	}

	if err := validateQueryRequest(in); err != nil {
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

	totalNum, err := invocations.ReadTestResultCount(ctx, invs)
	if err != nil {
		return nil, err
	}

	return &pb.QueryTestResultStatisticsResponse{
		TotalTestResults: totalNum,
	}, nil
}
