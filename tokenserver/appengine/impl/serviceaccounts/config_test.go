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
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/policy"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

const fakeConfig = `
defaults {
	max_grant_validity_duration: 72000
	allowed_scope: "https://www.googleapis.com/default"
}
rules {
	name: "rule 1"
	owner: "developer@example.com"
	service_account: "abc@robots.com"
	service_account: "def@robots.com"
	service_account: "via-group1-and-rule1@robots.com"
	service_account_group: "account-group-1"
	allowed_scope: "https://www.googleapis.com/scope1"
	allowed_scope: "https://www.googleapis.com/scope2"
	end_user: "user:enduser@example.com"
	end_user: "group:enduser-group"
	proxy: "user:proxy@example.com"
	proxy: "group:proxy-group"
	trusted_proxy: "user:trusted-proxy@example.com"
}
rules {
	name: "rule 2"
	service_account: "xyz@robots.com"
	service_account: "via-group1-and-rule2@robots.com"
	service_account_group: "account-group-2"
}`

func TestRules(t *testing.T) {
	t.Parallel()

	ctx := auth.WithState(context.Background(), &authtest.FakeState{
		Identity: "user:unused@example.com",
		FakeDB: authtest.NewFakeDB(
			authtest.MockMembership("user:via-group1@robots.com", "account-group-1"),
			authtest.MockMembership("user:via-group2@robots.com", "account-group-2"),
			authtest.MockMembership("user:via-both@robots.com", "account-group-1"),
			authtest.MockMembership("user:via-both@robots.com", "account-group-2"),
			authtest.MockMembership("user:via-group1-and-rule1@robots.com", "account-group-1"),
			authtest.MockMembership("user:via-group1-and-rule2@robots.com", "account-group-1"),
		),
	})

	Convey("Loads", t, func() {
		cfg, err := loadConfig(ctx, fakeConfig)
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)

		rule, err := cfg.Rule(ctx, "abc@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule, ShouldNotBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 1")

		scopes := rule.AllowedScopes.ToSlice()
		sort.Strings(scopes)
		So(scopes, ShouldResemble, []string{
			"https://www.googleapis.com/default",
			"https://www.googleapis.com/scope1",
			"https://www.googleapis.com/scope2",
		})

		So(rule.CheckScopes([]string{"https://www.googleapis.com/scope1"}), ShouldBeNil)
		So(
			rule.CheckScopes([]string{"https://www.googleapis.com/scope1", "unknown_scope"}),
			ShouldErrLike,
			`following scopes are not allowed by the rule "rule 1" - ["unknown_scope"]`,
		)

		So(rule.EndUsers.ToStrings(), ShouldResemble, []string{
			"group:enduser-group",
			"user:enduser@example.com",
		})
		So(rule.Proxies.ToStrings(), ShouldResemble, []string{
			"group:proxy-group",
			"user:proxy@example.com",
		})
		So(rule.TrustedProxies.ToStrings(), ShouldResemble, []string{
			"user:trusted-proxy@example.com",
		})
		So(rule.Rule.MaxGrantValidityDuration, ShouldEqual, 72000)
	})

	Convey("Rule picker works", t, func() {
		cfg, err := loadConfig(ctx, fakeConfig)
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)

		rule, err := cfg.Rule(ctx, "abc@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 1")

		rule, err = cfg.Rule(ctx, "def@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 1")

		rule, err = cfg.Rule(ctx, "xyz@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 2")

		rule, err = cfg.Rule(ctx, "via-group1@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 1")

		rule, err = cfg.Rule(ctx, "via-group2@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 2")

		// Note: "rule 2" is not visible to proxy@example.com.
		rule, err = cfg.Rule(ctx, "via-both@robots.com", "user:proxy@example.com")
		So(status.Code(err), ShouldEqual, codes.InvalidArgument)
		So(err, ShouldErrLike, `matches 2 rules in the config rev fake-revision: "rule 1" and 1 more`)

		// Note: no rules are visible to the proxy at all.
		rule, err = cfg.Rule(ctx, "via-both@robots.com", "user:unknown@example.com")
		So(status.Code(err), ShouldEqual, codes.PermissionDenied)
		So(err, ShouldErrLike, `unknown service account`)

		rule, err = cfg.Rule(ctx, "via-group1-and-rule1@robots.com", "user:proxy@example.com")
		So(err, ShouldBeNil)
		So(rule.Rule.Name, ShouldEqual, "rule 1")

		rule, err = cfg.Rule(ctx, "via-group1-and-rule2@robots.com", "user:proxy@example.com")
		So(err, ShouldErrLike, `matches 2 rules in the config rev fake-revision: "rule 1" and 1 more`)

		rule, err = cfg.Rule(ctx, "unknown@robots.com", "user:proxy@example.com")
		So(status.Code(err), ShouldEqual, codes.PermissionDenied)
		So(err, ShouldErrLike, `unknown service account`)
	})

	Convey("Check works", t, func() {
		cfg, err := loadConfig(ctx, fakeConfig)
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)

		Convey("Happy path using 'proxy'", func() {
			r, err := cfg.Check(ctx, &RulesQuery{
				ServiceAccount: "abc@robots.com",
				Proxy:          "user:proxy@example.com",
				EndUser:        "user:enduser@example.com",
			})
			So(err, ShouldBeNil)
			So(r.Rule.Name, ShouldEqual, "rule 1")
		})

		Convey("Happy path using 'trusted_proxy'", func() {
			r, err := cfg.Check(ctx, &RulesQuery{
				ServiceAccount: "abc@robots.com",
				Proxy:          "user:trusted-proxy@example.com",
				EndUser:        "user:someone-random@example.com",
			})
			So(err, ShouldBeNil)
			So(r.Rule.Name, ShouldEqual, "rule 1")
		})

		Convey("Unknown service account", func() {
			_, err := cfg.Check(ctx, &RulesQuery{
				ServiceAccount: "unknown@robots.com",
				Proxy:          "user:proxy@example.com",
				EndUser:        "user:enduser@example.com",
			})
			So(err, ShouldBeRPCPermissionDenied, "unknown service account or not enough permissions to use it")
		})

		Convey("Unauthorized proxy", func() {
			_, err := cfg.Check(ctx, &RulesQuery{
				ServiceAccount: "abc@robots.com",
				Proxy:          "user:unknown@example.com",
				EndUser:        "user:enduser@example.com",
			})
			So(err, ShouldBeRPCPermissionDenied, "unknown service account or not enough permissions to use it")
		})

		Convey("Unauthorized end user", func() {
			_, err := cfg.Check(ctx, &RulesQuery{
				ServiceAccount: "abc@robots.com",
				Proxy:          "user:proxy@example.com",
				EndUser:        "user:unknown@example.com",
			})
			So(err, ShouldBeRPCPermissionDenied,
				`per rule "rule 1" the user "user:unknown@example.com" is not authorized to use the service account "abc@robots.com"`)
		})
	})
}

func loadConfig(ctx context.Context, text string) (*Rules, error) {
	cfg := &admin.ServiceAccountsPermissions{}
	err := proto.UnmarshalText(text, cfg)
	if err != nil {
		return nil, err
	}
	rules, err := prepareRules(ctx, policy.ConfigBundle{serviceAccountsCfg: cfg}, "fake-revision")
	if err != nil {
		return nil, err
	}
	return rules.(*Rules), nil
}
