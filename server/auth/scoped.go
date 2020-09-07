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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/trace"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/caching"

	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"
)

const (
	// MaxScopedTokenTTL is maximum allowed token lifetime that can be
	// requested via MintScopedToken.
	MaxScopedTokenTTL = 15 * time.Minute
)

// scopedTokenMinterClient is subset of minter.TokenMinterClient we use.
type scopedTokenMinterClient interface {
	MintProjectToken(context.Context, *minter.MintProjectTokenRequest, ...grpc.CallOption) (*minter.MintProjectTokenResponse, error)
}

// ProjectTokenParams defines the parameters to create project scoped service account OAuth2 tokens.
type ProjectTokenParams struct {
	// LuciProject is the name of the LUCI project for which a token will be obtained.
	LuciProject string

	// OAuthScopes resemble the requested OAuth scopes for which the token is valid.
	OAuthScopes []string

	// MinTTL defines a minimally acceptable token lifetime.
	//
	// The returned token will be valid for at least MinTTL, but no longer than
	// MaxScopedTokenTTL (which is 15min).
	//
	// Default is 2 min.
	MinTTL time.Duration

	// rpcClient is token server RPC client to use.
	//
	// Mocked in tests.
	rpcClient scopedTokenMinterClient
}

// scopedTokenCache is used to store project scoped tokens in the cache.
//
// The token is stored in OAuth2Token field.
var scopedTokenCache = newTokenCache(tokenCacheConfig{
	Kind:                         "scoped",
	Version:                      2,
	ProcessLRUCache:              caching.RegisterLRUCache(8192),
	ExpiryRandomizationThreshold: MaxScopedTokenTTL / 10, // 10%
})

