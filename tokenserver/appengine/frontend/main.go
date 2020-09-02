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

// Binary frontend implements HTTP server that handles requests to 'default'
// module.
package main

import (
	"context"
	"net/http"

	"github.com/golang/protobuf/proto"

	"google.golang.org/appengine"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer"

	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"

	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/services/admin/adminsrv"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/services/admin/certauthorities"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/services/minter/tokenminter"
)

// adminPrelude returns a prelude that authorizes only administrators.
func adminPrelude(serviceName string) func(context.Context, string, proto.Message) (context.Context, error) {
	return func(c context.Context, method string, _ proto.Message) (context.Context, error) {
		logging.Infof(c, "%s: %q is calling %q", serviceName, auth.CurrentIdentity(c), method)
		switch admin, err := auth.IsMember(c, "administrators"); {
		case err != nil:
			return nil, status.Errorf(codes.Internal, "can't check ACL - %s", err)
		case !admin:
			return nil, status.Errorf(codes.PermissionDenied, "not an admin")
		}
		return c, nil
	}
}

func main() {
	r := router.New()
	base := standard.Base()

	// Register config validation rules.
	adminSrv := adminsrv.NewServer()
	tokenminter := tokenminter.NewServer()
	adminSrv.ImportCAConfigsRPC.SetupConfigValidation(&validation.Rules)
	adminSrv.ImportDelegationConfigsRPC.SetupConfigValidation(&validation.Rules)
	adminSrv.ImportServiceAccountsConfigsRPC.SetupConfigValidation(&validation.Rules)
	adminSrv.ImportProjectIdentityConfigsRPC.SetupConfigValidation(&validation.Rules)
	adminSrv.ImportProjectOwnedAccountsConfigsRPC.SetupConfigValidation(&validation.Rules)

	// Install auth, config and tsmon handlers.
	standard.InstallHandlers(r)

	// Serve the RPC Explorer UI.
	rpcexplorer.Install(r)

	// The service has no UI, so just redirect to the RPC Explorer.
	r.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
		http.Redirect(c.Writer, c.Request, "/rpcexplorer/", http.StatusFound)
	})

	// Install all RPC servers. Catch panics, report metrics to tsmon (including
	// panics themselves, as Internal errors).
	api := prpc.Server{
		UnaryServerInterceptor: grpcutil.ChainUnaryServerInterceptors(
			grpcmon.UnaryServerInterceptor,
			grpcutil.UnaryServerPanicCatcherInterceptor,
		),
	}
	admin.RegisterCertificateAuthoritiesServer(&api, &admin.DecoratedCertificateAuthorities{
		Service: certauthorities.NewServer(),
		Prelude: adminPrelude("admin.CertificateAuthorities"),
	})
	admin.RegisterAdminServer(&api, &admin.DecoratedAdmin{
		Service: adminSrv,
		Prelude: adminPrelude("admin.Admin"),
	})
	minter.RegisterTokenMinterServer(&api, tokenminter) // auth inside
	discovery.Enable(&api)
	api.InstallHandlers(r, base)

	// Expose all this stuff.
	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}
