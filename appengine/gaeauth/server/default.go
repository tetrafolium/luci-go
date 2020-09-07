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

package server

import (
	"context"

	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/openid"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/warmup"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server/internal/authdbimpl"
)

// CookieAuth is default cookie-based auth method to use on GAE.
//
// By default on the dev server it is based on dev server cookies (implemented
// by UsersAPIAuthMethod), in prod it is based on OpenID (implemented by
// *openid.CookieAuthMethod).
//
// Works only if appropriate handlers have been installed into the router. See
// InstallHandlers.
//
// It is allowed to assign to CookieAuth (e.g. to install a tweaked auth method)
// before InstallHandlers is called.
var CookieAuth auth.Method

// InstallHandlers installs HTTP handlers for various default routes related
// to authentication system.
//
// Must be installed in server HTTP router for authentication to work.
func InstallHandlers(r *router.Router, base router.MiddlewareChain) {
	if m, ok := CookieAuth.(auth.HasHandlers); ok {
		m.InstallHandlers(r, base)
	}
	auth.InstallHandlers(r, base)
	authdbimpl.InstallHandlers(r, base)
}

func init() {
	warmup.Register("appengine/gaeauth/server", func(ctx context.Context) error {
		if m, ok := CookieAuth.(auth.Warmable); ok {
			return m.Warmup(ctx)
		}
		return nil
	})

	// Flip to true to enable OpenID login on devserver for debugging. Requires
	// a configuration (see /admin/portal/openid_auth page).
	const useOIDOnDevServer = false

	if appengine.IsDevAppServer() && !useOIDOnDevServer {
		CookieAuth = UsersAPIAuthMethod{}
	} else {
		CookieAuth = &openid.CookieAuthMethod{
			SessionStore:        &SessionStore{Prefix: "openid"},
			IncompatibleCookies: []string{"SACSID", "dev_appserver_login"},
			Insecure:            appengine.IsDevAppServer(), // for http:// cookie
		}
	}
}
