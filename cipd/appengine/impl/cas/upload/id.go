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

package upload

import (
	"context"
	"strconv"
	"time"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/tokens"
)

// opToken describes how to generate HMAC-protected upload operation IDs
// returned to clients.
var opToken = tokens.TokenKind{
	Algo:       tokens.TokenAlgoHmacSHA256,
	Expiration: 5 * time.Hour,
	SecretKey:  "cipd_upload_op_id_key",
	Version:    1,
}

// NewOpID returns new unique upload operation ID.
func NewOpID(ctx context.Context) (int64, error) {
	// Note: AllocateIDs modifies passed slice in place, by replacing the keys
	// there.
	keys := []*datastore.Key{
		datastore.NewKey(ctx, "cas.UploadOperation", "", 0, nil),
	}
	if err := datastore.AllocateIDs(ctx, keys); err != nil {
		return 0, errors.Annotate(err, "failed to generate upload operation ID").
			Tag(transient.Tag).Err()
	}
	return keys[0].IntID(), nil
}

// WrapOpID returns HMAC-protected string that embeds upload operation ID.
//
// The string is bound to the given caller, i.e UnwrapOpID will correctly
// validate HMAC only if it receives the exact same caller.
func WrapOpID(ctx context.Context, id int64, caller identity.Identity) (string, error) {
	return opToken.Generate(ctx, []byte(caller), map[string]string{
		"id": strconv.FormatInt(id, 10),
	}, 0)
}

// UnwrapOpID extracts upload operation ID from a HMAC-protected string.
func UnwrapOpID(ctx context.Context, token string, caller identity.Identity) (int64, error) {
	body, err := opToken.Validate(ctx, token, []byte(caller))
	if err != nil {
		return 0, errors.Annotate(err, "failed to validate upload operation ID token").Err()
	}
	id, err := strconv.ParseInt(body["id"], 10, 64)
	if err != nil {
		return 0, errors.Annotate(err, "invalid upload operation ID %q", body["id"]).Err()
	}
	return id, nil
}
