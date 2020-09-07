// Copyright 2019 The LUCI Authors.
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

// Package rpcexplorer contains complied RPCExplorer web app.
//
// Linking to this package will add 7MB to your binary.
package rpcexplorer

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer/internal"
)

// Install adds routes to serve RPCExplorer web app from "/rpcexplorer".
func Install(r *router.Router) {
	r.GET("/rpcexplorer", router.MiddlewareChain{}, func(c *router.Context) {
		http.Redirect(c.Writer, c.Request, "/rpcexplorer/", http.StatusMovedPermanently)
	})

	// Everything under "services/" should load the main web app (it then itself
	// routes the request based on its URL). Everything else is assumed to be
	// a static resource loaded from assets bundle.
	r.GET("/rpcexplorer/*path", router.MiddlewareChain{}, func(c *router.Context) {
		path := strings.TrimPrefix(c.Params.ByName("path"), "/")
		if path == "" || path == "services" || strings.HasPrefix(path, "services/") {
			path = "index.html"
		}

		hash := internal.GetAssetSHA256(path)
		if hash == nil {
			http.Error(c.Writer, "404 page not found", http.StatusNotFound)
			return
		}

		c.Writer.Header().Set("ETag", fmt.Sprintf("%q", hex.EncodeToString(hash)))
		http.ServeContent(
			c.Writer, c.Request, path, time.Time{},
			strings.NewReader(internal.GetAssetString(path)))
	})
}
