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

package monitoring

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/config/impl/memory"
	gae "github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/caching"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	Convey("With mocks", t, func() {
		configs := map[config.Set]memory.Files{
			"services/${appid}": map[string]string{},
		}
		mockConfig := func(body string) {
			configs["services/${appid}"][cachedCfg.Path] = body
		}

		ctx := gae.Use(context.Background())
		ctx = cfgclient.Use(ctx, memory.New(configs))
		ctx = caching.WithEmptyProcessCache(ctx)

		Convey("No config", func() {
			So(ImportConfig(ctx), ShouldErrLike, "no such config")

			cfg, err := monitoringConfig(ctx)
			So(err, ShouldBeNil)
			So(cfg, ShouldBeNil)
		})

		Convey("Broken config", func() {
			mockConfig("broken")
			So(ImportConfig(ctx), ShouldErrLike, "validation errors")
		})

		Convey("Good config", func() {
			mockConfig(`
				client_monitoring_config {
					ip_whitelist: "ignored"
					label: "ignored-label"
				}
				client_monitoring_config {
					ip_whitelist: "bots"
					label: "bots-label"
				}
			`)
			So(ImportConfig(ctx), ShouldBeNil)

			Convey("Has matching entry", func() {
				e, err := monitoringConfig(auth.WithState(ctx, &authtest.FakeState{
					PeerIPWhitelists: []string{"bots"},
				}))
				So(err, ShouldBeNil)
				So(e.Label, ShouldEqual, "bots-label")
			})

			Convey("No matching entry", func() {
				e, err := monitoringConfig(auth.WithState(ctx, &authtest.FakeState{
					PeerIPWhitelists: []string{"something-else"},
				}))
				So(err, ShouldBeNil)
				So(e, ShouldBeNil)
			})
		})
	})
}
