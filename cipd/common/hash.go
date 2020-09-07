// Copyright 2017 The LUCI Authors.
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

package common

import (
	"encoding/hex"
	"fmt"
	"hash"
	"reflect"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	api "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"
)

// DefaultHashAlgo is a hash algorithm to use for deriving IDs of new package
// instances. Older existing instances are allowed to use some other hash algo.
const DefaultHashAlgo = api.HashAlgo_SHA256

// Supported algo => its digest length (in hex encoding) + factory function.
//
// Actual entries are added in individual hash_*.go files, so that supported
// hashes can be turned off by build flags or by omitting files when vendoring.
var supportedAlgos = make([]struct {
	hash         func() hash.Hash
	typ          reflect.Type
	hexDigestLen int
}, len(api.HashAlgo_value))

// registerHash is used from hash_*.go files to update supportedAlgos.
func registerHashAlgo(algo api.HashAlgo, h func() hash.Hash) {
	if supportedAlgos[algo].hash != nil {
		panic(fmt.Sprintf("hash algo %s is already registered", algo))
	}
	inst := h()
	supportedAlgos[algo].hash = h
	supportedAlgos[algo].typ = reflect.TypeOf(inst)
	supportedAlgos[algo].hexDigestLen = 2 * len(inst.Sum(nil))
}

// NewHash returns a hash implementation or an error if the algo is unknown.
func NewHash(algo api.HashAlgo) (hash.Hash, error) {
	if err := ValidateHashAlgo(algo); err != nil {
		return nil, err
	}
	return supportedAlgos[algo].hash(), nil
}

// MustNewHash as like NewHash, but panics on errors.
//
// Appropriate for cases when the hash algo has already been validated.
func MustNewHash(algo api.HashAlgo) hash.Hash {
	h, err := NewHash(algo)
	if err != nil {
		panic(err)
	}
	return h
}

// ValidateHashAlgo returns a grpc-annotated error if the given algo is invalid,
// e.g. either unspecified or not known to the current version of the code.
//
// Errors have InvalidArgument grpc code.
func ValidateHashAlgo(h api.HashAlgo) error {
	switch {
	case h == api.HashAlgo_HASH_ALGO_UNSPECIFIED:
		return errors.Reason("the hash algorithm is not specified or unrecognized").
			Tag(grpcutil.InvalidArgumentTag).Err()
	case int(h) >= len(supportedAlgos) || supportedAlgos[h].hash == nil:
		return errors.Reason("unsupported hash algorithm %d", h).
			Tag(grpcutil.InvalidArgumentTag).Err()
	}
	return nil
}

// HexDigest returns a digest string as it is used in ObjectRef.
func HexDigest(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

// ObjectRefFromHash returns ObjectRef based on the given hash.Hash.
//
// `h` must be one of the supported hashes (as created by `NewHash`). Panics
// otherwise.
func ObjectRefFromHash(h hash.Hash) *api.ObjectRef {
	typ := reflect.TypeOf(h)
	for algo, props := range supportedAlgos {
		if props.typ == typ {
			return &api.ObjectRef{
				HashAlgo:  api.HashAlgo(algo),
				HexDigest: HexDigest(h),
			}
		}
	}
	panic(fmt.Sprintf("unknown hash algo %T", h))
}
