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

// Binary default is a simple AppEngine LUCI service. It supplies basic LUCI
// service frontend and backend functionality.
//
// No RPC requests should target this service; instead, they are redirected to
// the appropriate service via "dispatch.yaml".
package main

import (
	"net/http"

	"google.golang.org/appengine"

	logsPb "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/logs/v1"
	registrationPb "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/registration/v1"
	servicesPb "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/server/config"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/router"
)

// Run installs and executes this site.
func main() {
	r := router.New()

	// Standard HTTP endpoints.
	standard.InstallHandlers(r)

	// Register all of the handlers that we want to show up in RPC explorer (via
	// pRPC discovery).
	//
	// Note that most of these services have dedicated service handlers, and any
	// RPCs sent to this module will automatically be routed to them via
	// "dispatch.yaml".
	svr := &prpc.Server{
		UnaryServerInterceptor: grpcmon.UnaryServerInterceptor,
	}
	logsPb.RegisterLogsServer(svr, dummyLogsService)
	registrationPb.RegisterRegistrationServer(svr, dummyRegistrationService)
	servicesPb.RegisterServicesServer(svr, dummyServicesService)
	discovery.Enable(svr)

	base := standard.Base().Extend(config.Middleware(&config.Store{}))
	svr.InstallHandlers(r, base)

	r.GET("/admin/cron/sync-configs", base, func(c *router.Context) {
		config.Sync(c.Context)
	})

	r.POST("/admin/cron/stats/:stat/:namespace", base, cronStatsNSHandler)
	r.GET("/admin/cron/stats", base, cronStatsHandler)

	// Redirect "/" to "/app/".
	r.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
		http.Redirect(c.Writer, c.Request, "/app/", http.StatusFound)
	})
	// Redirect "/v/?s=..." to "/logs/..."
	r.GET("/v/", router.MiddlewareChain{}, func(c *router.Context) {
		path := "/logs/" + c.Request.URL.Query().Get("s")
		http.Redirect(c.Writer, c.Request, path, http.StatusFound)
	})

	http.Handle("/", r)
	appengine.Main()
}
