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

package main

import (
	"context"
	"net/http"

	logsPb "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/logs/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/flex"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/flex/logs"
	"github.com/tetrafolium/luci-go/logdog/server/config"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server"
	flexMW "github.com/tetrafolium/luci-go/appengine/gaemiddleware/flex"
	commonAuth "github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/logging/gologger"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/router"
)

// Run installs and executes this site.
func main() {
	mathrand.SeedRandomly()

	// Setup process global Context.
	c := context.Background()
	c = gologger.StdConfig.Use(c) // Log to STDERR.

	// Cache for configs.
	configStore := config.Store{}

	// TODO(dnj): We currently instantiate global instances of several services,
	// with the current service configuration paramers (e.g., name of BigTable
	// table, etc.).
	//
	// We should monitor config and kill a Flex instance if it's been observed to
	// change. It would respawn, reload the new config, and then be good to go
	// until the next change.
	//
	// As things stand, this configuration basically never changes, so this is
	// not terribly important. However, it's worth noting that we should do this,
	// and that here is probably the right place to kick off such a goroutine.

	// Standard HTTP endpoints using flex LogDog services singleton.
	r := router.NewWithRootContext(c)
	flexMW.ReadOnlyFlex.InstallHandlers(r)

	// Setup the global services, such as auth, luci-config.
	c = flexMW.WithGlobal(c)
	c = config.WithStore(c, &configStore)
	gsvc, err := flex.NewGlobalServices(c)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to setup Flex services.")
		panic(err)
	}
	defer gsvc.Close()
	baseMW := flexMW.ReadOnlyFlex.Base().Extend(
		config.Middleware(&configStore),
		gsvc.Base,
	)

	// Set up PRPC server.
	svr := &prpc.Server{
		AccessControl: accessControl,
	}
	logsServer := logs.New()
	logsPb.RegisterLogsServer(svr, logsServer)
	discovery.Enable(svr)
	svr.InstallHandlers(r, baseMW)

	// Setup HTTP endpoints.
	// We support OpenID (cookie) auth for browsers and OAuth2 for everything else.
	httpMW := baseMW.Extend(
		auth.Authenticate(
			server.CookieAuth,
			&auth.GoogleOAuth2Method{Scopes: []string{commonAuth.OAuthScopeEmail}}))
	r.GET("/logs/*path", httpMW, logs.GetHandler)

	// Run forever.
	logging.Infof(c, "Listening on port 8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logging.WithError(err).Errorf(c, "Failed HTTP listen.")
		panic(err)
	}
}

func accessControl(c context.Context, origin string) bool {
	cfg, err := config.Config(c)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to get config for access control check.")
		return false
	}

	ccfg := cfg.GetCoordinator()
	if ccfg == nil {
		return false
	}

	for _, o := range ccfg.RpcAllowOrigins {
		if o == origin {
			return true
		}
	}
	return false
}
