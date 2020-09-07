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
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// validateCreateTestExonerationRequest returns a non-nil error if req is invalid.
func validateCreateTestExonerationRequest(req *pb.CreateTestExonerationRequest, requireInvocation bool) error {
	if requireInvocation || req.Invocation != "" {
		if err := pbutil.ValidateInvocationName(req.Invocation); err != nil {
			return errors.Annotate(err, "invocation").Err()
		}
	}

	ex := req.GetTestExoneration()
	if err := pbutil.ValidateTestID(ex.GetTestId()); err != nil {
		return errors.Annotate(err, "test_exoneration: test_id").Err()
	}
	if err := pbutil.ValidateVariant(ex.GetVariant()); err != nil {
		return errors.Annotate(err, "test_exoneration: variant").Err()
	}

	if err := pbutil.ValidateRequestID(req.RequestId); err != nil {
		return errors.Annotate(err, "request_id").Err()
	}
	return nil
}

// CreateTestExoneration implements pb.RecorderServer.
func (s *recorderServer) CreateTestExoneration(ctx context.Context, in *pb.CreateTestExonerationRequest) (*pb.TestExoneration, error) {
	if err := validateCreateTestExonerationRequest(in, true); err != nil {
		return nil, appstatus.BadRequest(err)
	}
	invID := invocations.MustParseName(in.Invocation)

	ret, mutation := insertTestExoneration(ctx, invID, in.RequestId, 0, in.TestExoneration)
	err := mutateInvocation(ctx, invID, func(ctx context.Context) error {
		span.BufferWrite(ctx, mutation)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func insertTestExoneration(ctx context.Context, invID invocations.ID, requestID string, ordinal int, body *pb.TestExoneration) (ret *pb.TestExoneration, mutation *spanner.Mutation) {
	// Compute exoneration ID and choose Insert vs InsertOrUpdate.
	var exonerationIDSuffix string
	mutFn := spanner.InsertMap
	if requestID == "" {
		// Use a random id.
		exonerationIDSuffix = "r:" + uuid.New().String()
	} else {
		// Use a deterministic id.
		exonerationIDSuffix = "d:" + deterministicExonerationIDSuffix(ctx, requestID, ordinal)
		mutFn = spanner.InsertOrUpdateMap
	}

	exonerationID := fmt.Sprintf("%s:%s", pbutil.VariantHash(body.Variant), exonerationIDSuffix)
	ret = &pb.TestExoneration{
		Name:            pbutil.TestExonerationName(string(invID), body.TestId, exonerationID),
		TestId:          body.TestId,
		Variant:         body.Variant,
		ExonerationId:   exonerationID,
		ExplanationHtml: body.ExplanationHtml,
	}
	mutation = mutFn("TestExonerations", spanutil.ToSpannerMap(map[string]interface{}{
		"InvocationId":    invID,
		"TestId":          ret.TestId,
		"ExonerationId":   exonerationID,
		"Variant":         ret.Variant,
		"VariantHash":     pbutil.VariantHash(ret.Variant),
		"ExplanationHTML": spanutil.Compressed(ret.ExplanationHtml),
	}))
	return
}

func deterministicExonerationIDSuffix(ctx context.Context, requestID string, ordinal int) string {
	h := sha512.New()
	// Include current identity, so that two separate clients
	// do not override each other's test exonerations even if
	// they happened to produce identical request ids.
	// The alternative is to use remote IP address, but it is not
	// implemented in pRPC.
	fmt.Fprintln(h, auth.CurrentIdentity(ctx))
	fmt.Fprintln(h, requestID)
	fmt.Fprintln(h, ordinal)
	return hex.EncodeToString(h.Sum(nil))
}
