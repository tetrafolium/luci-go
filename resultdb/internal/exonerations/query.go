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

package exonerations

import (
	"context"

	"cloud.google.com/go/spanner"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/pagination"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// Query specifies test exonerations to fetch.
type Query struct {
	InvocationIDs invocations.IDSet
	Predicate     *pb.TestExonerationPredicate
	PageSize      int // must be positive
	PageToken     string
}

// Fetch returns a page test of exonerations matching the query.
// Returned test exonerations are ordered by invocation ID, test ID and
// exoneration ID.
func (q *Query) Fetch(ctx context.Context) (tes []*pb.TestExoneration, nextPageToken string, err error) {
	if q.PageSize <= 0 {
		panic("PageSize <= 0")
	}

	st := spanner.NewStatement(`
		SELECT InvocationId, TestId, ExonerationId, Variant, VariantHash, ExplanationHtml
		FROM TestExonerations
		WHERE InvocationId IN UNNEST(@invIDs)
			# Skip test exonerations after the one specified in the page token.
			AND (
				(InvocationId > @afterInvocationId) OR
				(InvocationId = @afterInvocationId AND TestId > @afterTestId) OR
				(InvocationId = @afterInvocationId AND TestId = @afterTestId AND ExonerationID > @afterExonerationID)
		  )
		ORDER BY InvocationId, TestId, ExonerationId
		LIMIT @limit
	`)
	st.Params["invIDs"] = q.InvocationIDs
	st.Params["limit"] = q.PageSize
	err = invocations.TokenToMap(q.PageToken, st.Params, "afterInvocationId", "afterTestId", "afterExonerationID")
	if err != nil {
		return
	}

	// TODO(nodir): add support for q.Predicate.TestId.
	// TODO(nodir): add support for q.Predicate.Variant.

	tes = make([]*pb.TestExoneration, 0, q.PageSize)
	var b spanutil.Buffer
	var explanationHTML spanutil.Compressed
	err = spanutil.Query(ctx, st, func(row *spanner.Row) error {
		var invID invocations.ID
		ex := &pb.TestExoneration{}
		err := b.FromSpanner(row, &invID, &ex.TestId, &ex.ExonerationId, &ex.Variant, &ex.VariantHash, &explanationHTML)
		if err != nil {
			return err
		}
		ex.Name = pbutil.TestExonerationName(string(invID), ex.TestId, ex.ExonerationId)
		ex.ExplanationHtml = string(explanationHTML)
		tes = append(tes, ex)
		return nil
	})
	if err != nil {
		tes = nil
		return
	}

	// If we got pageSize results, then we haven't exhausted the collection and
	// need to return the next page token.
	if len(tes) == q.PageSize {
		last := tes[q.PageSize-1]
		invID, testID, exID := MustParseName(last.Name)
		nextPageToken = pagination.Token(string(invID), testID, exID)
	}
	return
}
