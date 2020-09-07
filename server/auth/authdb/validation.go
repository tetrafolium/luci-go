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

package authdb

import (
	"fmt"
	"net"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
)

// validateAuthDB returns nil if AuthDB looks correct.
func validateAuthDB(db *protocol.AuthDB) error {
	groups := make(map[string]*protocol.AuthGroup, len(db.GetGroups()))
	for _, g := range db.GetGroups() {
		groups[g.GetName()] = g
	}
	for name := range groups {
		if err := validateAuthGroup(name, groups); err != nil {
			return err
		}
	}
	for _, wl := range db.GetIpWhitelists() {
		if err := validateIPWhitelist(wl); err != nil {
			return fmt.Errorf("auth: bad IP whitlist %q - %s", wl.GetName(), err)
		}
	}
	return nil
}

// validateAuthGroup returns nil if AuthGroup looks correct.
func validateAuthGroup(name string, groups map[string]*protocol.AuthGroup) error {
	g := groups[name]

	for _, ident := range g.GetMembers() {
		if _, err := identity.MakeIdentity(ident); err != nil {
			return fmt.Errorf("auth: invalid identity %q in group %q - %s", ident, name, err)
		}
	}

	for _, glob := range g.GetGlobs() {
		if _, err := identity.MakeGlob(glob); err != nil {
			return fmt.Errorf("auth: invalid glob %q in group %q - %s", glob, name, err)
		}
	}

	for _, nested := range g.GetNested() {
		if groups[nested] == nil {
			return fmt.Errorf("auth: unknown nested group %q in group %q", nested, name)
		}
	}

	if cycle := findGroupCycle(name, groups); len(cycle) != 0 {
		return fmt.Errorf("auth: dependency cycle found - %v", cycle)
	}

	return nil
}

// findGroupCycle searches for a group dependency cycle that contains group
// `name`. Returns list of groups that form the cycle if found, empty list
// if no cycles. Unknown groups are considered empty.
func findGroupCycle(name string, groups map[string]*protocol.AuthGroup) []string {
	// Set of groups that are completely explored (all subtree is traversed).
	visited := map[string]bool{}

	// Stack of groups that are being explored now. In case a cycle is detected
	// it would contain that cycle.
	var visiting []string

	// Recursively explores `group` subtree, returns true if finds a cycle.
	var visit func(string) bool
	visit = func(group string) bool {
		g := groups[group]
		if g == nil {
			visited[group] = true
			return false
		}
		visiting = append(visiting, group)
		for _, nested := range g.GetNested() {
			// Cross edge. Can happen in diamond-like graph, not a cycle.
			if visited[nested] {
				continue
			}
			// Is `group` references its own ancestor -> cycle is detected.
			for _, v := range visiting {
				if v == nested {
					return true
				}
			}
			// Explore subtree.
			if visit(nested) {
				return true
			}
		}
		visiting = visiting[:len(visiting)-1]
		visited[group] = true
		return false
	}

	visit(name)
	return visiting // will contain a cycle, if any
}

// validateIPWhitelist checks IPs in the whitelist are parsable.
func validateIPWhitelist(wl *protocol.AuthIPWhitelist) error {
	for _, subnet := range wl.GetSubnets() {
		if _, _, err := net.ParseCIDR(subnet); err != nil {
			return fmt.Errorf("bad subnet %q - %s", subnet, err)
		}
	}
	return nil
}
