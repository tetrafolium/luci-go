// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package serviceaccounts

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth/signing"

	"github.com/tetrafolium/luci-go/tokenserver/api"
	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
)

// InspectOAuthTokenGrantRPC implements admin.InspectOAuthTokenGrant method.
type InspectOAuthTokenGrantRPC struct {
	// Signer is mocked in tests.
	//
	// In prod it is gaesigner.Signer.
	Signer signing.Signer

	// Rules returns service account rules to use for the request.
	//
	// In prod it is GlobalRulesCache.Rules.
	Rules func(context.Context) (*Rules, error)
}

// InspectOAuthTokenGrant decodes the given OAuth token grant.
func (r *InspectOAuthTokenGrantRPC) InspectOAuthTokenGrant(c context.Context, req *admin.InspectOAuthTokenGrantRequest) (*admin.InspectOAuthTokenGrantResponse, error) {
	inspection, err := InspectGrant(c, r.Signer, req.Token)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	resp := &admin.InspectOAuthTokenGrantResponse{
		Valid:            inspection.Signed && inspection.NonExpired,
		Signed:           inspection.Signed,
		NonExpired:       inspection.NonExpired,
		InvalidityReason: inspection.InvalidityReason,
	}

	addInvalidityReason := func(msg string, args ...interface{}) {
		reason := fmt.Sprintf(msg, args...)
		resp.Valid = false
		if resp.InvalidityReason == "" {
			resp.InvalidityReason = reason
		} else {
			resp.InvalidityReason += "; " + reason
		}
	}

	if env, _ := inspection.Envelope.(*tokenserver.OAuthTokenGrantEnvelope); env != nil {
		resp.SigningKeyId = env.KeyId
	}

	// Examine the body, even if the token is expired or unsigned. This helps to
	// debug expired or unsigned tokens...
	resp.TokenBody, _ = inspection.Body.(*tokenserver.OAuthTokenGrantBody)
	if resp.TokenBody != nil {
		rules, err := r.Rules(c)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load service accounts rules")
		}

		// Always return the rule that matches the service account, even if the
		// token itself is not allowed by it (we check it separately below).
		rule, err := rules.Rule(c, resp.TokenBody.ServiceAccount, "")
		switch {
		case status.Code(err) == codes.PermissionDenied:
			s, _ := status.FromError(err)
			addInvalidityReason("%s", s.Message())
			return resp, nil
		case err != nil: // usually codes.Internal
			return nil, err
		default:
			resp.MatchingRule = rule.Rule
		}

		q := &RulesQuery{
			ServiceAccount: resp.TokenBody.ServiceAccount,
			Rule:           rule,
			Proxy:          identity.Identity(resp.TokenBody.Proxy),
			EndUser:        identity.Identity(resp.TokenBody.EndUser),
		}
		switch _, err = rules.Check(c, q); {
		case err == nil:
			resp.AllowedByRules = true
		case status.Code(err) == codes.Internal:
			return nil, err // a transient error when checking rules
		default: // fatal gRPC error => the rules forbid the token
			addInvalidityReason("not allowed by the rules (rev %s)", rules.ConfigRevision())
		}
	}

	return resp, nil
}
