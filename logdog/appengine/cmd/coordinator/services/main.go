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
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/prpc"

	registrationPb "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/registration/v1"
	servicesPb "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/endpoints/registration"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/endpoints/services"
	"github.com/tetrafolium/luci-go/logdog/server/config"
	"github.com/tetrafolium/luci-go/server/router"
)

// Run installs and executes this site.
func main() {
	// needed for LeaseArchiveTasks to pick random queues.
	mathrand.SeedRandomly()

	r := router.New()

	// Setup Cloud Endpoints.
	svr := prpc.Server{
		UnaryServerInterceptor: grpcmon.UnaryServerInterceptor,
	}
	servicesPb.RegisterServicesServer(&svr, services.New(services.ServerSettings{
		// 4 is very likely overkill. Until 2020Q3, Logdog was essentially fine
		// running on a single queue.
		NumQueues: 4,
	}))
	registrationPb.RegisterRegistrationServer(&svr, registration.New())

	// Standard HTTP endpoints.
	base := standard.Base().Extend(config.Middleware(&config.Store{}))
	svr.InstallHandlers(r, base)
	standard.InstallHandlers(r)

	http.Handle("/", r)
	appengine.Main()
}
