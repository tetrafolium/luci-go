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

package rpc

import (
	"context"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"

	"github.com/tetrafolium/luci-go/machine-db/api/crimson/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/database"
)

// ListVLANs handles a request to retrieve VLANs.
func (*Service) ListVLANs(c context.Context, req *crimson.ListVLANsRequest) (*crimson.ListVLANsResponse, error) {
	ids := make(map[int64]struct{}, len(req.Ids))
	for _, id := range req.Ids {
		ids[id] = struct{}{}
	}
	vlans, err := listVLANs(c, ids, stringset.NewFromSlice(req.Aliases...))
	if err != nil {
		return nil, err
	}
	return &crimson.ListVLANsResponse{
		Vlans: vlans,
	}, nil
}

// listVLANs returns a slice of VLANs in the database.
// VLANs matching either a given ID or a given alias are returned. Specify no IDs or aliases to return all VLANs.
func listVLANs(c context.Context, ids map[int64]struct{}, aliases stringset.Set) ([]*crimson.VLAN, error) {
	db := database.Get(c)
	rows, err := db.QueryContext(c, `
		SELECT id, alias, state, cidr_block
		FROM vlans
	`)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch VLANs").Err()
	}
	defer rows.Close()

	var vlans []*crimson.VLAN
	for rows.Next() {
		vlan := &crimson.VLAN{}
		if err = rows.Scan(&vlan.Id, &vlan.Alias, &vlan.State, &vlan.CidrBlock); err != nil {
			return nil, errors.Annotate(err, "failed to fetch VLAN").Err()
		}
		// VLAN may match either the given IDs or aliases.
		// If both IDs and aliases are empty, consider all VLANs to match.
		if _, ok := ids[vlan.Id]; ok || aliases.Has(vlan.Alias) || (len(ids) == 0 && aliases.Len() == 0) {
			vlans = append(vlans, vlan)
		}
	}
	return vlans, nil
}
