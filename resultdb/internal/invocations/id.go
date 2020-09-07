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

package invocations

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"cloud.google.com/go/spanner"

	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
)

// ID can convert an invocation id to various formats.
type ID string

// ToSpanner implements span.Value.
func (id ID) ToSpanner() interface{} {
	return id.RowID()
}

// SpannerPtr implements span.Ptr.
func (id *ID) SpannerPtr(b *spanutil.Buffer) interface{} {
	return &b.NullString
}

// FromSpanner implements span.Ptr.
func (id *ID) FromSpanner(b *spanutil.Buffer) error {
	*id = ""
	if b.NullString.Valid {
		*id = IDFromRowID(b.NullString.StringVal)
	}
	return nil
}

// MustParseName converts an invocation name to an ID.
// Panics if the name is invalid. Useful for situations when name was already
// validated.
func MustParseName(name string) ID {
	id, err := pbutil.ParseInvocationName(name)
	if err != nil {
		panic(err)
	}
	return ID(id)
}

// IDFromRowID converts a Spanner-level row ID to an ID.
func IDFromRowID(rowID string) ID {
	return ID(stripHashPrefix(rowID))
}

// Name returns an invocation name.
func (id ID) Name() string {
	return pbutil.InvocationName(string(id))
}

// RowID returns an invocation ID used in spanner rows.
// If id is empty, returns "".
func (id ID) RowID() string {
	if id == "" {
		return ""
	}
	return prefixWithHash(string(id))
}

// Key returns a invocation spanner key.
func (id ID) Key(suffix ...interface{}) spanner.Key {
	ret := make(spanner.Key, 1+len(suffix))
	ret[0] = id.RowID()
	copy(ret[1:], suffix)
	return ret
}

// IDSet is an unordered set of invocation ids.
type IDSet map[ID]struct{}

// NewIDSet creates an IDSet from members.
func NewIDSet(ids ...ID) IDSet {
	ret := make(IDSet, len(ids))
	for _, id := range ids {
		ret.Add(id)
	}
	return ret
}

// Add adds id to the set.
func (s IDSet) Add(id ID) {
	s[id] = struct{}{}
}

// Union adds other ids.
func (s IDSet) Union(other IDSet) {
	for id := range other {
		s.Add(id)
	}
}

// Remove removes id from the set if it was present.
func (s IDSet) Remove(id ID) {
	delete(s, id)
}

// Has returns true if id is in the set.
func (s IDSet) Has(id ID) bool {
	_, ok := s[id]
	return ok
}

// String implements fmt.Stringer.
func (s IDSet) String() string {
	strs := make([]string, 0, len(s))
	for id := range s {
		strs = append(strs, string(id))
	}
	sort.Strings(strs)
	return fmt.Sprintf("%q", strs)
}

// Keys returns a spanner.KeySet.
func (s IDSet) Keys(suffix ...interface{}) spanner.KeySet {
	ret := spanner.KeySets()
	for id := range s {
		ret = spanner.KeySets(id.Key(suffix...), ret)
	}
	return ret
}

// ToSpanner implements span.Value.
func (s IDSet) ToSpanner() interface{} {
	ret := make([]string, 0, len(s))
	for id := range s {
		ret = append(ret, id.RowID())
	}
	sort.Strings(ret)
	return ret
}

// SpannerPtr implements span.Ptr.
func (s *IDSet) SpannerPtr(b *spanutil.Buffer) interface{} {
	return &b.StringSlice
}

// FromSpanner implements span.Ptr.
func (s *IDSet) FromSpanner(b *spanutil.Buffer) error {
	*s = make(IDSet, len(b.StringSlice))
	for _, rowID := range b.StringSlice {
		s.Add(IDFromRowID(rowID))
	}
	return nil
}

// ParseNames converts invocation names to IDSet.
func ParseNames(names []string) (IDSet, error) {
	ids := make(IDSet, len(names))
	for _, name := range names {
		id, err := pbutil.ParseInvocationName(name)
		if err != nil {
			return nil, err
		}
		ids.Add(ID(id))
	}
	return ids, nil
}

// MustParseNames converts invocation names to IDSet.
// Panics if a name is invalid. Useful for situations when names were already
// validated.
func MustParseNames(names []string) IDSet {
	ids, err := ParseNames(names)
	if err != nil {
		panic(err)
	}
	return ids
}

// Names returns a sorted slice of invocation names.
func (s IDSet) Names() []string {
	names := make([]string, 0, len(s))
	for id := range s {
		names = append(names, id.Name())
	}
	sort.Strings(names)
	return names
}

// Batches splits s into batches.
// The batches are sorted by RowID(), such that interval (minRowID, maxRowID)
// of each batch does not overlap with any other batch.
//
// The size of batch is hardcoded 50, because that's the maximum parallelism
// we get from Cloud Spanner.
func (s IDSet) Batches() []IDSet {
	return s.batches(50)
}

func (s IDSet) batches(size int) []IDSet {
	ids := s.SortByRowID()
	batches := make([]IDSet, 0, 1+len(ids)/size)
	for len(ids) > 0 {
		batchSize := size
		if batchSize > len(ids) {
			batchSize = len(ids)
		}
		batches = append(batches, NewIDSet(ids[:batchSize]...))
		ids = ids[batchSize:]
	}
	return batches
}

// SortByRowID returns IDs in the set sorted by row id.
func (s IDSet) SortByRowID() []ID {
	rowIDs := make([]string, 0, len(s))
	for id := range s {
		rowIDs = append(rowIDs, id.RowID())
	}
	sort.Strings(rowIDs)

	ret := make([]ID, len(rowIDs))
	for i, rowID := range rowIDs {
		ret[i] = ID(stripHashPrefix(rowID))
	}
	return ret
}

// hashPrefixBytes is the number of bytes of sha256 to prepend to a PK
// to achieve even distribution.
const hashPrefixBytes = 4

func prefixWithHash(s string) string {
	h := sha256.Sum256([]byte(s))
	prefix := hex.EncodeToString(h[:hashPrefixBytes])
	return fmt.Sprintf("%s:%s", prefix, s)
}

func stripHashPrefix(s string) string {
	expectedPrefixLen := hex.EncodedLen(hashPrefixBytes) + 1 // +1 for separator
	if len(s) < expectedPrefixLen {
		panic(fmt.Sprintf("%q is too short", s))
	}
	return s[expectedPrefixLen:]
}
