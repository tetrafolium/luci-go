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

// Package realmset provides queryable representation of LUCI Realms DB.
//
// Used internally by authdb.Snapshot.
package realmset

import (
	"strings"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"

	"github.com/tetrafolium/luci-go/server/auth/realms"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"

	"github.com/tetrafolium/luci-go/server/auth/authdb/internal/graph"
)

// ExpectedAPIVersion is the supported value of api_version field.
//
// See Build implementation for details.
const ExpectedAPIVersion = 1

// Realms is a queryable representation of realms.Realms proto.
type Realms struct {
	perms  map[string]PermissionIndex       // permission name -> its index
	names  stringset.Set                    // just names of all defined realms
	realms map[realmAndPerm]groupsAndIdents // <realm, perm> -> who has it
	data   map[string]*protocol.RealmData   // per-realm attached RealmData
}

// PermissionIndex is used in place of permission names.
//
// Note: should match an int type used in `permissions` field in the proto.
type PermissionIndex uint32

// realmAndPerm is used as a composite key in `realms` map.
type realmAndPerm struct {
	realm string
	perm  PermissionIndex
}

// groupsAndIdents is used as a value in `realms` map.
type groupsAndIdents struct {
	groups graph.SortedNodeSet
	idents stringset.Set
}

// PermissionIndex returns an index of the given permission.
//
// It can be passed to QueryAuthorized. Returns (0, false) if there's no such
// permission in the Realms DB.
func (r *Realms) PermissionIndex(perm realms.Permission) (idx PermissionIndex, ok bool) {
	idx, ok = r.perms[perm.Name()]
	return
}

// HasRealm returns true if the given realm exists in the DB.
func (r *Realms) HasRealm(realm string) bool {
	return r.names.Has(realm)
}

// Data returns RealmData attached to a realm or nil if none.
func (r *Realms) Data(realm string) *protocol.RealmData {
	return r.data[realm]
}

// QueryAuthorized returns a representation of principals that have the
// requested permission in the given realm.
//
// The permission should be given as its index obtained via PermissionIndex.
//
// The realm name is not validated. Unknown or invalid realms are silently
// treated as empty. No fallback to @root happens.
//
// Returns a set of groups with principals that have the permission and a set
// of identity strings that were specified in the realm ACL directly (not via
// a group). nils are used in place of empty sets.
//
// The set of groups is represented by a sorted slice of group indexes in a
// graph.QueryableGraph which was passed to Build().
func (r *Realms) QueryAuthorized(realm string, perm PermissionIndex) (graph.SortedNodeSet, stringset.Set) {
	out := r.realms[realmAndPerm{realm, perm}]
	return out.groups, out.idents
}

// Build constructs Realms from the proto message and the group graph.
func Build(r *protocol.Realms, qg *graph.QueryableGraph) (*Realms, error) {
	// Do not use realms.Realms we don't understand. Better to go offline
	// completely than mistakenly allow access to something private by
	// misinterpreting realm rules (e.g. if a new hypothetical DENY rule is
	// misinterpreted as ALLOW).
	//
	// Bumping `api_version` (if it ever happens) should be done extremely
	// carefully in multiple stages:
	//   1. Update components.auth to understand both new and old api_version.
	//   2. Redeploy *everything*.
	//   3. Update Auth Service to generate realms.Realms using the new API.
	if r.ApiVersion != ExpectedAPIVersion {
		return nil, errors.Reason(
			"Realms proto has api_version %d not compatible with this service (it expects %d)",
			r.ApiVersion, ExpectedAPIVersion).Err()
	}

	// Build map: permission name -> its index (since Binding messages operate
	// with indexes). Using ints as keys is also slightly faster than strings.
	perms := make(map[string]PermissionIndex, len(r.Permissions))
	for idx, perm := range r.Permissions {
		perms[perm.Name] = PermissionIndex(idx)
	}

	// Gather names of all realms for HasRealm check.
	names := stringset.New(len(r.Realms))
	for _, realm := range r.Realms {
		names.Add(realm.Name)
	}

	// This is the `realms` map under construction. We'll shrink its memory
	// footprint at the end. Just like `realms` it uses a composite key
	// realmAndPerm as a more memory-efficient alternative to a map of maps
	// (realm -> perm -> principals).
	type principalSet struct {
		groups graph.NodeSet
		idents stringset.Set
	}
	realmsToBe := map[realmAndPerm]principalSet{}

	// interner is used to deduplicate memory used to store identity names.
	interner := stringInterner{}

	// Visit all bindings in all realms and update principal sets in realmsToBe.
	for _, realm := range r.Realms {
		for _, binding := range realm.Bindings {
			// Categorize 'principals' into groups and identity strings.
			groups, idents := categorizePrincipals(binding.Principals, qg, interner)

			// Add them into the corresponding principal sets in realmsToBe.
			for _, permIdx := range binding.Permissions {
				key := realmAndPerm{realm.Name, PermissionIndex(permIdx)}

				ps, ok := realmsToBe[key]
				if !ok {
					ps = principalSet{
						groups: make(graph.NodeSet, len(groups)),
						idents: stringset.New(len(idents)),
					}
					realmsToBe[key] = ps
				}

				for _, idx := range groups {
					ps.groups.Add(idx)
				}
				for _, ident := range idents {
					ps.idents.Add(ident)
				}
			}
		}
	}

	// Replace identically looking group sets with references to a single copy.
	// Also throw away zero-length sets, just a waste of memory. Finally, for
	// identity sets we just keep using stringset.Set we've built as is, assuming
	// using identities in Realm ACLs directly is rare and not worth optimizing
	// for (on top of the string interning optimization we've already done).
	realms := make(map[realmAndPerm]groupsAndIdents, len(realmsToBe))
	dedupper := graph.NodeSetDedupper{}
	for key, sets := range realmsToBe {
		var groups graph.SortedNodeSet
		if len(sets.groups) > 0 {
			groups = dedupper.Dedup(sets.groups)
		}
		var idents stringset.Set
		if sets.idents.Len() > 0 {
			idents = sets.idents
		}
		realms[key] = groupsAndIdents{groups: groups, idents: idents}
	}

	// Extract attached per-realm data into a queryable map.
	count := 0
	for _, realm := range r.Realms {
		if realm.Data != nil {
			count++
		}
	}
	data := make(map[string]*protocol.RealmData, count)
	for _, realm := range r.Realms {
		if realm.Data != nil {
			data[realm.Name] = realm.Data
		}
	}

	return &Realms{
		perms:  perms,
		names:  names,
		realms: realms,
		data:   data,
	}, nil
}

// stringInterner implements string interning to save some memory.
type stringInterner map[string]string

// intern returns an interned copy of 's'.
func (si stringInterner) intern(s string) string {
	if existing, ok := si[s]; ok {
		return existing
	}
	si[s] = s
	return s
}

// categorizePrincipals splits a list of principals into a list of groups
// (identified by their indexes in a QueryableGraph) and list of identity names.
//
// Unknown groups are silently skipped.
func categorizePrincipals(p []string, qg *graph.QueryableGraph, interner stringInterner) (groups []graph.NodeIndex, idents []string) {
	for _, principal := range p {
		if strings.HasPrefix(principal, "group:") {
			if idx, ok := qg.GroupIndex(strings.TrimPrefix(principal, "group:")); ok {
				groups = append(groups, idx)
			}
		} else {
			idents = append(idents, interner.intern(principal))
		}
	}
	return
}
