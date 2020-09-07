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

// Package tokenminter implements TokenMinter API.
//
// This is main public API of The Token Server.
package tokenminter

import (
	"github.com/tetrafolium/luci-go/appengine/gaeauth/server/gaesigner"
	"github.com/tetrafolium/luci-go/server/auth"

	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/certchecker"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/delegation"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/machinetoken"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/projectscope"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/serviceaccounts"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/serviceaccountsv2"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/projectidentity"
)

// Server implements minter.TokenMinterServer RPC interface.
//
// This is just an assembly of individual method implementations, properly
// configured for use in GAE prod setting.
type serverImpl struct {
	machinetoken.MintMachineTokenRPC
	delegation.MintDelegationTokenRPC
	serviceaccounts.MintOAuthTokenGrantRPC
	serviceaccounts.MintOAuthTokenViaGrantRPC
	projectscope.MintProjectTokenRPC
	serviceaccountsv2.MintServiceAccountTokenRPC
}

// NewServer returns prod TokenMinterServer implementation.
//
// It does all authorization checks inside.
func NewServer() minter.TokenMinterServer {
	return &serverImpl{
		MintMachineTokenRPC: machinetoken.MintMachineTokenRPC{
			Signer:           gaesigner.Signer{},
			CheckCertificate: certchecker.CheckCertificate,
			LogToken:         machinetoken.LogToken,
		},
		MintDelegationTokenRPC: delegation.MintDelegationTokenRPC{
			Signer:   gaesigner.Signer{},
			Rules:    delegation.GlobalRulesCache.Rules,
			LogToken: delegation.LogToken,
		},
		MintOAuthTokenGrantRPC: serviceaccounts.MintOAuthTokenGrantRPC{
			Signer:   gaesigner.Signer{},
			Rules:    serviceaccounts.GlobalRulesCache.Rules,
			LogGrant: serviceaccounts.LogGrant,
		},
		MintOAuthTokenViaGrantRPC: serviceaccounts.MintOAuthTokenViaGrantRPC{
			Signer:          gaesigner.Signer{},
			Rules:           serviceaccounts.GlobalRulesCache.Rules,
			MintAccessToken: auth.MintAccessTokenForServiceAccount,
			LogOAuthToken:   serviceaccounts.LogOAuthToken,
		},
		MintProjectTokenRPC: projectscope.MintProjectTokenRPC{
			Signer:            gaesigner.Signer{},
			MintAccessToken:   auth.MintAccessTokenForServiceAccount,
			ProjectIdentities: projectidentity.ProjectIdentities,
			LogToken:          projectscope.LogToken,
		},
		MintServiceAccountTokenRPC: serviceaccountsv2.MintServiceAccountTokenRPC{
			Signer:          gaesigner.Signer{},
			Mapping:         serviceaccountsv2.GlobalMappingCache.Mapping,
			MintAccessToken: auth.MintAccessTokenForServiceAccount,
			MintIDToken:     auth.MintIDTokenForServiceAccount,
			LogToken:        serviceaccountsv2.LogToken,
		},
	}
}
