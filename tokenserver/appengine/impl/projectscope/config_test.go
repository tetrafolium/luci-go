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

package projectscope

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	configset "github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/config/impl/memory"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/projectidentity"

	. "github.com/smartystreets/goconvey/convey"
)

const fakeConfig = `
	projects {
		id: "id1"
		config_location {
			url: "https://some/repo"
			storage_type: GITILES
		}
		identity_config {
			service_account_email: "foo@bar.com"
		}
	}
	projects {
		id: "id2"
		config_location {
			url: "https://some/other/repo"
			storage_type: GITILES
		}
		identity_config {
			service_account_email: "bar@bar.com"
		}
	}
`

func TestRules(t *testing.T) {
	t.Parallel()

	ctx := auth.WithState(gaetesting.TestingContext(), &authtest.FakeState{
		Identity: "user:unused@example.com",
	})
	storage := projectidentity.ProjectIdentities(ctx)

	Convey("Loads", t, func() {
		ctx = prepareCfg(ctx, fakeConfig)
		_, err := ImportConfigs(ctx)
		So(err, ShouldBeNil)

		expected := map[string]string{
			"id1": "foo@bar.com",
			"id2": "bar@bar.com",
		}

		for project, email := range expected {
			identity, err := storage.LookupByProject(ctx, project)
			So(err, ShouldBeNil)
			So(identity, ShouldResemble, &projectidentity.ProjectIdentity{
				Project: project,
				Email:   email,
			})
		}

	})

}

// prepareCfg injects config.Backend implementation with a bunch of
// config files.
func prepareCfg(c context.Context, configFile string) context.Context {
	return cfgclient.Use(c, memory.New(map[configset.Set]memory.Files{
		"services/${config_service_appid}": {
			"projects.cfg": configFile,
		},
	}))
}
