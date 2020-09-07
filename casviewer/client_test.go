// Copyright 2020 The LUCI Authors.
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

package casviewer

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/router"
)

func TestClient(t *testing.T) {
	t.Parallel()

	Convey("ClientCache", t, func() {
		Convey("Get", func() {
			c := newContext()
			inst1 := "projects/test-proj/instances/inst1"
			inst2 := "projects/test-proj/instances/inst2"

			// First time, it creates a new client.
			cl1, err := GetClient(c.Context, inst1)
			So(err, ShouldBeNil)
			So(cl1, ShouldNotBeNil)

			// The client should be reused for the same instance.
			cl2, err := GetClient(c.Context, inst1)
			So(err, ShouldBeNil)
			So(cl2, ShouldEqual, cl1)

			// A new client for a different instance will be created.
			cl3, err := GetClient(c.Context, inst2)
			So(err, ShouldBeNil)
			So(cl3, ShouldNotBeNil)
			So(cl3, ShouldNotEqual, cl1)
		})

		Convey("Clear", func() {
			c := newContext()
			inst1 := "projects/test-proj/instances/inst1"
			inst2 := "projects/test-proj/instances/inst2"

			// Create clients.
			var err error
			_, err = GetClient(c.Context, inst1)
			So(err, ShouldBeNil)
			_, err = GetClient(c.Context, inst2)
			So(err, ShouldBeNil)

			cc, err := clientCache(c.Context)
			So(err, ShouldBeNil)
			cc.Clear()

			So(cc.clients, ShouldBeEmpty)
		})
	})
}

// newContext creats a fake context.
func newContext() *router.Context {
	cc := NewClientCache(context.Background())

	ctx := context.Background()
	ctx = authtest.MockAuthConfig(ctx)
	c := &router.Context{
		Context: ctx,
	}
	withClientCacheMW(cc)(c, func(_ *router.Context) {})
	return c
}
