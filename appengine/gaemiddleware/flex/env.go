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

// Package flex exposes gaemiddleware Environments for AppEngine's Flex
// environment.
package flex

import (
	"context"
	"net/http"
	"strings"

	"github.com/tetrafolium/luci-go/common/data/caching/lru"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/tsmon/monitor"
	"github.com/tetrafolium/luci-go/common/tsmon/target"
	"github.com/tetrafolium/luci-go/config/appengine/gaeconfig"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authdb"
	"github.com/tetrafolium/luci-go/server/pprof"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/tsmon"

	authClient "github.com/tetrafolium/luci-go/appengine/gaeauth/client"
	gaeauth "github.com/tetrafolium/luci-go/appengine/gaeauth/server"
	"github.com/tetrafolium/luci-go/appengine/gaeauth/server/gaesigner"
	"github.com/tetrafolium/luci-go/appengine/gaemiddleware"
	gaetsmon "github.com/tetrafolium/luci-go/appengine/tsmon"

	"github.com/tetrafolium/luci-go/gae/impl/cloud"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/gae/service/info"

	"cloud.google.com/go/compute/metadata"
)

var (
	// ListeningAddr is a address to bind the listening socket to.
	ListeningAddr string

	// globalFlex is the global luci/gae cloud Flex services definition.
	globalFlex *cloud.Flex

	// globalFlexConfig is a process-wide Flex environment configuration.
	globalFlexConfig *cloud.Config

	// globalAuthConfig is configuration of the server/auth library.
	//
	// It specifies concrete GAE-based implementations for various interfaces
	// used by the library.
	//
	// It is indirectly stateful (since NewDBCache returns a stateful object that
	// keeps AuthDB cache in local memory), and thus it's defined as a long living
	// global variable.
	//
	// Used in prod contexts only.
	globalAuthConfig = auth.Config{
		DBProvider:          authdb.NewDBCache(gaeauth.GetAuthDB),
		Signer:              gaesigner.Signer{},
		AccessTokenProvider: authClient.GetAccessToken,
		AnonymousTransport:  func(context.Context) http.RoundTripper { return http.DefaultTransport },
		IsDevMode:           !metadata.OnGCE(),
	}

	// globalTsMonState holds configuration and state related to time series
	// monitoring.
	//
	// TODO(vadimsh): We can flush asynchronously on Flex. Unfortunately, the
	// request context is canceled as soon as the request ends, killing all
	// outliving goroutines. We either need a way to detach it, or run the flush
	// using the global context (the logging is moot in this case, since we'll
	// have no trace ID for logs).
	globalTsMonState = &tsmon.State{
		CustomMonitor: tsmonDebugMonitor(),
		Target: func(ctx context.Context) target.Task {
			return target.Task{
				DataCenter:  "appengine",
				ServiceName: info.AppID(ctx),
				JobName:     info.ModuleName(ctx),
				HostName:    strings.SplitN(info.VersionID(ctx), ".", 2)[0],
			}
		},
		InstanceID:        info.InstanceID,
		TaskNumAllocator:  gaetsmon.DatastoreTaskNumAllocator{},
		FlushInMiddleware: true,
	}
)

// tsmonDebugMonitor returns debug monitor when running on dev server.
func tsmonDebugMonitor() monitor.Monitor {
	if !metadata.OnGCE() {
		return monitor.NewDebugMonitor("")
	}
	return nil
}

func init() {
	if metadata.OnGCE() {
		ListeningAddr = ":8080" // assume real Flex
	} else {
		ListeningAddr = "127.0.0.1:8080" // dev environment
	}
}

// ReadOnlyFlex is an Environment designed for cooperative Flex support
// environments.
var ReadOnlyFlex = gaemiddleware.Environment{
	MemcacheAvailable: false,
	DSReadOnly:        true,
	DSReadOnlyPredicate: func(k *datastore.Key) (isRO bool) {
		// HACK(vadimsh): This is needed to allow tsmon middleware to bypass
		// read-only filter on the datastore. It needs writable access to the
		// datastore for DatastoreTaskNumAllocator to work. It doesn't rely on
		// dscache, and thus it is safe to mutate datastore from Flex side.
		return k.Namespace() != gaetsmon.DatastoreNamespace
	},
	Prepare: func(ctx context.Context) {
		globalFlex = &cloud.Flex{
			Cache: lru.New(65535),
		}
		var err error
		if globalFlexConfig, err = globalFlex.Configure(ctx); err != nil {
			panic(errors.Annotate(err, "could not create Flex config").Err())
		}
	},
	WithInitialRequest: func(ctx context.Context, req *http.Request) context.Context {
		// Install our Cloud services.
		if globalFlexConfig == nil {
			// This can happen when Prepare fails.
			panic("global Flex config is not initialized")
		}
		return globalFlexConfig.Use(ctx, globalFlex.Request(ctx, req))
	},
	WithConfig: gaeconfig.Use,
	WithAuth: func(ctx context.Context) context.Context {
		return auth.Initialize(ctx, &globalAuthConfig)
	},
	ExtraMiddleware: router.NewMiddlewareChain(
		flexFoundationMiddleware,
		globalTsMonState.Middleware,
	),
	ExtraHandlers: func(r *router.Router, base router.MiddlewareChain) {
		gaeauth.InstallHandlers(r, base)
		pprof.InstallHandlers(r, base)
		// Install a handler for basic health checking. We respond with HTTP 200 to
		// indicate that we're always healthy.
		r.GET("/_ah/health", router.MiddlewareChain{},
			func(c *router.Context) { c.Writer.WriteHeader(http.StatusOK) })
	},
}

func flexFoundationMiddleware(c *router.Context, next router.Handler) {
	sr := cloud.ScopedRequestHandler{
		CapturePanics: true,
	}
	sr.Handle(c.Context, c.Writer, func(ctx context.Context, rw http.ResponseWriter) {
		c.Context = ctx
		c.Writer = rw
		next(c)
	})
}

// WithGlobal returns a Context that is not attached to a specific request.
func WithGlobal(ctx context.Context) context.Context {
	return ReadOnlyFlex.With(ctx, &http.Request{})
}
