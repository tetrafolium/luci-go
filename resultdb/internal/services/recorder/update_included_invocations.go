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

package recorder

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// validateUpdateIncludedInvocationsRequest returns a non-nil error if req is
// determined to be invalid.
func validateUpdateIncludedInvocationsRequest(req *pb.UpdateIncludedInvocationsRequest) error {
	if _, err := pbutil.ParseInvocationName(req.IncludingInvocation); err != nil {
		return errors.Annotate(err, "including_invocation").Err()
	}
	for _, name := range req.AddInvocations {
		if name == req.IncludingInvocation {
			return errors.Reason("cannot include itself").Err()
		}
		if _, err := pbutil.ParseInvocationName(name); err != nil {
			return errors.Annotate(err, "add_invocations: %q", name).Err()
		}
	}

	for _, name := range req.RemoveInvocations {
		if _, err := pbutil.ParseInvocationName(name); err != nil {
			return errors.Annotate(err, "remove_invocations: %q", name).Err()
		}
	}

	both := stringset.NewFromSlice(req.AddInvocations...).Intersect(stringset.NewFromSlice(req.RemoveInvocations...)).ToSortedSlice()
	if len(both) > 0 {
		return errors.Reason("cannot add and remove the same invocation(s) at the same time: %q", both).Err()
	}
	return nil
}

// UpdateIncludedInvocations implements pb.RecorderServer.
func (s *recorderServer) UpdateIncludedInvocations(ctx context.Context, in *pb.UpdateIncludedInvocationsRequest) (*empty.Empty, error) {
	if err := validateUpdateIncludedInvocationsRequest(in); err != nil {
		return nil, appstatus.BadRequest(err)
	}
	including := invocations.MustParseName(in.IncludingInvocation)
	add := invocations.MustParseNames(in.AddInvocations)
	remove := invocations.MustParseNames(in.RemoveInvocations)

	err := mutateInvocation(ctx, including, func(ctx context.Context) error {
		// Accumulate keys to remove in a single KeySet.
		ks := spanner.KeySets()
		for rInv := range remove {
			ks = spanner.KeySets(invocations.InclusionKey(including, rInv), ks)
		}
		ms := make([]*spanner.Mutation, 1, 1+len(add))
		ms[0] = spanner.Delete("IncludedInvocations", ks)

		switch states, err := invocations.ReadStateBatch(ctx, add); {
		case err != nil:
			return err
		// Ensure every included invocation exists.
		case len(states) != len(add):
			return appstatus.Errorf(codes.NotFound, "at least one of the included invocations does not exist")
		}
		for aInv := range add {
			ms = append(ms, spanutil.InsertOrUpdateMap("IncludedInvocations", map[string]interface{}{
				"InvocationId":         including,
				"IncludedInvocationId": aInv,
			}))
		}
		span.BufferWrite(ctx, ms...)
		return nil
	})

	return &empty.Empty{}, err
}
