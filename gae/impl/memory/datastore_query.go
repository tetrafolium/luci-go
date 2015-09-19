// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package memory

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	ds "github.com/luci/gae/service/datastore"
	"github.com/luci/gae/service/datastore/serialize"
	"github.com/luci/luci-go/common/cmpbin"
	"github.com/luci/luci-go/common/stringset"
)

// MaxQueryComponents was lifted from a hard-coded constant in dev_appserver.
// No idea if it's a real limit or just a convenience in the current dev
// appserver implementation.
const MaxQueryComponents = 100

// MaxIndexColumns is the maximum number of index columns we're willing to
// support.
const MaxIndexColumns = 64

// A queryCursor is:
//   {#orders} ++ IndexColumn* ++ RawRowData
//   IndexColumn will always contain __key__ as the last column, and so #orders
//     must always be >= 1
type queryCursor []byte

func newCursor(s string) (ds.Cursor, error) {
	d, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to Base64-decode cursor: %s", err)
	}
	c := queryCursor(d)
	if _, _, err := c.decode(); err != nil {
		return nil, err
	}
	return c, nil
}

func (q queryCursor) String() string { return base64.URLEncoding.EncodeToString([]byte(q)) }

// decode returns the encoded IndexColumns, the raw row (cursor) data, or an
// error.
func (q queryCursor) decode() ([]ds.IndexColumn, []byte, error) {
	buf := bytes.NewBuffer([]byte(q))
	count, _, err := cmpbin.ReadUint(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid cursor: bad prefix number")
	}

	if count == 0 || count > MaxIndexColumns {
		return nil, nil, fmt.Errorf("invalid cursor: bad column count %d", count)
	}

	if count == 0 {
		return nil, nil, fmt.Errorf("invalid cursor: zero prefix number")
	}

	cols := make([]ds.IndexColumn, count)
	for i := range cols {
		if cols[i], err = serialize.ReadIndexColumn(buf); err != nil {
			return nil, nil, fmt.Errorf("invalid cursor: unable to decode IndexColumn %d: %s", i, err)
		}
	}

	if cols[len(cols)-1].Property != "__key__" {
		return nil, nil, fmt.Errorf("invalid cursor: last column was not __key__: %v", cols[len(cols)-1])
	}

	return cols, buf.Bytes(), nil
}

func sortOrdersEqual(as, bs []ds.IndexColumn) bool {
	if len(as) != len(bs) {
		return false
	}
	for i, a := range as {
		if a != bs[i] {
			return false
		}
	}
	return true
}

func numComponents(fq *ds.FinalizedQuery) int {
	numComponents := len(fq.Orders())
	if p, _, _ := fq.IneqFilterLow(); p != "" {
		numComponents++
	}
	if p, _, _ := fq.IneqFilterHigh(); p != "" {
		numComponents++
	}
	for _, v := range fq.EqFilters() {
		numComponents += v.Len()
	}
	return numComponents
}

