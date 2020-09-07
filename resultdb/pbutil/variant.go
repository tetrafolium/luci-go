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

package pbutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"

	"github.com/tetrafolium/luci-go/common/errors"

	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// ValidateVariant returns an error if vr is invalid.
func ValidateVariant(vr *pb.Variant) error {
	for k, v := range vr.GetDef() {
		p := pb.StringPair{Key: k, Value: v}
		if err := ValidateStringPair(&p); err != nil {
			return errors.Annotate(err, "%q:%q", k, v).Err()
		}
	}
	return nil
}

// Variant creates a pb.Variant from a list of strings alternating
// key/value. Does not validate pairs.
// See also VariantFromStrings.
//
// Panics if an odd number of tokens is passed.
func Variant(pairs ...string) *pb.Variant {
	if len(pairs)%2 != 0 {
		panic(fmt.Sprintf("odd number of tokens in %q", pairs))
	}

	vr := &pb.Variant{Def: make(map[string]string, len(pairs)/2)}
	for i := 0; i < len(pairs); i += 2 {
		vr.Def[pairs[i]] = pairs[i+1]
	}
	return vr
}

var nonNilEmptyStringSlice = []string{}

// VariantToStrings returns a key:val string slice representation of the Variant.
// Never returns nil.
func VariantToStrings(vr *pb.Variant) []string {
	if len(vr.GetDef()) == 0 {
		return nonNilEmptyStringSlice
	}

	keys := SortedVariantKeys(vr)
	pairs := make([]string, len(keys))
	defMap := vr.GetDef()
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%s:%s", k, defMap[k])
	}
	return pairs
}

// VariantFromStrings returns a Variant proto given the key:val string slice of its contents.
//
// If a key appears multiple times, the last pair wins.
func VariantFromStrings(pairs []string) (*pb.Variant, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	def := make(map[string]string, len(pairs))
	for _, p := range pairs {
		pair, err := StringPairFromString(p)
		if err != nil {
			return nil, errors.Annotate(err, "pair %q", p).Err()
		}
		def[pair.Key] = pair.Value
	}
	return &pb.Variant{Def: def}, nil
}

// SortedVariantKeys returns the keys in the variant as a sorted slice.
func SortedVariantKeys(vr *pb.Variant) []string {
	keys := make([]string, 0, len(vr.GetDef()))
	for k := range vr.GetDef() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// VariantHash returns a short hash of the variant.
func VariantHash(vr *pb.Variant) string {
	h := sha256.New()
	for _, k := range SortedVariantKeys(vr) {
		io.WriteString(h, k)
		io.WriteString(h, ":")
		io.WriteString(h, vr.Def[k])
		io.WriteString(h, "\n")
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}
