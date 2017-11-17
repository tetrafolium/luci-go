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

package cloud

import (
	"net/http"
	"os"
	"strings"

	"go.chromium.org/luci/common/data/caching/lru"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/googleoauth"
	"go.chromium.org/luci/common/logging"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/datastore"
	cloudLogging "cloud.google.com/go/logging"
	iamAPI "google.golang.org/api/iam/v1"
	"google.golang.org/api/option"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Flex defines a Google AppEngine Flex Environment platform.
type Flex struct {
	// Cache is the process-global LRU cache instance that Flexi services can use
	// to cache data.
	//
	// If Cache is nil, a default cache will be used.
	Cache *lru.Cache
}

// Configure constructs a Config based on the current Flex environment.
//
// Configure will instantiate some cloud clients. It is the responsibility of
// the client to close those instances when finished.
//
// opts is the optional set of client options to pass to cloud platform clients
// that are instantiated.
func (f *Flex) Configure(c context.Context, opts ...option.ClientOption) (cfg *Config, err error) {
	// If running on GCE, assume we are on Flex and use the metadata server and
	// environ to get environment information. When running locally (e.g. during
	// development), we extract the email from the Default Application Credentials
	// and provide some fake defaults for non-essential parts of the config.
	cfg = &Config{}
	if metadata.OnGCE() {
		if cfg.ServiceAccountName, err = getMetadata("instance/service-accounts/default/email"); err != nil {
			return nil, err
		}
	} else {
		ts, err := google.DefaultTokenSource(c, iamAPI.CloudPlatformScope)
		if err != nil {
			return nil, errors.Annotate(err, "failed to get Application Default Credentials").Err()
		}
		if cfg.ServiceAccountName, err = getEmailFromTokenSource(c, ts); err != nil {
			return nil, err
		}
		logging.Infof(c, "Running locally as %q", cfg.ServiceAccountName)
		cfg.IsDev = true
		cfg.ServiceName = "local"
		cfg.VersionName = "tainted-local"
		cfg.InstanceID = "local"
	}

	err = getEnv(map[string]*string{
		"GCLOUD_PROJECT": &cfg.ProjectID,
		"GAE_SERVICE":    &cfg.ServiceName,
		"GAE_VERSION":    &cfg.VersionName,
		"GAE_INSTANCE":   &cfg.InstanceID,
	})
	if err != nil {
		return nil, err
	}

	gsp := GoogleServiceProvider{
		ServiceAccount: cfg.ServiceAccountName,
		Cache:          f.Cache,
	}
	if gsp.Cache == nil {
		gsp.Cache = lru.New(defaultGoogleServicesCacheSize)
	}
	cfg.ServiceProvider = &gsp

	// Augment our client options. First we clone them so we don't mutate our
	// caller's option set.
	ts, err := gsp.TokenSource(c, iamAPI.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	opts = append(make([]option.ClientOption, 0, len(opts)+1), opts...)
	opts = append(opts, option.WithTokenSource(ts))

	// Cloud Datastore Client.
	if cfg.DS, err = datastore.NewClient(c, cfg.ProjectID, opts...); err != nil {
		return nil, errors.Annotate(err, "failed to instantiate datastore client").Err()
	}

	// Cloud Logging Client, only when running for real.
	if !cfg.IsDev {
		if cfg.L, err = cloudLogging.NewClient(c, cfg.ProjectID, opts...); err != nil {
			return nil, errors.Annotate(err, "could not create logger").Err()
		}
	}

	return
}

// Request probes Request parameters from a AppEngine Flex Environment HTTP
// request.
func (*Flex) Request(req *http.Request) *Request {
	return &Request{
		TraceID: getCloudTraceContext(req),
	}
}

func getEnv(kv map[string]*string) error {
	for k, ptr := range kv {
		switch v := os.Getenv(k); {
		case v != "":
			*ptr = v
		case *ptr == "":
			return errors.Reason("missing required environment variable %q", k).Err()
		}
	}
	return nil
}

func getMetadata(key string) (string, error) {
	switch v, err := metadata.Get(key); {
	case err != nil:
		return "", errors.Annotate(err, "could not retrieve metadata value %q", key).Err()
	case v == "":
		return "", errors.Reason("missing metadata value %q", key).Err()
	default:
		return v, nil
	}
}

func getEmailFromTokenSource(c context.Context, ts oauth2.TokenSource) (string, error) {
	tok, err := ts.Token()
	if err != nil {
		return "", errors.Annotate(err, "failed to grab an access token").Err()
	}
	info, err := googleoauth.GetTokenInfo(c, googleoauth.TokenInfoParams{
		AccessToken: tok.AccessToken,
		Client:      http.DefaultClient,
	})
	if err != nil {
		return "", errors.Annotate(err, "failed to get the token info").Err()
	}
	return info.Email, nil
}

// getCloudTraceContext parses an "X-Cloud-Trace-Context" header.
//
// The "X-Cloud-Trace-Context" header is supplied with requests, and is
// formatted: "TRACE-ID/SPAN_ID;o=TRACE_TRUE".
//
// The "TRACE-ID" should be a unique 32-character hex value representing an
// ID that is unique between requests, so we will use that as a base for
// per-request uniqueness.
//
// https://cloud.google.com/trace/docs/faq
func getCloudTraceContext(req *http.Request) string {
	v := req.Header.Get("X-Cloud-Trace-Context")
	if v == "" {
		return ""
	}
	return strings.SplitN(v, "/", 2)[0]
}
