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

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"

	"golang.org/x/oauth2"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/common/retry/transient"
)

// A client with more relaxed timeouts compared to the default one, which was
// observed to timeout often on GKE when using Workload Identities.
var metadataClient = metadata.NewClient(&http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		ResponseHeaderTimeout: 15 * time.Second, // default is 2
	},
})

// GKE metadata servers is grumpy when it is called concurrently. We use the
// global mutex to serialize calls to it from within this process.
var globalGCELock sync.Mutex

type gceTokenProvider struct {
	account  string
	email    string
	cacheKey CacheKey
}

// NewGCETokenProvider returns TokenProvider that knows how to use GCE metadata
// server.
func NewGCETokenProvider(ctx context.Context, account string, scopes []string) (TokenProvider, error) {
	// When running on GKE using Workload Identities, the metadata is served by
	// gke-metadata-server pod, which may be very slow, especially when the node
	// has just started. We'll wait for it to become responsive by retrying
	// transient errors a bunch of times.
	var p TokenProvider
	err := retry.Retry(ctx, transient.Only(retryParams), func() error {
		var err error
		p, err = attemptInit(ctx, account, scopes)
		return err
	}, retry.LogCallback(ctx, "initializing GCE token provider"))
	return p, err
}

// retryParams defines the retry strategy for attemptInit.
func retryParams() retry.Iterator {
	return &retry.ExponentialBackoff{
		Limited: retry.Limited{
			Delay:    100 * time.Millisecond,
			MaxTotal: 5 * time.Minute,
			Retries:  -1, // until the overall MaxTotal timeout
		},
		Multiplier: 2,
		MaxDelay:   10 * time.Second,
	}
}

// attemptInit attempts to initialize GCE token provider.
func attemptInit(ctx context.Context, account string, scopes []string) (TokenProvider, error) {
	// This mutex is used to avoid hitting GKE metadata server concurrently if
	// we have a stampede of goroutines. It doesn't actually protect any shared
	// state in the current process.
	globalGCELock.Lock()
	defer globalGCELock.Unlock()

	if account == "" {
		account = "default"
	}

	// Grab an email associated with the account. This must not be failing on
	// a healthy VM if the account is present. If it does, the metadata server is
	// broken.
	email, err := metadataClient.Email(account)
	if err != nil {
		// Note: we purposefully delay this check only after the first call to
		// the metadata fails because metadata.OnGCE was observed to often report
		// "false" when running on GKE due to gke-metadata-server being slow. Our
		// metadataClient has (much) higher timeouts that the client used by
		// metadata.OnGCE, and it handles slow gke-metadata-server better. So if we
		// end up here and metadata.OnGCE also says "false", then we are not on GCE
		// with high probability. The downside is that it may take up to 15 sec to
		// detect this (or whatever ResponseHeaderTimeout in metadataClient is).
		if !metadata.OnGCE() {
			return nil, ErrBadCredentials
		}
		if _, yep := err.(metadata.NotDefinedError); yep {
			return nil, ErrInsufficientAccess
		}
		return nil, transient.Tag.Apply(err)
	}

	// Ensure the account has requested scopes. Assume 'cloud-platform' scope
	// covers all possible scopes. This is important when using GKE Workload
	// Identities: the metadata server always reports only 'cloud-platform' scope
	// there. Its presence should be enough to cover all scopes used in practice.
	// The exception is non-cloud scopes (like gerritcodereview or G Suite). To
	// use such scopes, one will have to use impersonation through Cloud IAM APIs,
	// which *are* covered by cloud-platform (see ActAsServiceAccount in auth.go).
	availableScopes, err := metadataClient.Scopes(account)
	if err != nil {
		return nil, transient.Tag.Apply(err)
	}
	availableSet := stringset.NewFromSlice(availableScopes...)
	if !availableSet.Has("https://www.googleapis.com/auth/cloud-platform") {
		for _, requested := range scopes {
			if !availableSet.Has(requested) {
				logging.Warningf(ctx, "GCE service account %q doesn't have required scope %q (all scopes: %q)", account, requested, availableScopes)
				return nil, ErrInsufficientAccess
			}
		}
	}

	return &gceTokenProvider{
		account: account,
		email:   email,
		cacheKey: CacheKey{
			Key:    fmt.Sprintf("gce/%s", account),
			Scopes: scopes,
		},
	}, nil
}

func (p *gceTokenProvider) RequiresInteraction() bool {
	return false
}

func (p *gceTokenProvider) Lightweight() bool {
	return true
}

func (p *gceTokenProvider) Email() string {
	return p.email
}

func (p *gceTokenProvider) CacheKey(ctx context.Context) (*CacheKey, error) {
	return &p.cacheKey, nil
}

func (p *gceTokenProvider) MintToken(ctx context.Context, base *Token) (*Token, error) {
	// This mutex is used to avoid hitting GKE metadata server concurrently if
	// we have a stampede of goroutines. It doesn't actually protect any shared
	// state in the current process.
	globalGCELock.Lock()
	defer globalGCELock.Unlock()

	// Note: this code is very similar to ComputeTokenSource(p.account).Token()
	// from [1], except it uses our custom metadataClient which is more forgiving
	// of the slowness of the gke-metadata-server.
	//
	// [1]: https://github.com/golang/oauth2/blob/master/google/google.go

	tokenJSON, err := metadataClient.Get("instance/service-accounts/" + p.account + "/token")
	if err != nil {
		return nil, errors.Annotate(err, "auth/gce: metadata server call failed").Tag(transient.Tag).Err()
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		ExpiresInSec int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	switch err = json.NewDecoder(strings.NewReader(tokenJSON)).Decode(&res); {
	case err != nil:
		return nil, errors.Annotate(err, "auth/gce: invalid token JSON from metadata").Tag(transient.Tag).Err()
	case res.ExpiresInSec == 0 || res.AccessToken == "":
		return nil, errors.Reason("auth/gce: incomplete token received from metadata").Tag(transient.Tag).Err()
	}

	tok := oauth2.Token{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
		Expiry:      clock.Now(ctx).Add(time.Duration(res.ExpiresInSec) * time.Second),
	}

	return &Token{
		// Replicate the hidden magic state added by computeSource.Token().
		Token: *tok.WithExtra(map[string]interface{}{
			"oauth2.google.tokenSource":    "compute-metadata",
			"oauth2.google.serviceAccount": p.account,
		}),
		Email: p.Email(),
	}, nil
}

func (p *gceTokenProvider) RefreshToken(ctx context.Context, prev, base *Token) (*Token, error) {
	// Minting and refreshing on GCE is the same thing: a call to metadata server.
	return p.MintToken(ctx, base)
}
