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

// Spanner contains Transactional Enqueue support for Cloud Spanner.
//
// Importing this package adds Cloud Spanner transactions support to server/tq's
// AddTask. Works only for transactions initiated via server/span library
// (see ReadWriteTransaction there).
//
// This package is normally imported unnamed:
//
//   import _ "github.com/tetrafolium/luci-go/server/tq/txn/spanner"
package spanner

import (
	"context"

	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/server/tq/internal/db"
)

var impl spanDB

func init() {
	db.Register(db.Impl{
		Kind: impl.Kind(),
		ProbeForTxn: func(ctx context.Context) db.DB {
			if span.RW(ctx) != nil {
				return impl
			}
			return nil
		},
		NonTxn: func(ctx context.Context) db.DB {
			return impl
		},
	})
}
