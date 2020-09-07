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

package spanutil

import (
	"context"
	"reflect"

	"cloud.google.com/go/spanner"
	"github.com/golang/protobuf/proto"
	"google.golang.org/api/iterator"

	"github.com/tetrafolium/luci-go/server/span"
)

// ErrNoResults is an error returned when a query unexpectedly has no results.
var ErrNoResults = iterator.Done

// This file implements utility functions that make spanner API slightly easier
// to use.

func slices(m map[string]interface{}) (keys []string, values []interface{}) {
	keys = make([]string, 0, len(m))
	values = make([]interface{}, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return
}

// ReadRow reads a single row from the database and reads its values.
// ptrMap must map from column names to pointers where the values will be
// written.
func ReadRow(ctx context.Context, table string, key spanner.Key, ptrMap map[string]interface{}) error {
	columns, ptrs := slices(ptrMap)
	row, err := span.ReadRow(ctx, table, key, columns)
	if err != nil {
		return err
	}

	return FromSpanner(row, ptrs...)
}

// QueryFirstRow executes a query, reads the first row into ptrs and stops the
// iterator. Returns ErrNoResults if the query does not return at least one row.
func QueryFirstRow(ctx context.Context, st spanner.Statement, ptrs ...interface{}) error {
	st.Params = ToSpannerMap(st.Params)
	it := span.Query(ctx, st)
	defer it.Stop()
	row, err := it.Next()
	if err != nil {
		return err
	}
	return FromSpanner(row, ptrs...)
}

// Query executes a query.
// Ensures st.Params are Spanner-compatible by modifying st.Params in place.
func Query(ctx context.Context, st spanner.Statement, fn func(row *spanner.Row) error) error {
	st.Params = ToSpannerMap(st.Params)
	return span.Query(ctx, st).Do(fn)
}

func isMessageNil(m proto.Message) bool {
	return reflect.ValueOf(m).IsNil()
}
