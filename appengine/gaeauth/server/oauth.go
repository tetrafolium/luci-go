// Copyright 2015 The LUCI Authors.
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

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"github.com/tetrafolium/luci-go/gae/service/info"
	"github.com/tetrafolium/luci-go/gae/service/user"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/server/auth"
)

var errBadAuthHeader = errors.New("oauth: bad Authorization header")

// EmailScope is a scope used to identifies user's email. Present in most tokens
// by default. Can be used as a base scope for authentication.
const EmailScope = "https://www.googleapis.com/auth/userinfo.email"

// OAuth2Method implements auth.Method on top of GAE OAuth2 API. It doesn't
// implement auth.UsersAPI.
type OAuth2Method struct {
	// Scopes is a list of OAuth scopes to check when authenticating the token.
	Scopes []string
}

var _ auth.UserCredentialsGetter = (*OAuth2Method)(nil)

// Authenticate extracts peer's identity from the incoming request.
func (m *OAuth2Method) Authenticate(ctx context.Context, r *http.Request) (*auth.User, error) {
	if info.IsDevAppServer(ctx) {
		// On "dev_appserver", we verify OAuth2 tokens using Google's OAuth2
		// verification endpoint.
		//
		// It is slow as hell on "dev_appserver", but good enough for local manual
		// testing.
		devMethod := auth.GoogleOAuth2Method{
			Scopes: m.Scopes,
		}
		return devMethod.Authenticate(ctx, r)
	}

	header := r.Header.Get("Authorization")
	if header == "" || len(m.Scopes) == 0 {
		return nil, nil // this method is not applicable
	}

	// GetOAuthUser RPC is notoriously flaky. Do a bunch of retries on errors.
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		var u *user.User
		u, err = user.CurrentOAuth(ctx, m.Scopes...)
		if err != nil {
			if isFatalOAuthErr(err) {
				return nil, err
			}
			logging.Warningf(ctx, "oauth: failed to execute GetOAuthUser - %s", err)
			continue
		}
		if u.ClientID == "" {
			return nil, fmt.Errorf("oauth: ClientID is unexpectedly empty")
		}
		id, idErr := identity.MakeIdentity("user:" + u.Email)
		if idErr != nil {
			return nil, idErr
		}
		return &auth.User{
			Identity:  id,
			Superuser: u.Admin,
			Email:     u.Email,
			ClientID:  u.ClientID,
		}, nil
	}
	return nil, transient.Tag.Apply(err)
}

// isFatalOAuthErr examines the error message to figure out whether the error
// from the OAuth service is fatal.
func isFatalOAuthErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "OAUTH_INVALID_TOKEN") || strings.Contains(msg, "OAUTH_INVALID_REQUEST")
}

// GetUserCredentials implements auth.UserCredentialsGetter.
func (m *OAuth2Method) GetUserCredentials(ctx context.Context, r *http.Request) (*oauth2.Token, error) {
	chunks := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(chunks) != 2 || (chunks[0] != "OAuth" && chunks[0] != "Bearer") {
		return nil, errBadAuthHeader
	}
	return &oauth2.Token{
		AccessToken: chunks[1],
		TokenType:   "Bearer",
	}, nil
}
