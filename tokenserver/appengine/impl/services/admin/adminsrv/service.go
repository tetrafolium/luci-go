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

// Package adminsrv implements Admin API.
//
// Code defined here is either invoked by an administrator or by the service
// itself (via cron jobs or task queues).
package adminsrv

import (
	"github.com/tetrafolium/luci-go/appengine/gaeauth/server/gaesigner"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/certconfig"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/delegation"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/machinetoken"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/projectscope"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/serviceaccounts"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/serviceaccountsv2"
)

// AdminServer implements admin.AdminServer RPC interface.
type AdminServer struct {
	certconfig.ImportCAConfigsRPC
	delegation.ImportDelegationConfigsRPC
	delegation.InspectDelegationTokenRPC
	machinetoken.InspectMachineTokenRPC
	serviceaccounts.ImportServiceAccountsConfigsRPC
	serviceaccounts.InspectOAuthTokenGrantRPC
	projectscope.ImportProjectIdentityConfigsRPC
	serviceaccountsv2.ImportProjectOwnedAccountsConfigsRPC
}

// NewServer returns prod AdminServer implementation.
//
// It assumes authorization has happened already.
func NewServer() *AdminServer {
	signer := gaesigner.Signer{}
	return &AdminServer{
		ImportDelegationConfigsRPC: delegation.ImportDelegationConfigsRPC{
			RulesCache: delegation.GlobalRulesCache,
		},
		InspectDelegationTokenRPC: delegation.InspectDelegationTokenRPC{
			Signer: signer,
		},
		InspectMachineTokenRPC: machinetoken.InspectMachineTokenRPC{
			Signer: signer,
		},
		ImportServiceAccountsConfigsRPC: serviceaccounts.ImportServiceAccountsConfigsRPC{
			RulesCache: serviceaccounts.GlobalRulesCache,
		},
		InspectOAuthTokenGrantRPC: serviceaccounts.InspectOAuthTokenGrantRPC{
			Signer: signer,
			Rules:  serviceaccounts.GlobalRulesCache.Rules,
		},
		ImportProjectIdentityConfigsRPC: projectscope.ImportProjectIdentityConfigsRPC{},
		ImportProjectOwnedAccountsConfigsRPC: serviceaccountsv2.ImportProjectOwnedAccountsConfigsRPC{
			MappingCache: serviceaccountsv2.GlobalMappingCache,
		},
	}
}
