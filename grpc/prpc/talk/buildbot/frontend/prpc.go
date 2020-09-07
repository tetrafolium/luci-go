// Copyright 2016 The LUCI Authors.
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
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/router"

	buildbot "github.com/tetrafolium/luci-go/grpc/prpc/talk/buildbot/proto"
)

func InstallAPIRoutes(r *router.Router, base router.MiddlewareChain) {
	server := &prpc.Server{}
	buildbot.RegisterBuildbotServer(server, &buildbotService{})
	discovery.Enable(server)
	server.InstallHandlers(r, base)
}
