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

package authdb

import (
	"context"
	"errors"
	"net"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth/realms"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
	"github.com/tetrafolium/luci-go/server/auth/signing"
)

var errNotImplementedInDev = errors.New("this feature is not available in development mode")

// DevServerDB implements authdb.DB by allowing everything.
//
// It can be used locally during development to skip fully configuring auth.
// Must not be used for real production applications.
type DevServerDB struct{}

func (DevServerDB) IsAllowedOAuthClientID(ctx context.Context, email, clientID string) (bool, error) {
	return true, nil
}

func (DevServerDB) IsInternalService(ctx context.Context, hostname string) (bool, error) {
	return false, nil
}

func (DevServerDB) IsMember(ctx context.Context, id identity.Identity, groups []string) (bool, error) {
	if len(groups) == 0 {
		return false, nil
	}
	return id.Kind() != identity.Anonymous, nil
}

func (DevServerDB) CheckMembership(ctx context.Context, id identity.Identity, groups []string) ([]string, error) {
	if id.Kind() == identity.Anonymous {
		return nil, nil
	}
	return groups, nil
}

func (DevServerDB) HasPermission(ctx context.Context, id identity.Identity, perm realms.Permission, realm string) (bool, error) {
	return id.Kind() != identity.Anonymous, nil
}

func (DevServerDB) GetCertificates(ctx context.Context, id identity.Identity) (*signing.PublicCertificates, error) {
	return nil, errNotImplementedInDev
}

func (DevServerDB) GetWhitelistForIdentity(ctx context.Context, ident identity.Identity) (string, error) {
	return "", nil
}

func (DevServerDB) IsInWhitelist(ctx context.Context, ip net.IP, whitelist string) (bool, error) {
	return false, nil
}

func (DevServerDB) GetAuthServiceURL(ctx context.Context) (string, error) {
	return "", errNotImplementedInDev
}

func (DevServerDB) GetTokenServiceURL(ctx context.Context) (string, error) {
	return "", errNotImplementedInDev
}

func (DevServerDB) GetRealmData(ctx context.Context, realm string) (*protocol.RealmData, error) {
	return &protocol.RealmData{}, nil
}
