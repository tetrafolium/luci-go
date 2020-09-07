// Copyright 2017 The LUCI Authors.
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

package openid

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
)

// IDToken is verified deserialized ID token.
//
// See https://developers.google.com/identity/protocols/OpenIDConnect.
type IDToken struct {
	Iss           string `json:"iss"`
	AtHash        string `json:"at_hash"`
	EmailVerified bool   `json:"email_verified"`
	Sub           string `json:"sub"`
	Azp           string `json:"azp"`
	Email         string `json:"email"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Name          string `json:"name"`
	Aud           string `json:"aud"`
	Iat           int64  `json:"iat"`
	Exp           int64  `json:"exp"`
	Nonce         string `json:"nonce"`
	Hd            string `json:"hd"`
}

const allowedClockSkew = 30 * time.Second

// VerifyIDToken deserializes and verifies the ID token.
//
// It checks the signature, expiration time and verifies fields `iss` and
// `email_verified`.
//
// It checks `aud` and `sub` are present, but does NOT verify them any further.
// It is the caller's responsibility to do so.
//
// This is a fast local operation.
func VerifyIDToken(ctx context.Context, token string, keys *JSONWebKeySet, issuer string) (*IDToken, error) {
	// See https://developers.google.com/identity/protocols/OpenIDConnect#validatinganidtoken

	body, err := keys.VerifyJWT(token)
	if err != nil {
		return nil, err
	}
	tok := &IDToken{}
	if err := json.Unmarshal(body, tok); err != nil {
		return nil, errors.Annotate(err, "bad ID token - not JSON").Err()
	}

	exp := time.Unix(tok.Exp, 0)
	now := clock.Now(ctx)

	switch {
	case tok.Iss != issuer && "https://"+tok.Iss != issuer:
		return nil, errors.Reason("bad ID token - expecting issuer %q, got %q", issuer, tok.Iss).Err()
	case exp.Add(allowedClockSkew).Before(now):
		return nil, errors.Reason("bad ID token - expired %s ago", now.Sub(exp)).Err()
	case !tok.EmailVerified:
		return nil, errors.Reason("bad ID token - the email %q is not verified", tok.Email).Err()
	case tok.Aud == "":
		return nil, errors.Reason("bad ID token - the audience is missing").Err()
	case tok.Sub == "":
		return nil, errors.Reason("bad ID token - the subject is missing").Err()
	}

	return tok, nil
}
