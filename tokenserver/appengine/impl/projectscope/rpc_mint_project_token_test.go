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
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/testing/assertions"
	"github.com/tetrafolium/luci-go/gae/service/info"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/auth/signing/signingtest"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/projectidentity"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	authorizedGroups = []string{projectActorsGroup}
)

func testMintAccessToken(ctx context.Context, params auth.MintAccessTokenParams) (*auth.Token, error) {
	return &auth.Token{
		Token:  "",
		Expiry: time.Now().UTC(),
	}, nil
}

func testingContext(caller identity.Identity) context.Context {
	ctx := gaetesting.TestingContext()
	ctx = logging.SetLevel(ctx, logging.Debug)
	ctx = info.GetTestable(ctx).SetRequestID("gae-request-id")
	ctx, _ = testclock.UseTime(ctx, testclock.TestTimeUTC)
	return auth.WithState(ctx, &authtest.FakeState{
		Identity:       caller,
		IdentityGroups: authorizedGroups,
	})

}

func testMintAccessTokenWithError(ctx context.Context, params auth.MintAccessTokenParams) (*oauth2.Token, error) {
	return nil, fmt.Errorf("Intended error for testing")
}

func newTestMintProjectTokenRPC() *MintProjectTokenRPC {
	rpc := MintProjectTokenRPC{
		Signer:            signingtest.NewSigner(nil),
		MintAccessToken:   testMintAccessToken,
		ProjectIdentities: projectidentity.ProjectIdentities,
	}
	return &rpc
}

func TestMintProjectToken(t *testing.T) {

	t.Parallel()
	ctx := testingContext("service@example.com")
	member, err := auth.IsMember(ctx, projectActorsGroup)

	Convey("initialize rpc handler", t, func() {
		rpc := newTestMintProjectTokenRPC()

		Convey("validateRequest works", func() {

			Convey("empty fields", func() {
				req := &minter.MintProjectTokenRequest{
					LuciProject:         "",
					OauthScope:          []string{},
					MinValidityDuration: 7200,
				}
				_, err := rpc.MintProjectToken(ctx, req)
				So(err, ShouldNotBeNil)
			})

			Convey("empty project", func() {

				req := &minter.MintProjectTokenRequest{
					LuciProject:         "",
					OauthScope:          []string{"https://www.googleapis.com/auth/cloud-platform"},
					MinValidityDuration: 1800,
				}
				_, err := rpc.MintProjectToken(ctx, req)
				So(err, assertions.ShouldErrLike, `luci_project is empty`)
			})

			Convey("empty scopes", func() {

				req := &minter.MintProjectTokenRequest{
					LuciProject:         "foo-project",
					OauthScope:          []string{},
					MinValidityDuration: 1800,
				}

				_, err := rpc.MintProjectToken(ctx, req)
				So(err, assertions.ShouldErrLike, `oauth_scope is required`)
			})

			Convey("returns nil for valid request", func() {
				req := &minter.MintProjectTokenRequest{
					LuciProject:         "test-project",
					OauthScope:          []string{"https://www.googleapis.com/auth/cloud-platform"},
					MinValidityDuration: 3600,
				}
				_, err := rpc.MintProjectToken(ctx, req)
				So(err, assertions.ShouldErrLike, "min_validity_duration must not exceed 1800")
			})
		})

		Convey("MintProjectToken does not return errors with valid input", func() {
			So(err, ShouldBeNil)
			So(member, ShouldBeTrue)

			identity, err := rpc.ProjectIdentities(ctx).Create(
				ctx,
				&projectidentity.ProjectIdentity{Project: "service-project", Email: "foo@bar.com"})
			So(err, ShouldBeNil)
			So(identity, ShouldNotBeNil)

			req := &minter.MintProjectTokenRequest{
				LuciProject: "service-project",
				OauthScope:  []string{"https://www.googleapis.com/auth/cloud-platform"},
			}
			resp, err := rpc.MintProjectToken(ctx, req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)

		})

	})

}
