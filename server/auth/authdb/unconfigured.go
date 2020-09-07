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

package authdb

import (
	"context"
	"net"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/server/auth/realms"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
	"github.com/tetrafolium/luci-go/server/auth/signing"
)

// UnconfiguredDB is an empty authdb.DB that logs and rejects most checks.
//
// What checks are logged are based on the following criteria: if a server has
// UnconfiguredDB installed, and it totally ignores authentication and
// authorization (for example, it is a localhost server), then no logging should
// be emitted. In practice it means we don't log in GetWhitelistForIdentity
// only (it is called for all incoming requests).
type UnconfiguredDB struct {
	Error error // an error to return, must be non-nil
}

func (db UnconfiguredDB) log(ctx context.Context, method string) {
	logging.Errorf(ctx, "UnconfiguredDB.%s: %s", method, db.Error)
	if db.Error == nil {
		panic("UnconfiguredDB.Error must not be nil")
	}
}

func (db UnconfiguredDB) IsAllowedOAuthClientID(ctx context.Context, email, clientID string) (bool, error) {
	db.log(ctx, "IsAllowedOAuthClientID")
	return false, db.Error
}

func (db UnconfiguredDB) IsInternalService(ctx context.Context, hostname string) (bool, error) {
	db.log(ctx, "IsInternalService")
	return false, db.Error
}

func (db UnconfiguredDB) IsMember(ctx context.Context, id identity.Identity, groups []string) (bool, error) {
	db.log(ctx, "IsMember")
	return false, db.Error
}

func (db UnconfiguredDB) CheckMembership(ctx context.Context, id identity.Identity, groups []string) ([]string, error) {
	db.log(ctx, "CheckMembership")
	return nil, db.Error
}

func (db UnconfiguredDB) HasPermission(ctx context.Context, id identity.Identity, perm realms.Permission, realm string) (bool, error) {
	db.log(ctx, "HasPermission")
	return false, db.Error
}

func (db UnconfiguredDB) GetCertificates(ctx context.Context, id identity.Identity) (*signing.PublicCertificates, error) {
	db.log(ctx, "GetCertificates")
	return nil, db.Error
}

func (db UnconfiguredDB) GetWhitelistForIdentity(ctx context.Context, ident identity.Identity) (string, error) {
	// GetWhitelistForIdentity is called for ALL incoming requests. Let them pass.
	return "", nil
}

func (db UnconfiguredDB) IsInWhitelist(ctx context.Context, ip net.IP, whitelist string) (bool, error) {
	db.log(ctx, "IsInWhitelist")
	return false, db.Error
}

func (db UnconfiguredDB) GetAuthServiceURL(ctx context.Context) (string, error) {
	db.log(ctx, "GetAuthServiceURL")
	return "", db.Error
}

func (db UnconfiguredDB) GetTokenServiceURL(ctx context.Context) (string, error) {
	db.log(ctx, "GetTokenServiceURL")
	return "", db.Error
}

func (db UnconfiguredDB) GetRealmData(ctx context.Context, realm string) (*protocol.RealmData, error) {
	db.log(ctx, "GetRealmData")
	return nil, db.Error
}
