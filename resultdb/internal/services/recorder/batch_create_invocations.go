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

package recorder

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// validateBatchCreateInvocationsRequest checks that the individual requests
// are valid, that they match the batch request requestID and that their names
// are not repeated.
func validateBatchCreateInvocationsRequest(
	now time.Time, reqs []*pb.CreateInvocationRequest, requestID string) (invocations.IDSet, error) {
	if err := pbutil.ValidateRequestID(requestID); err != nil {
		return nil, errors.Annotate(err, "request_id").Err()
	}

	if err := pbutil.ValidateBatchRequestCount(len(reqs)); err != nil {
		return nil, err
	}

	idSet := make(invocations.IDSet, len(reqs))
	for i, req := range reqs {
		if err := validateCreateInvocationRequest(req, now); err != nil {
			return nil, errors.Annotate(err, "requests[%d]", i).Err()
		}

		// If there's multiple `CreateInvocationRequest`s their request id
		// must either be empty or match the one in the batch request.
		if req.RequestId != "" && req.RequestId != requestID {
			return nil, errors.Reason("requests[%d].request_id: %q does not match request_id %q", i, requestID, req.RequestId).Err()
		}

		invID := invocations.ID(req.InvocationId)
		if idSet.Has(invID) {
			return nil, errors.Reason("requests[%d].invocation_id: duplicated invocation id %q", i, req.InvocationId).Err()
		}
		idSet.Add(invID)
	}
	return idSet, nil
}

// BatchCreateInvocations implements pb.RecorderServer.
func (s *recorderServer) BatchCreateInvocations(ctx context.Context, in *pb.BatchCreateInvocationsRequest) (*pb.BatchCreateInvocationsResponse, error) {
	now := clock.Now(ctx).UTC()
	for i, r := range in.Requests {
		if err := verifyCreateInvocationPermissions(ctx, r); err != nil {
			return nil, errors.Annotate(err, "requests[%d]", i).Err()
		}

	}

	idSet, err := validateBatchCreateInvocationsRequest(now, in.Requests, in.RequestId)
	if err != nil {
		return nil, appstatus.BadRequest(err)
	}

	invs, tokens, err := s.createInvocations(ctx, in.Requests, in.RequestId, now, idSet)
	if err != nil {
		return nil, err
	}
	return &pb.BatchCreateInvocationsResponse{Invocations: invs, UpdateTokens: tokens}, nil
}

// createInvocations is a shared implementation for CreateInvocation and BatchCreateInvocations RPCs.
func (s *recorderServer) createInvocations(ctx context.Context, reqs []*pb.CreateInvocationRequest, requestID string, now time.Time, idSet invocations.IDSet) ([]*pb.Invocation, []string, error) {
	createdBy := string(auth.CurrentIdentity(ctx))
	ms := s.createInvocationsRequestsToMutations(ctx, now, reqs, requestID, createdBy)

	var err error
	deduped := false
	_, err = span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		deduped, err = deduplicateCreateInvocations(ctx, idSet, requestID, createdBy)
		if err != nil {
			return err
		}
		if !deduped {
			span.BufferWrite(ctx, ms...)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if !deduped {
		for _, r := range reqs {
			spanutil.IncRowCount(ctx, 1, spanutil.Invocations, spanutil.Inserted, r.Invocation.GetRealm())
		}
	}

	return getCreatedInvocationsAndUpdateTokens(ctx, idSet, reqs)
}

// createInvocationsRequestsToMutations computes a database mutation for
// inserting a row for each invocation creation requested.
func (s *recorderServer) createInvocationsRequestsToMutations(ctx context.Context, now time.Time, reqs []*pb.CreateInvocationRequest, requestID, createdBy string) []*spanner.Mutation {

	ms := make([]*spanner.Mutation, len(reqs))
	// Compute mutations
	for i, req := range reqs {

		// Prepare the invocation we will save to spanner.
		inv := &pb.Invocation{
			Name:             invocations.ID(req.InvocationId).Name(),
			State:            pb.Invocation_ACTIVE,
			Deadline:         req.Invocation.GetDeadline(),
			Tags:             req.Invocation.GetTags(),
			BigqueryExports:  req.Invocation.GetBigqueryExports(),
			CreatedBy:        createdBy,
			ProducerResource: req.Invocation.GetProducerResource(),
			Realm:            req.Invocation.GetRealm(),
		}

		// Ensure the invocation has a deadline.
		if inv.Deadline == nil {
			inv.Deadline = pbutil.MustTimestampProto(now.Add(defaultInvocationDeadlineDuration))
		}

		pbutil.NormalizeInvocation(inv)
		// Create a mutation to create the invocation.
		ms[i] = spanutil.InsertMap("Invocations", s.rowOfInvocation(ctx, inv, requestID))
	}
	return ms
}

// getCreatedInvocationsAndUpdateTokens reads the full details of the
// invocations just created in a separate read-only transaction, and
// generates an update token for each.
func getCreatedInvocationsAndUpdateTokens(ctx context.Context, idSet invocations.IDSet, reqs []*pb.CreateInvocationRequest) ([]*pb.Invocation, []string, error) {
	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	invMap, err := invocations.ReadBatch(ctx, idSet)
	if err != nil {
		return nil, nil, err
	}

	// Arrange them in same order as the incoming requests.
	// Ordering is important to match the tokens.
	invs := make([]*pb.Invocation, len(reqs))
	for i, req := range reqs {
		invs[i] = invMap[invocations.ID(req.InvocationId)]
	}

	tokens, err := generateTokens(ctx, invs)
	if err != nil {
		return nil, nil, err
	}
	return invs, tokens, nil
}

// deduplicateCreateInvocations checks if the invocations have already been
// created with the given requestID and current requester.
// Returns a true if they have.
func deduplicateCreateInvocations(ctx context.Context, idSet invocations.IDSet, requestID, createdBy string) (bool, error) {
	invCount := 0
	columns := []string{"InvocationId", "CreateRequestId", "CreatedBy"}
	err := span.Read(ctx, "Invocations", idSet.Keys(), columns).Do(func(r *spanner.Row) error {
		var invID invocations.ID
		var rowRequestID spanner.NullString
		var rowCreatedBy spanner.NullString
		switch err := spanutil.FromSpanner(r, &invID, &rowRequestID, &rowCreatedBy); {
		case err != nil:
			return err
		case !rowRequestID.Valid || rowRequestID.StringVal != requestID:
			return invocationAlreadyExists(invID)
		case rowCreatedBy.StringVal != createdBy:
			return invocationAlreadyExists(invID)
		default:
			invCount++
			return nil
		}
	})
	switch {
	case err != nil:
		return false, err
	case invCount == len(idSet):
		// All invocations were previously created with this request id.
		return true, nil
	case invCount == 0:
		// None of the invocations exist already.
		return false, nil
	default:
		// Could happen if someone sent two different but overlapping batch create
		// requests, but reused the request_id.
		return false, appstatus.Errorf(codes.AlreadyExists, "some, but not all of the invocations already created with this request id")
	}
}

// generateTokens generates an update token for each invocation.
func generateTokens(ctx context.Context, invs []*pb.Invocation) ([]string, error) {
	ret := make([]string, len(invs))
	for i, inv := range invs {
		updateToken, err := generateInvocationToken(ctx, invocations.MustParseName(inv.Name))
		if err != nil {
			return nil, err
		}
		ret[i] = updateToken
	}
	return ret, nil
}
