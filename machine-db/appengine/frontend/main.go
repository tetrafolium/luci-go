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

// Package main contains the Machine Database AppEngine front end.
package main

import (
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/router"

	"github.com/tetrafolium/luci-go/machine-db/api/crimson/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/config"
	"github.com/tetrafolium/luci-go/machine-db/appengine/database"
	"github.com/tetrafolium/luci-go/machine-db/appengine/rpc"
	"github.com/tetrafolium/luci-go/machine-db/appengine/ui"
)

func main() {
	mathrand.SeedRandomly()
	databaseMiddleware := standard.Base().Extend(database.WithMiddleware)

	srv := rpc.NewServer()

	r := router.New()
	standard.InstallHandlers(r)
	config.InstallHandlers(r, databaseMiddleware)
	ui.InstallHandlers(r, databaseMiddleware, srv, "templates")

	api := prpc.Server{
		// Install an interceptor capable of reporting tsmon metrics.
		UnaryServerInterceptor: grpcmon.UnaryServerInterceptor,
	}
	crimson.RegisterCrimsonServer(&api, srv)
	discovery.Enable(&api)
	api.InstallHandlers(r, databaseMiddleware)

	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}
