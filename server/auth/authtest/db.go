// Copyright 2015 The LUCI Authors.
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

package authtest

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authdb"
	"github.com/tetrafolium/luci-go/server/auth/realms"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
	"github.com/tetrafolium/luci-go/server/auth/signing"
)

// FakeDB implements authdb.DB by mocking membership and permission checks.
//
// Initialize it with a bunch of mocks like:
//
// db := authtest.NewFakeDB(
//   authtest.MockMembership("user:a@example.com", "group"),
//   authtest.MockPermission("user:a@example.com", "proj:realm", perm),
//   ...
// )
//
// The list of mocks can also be extended later via db.AddMocks(...).
type FakeDB struct {
	m         sync.RWMutex
	err       error                              // if not nil, return this error
	perID     map[identity.Identity]*mockedForID // id => groups and perms it has
	ips       map[string]stringset.Set           // IP => whitelists it belongs to
	realmData map[string]*protocol.RealmData     // realm name => data
}

var _ authdb.DB = (*FakeDB)(nil)

// mockedForID is mocked groups and permissions of some identity.
type mockedForID struct {
	groups stringset.Set // a set of group names
	perms  stringset.Set // a set of "<realm>\t<perm>" strings
}

// mockedPermKey is used as a key in mocked.perms map.
func mockedPermKey(realm string, perm realms.Permission) string {
	return fmt.Sprintf("%s\t%s", realm, perm)
}

// MockedDatum is a return value of various Mock* constructors.
type MockedDatum struct {
	// apply mutates the db to apply the mock, called under the write lock.
	apply func(db *FakeDB)
}

// MockMembership modifies db to make IsMember(id, group) == true.
func MockMembership(id identity.Identity, group string) MockedDatum {
	return MockedDatum{
		apply: func(db *FakeDB) { db.mockedForID(id).groups.Add(group) },
	}
}

// MockPermission modifies db to make HasPermission(id, realm, perm) == true.
//
// Panics if `realm` is not a valid globally scoped realm, i.e. it doesn't look
// like "<project>:<realm>".
func MockPermission(id identity.Identity, realm string, perm realms.Permission) MockedDatum {
	if err := realms.ValidateRealmName(realm, realms.GlobalScope); err != nil {
		panic(err)
	}
	return MockedDatum{
		apply: func(db *FakeDB) { db.mockedForID(id).perms.Add(mockedPermKey(realm, perm)) },
	}
}

// MockRealmData modifies what db's GetRealmData returns.
//
// Panics if `realm` is not a valid globally scoped realm, i.e. it doesn't look
// like "<project>:<realm>".
func MockRealmData(realm string, data *protocol.RealmData) MockedDatum {
	if err := realms.ValidateRealmName(realm, realms.GlobalScope); err != nil {
		panic(err)
	}
	return MockedDatum{
		apply: func(db *FakeDB) {
			if db.realmData == nil {
				db.realmData = make(map[string]*protocol.RealmData, 1)
			}
			db.realmData[realm] = data
		},
	}
}

// MockIPWhitelist modifies db to make IsInWhitelist(ip, whitelist) == true.
//
// Panics if `ip` is not a valid IP address.
func MockIPWhitelist(ip, whitelist string) MockedDatum {
	if net.ParseIP(ip) == nil {
		panic(fmt.Sprintf("%q is not a valid IP address", ip))
	}
	return MockedDatum{
		apply: func(db *FakeDB) {
			wl, ok := db.ips[ip]
			if !ok {
				wl = stringset.New(1)
				if db.ips == nil {
					db.ips = make(map[string]stringset.Set, 1)
				}
				db.ips[ip] = wl
			}
			wl.Add(whitelist)
		},
	}
}

// MockError modifies db to make its methods return this error.
//
// `err` may be nil, in which case the previously mocked error is removed.
func MockError(err error) MockedDatum {
	return MockedDatum{
		apply: func(db *FakeDB) { db.err = err },
	}
}

// NewFakeDB creates a FakeDB populated with the given mocks.
//
// Construct mocks using MockMembership, MockPermission, MockIPWhitelist and
// MockError functions.
func NewFakeDB(mocks ...MockedDatum) *FakeDB {
	db := &FakeDB{}
	db.AddMocks(mocks...)
	return db
}