func reduce(fq *ds.FinalizedQuery, ns string, isTxn bool) (*reducedQuery, error) {
	if err := fq.Valid(globalAppID, ns); err != nil {
		return nil, err
	}
	if isTxn && fq.Ancestor() == nil {
		return nil, fmt.Errorf("queries within a transaction must include an Ancestor filter")
	}
	if num := numComponents(fq); num > MaxQueryComponents {
		return nil, fmt.Errorf(
			"gae/memory: query is too large. may not have more than "+
				"%d filters + sort orders + ancestor total: had %d",
			MaxQueryComponents, num)
	}

	ret := &reducedQuery{
		ns:           ns,
		kind:         fq.Kind(),
		suffixFormat: fq.Orders(),
	}

	eqFilts := fq.EqFilters()
	ret.eqFilters = make(map[string]stringset.Set, len(eqFilts))
	for prop, vals := range eqFilts {
		sVals := stringset.New(len(vals))
		for _, v := range vals {
			sVals.Add(string(serialize.ToBytes(v)))
		}
		ret.eqFilters[prop] = sVals
	}

	// Pick up the start/end range from the inequalities, if any.
	//
	// start and end in the reducedQuery are normalized so that `start >=
	// X < end`. Because of that, we need to tweak the inequality filters
	// contained in the query if they use the > or <= operators.
	startD := []byte(nil)
	endD := []byte(nil)
	if ineqProp := fq.IneqFilterProp(); ineqProp != "" {
		_, startOp, startV := fq.IneqFilterLow()
		if startOp != "" {
			startD = serialize.ToBytes(startV)
			if startOp == ">" {
				startD = increment(startD)
			}
		}

		_, endOp, endV := fq.IneqFilterHigh()
		if endOp != "" {
			endD = serialize.ToBytes(endV)
			if endOp == "<=" {
				endD = increment(endD)
			}
		}

		// The inequality is specified in natural (ascending) order in the query's
		// Filter syntax, but the order information may indicate to use a descending
		// index column for it. If that's the case, then we must invert, swap and
		// increment the inequality endpoints.
		//
		// Invert so that the desired numbers are represented correctly in the index.
		// Swap so that our iterators still go from >= start to < end.
		// Increment so that >= and < get correctly bounded (since the iterator is
		// still using natrual bytes ordering)
		if ret.suffixFormat[0].Descending {
			hi, lo := []byte(nil), []byte(nil)
			if len(startD) > 0 {
				lo = increment(invert(startD))
			}
			if len(endD) > 0 {
				hi = increment(invert(endD))
			}
			endD, startD = lo, hi
		}
	}

	// Now we check the start and end cursors.
	//
	// Cursors are composed of a list of IndexColumns at the beginning, followed
	// by the raw bytes to use for the suffix. The cursor is only valid if all of
	// its IndexColumns match our proposed suffixFormat, as calculated above.
	//
	// Cursors are mutually exclusive with the start/end we picked up from the
	// inequality. In a well formed query, they indicate a subset of results
	// bounded by the inequality. Technically if the start cursor is not >= the
	// low bound, or the end cursor is < the high bound, it's an error, but for
	// simplicity we just cap to the narrowest intersection of the inequality and
	// cursors.
	ret.start = startD
	ret.end = endD
	if start, end := fq.Bounds(); start != nil || end != nil {
		if start != nil {
			if c, ok := start.(queryCursor); ok {
				startCols, startD, err := c.decode()
				if err != nil {
					return nil, err
				}

				if !sortOrdersEqual(startCols, ret.suffixFormat) {
					return nil, errors.New("gae/memory: start cursor is invalid for this query")
				}
				if ret.start == nil || bytes.Compare(ret.start, startD) < 0 {
					ret.start = startD
				}
			} else {
				return nil, errors.New("gae/memory: bad cursor type")
			}
		}

		if end != nil {
			if c, ok := end.(queryCursor); ok {
				endCols, endD, err := c.decode()
				if err != nil {
					return nil, err
				}

				if !sortOrdersEqual(endCols, ret.suffixFormat) {
					return nil, errors.New("gae/memory: end cursor is invalid for this query")
				}
				if ret.end == nil || bytes.Compare(endD, ret.end) < 0 {
					ret.end = endD
				}
			} else {
				return nil, errors.New("gae/memory: bad cursor type")
			}
		}
	}

	// Finally, verify that we could even /potentially/ do work. If we have
	// overlapping range ends, then we don't have anything to do.
	if ret.end != nil && bytes.Compare(ret.start, ret.end) >= 0 {
		return nil, ds.ErrNullQuery
	}

	ret.numCols = len(ret.suffixFormat)
	for prop, vals := range ret.eqFilters {
		if len(ret.suffixFormat) == 1 && prop == "__ancestor__" {
			continue
		}
		ret.numCols += vals.Len()
	}

	return ret, nil
}
