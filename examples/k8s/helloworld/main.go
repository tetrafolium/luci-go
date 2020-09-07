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

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/gomodule/redigo/redis"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/trace"

	"github.com/tetrafolium/luci-go/server"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/gaeemulation"
	"github.com/tetrafolium/luci-go/server/module"
	"github.com/tetrafolium/luci-go/server/redisconn"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/secrets"

	"github.com/tetrafolium/luci-go/examples/k8s/helloworld/apipb"
)

func main() {
	// Additional modules that extend the server functionality.
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
		redisconn.NewModuleFromFlags(),
		secrets.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		// pRPC example.
		apipb.RegisterGreeterServer(srv.PRPC, &greeterServer{})

		// Logging and tracing example.
		srv.Routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
			logging.Debugf(c.Context, "Hello debug world")

			ctx, span := trace.StartSpan(c.Context, "Testing")
			logging.Infof(ctx, "Hello info world")
			time.Sleep(100 * time.Millisecond)
			span.End(nil)

			logging.Warningf(c.Context, "Hello warning world")
			c.Writer.Write([]byte("Hello, world"))

			logging.WithError(fmt.Errorf("boom")).Errorf(c.Context, "Hello error world")
		})

		// Authentication example (using Google OAuth2 access tokens).
		mw := router.NewMiddlewareChain(auth.Authenticate(&auth.GoogleOAuth2Method{
			Scopes: []string{"https://www.googleapis.com/auth/userinfo.email"},
		}))
		srv.Routes.GET("/who", mw, func(c *router.Context) {
			logging.Infof(c.Context, "Authenticated as %s", auth.CurrentIdentity(c.Context))
			fmt.Fprintf(c.Writer, "Authenticated as %s\n", auth.CurrentIdentity(c.Context))
		})

		// Redis example.
		//
		// To run Redis for tests locally (in particular on OSX):
		//   docker run --name redis -p 6379:6379 --restart always --detach redis
		//
		// Then launch the example with "... -redis-addr :6379".
		//
		// Note that it makes Redis port available on 0.0.0.0. This is a necessity
		// when using Docker-for-Mac. Don't put any sensitive stuff there (or make
		// sure your firewall is configured to block external connections).
		srv.Routes.GET("/redis", router.MiddlewareChain{}, func(c *router.Context) {
			conn, err := redisconn.Get(c.Context)
			if err != nil {
				http.Error(c.Writer, err.Error(), 500)
				return
			}
			defer conn.Close()
			n, err := redis.Int(conn.Do("INCR", "testKey"))
			if err != nil {
				http.Error(c.Writer, err.Error(), 500)
				return
			}
			fmt.Fprintf(c.Writer, "%d", n)
		})

		return nil
	})
}

type greeterServer struct{}

func (*greeterServer) SayHi(ctx context.Context, _ *empty.Empty) (*empty.Empty, error) {
	logging.Infof(ctx, "Hi")
	time.Sleep(100 * time.Millisecond)
	return &empty.Empty{}, nil
}
