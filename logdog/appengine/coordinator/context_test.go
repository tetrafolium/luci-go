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

package coordinator

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/auth/identity"
	cfglib "github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	cfgmem "github.com/tetrafolium/luci-go/config/impl/memory"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/logdog/api/config/svcconfig"
	"github.com/tetrafolium/luci-go/logdog/server/config"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestWithProjectNamespace(t *testing.T) {
	t.Parallel()

	Convey(`A testing environment`, t, func() {
		ctx := context.Background()
		ctx = memory.Use(ctx)

		// Load fake project configs into the datastore cache.
		ctx = cfgclient.Use(ctx, projectConfigs(map[string]*svcconfig.ProjectConfig{
			"all-access": {
				ReaderAuthGroups: []string{"all"},
				WriterAuthGroups: []string{"all"},
			},
			"exclusive-access": {
				ReaderAuthGroups: []string{"auth"},
				WriterAuthGroups: []string{"auth"},
			},
		}))
		config.Sync(ctx)

		// Make them available to handlers.
		ctx = config.WithStore(ctx, &config.Store{NoCache: true})

		// Fake authentication state.
		as := authtest.FakeState{
			IdentityGroups: []string{"all"},
		}
		ctx = auth.WithState(ctx, &as)

		Convey(`When using NamespaceAccessNoAuth with anonymous identity`, func() {
			So(auth.CurrentIdentity(ctx).Kind(), ShouldEqual, identity.Anonymous)

			Convey(`Can enter exclusive namespace`, func() {
				So(WithProjectNamespace(&ctx, "exclusive-access", NamespaceAccessNoAuth), ShouldBeNil)
				So(CurrentProject(ctx), ShouldEqual, "exclusive-access")
			})

			Convey(`Will fail to enter a namespace for a non-existent project with Unauthenticated.`, func() {
				So(WithProjectNamespace(&ctx, "does-not-exist", NamespaceAccessNoAuth), ShouldBeRPCUnauthenticated)
			})
		})

		Convey(`When using NamespaceAccessAllTesting with anonymous identity`, func() {
			So(auth.CurrentIdentity(ctx).Kind(), ShouldEqual, identity.Anonymous)

			Convey(`Can enter exclusive namespace`, func() {
				So(WithProjectNamespace(&ctx, "exclusive-access", NamespaceAccessAllTesting), ShouldBeNil)
				So(CurrentProject(ctx), ShouldEqual, "exclusive-access")
			})

			Convey(`Will fail to enter a namespace for a non-existent project.`, func() {
				So(WithProjectNamespace(&ctx, "does-not-exist", NamespaceAccessAllTesting), ShouldBeNil)
				So(CurrentProject(ctx), ShouldEqual, "does-not-exist")
			})
		})

		for _, tc := range []struct {
			testName string
			access   NamespaceAccessType
		}{
			{"READ", NamespaceAccessREAD},
			{"WRITE", NamespaceAccessWRITE},
		} {
			Convey(fmt.Sprintf(`When requesting %s access`, tc.testName), func() {

				Convey(`When logged in`, func() {
					id, err := identity.MakeIdentity("user:testing@example.com")
					if err != nil {
						panic(err)
					}
					as.Identity = id

					Convey(`Will successfully access public project.`, func() {
						So(WithProjectNamespace(&ctx, "all-access", tc.access), ShouldBeNil)
					})

					Convey(`When user is a member of exclusive group`, func() {
						as.IdentityGroups = append(as.IdentityGroups, "auth")

						Convey(`Can access exclusive namespace.`, func() {
							So(WithProjectNamespace(&ctx, "exclusive-access", tc.access), ShouldBeNil)
							So(CurrentProject(ctx), ShouldEqual, "exclusive-access")
						})

						Convey(`Will fail to access non-existent project with PermissionDenied.`, func() {
							So(WithProjectNamespace(&ctx, "does-not-exist", tc.access), ShouldBeRPCPermissionDenied)
						})
					})

					Convey(`Will fail to access exclusive project with PermissionDenied.`, func() {
						So(WithProjectNamespace(&ctx, "exclusive-access", tc.access), ShouldBeRPCPermissionDenied)
					})

					Convey(`Will fail to access non-existent project with PermissionDenied.`, func() {
						So(WithProjectNamespace(&ctx, "does-not-exist", tc.access), ShouldBeRPCPermissionDenied)
					})
				})

				Convey(`Will successfully access public project.`, func() {
					So(WithProjectNamespace(&ctx, "all-access", tc.access), ShouldBeNil)
				})

				Convey(`Will fail to access exclusive project with Unauthenticated.`, func() {
					So(WithProjectNamespace(&ctx, "exclusive-access", tc.access), ShouldBeRPCUnauthenticated)
				})

				Convey(`Will fail to access non-existent project with Unauthenticated.`, func() {
					So(WithProjectNamespace(&ctx, "does-not-exist", tc.access), ShouldBeRPCUnauthenticated)
				})
			})
		}
	})
}

func projectConfigs(p map[string]*svcconfig.ProjectConfig) cfglib.Interface {
	configs := make(map[cfglib.Set]cfgmem.Files, len(p))
	for projectID, cfg := range p {
		configs[cfglib.ProjectSet(projectID)] = cfgmem.Files{
			"${appid}.cfg": proto.MarshalTextString(cfg),
		}
	}
	return cfgmem.New(configs)
}
