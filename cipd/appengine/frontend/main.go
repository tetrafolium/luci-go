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

// Binary frontend implements HTTP server that handles requests to 'default'
// module.
package main

import (
	"net/http"

	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server"
	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer"

	adminapi "github.com/tetrafolium/luci-go/cipd/api/admin/v1"
	pubapi "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"
	"github.com/tetrafolium/luci-go/cipd/appengine/impl"
	"github.com/tetrafolium/luci-go/cipd/appengine/ui"
)

func main() {
	r := router.New()

	// Install auth, config and tsmon handlers.
	standard.InstallHandlers(r)

	// RPC Explorer UI.
	rpcexplorer.Install(r)

	// Register non-pRPC routes, such as the client bootstrap handler and routes
	// to support minimal subset of legacy API required to let old CIPD clients
	// fetch packages and self-update.
	impl.PublicRepo.InstallHandlers(r, standard.Base().Extend(
		auth.Authenticate(&server.OAuth2Method{
			Scopes: []string{server.EmailScope},
		}),
	))

	// UI pages.
	ui.InstallHandlers(r, standard.Base(), "templates")

	// Install all RPC servers. Catch panics, report metrics to tsmon (including
	// panics themselves, as Internal errors).
	srv := &prpc.Server{
		UnaryServerInterceptor: grpcutil.ChainUnaryServerInterceptors(
			grpcmon.UnaryServerInterceptor,
			grpcutil.UnaryServerPanicCatcherInterceptor,
		),
	}
	adminapi.RegisterAdminServer(srv, impl.AdminAPI)
	pubapi.RegisterStorageServer(srv, impl.PublicCAS)
	pubapi.RegisterRepositoryServer(srv, impl.PublicRepo)
	discovery.Enable(srv)

	srv.InstallHandlers(r, standard.Base())
	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}
