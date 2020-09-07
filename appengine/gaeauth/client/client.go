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

// Package client implements OAuth2 authentication for outbound connections
// from Appengine using the application services account.
package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/tetrafolium/luci-go/gae/service/info"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/server/caching"
)

// GetAccessToken returns an OAuth access token representing app's service
// account.
//
// If scopes is empty, uses auth.OAuthScopeEmail scope.
//
// Implements a caching layer on top of GAE's GetAccessToken RPC. May return
// transient errors.
func GetAccessToken(ctx context.Context, scopes []string) (*oauth2.Token, error) {
	scopes, cacheKey := normalizeScopes(scopes)

	// Try to find the token in the local memory first. If it expires soon,
	// refresh it earlier with some probability. That avoids a situation when
	// parallel requests that use access tokens suddenly see the cache expired
	// and rush to refresh the token all at once.
	lru := tokensCache.LRU(ctx)
	if tokIface, ok := lru.Get(ctx, cacheKey); ok {
		tok := tokIface.(*oauth2.Token)
		if !closeToExpRandomized(ctx, tok.Expiry) {
			return tok, nil
		}
	}

	tokIface, err := lru.Create(ctx, cacheKey, func() (interface{}, time.Duration, error) {
		// The token needs to be refreshed.
		logging.Debugf(ctx, "Getting an access token for scopes %q", strings.Join(scopes, ", "))
		accessToken, exp, err := info.AccessToken(ctx, scopes...)
		if err != nil {
			return nil, 0, transient.Tag.Apply(err)
		}
		now := clock.Now(ctx)
		logging.Debugf(ctx, "The token expires in %s", exp.Sub(now))

		// Prematurely expire it to guarantee all returned token live for at least
		// 'expirationMinLifetime'.
		tok := &oauth2.Token{
			AccessToken: accessToken,
			Expiry:      exp.Add(-expirationMinLifetime),
			TokenType:   "Bearer",
		}

		return tok, now.Sub(tok.Expiry), nil
	})
	if err != nil {
		return nil, err
	}
	return tokIface.(*oauth2.Token), nil
}

// NewTokenSource makes oauth2.TokenSource implemented on top of GetAccessToken.
//
// It is bound to the given context.
func NewTokenSource(ctx context.Context, scopes []string) oauth2.TokenSource {
	return &tokenSource{ctx, scopes}
}

type tokenSource struct {
	ctx    context.Context
	scopes []string
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	return GetAccessToken(ts.ctx, ts.scopes)
}

//// Internal stuff.

// normalized scopes string => *oauth2.Token.
var tokensCache = caching.RegisterLRUCache(100)

const (
	// expirationMinLifetime is minimal possible lifetime of a returned token.
	expirationMinLifetime = 2 * time.Minute
	// expirationRandomization defines how much to randomize expiration time.
	expirationRandomization = 3 * time.Minute
)

func normalizeScopes(scopes []string) (normalized []string, cacheKey string) {
	if len(scopes) == 0 {
		scopes = []string{auth.OAuthScopeEmail}
	} else {
		set := stringset.New(len(scopes))
		for _, s := range scopes {
			if strings.ContainsRune(s, '\n') {
				panic(fmt.Errorf("invalid scope %q", s))
			}
			set.Add(s)
		}
		scopes = set.ToSlice()
		sort.Strings(scopes)
	}
	return scopes, strings.Join(scopes, "\n")
}

func closeToExpRandomized(ctx context.Context, exp time.Time) bool {
	switch now := clock.Now(ctx); {
	case now.After(exp):
		return true // expired already
	case now.Add(expirationRandomization).Before(exp):
		return false // far from expiration
	default:
		// The expiration is close enough. Do the randomization.
		rnd := time.Duration(mathrand.Int63n(ctx, int64(expirationRandomization)))
		return now.Add(rnd).After(exp)
	}
}