// MintProjectToken returns a LUCI project-scoped OAuth2 token that can be used
// to access external resources on behalf of the project.
//
// It protects against accidental cross-project resource access. A token
// is targeted to some single specific LUCI project. The token is cached
// internally. Same token may be returned by multiple calls, if its lifetime
// allows.
func MintProjectToken(ctx context.Context, p ProjectTokenParams) (_ *Token, err error) {
	ctx, span := trace.StartSpan(ctx, "github.com/tetrafolium/luci-go/server/auth.MintProjectToken")
	span.Attribute("cr.dev/project", p.LuciProject)
	defer func() { span.End(err) }()

	report := durationReporter(ctx, mintProjectTokenDuration)

	// Validate TTL is sane.
	if p.MinTTL == 0 {
		p.MinTTL = 2 * time.Minute
	}
	if p.MinTTL < 30*time.Second || p.MinTTL > MaxScopedTokenTTL {
		report(ErrBadTokenTTL, "ERROR_BAD_TTL")
		return nil, ErrBadTokenTTL
	}

	// Config contains the cache implementation.
	cfg := getConfig(ctx)
	if cfg == nil {
		report(ErrNotConfigured, "ERROR_NOT_CONFIGURED")
		return nil, ErrNotConfigured
	}

	// The state carries ID of the current user and URL of the token service.
	state := GetState(ctx)
	if state == nil {
		report(ErrNotConfigured, "ERROR_NO_AUTH_STATE")
		return nil, ErrNotConfigured
	}

	// Grab hostname of the token service we received from the auth service.
	tokenServiceURL, err := state.DB().GetTokenServiceURL(ctx)
	switch {
	case err != nil:
		report(err, "ERROR_AUTH_DB")
		return nil, err
	case tokenServiceURL == "":
		report(ErrTokenServiceNotConfigured, "ERROR_NO_TOKEN_SERVICE")
		return nil, ErrTokenServiceNotConfigured
	case !strings.HasPrefix(tokenServiceURL, "https://"):
		// Note: this never actually happens.
		logging.Errorf(ctx, "Bad token service URL: %s", tokenServiceURL)
		report(ErrTokenServiceNotConfigured, "ERROR_NOT_HTTPS_TOKEN_SERVICE")
		return nil, ErrTokenServiceNotConfigured
	}
	tokenServiceHost := tokenServiceURL[len("https://"):]

	ctx = logging.SetFields(ctx, logging.Fields{
		"token":   "scoped",
		"project": p.LuciProject,
	})

	cacheKey := fmt.Sprintf("%s\n%s\n",
		p.LuciProject, strings.Join(p.OAuthScopes, "\n"))

	cached, err, label := scopedTokenCache.fetchOrMintToken(ctx, &fetchOrMintTokenOp{
		CacheKey:    cacheKey,
		MinTTL:      p.MinTTL,
		MintTimeout: cfg.adjustedTimeout(10 * time.Second),

		// Mint is called on cache miss, under the lock.
		Mint: func(ctx context.Context) (t *cachedToken, err error, label string) {
			// Grab a token server client (or its mock).
			rpcClient := p.rpcClient
			if rpcClient == nil {
				transport, err := GetRPCTransport(ctx, AsSelf)
				if err != nil {
					return nil, err, "ERROR_NO_TRANSPORT"
				}
				rpcClient = minter.NewTokenMinterPRPCClient(&prpc.Client{
					C:    &http.Client{Transport: transport},
					Host: tokenServiceHost,
					Options: &prpc.Options{
						Retry: func() retry.Iterator {
							return &retry.ExponentialBackoff{
								Limited: retry.Limited{
									Delay:   50 * time.Millisecond,
									Retries: 5,
								},
							}
						},
					},
				})
			}

			// The actual RPC call.
			resp, err := rpcClient.MintProjectToken(ctx, &minter.MintProjectTokenRequest{
				LuciProject:         p.LuciProject,
				OauthScope:          p.OAuthScopes,
				MinValidityDuration: int64(MaxScopedTokenTTL.Seconds()),
			})

			// TODO(fmatenaar): This is valid during scoped-account migration and
			// should be removed eventually after migration is finished for all
			// projects.
			//
			// Cache the "NotFound" response and indicate it in the cached token.
			now := clock.Now(ctx).UTC()
			if err != nil && status.Code(err) == codes.NotFound {
				logging.Warningf(ctx, "Received NOT_FOUND from token-server, caching")
				exp := now.Add(5 * time.Minute).UTC()
				return &cachedToken{
					Created:              now,
					Expiry:               exp,
					ProjectScopeFallback: true,
					OAuth2Token:          "",
				}, nil, "FALLBACK_PROJECT_NOT_FOUND"
			}

			if err != nil {
				err = grpcutil.WrapIfTransient(err)
				if transient.Tag.In(err) {
					return nil, err, "ERROR_TRANSIENT_IN_MINTING"
				}
				return nil, err, "ERROR_MINTING"
			}

			// Sanity checks. A correctly working token server should not trigger
			// them.
			good := false
			switch {
			case resp.AccessToken == "":
				logging.Errorf(ctx, "No access token in the response")
			case resp.ServiceAccountEmail == "":
				logging.Errorf(ctx, "No service account email in the response")
			case resp.Expiry == nil:
				logging.Errorf(ctx, "No expiration in the response")
			default:
				good = true
			}
			if !good {
				return nil, ErrBrokenTokenService, "ERROR_BROKEN_TOKEN_SERVICE"
			}

			exp := time.Unix(resp.Expiry.Seconds, 0).UTC()

			// Log details about the new token.
			logging.Fields{
				"service_account": resp.ServiceAccountEmail,
				"expiry":          exp.Sub(now),
				"fingerprint":     tokenFingerprint(resp.AccessToken),
			}.Debugf(ctx, "Minted new project scoped service account token")

			return &cachedToken{
				Created:     now,
				Expiry:      exp,
				OAuth2Token: resp.AccessToken,
			}, nil, "SUCCESS_CACHE_MISS"
		},
	})

	report(err, label)
	if err != nil {
		return nil, err
	}
	// TODO(fmatenaar): Remove this when scoped service accounts have been
	// migrated.
	if cached.OAuth2Token == "" {
		return nil, nil
	}
	return &Token{
		Token:  cached.OAuth2Token,
		Expiry: cached.Expiry,
	}, nil
}
