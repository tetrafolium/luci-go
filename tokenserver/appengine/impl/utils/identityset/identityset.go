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

// Package identityset implements a set-like structure for identity.Identity.
package identityset

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth"
)

type identSet map[identity.Identity]struct{}
type groupSet map[string]struct{}

// Set is a set of identities represented as sets of IDs and groups.
//
// Groups are generally not expanded, but treated as different kind of items,
// so essentially this struct represents two sets: a set of explicitly specified
// identities, and a set of groups. The exception to this rule is IsMember
// function that looks inside the groups.
type Set struct {
	All    bool     // if true, this set contains all possible identities
	IDs    identSet // set of identity.Identity strings
	Groups groupSet // set of group names
}

// AddIdentity adds a single identity to the set.
//
// The receiver must not be nil.
func (s *Set) AddIdentity(id identity.Identity) {
	if !s.All {
		if s.IDs == nil {
			s.IDs = make(identSet, 1)
		}
		s.IDs[id] = struct{}{}
	}
}

// AddGroup adds a single group to the set.
//
// The receiver must not be nil.
func (s *Set) AddGroup(group string) {
	if !s.All {
		if s.Groups == nil {
			s.Groups = make(groupSet, 1)
		}
		s.Groups[group] = struct{}{}
	}
}

// IsEmpty returns true if this set is empty.
//
// 'nil' receiver value is valid and represents an empty set.
func (s *Set) IsEmpty() bool {
	return s == nil || (!s.All && len(s.IDs) == 0 && len(s.Groups) == 0)
}

// IsMember returns true if the given identity is in the set.
//
// It looks inside the groups too.
//
// 'nil' receiver value is valid and represents an empty set.
func (s *Set) IsMember(c context.Context, id identity.Identity) (bool, error) {
	if s == nil {
		return false, nil
	}

	if s.All {
		return true, nil
	}

	if _, ok := s.IDs[id]; ok {
		return true, nil
	}

	if len(s.Groups) != 0 {
		groups := make([]string, 0, len(s.Groups))
		for gr := range s.Groups {
			groups = append(groups, gr)
		}
		return auth.GetState(c).DB().IsMember(c, id, groups)
	}

	return false, nil
}

// IsSubset returns true if this set if a subset of another set.
//
// Two equal sets are considered subsets of each other.
//
// It doesn't attempt to expand groups. Compares IDs and Groups sets separately,
// as independent kinds of entities.
//
// 'nil' receiver and argument values are valid and represent empty sets.
func (s *Set) IsSubset(superset *Set) bool {
	// An empty set is subset of any other set (including empty sets).
	if s.IsEmpty() {
		return true
	}

	// An empty set is not a superset of any non-empty set.
	if superset.IsEmpty() {
		return false
	}

	// The universal set is subset of only itself.
	if s.All {
		return superset.All
	}

	// The universal set is superset of any other set.
	if superset.All {
		return true
	}

	// Is s.IDs a subset of superset.IDs?
	if len(superset.IDs) < len(s.IDs) {
		return false
	}
	for id := range s.IDs {
		if _, ok := superset.IDs[id]; !ok {
			return false
		}
	}

	// Is s.Groups a subset of superset.Groups?
	if len(superset.Groups) < len(s.Groups) {
		return false
	}
	for group := range s.Groups {
		if _, ok := superset.Groups[group]; !ok {
			return false
		}
	}

	return true
}

// IsSuperset returns true if this set is a super set of another set.
//
// Two equal sets are considered supersets of each other.
//
// 'nil' receiver and argument values are valid and represent empty sets.
func (s *Set) IsSuperset(subset *Set) bool {
	return subset.IsSubset(s)
}

// ToStrings returns a sorted list of strings representing this set.
//
// See 'FromStrings' for the format of this list.
func (s *Set) ToStrings() []string {
	if s.IsEmpty() {
		return nil
	}
	if s.All {
		return []string{"*"}
	}
	out := make([]string, 0, len(s.IDs)+len(s.Groups))
	for ident := range s.IDs {
		out = append(out, string(ident))
	}
	for group := range s.Groups {
		out = append(out, "group:"+group)
	}
	sort.Strings(out)
	return out
}

// FromStrings constructs a Set by parsing a slice of strings.
//
// Each string is either:
//  * "<kind>:<id>" identity string.
//  * "group:<name>" group reference.
//  * "*" token to mean "All identities".
//
// Any string that matches 'skip' predicate is skipped.
func FromStrings(str []string, skip func(string) bool) (*Set, error) {
	set := &Set{}
	for _, s := range str {
		if skip != nil && skip(s) {
			continue
		}
		switch {
		case s == "*":
			set.All = true
		case strings.HasPrefix(s, "group:"):
			gr := strings.TrimPrefix(s, "group:")
			if gr == "" {
				return nil, fmt.Errorf("invalid entry %q", s)
			}
			set.AddGroup(gr)
		default:
			id, err := identity.MakeIdentity(s)
			if err != nil {
				return nil, err
			}
			set.AddIdentity(id)
		}
	}
	// If '*' was used, separately listed IDs and groups are redundant.
	if set.All {
		set.Groups = nil
		set.IDs = nil
	}
	return set, nil
}

// Union returns a union of a list of sets.
func Union(sets ...*Set) *Set {
	estimateIDs := 0
	estimateGroups := 0

	for _, s := range sets {
		if s == nil {
			continue
		}
		if s.All {
			return &Set{All: true}
		}
		if len(s.IDs) > estimateIDs {
			estimateIDs = len(s.IDs)
		}
		if len(s.Groups) > estimateGroups {
			estimateGroups = len(s.Groups)
		}
	}

	union := &Set{}
	if estimateIDs != 0 {
		union.IDs = make(identSet, estimateIDs)
	}
	if estimateGroups != 0 {
		union.Groups = make(groupSet, estimateGroups)
	}

	for _, s := range sets {
		if s == nil {
			continue
		}
		for ident := range s.IDs {
			union.IDs[ident] = struct{}{}
		}
		for group := range s.Groups {
			union.Groups[group] = struct{}{}
		}
	}

	return union
}

// Extend returns a throw-away set that has one additional member.
//
// The returned set must not be modified, since it references data of the
// original set (to avoid unnecessary copying).
func Extend(orig *Set, id identity.Identity) *Set {
	if orig.IsEmpty() {
		return &Set{
			IDs: identSet{
				id: struct{}{},
			},
		}
	}
	if orig.All {
		return orig
	}
	if _, ok := orig.IDs[id]; ok {
		return orig
	}

	// Need to make a copy of orig.IDs to add 'id' there.
	extended := make(identSet, len(orig.IDs)+1)
	for origID := range orig.IDs {
		extended[origID] = struct{}{}
	}
	extended[id] = struct{}{}

	return &Set{
		IDs:    extended,
		Groups: orig.Groups,
	}
}