// AddMocks applies a bunch of mocks to the state in the db.
func (db *FakeDB) AddMocks(mocks ...MockedDatum) {
	db.m.Lock()
	defer db.m.Unlock()
	for _, m := range mocks {
		m.apply(db)
	}
}

// Use installs the fake db into the context.
//
// Note that if you use auth.WithState(ctx, &authtest.FakeState{...}), you don't
// need this method. Modify FakeDB in the FakeState instead. See its doc for
// some examples.
func (db *FakeDB) Use(ctx context.Context) context.Context {
	return auth.ModifyConfig(ctx, func(cfg auth.Config) auth.Config {
		cfg.DBProvider = func(context.Context) (authdb.DB, error) {
			return db, nil
		}
		return cfg
	})
}

// IsMember is part of authdb.DB interface.
func (db *FakeDB) IsMember(ctx context.Context, id identity.Identity, groups []string) (bool, error) {
	hits, err := db.CheckMembership(ctx, id, groups)
	if err != nil {
		return false, err
	}
	return len(hits) > 0, nil
}

// CheckMembership is part of authdb.DB interface.
func (db *FakeDB) CheckMembership(ctx context.Context, id identity.Identity, groups []string) (out []string, err error) {
	db.m.RLock()
	defer db.m.RUnlock()

	if db.err != nil {
		return nil, db.err
	}

	if mocked := db.perID[id]; mocked != nil {
		for _, group := range groups {
			if mocked.groups.Has(group) {
				out = append(out, group)
			}
		}
	}

	return
}

// HasPermission is part of authdb.DB interface.
func (db *FakeDB) HasPermission(ctx context.Context, id identity.Identity, perm realms.Permission, realm string) (bool, error) {
	db.m.RLock()
	defer db.m.RUnlock()

	if db.err != nil {
		return false, db.err
	}

	if mocked := db.perID[id]; mocked != nil {
		if mocked.perms.Has(mockedPermKey(realm, perm)) {
			return true, nil
		}
	}

	return false, nil
}

// IsAllowedOAuthClientID is part of authdb.DB interface.
func (db *FakeDB) IsAllowedOAuthClientID(ctx context.Context, email, clientID string) (bool, error) {
	return true, nil
}

// IsInternalService is part of authdb.DB interface.
func (db *FakeDB) IsInternalService(ctx context.Context, hostname string) (bool, error) {
	return false, nil
}

// GetCertificates is part of authdb.DB interface.
func (db *FakeDB) GetCertificates(ctx context.Context, id identity.Identity) (*signing.PublicCertificates, error) {
	return nil, fmt.Errorf("GetCertificates is not implemented by FakeDB")
}

// GetWhitelistForIdentity is part of authdb.DB interface.
func (db *FakeDB) GetWhitelistForIdentity(ctx context.Context, ident identity.Identity) (string, error) {
	return "", nil
}

// IsInWhitelist is part of authdb.DB interface.
func (db *FakeDB) IsInWhitelist(ctx context.Context, ip net.IP, whitelist string) (bool, error) {
	db.m.RLock()
	defer db.m.RUnlock()
	if db.err != nil {
		return false, db.err
	}
	return db.ips[ip.String()].Has(whitelist), nil
}

// GetAuthServiceURL is part of authdb.DB interface.
func (db *FakeDB) GetAuthServiceURL(ctx context.Context) (string, error) {
	return "", fmt.Errorf("GetAuthServiceURL is not implemented by FakeDB")
}

// GetTokenServiceURL is part of authdb.DB interface.
func (db *FakeDB) GetTokenServiceURL(ctx context.Context) (string, error) {
	return "", fmt.Errorf("GetTokenServiceURL is not implemented by FakeDB")
}

// GetRealmData is part of authdb.DB interface.
func (db *FakeDB) GetRealmData(ctx context.Context, realm string) (*protocol.RealmData, error) {
	db.m.RLock()
	defer db.m.RUnlock()
	return db.realmData[realm], nil
}

// mockedForID returns db.perID[id], initializing it if necessary.
//
// Called under the write lock.
func (db *FakeDB) mockedForID(id identity.Identity) *mockedForID {
	m, ok := db.perID[id]
	if !ok {
		m = &mockedForID{
			groups: stringset.New(1),
			perms:  stringset.New(1),
		}
		if db.perID == nil {
			db.perID = make(map[identity.Identity]*mockedForID, 1)
		}
		db.perID[id] = m
	}
	return m
}
