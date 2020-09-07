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

package delegation

import (
	"context"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/config/impl/memory"
	admin "github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/policy"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestImportDelegationConfigs(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		ctx := gaetesting.TestingContext()
		ctx, clk := testclock.UseTime(ctx, testclock.TestTimeUTC)

		ctx = prepareCfg(ctx, `rules {
			name: "rule 1"
			requestor: "user:some-user@example.com"
			target_service: "service:some-service"
			allowed_to_impersonate: "group:some-group"
			allowed_audience: "REQUESTOR"
			max_validity_duration: 86400
		}`)

		rules := NewRulesCache()
		rpc := ImportDelegationConfigsRPC{RulesCache: rules}

		// No config.
		r, err := rules.Rules(ctx)
		So(err, ShouldEqual, policy.ErrNoPolicy)

		resp, err := rpc.ImportDelegationConfigs(ctx, nil)
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &admin.ImportedConfigs{
			Revision: "669865709d488bd6fbcec4fd87d6d1070253f42f",
		})

		// Have config now.
		r, err = rules.Rules(ctx)
		So(err, ShouldBeNil)
		So(r.rules[0].rule.Name, ShouldEqual, "rule 1")
		So(r.revision, ShouldEqual, "669865709d488bd6fbcec4fd87d6d1070253f42f")

		// Noop import.
		resp, err = rpc.ImportDelegationConfigs(ctx, nil)
		So(err, ShouldBeNil)
		So(resp.Revision, ShouldEqual, "669865709d488bd6fbcec4fd87d6d1070253f42f")

		// Try to import completely broken config.
		ctx = prepareCfg(ctx, `I'm broken`)
		_, err = rpc.ImportDelegationConfigs(ctx, nil)
		So(err, ShouldErrLike, `line 1.0: unknown field name`)

		// Old config is not replaced.
		r, _ = rules.Rules(ctx)
		So(r.revision, ShouldEqual, "669865709d488bd6fbcec4fd87d6d1070253f42f")

		// Try to import a config that doesn't pass validation.
		ctx = prepareCfg(ctx, `rules {
			name: "rule 1"
		}`)
		_, err = rpc.ImportDelegationConfigs(ctx, nil)
		So(err, ShouldErrLike, `"requestor" is required (and 4 other errors)`)

		// Old config is not replaced.
		r, _ = rules.Rules(ctx)
		So(r.revision, ShouldEqual, "669865709d488bd6fbcec4fd87d6d1070253f42f")

		// Roll time to expire local rules cache.
		clk.Add(10 * time.Minute)

		// Have new config now!
		ctx = prepareCfg(ctx, `rules {
			name: "rule 2"
			requestor: "user:some-user@example.com"
			target_service: "service:some-service"
			allowed_to_impersonate: "group:some-group"
			allowed_audience: "REQUESTOR"
			max_validity_duration: 86400
		}`)

		// Import it.
		resp, err = rpc.ImportDelegationConfigs(ctx, nil)
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &admin.ImportedConfigs{
			Revision: "7826cf5f5d3922ff363a2df0c363df088a81f261",
		})

		// It is now active.
		r, err = rules.Rules(ctx)
		So(err, ShouldBeNil)
		So(r.rules[0].rule.Name, ShouldEqual, "rule 2")
		So(r.revision, ShouldEqual, "7826cf5f5d3922ff363a2df0c363df088a81f261")
	})
}

func prepareCfg(c context.Context, configFile string) context.Context {
	return cfgclient.Use(c, memory.New(map[config.Set]memory.Files{
		"services/${appid}": {
			"delegation.cfg": configFile,
		},
	}))
}
