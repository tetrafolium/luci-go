// Copyright 2017 The LUCI Authors.
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

package serviceaccounts

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/gae/service/info"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authdb"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/auth/signing/signingtest"
	"github.com/tetrafolium/luci-go/tokenserver/api"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/google"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func testingContext(caller identity.Identity) context.Context {
	ctx := gaetesting.TestingContext()
	ctx = logging.SetLevel(ctx, logging.Debug)
	ctx = info.GetTestable(ctx).SetRequestID("gae-request-id")
	ctx, _ = testclock.UseTime(ctx, testclock.TestTimeUTC)
	return auth.WithState(ctx, &authtest.FakeState{
		Identity:       caller,
		PeerIPOverride: net.ParseIP("127.10.10.10"),
		FakeDB:         &authdb.SnapshotDB{Rev: 1234},
	})
}

func testingSigner() *signingtest.Signer {
	return signingtest.NewSigner(&signing.ServiceInfo{
		ServiceAccountName: "signer@testing.host",
		AppID:              "unit-tests",
		AppVersion:         "mocked-ver",
	})
}

func TestMintOAuthTokenGrant(t *testing.T) {
	t.Parallel()

	ctx := testingContext("user:requestor@example.com")

	Convey("with mocked config and state", t, func() {
		cfg, err := loadConfig(ctx, `rules {
			name: "rule 1"
			service_account: "account@robots.com"
			proxy: "user:requestor@example.com"
			end_user: "user:enduser@example.com"
			max_grant_validity_duration: 7200
		}`)
		So(err, ShouldBeNil)

		var lastParams *mintParams
		var lastBody *tokenserver.OAuthTokenGrantBody
		mintMock := func(c context.Context, p *mintParams) (*minter.MintOAuthTokenGrantResponse, *tokenserver.OAuthTokenGrantBody, error) {
			lastParams = p
			now := clock.Now(c)
			expiry := now.Add(time.Duration(p.validityDuration) * time.Second)
			lastBody = &tokenserver.OAuthTokenGrantBody{
				TokenId:          12345,
				ServiceAccount:   p.serviceAccount,
				Proxy:            string(p.proxyID),
				EndUser:          string(p.endUserID),
				IssuedAt:         google.NewTimestamp(now),
				ValidityDuration: p.validityDuration,
			}
			return &minter.MintOAuthTokenGrantResponse{
				GrantToken:     "valid_token",
				Expiry:         google.NewTimestamp(expiry),
				ServiceVersion: p.serviceVer,
			}, lastBody, nil
		}

		var loggedInfo *MintedGrantInfo
		rpc := MintOAuthTokenGrantRPC{
			Signer: testingSigner(),
			Rules:  func(context.Context) (*Rules, error) { return cfg, nil },
			LogGrant: func(c context.Context, i *MintedGrantInfo) error {
				loggedInfo = i
				return nil
			},
			mintMock: mintMock,
		}

		Convey("Happy path", func() {
			req := &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "account@robots.com",
				EndUser:        "user:enduser@example.com",
				AuditTags:      []string{"k1:v1", "k2:v2"},
			}
			resp, err := rpc.MintOAuthTokenGrant(ctx, req)
			So(err, ShouldBeNil)
			So(resp.GrantToken, ShouldEqual, "valid_token")
			So(resp.ServiceVersion, ShouldEqual, "unit-tests/mocked-ver")
			So(lastParams, ShouldResemble, &mintParams{
				serviceAccount:   "account@robots.com",
				proxyID:          "user:requestor@example.com",
				endUserID:        "user:enduser@example.com",
				validityDuration: 3600, // default
				serviceVer:       "unit-tests/mocked-ver",
			})

			// LogGrant called.
			So(loggedInfo, ShouldResemble, &MintedGrantInfo{
				Request:   req,
				Response:  resp,
				GrantBody: lastBody,
				ConfigRev: cfg.revision,
				Rule:      cfg.rulesPerAcc["account@robots.com"].Rule,
				PeerIP:    net.ParseIP("127.10.10.10"),
				RequestID: "gae-request-id",
				AuthDBRev: 1234,
			})
		})

		Convey("Empty service account", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				EndUser: "user:enduser@example.com",
			})
			So(err, ShouldBeRPCInvalidArgument, "service_account is required")
		})

		Convey("Negative validity", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount:   "account@robots.com",
				EndUser:          "user:enduser@example.com",
				ValidityDuration: -1,
			})
			So(err, ShouldBeRPCInvalidArgument, "validity_duration must be positive")
		})

		Convey("Empty end-user", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "account@robots.com",
			})
			So(err, ShouldBeRPCInvalidArgument, "end_user is required")
		})

		Convey("Bad end-user", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "account@robots.com",
				EndUser:        "blah",
			})
			So(err, ShouldBeRPCInvalidArgument, "bad identity string")
		})

		Convey("Bad audit tags", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "account@robots.com",
				EndUser:        "user:enduser@example.com",
				AuditTags:      []string{"not-kv-pair"},
			})
			So(err, ShouldBeRPCInvalidArgument, "bad audit_tags - tag #1")
		})

		Convey("Unknown rule", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "unknown@robots.com",
				EndUser:        "user:enduser@example.com",
			})
			So(err, ShouldBeRPCPermissionDenied, "unknown service account or not enough permissions to use it")
		})

		Convey("Unauthorized caller", func() {
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:unknown@example.com",
			})
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "account@robots.com",
				EndUser:        "user:enduser@example.com",
			})
			So(err, ShouldBeRPCPermissionDenied, "unknown service account or not enough permissions to use it")
		})

		Convey("Too high validity duration", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount:   "account@robots.com",
				EndUser:          "user:enduser@example.com",
				ValidityDuration: 7201,
			})
			So(err, ShouldBeRPCInvalidArgument, `per rule "rule 1" the validity duration should be <= 7200`)
		})

		Convey("Unauthorized end-user", func() {
			_, err := rpc.MintOAuthTokenGrant(ctx, &minter.MintOAuthTokenGrantRequest{
				ServiceAccount: "account@robots.com",
				EndUser:        "user:unknown@example.com",
			})
			So(err, ShouldBeRPCPermissionDenied,
				`per rule "rule 1" the user "user:unknown@example.com" is not authorized to use the service account "account@robots.com"`)
		})
	})
}
