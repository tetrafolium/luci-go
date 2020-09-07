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

package openid

import (
	"context"
	"net/http"
	"strings"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/server/auth"
)

// googleDiscoveryURL is URL of Google OpenID Connect discovery document.
const googleDiscoveryURL = "https://accounts.google.com/.well-known/openid-configuration"

// GoogleIDTokenAuthMethod implements auth.Method by checking `Authorization`
// header which is expected to have an OpenID Connect ID token signed by Google.
//
// The header value should have form "Bearer <base64 JWT>".
//
// There are two variants of tokens signed by Google:
//   * ID tokens identifying end users. They always have an OAuth2 Client ID as
//     an audience (`aud` field). Their `aud` is placed into User.ClientID, so
//     it is later checked against a whitelist of client IDs by the LUCI auth
//     stack.
//   * ID tokens identifying service accounts. They generally can have anything
//     at all as an audience, but usually have an URL of the service being
//     called. Their `aud` is checked against Audience list below.
type GoogleIDTokenAuthMethod struct {
	// Audience is a list of allowed audiences for tokens that identify Google
	// service accounts ("*.gserviceaccount.com" emails).
	Audience []string

	// AudienceCheck is an optional callback to use to check tokens audience in
	// case enumerating all expected audiences is not viable.
	//
	// Works in conjunction with Audience. Also, just like Audience, this check is
	// used only for tokens that identify service accounts.
	AudienceCheck func(ctx context.Context, r *http.Request, aud string) (valid bool, err error)

	// discoveryURL is used in tests to override googleDiscoveryURL.
	discoveryURL string
}

// Make sure all extra interfaces are implemented.
var _ interface {
	auth.Method
	auth.Warmable
} = (*GoogleIDTokenAuthMethod)(nil)

// AudienceMatchesHost can be used as a AudienceCheck callback.
//
// It verifies token's audience matches "Host" request header. Suitable for
// environments where "Host" header can be trusted.
func AudienceMatchesHost(ctx context.Context, r *http.Request, aud string) (valid bool, err error) {
	valid = aud == "https://"+r.Host || strings.HasPrefix(aud, "https://"+r.Host+"/")
	return
}

// Authenticate extracts user information from the incoming request.
//
// It returns:
//   * (*User, nil) on success.
//   * (nil, nil) if the method is not applicable.
//   * (nil, error) if the method is applicable, but credentials are invalid.
func (m *GoogleIDTokenAuthMethod) Authenticate(ctx context.Context, r *http.Request) (*auth.User, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, nil // this auth method is not applicable
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))

	// Grab (usually already cached) discovery document.
	doc, err := m.discoveryDoc(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "openid: failed to fetch the OpenID discovery doc").Err()
	}

	// Validate token's signature and expiration. Extract user info from it.
	tok, user, err := userFromIDToken(ctx, token, doc)
	if err != nil {
		return nil, err
	}

	// For tokens identifying end users, populate user.ClientID to let the LUCI
	// auth stack check it against a whitelist in the AuthDB (same way it does for
	// OAuth2 access tokens).
	if !strings.HasSuffix(user.Email, ".gserviceaccount.com") {
		user.ClientID = tok.Aud
		return user, nil
	}

	// For service accounts we want to check `aud` right here, since it is
	// generally not an OAuth Client ID and can be anything at all.
	for _, aud := range m.Audience {
		if tok.Aud == aud {
			return user, nil
		}
	}
	if m.AudienceCheck != nil {
		switch valid, err := m.AudienceCheck(ctx, r, tok.Aud); {
		case err != nil:
			return nil, err
		case valid:
			return user, nil
		}
	}

	logging.Errorf(ctx, "openid: token from %s has unrecognized audience %q", user.Email, tok.Aud)
	return nil, auth.ErrBadAudience
}

// Warmup prepares local caches. It's optional.
//
// Implements auth.Warmable.
func (m *GoogleIDTokenAuthMethod) Warmup(ctx context.Context) error {
	_, err := m.discoveryDoc(ctx)
	return err
}

// discoveryDoc fetches (and caches) the discovery document.
func (m *GoogleIDTokenAuthMethod) discoveryDoc(ctx context.Context) (*discoveryDoc, error) {
	url := googleDiscoveryURL
	if m.discoveryURL != "" {
		url = m.discoveryURL
	}
	return fetchDiscoveryDoc(ctx, url)
}
