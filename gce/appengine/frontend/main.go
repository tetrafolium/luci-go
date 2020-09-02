// Copyright 2018 The LUCI Authors.
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

// Package main is the main entry point for the app.
package main

import (
	"net/http"

	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer"

	server "github.com/tetrafolium/luci-go/gce/api/config/v1"
	"github.com/tetrafolium/luci-go/gce/api/instances/v1"
	"github.com/tetrafolium/luci-go/gce/api/projects/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/backend"
	"github.com/tetrafolium/luci-go/gce/appengine/config"
	"github.com/tetrafolium/luci-go/gce/appengine/rpc"
	"github.com/tetrafolium/luci-go/gce/vmtoken"
)

func main() {
	mathrand.SeedRandomly()
	api := prpc.Server{UnaryServerInterceptor: grpcmon.UnaryServerInterceptor}
	server.RegisterConfigurationServer(&api, rpc.NewConfigurationServer())
	instances.RegisterInstancesServer(&api, rpc.NewInstancesServer())
	projects.RegisterProjectsServer(&api, rpc.NewProjectsServer())
	discovery.Enable(&api)

	r := router.New()

	standard.InstallHandlers(r)
	rpcexplorer.Install(r)

	mw := standard.Base()
	api.InstallHandlers(r, mw.Extend(vmtoken.Middleware))
	backend.InstallHandlers(r, mw)
	config.InstallHandlers(r, mw)

	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}
