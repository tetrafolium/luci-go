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

// Package main implements HTTP server that handles requests to default module.
package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server"
	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	helloworld "github.com/tetrafolium/luci-go/examples/appengine/helloworld_standard/proto"
	"github.com/tetrafolium/luci-go/gae/service/info"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/templates"
)

// templateBundle is used to render HTML templates. It provides a base args
// passed to all templates.
var templateBundle = &templates.Bundle{
	Loader:    templates.FileSystemLoader("templates"),
	DebugMode: info.IsDevAppServer,
	DefaultArgs: func(c context.Context, e *templates.Extra) (templates.Args, error) {
		loginURL, err := auth.LoginURL(c, e.Request.URL.RequestURI())
		if err != nil {
			return nil, err
		}
		logoutURL, err := auth.LogoutURL(c, e.Request.URL.RequestURI())
		if err != nil {
			return nil, err
		}
		isAdmin, err := auth.IsMember(c, "administrators")
		if err != nil {
			return nil, err
		}
		return templates.Args{
			"AppVersion":  strings.Split(info.VersionID(c), ".")[0],
			"IsAnonymous": auth.CurrentIdentity(c) == "anonymous:anonymous",
			"IsAdmin":     isAdmin,
			"User":        auth.CurrentUser(c),
			"LoginURL":    loginURL,
			"LogoutURL":   logoutURL,
		}, nil
	},
}

// pageBase returns the middleware chain for page handlers.
func pageBase() router.MiddlewareChain {
	return standard.Base().Extend(
		templates.WithTemplates(templateBundle),
		auth.Authenticate(server.UsersAPIAuthMethod{}),
	)
}

// prpcBase returns the middleware chain for pRPC API handlers.
func prpcBase() router.MiddlewareChain {
	// OAuth 2.0 with email scope is registered as a default authenticator
	// by importing "github.com/tetrafolium/luci-go/appengine/gaeauth/server".
	// No need to setup an authenticator here.
	//
	// For authorization checks, we use per-service decorators; see
	// service registration code.
	return standard.Base()
}

//// Routes.

func checkAPIAccess(c context.Context, methodName string, req proto.Message) (context.Context, error) {
	// Implement authorization check here, for example:
	//
	// import "github.com/golang/protobuf/proto"
	// import "google.golang.org/grpc/codes"
	// import "github.com/tetrafolium/luci-go/grpc/grpcutil"
	//
	// hasAccess, err := auth.IsMember(c, "my-users")
	// if err != nil {
	//   return nil, grpcutil.Errf(codes.Internal, "%s", err)
	// }
	// if !hasAccess {
	//   return nil, grpcutil.Errf(codes.PermissionDenied, "%s is not allowed to call APIs", auth.CurrentIdentity(c))
	// }

	return c, nil
}

func init() {
	r := router.New()
	standard.InstallHandlers(r)
	r.GET("/", pageBase(), indexPage)
	r.GET("/test/*something", pageBase(), indexPage) // to test redirect on login

	var api prpc.Server
	helloworld.RegisterGreeterServer(&api, &helloworld.DecoratedGreeter{
		Service: &greeterService{},
		Prelude: checkAPIAccess,
	})
	discovery.Enable(&api)
	api.InstallHandlers(r, prpcBase())

	http.DefaultServeMux.Handle("/", r)
}

//// Handlers.

func indexPage(c *router.Context) {
	templates.MustRender(c.Context, c.Writer, "pages/index.html", nil)
}
