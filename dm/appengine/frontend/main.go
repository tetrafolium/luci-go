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

package main

import (
	"net/http"

	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/dm/appengine/deps"
	"github.com/tetrafolium/luci-go/dm/appengine/distributor"
	"github.com/tetrafolium/luci-go/dm/appengine/distributor/jobsim"
	"github.com/tetrafolium/luci-go/dm/appengine/distributor/swarming/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/mutate"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/tumble"
)

func main() {
	tmb := tumble.Service{}

	distributors := distributor.FactoryMap{}
	jobsim.AddFactory(distributors)
	swarming.AddFactory(distributors)

	reg := distributor.NewRegistry(distributors, mutate.FinishExecutionFn)

	basemw := standard.Base().Extend(func(c *router.Context, next router.Handler) {
		c.Context = distributor.WithRegistry(c.Context, reg)
		next(c)
	})

	r := router.New()

	svr := prpc.Server{}
	deps.RegisterDepsServer(&svr)
	discovery.Enable(&svr)

	distributor.InstallHandlers(r, basemw)
	svr.InstallHandlers(r, basemw)
	tmb.InstallHandlers(r, basemw)
	standard.InstallHandlers(r)

	http.Handle("/", r)
	appengine.Main()
}
