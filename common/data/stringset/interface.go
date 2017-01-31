// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stringset

// Set is the interface for all string set implementations in this package.
type Set interface {
	// Has returns true iff the Set contains value.
	Has(value string) bool

	// Add ensures that Set contains value, and returns true if it was added (i.e.
	// it returns false if the Set already contained the value).
	Add(value string) bool

	// Del removes value from the set, and returns true if it was deleted (i.e. it
	// returns false if the Set did not already contain the value).
	Del(value string) bool

	// Peek returns an arbitrary element from the set. If the set was empty, this
	// returns ("", false).
	Peek() (string, bool)

	// Peek removes and returns an arbitrary element from the set. If the set was
	// empty, this returns ("", false).
	Pop() (string, bool)

	// Iter calls `cb` for each item in the set. If `cb` returns false, the
	// iteration stops.
	Iter(cb func(string) bool)

	// Len returns the number of items in this set.
	Len() int

	// Dup returns a duplicate set.
	Dup() Set

	// ToSlice renders this set to a slice of all values.
	ToSlice() []string

	// Intersect returns a new Set which is the intersection of this set with the
	// other set.
	//
	// `other` must have the same underlying type as the current set, or this will
	// panic.
	Intersect(other Set) Set

	// Difference returns a new Set which is this set with all elements from other
	// removed (i.e. `self - other`).
	//
	// `other` must have the same underlying type as the current set, or this will
	// panic.
	Difference(other Set) Set

	// Union returns a new Set which contains all element from this set, as well
	// as all elements from the other set.
	//
	// `other` must have the same underlying type as the current set, or this will
	// panic.
	Union(other Set) Set
}
