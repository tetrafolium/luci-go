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

// ErroringDB implements DB by forbidding all access and returning errors.
type ErroringDB struct {
	Error error // returned by all calls
}

// IsAllowedOAuthClientID returns true if given OAuth2 client_id can be used
// to authenticate access for given email.
func (db ErroringDB) IsAllowedOAuthClientID(ctx context.Context, email, clientID string) (bool, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return false, db.Error
}

// IsInternalService returns true if the given hostname belongs to a service
// that is a part of the current LUCI deployment.
func (db ErroringDB) IsInternalService(ctx context.Context, hostname string) (bool, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return false, db.Error
}

// IsMember returns true if the given identity belongs to any of the groups.
func (db ErroringDB) IsMember(ctx context.Context, id identity.Identity, groups []string) (bool, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return false, db.Error
}

// CheckMembership returns groups from the given list the identity belongs to.
func (db ErroringDB) CheckMembership(ctx context.Context, id identity.Identity, groups []string) ([]string, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return nil, db.Error
}

// HasPermission returns true if the identity has the given permission in any
// of the realms.
func (db ErroringDB) HasPermission(ctx context.Context, id identity.Identity, perm realms.Permission, realm string) (bool, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return false, db.Error
}

// GetCertificates returns a bundle with certificates of a trusted signer.
func (db ErroringDB) GetCertificates(ctx context.Context, id identity.Identity) (*signing.PublicCertificates, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return nil, db.Error
}

// GetWhitelistForIdentity returns name of the IP whitelist to use to check
// IP of requests from given `ident`.
//
// It's used to restrict access for certain account to certain IP subnets.
//
// Returns ("", nil) if `ident` is not IP restricted.
func (db ErroringDB) GetWhitelistForIdentity(ctx context.Context, ident identity.Identity) (string, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return "", db.Error
}

// IsInWhitelist returns true if IP address belongs to given named IP whitelist.
//
// IP whitelist is a set of IP subnets. Unknown IP whitelists are considered
// empty. May return errors if underlying datastore has issues.
func (db ErroringDB) IsInWhitelist(ctx context.Context, ip net.IP, whitelist string) (bool, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return false, db.Error
}

// GetAuthServiceURL returns root URL ("https://<host>") of the auth service.
func (db ErroringDB) GetAuthServiceURL(ctx context.Context) (string, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return "", db.Error
}

// GetTokenServiceURL returns root URL ("https://<host>") of the token service.
func (db ErroringDB) GetTokenServiceURL(ctx context.Context) (string, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return "", db.Error
}

// GetRealmData returns data attached to a realm.
func (db ErroringDB) GetRealmData(ctx context.Context, realm string) (*protocol.RealmData, error) {
	logging.Errorf(ctx, "%s", db.Error)
	return nil, db.Error
}
