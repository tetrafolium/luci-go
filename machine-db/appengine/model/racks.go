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

package model

import (
	"context"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"

	"github.com/tetrafolium/luci-go/machine-db/api/config/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/database"
)

// Rack represents a row in the racks table.
type Rack struct {
	config.Rack
	DatacenterId int64
	Id           int64
}

// RacksTable represents the table of racks in the database.
type RacksTable struct {
	// datacenters is a map of datacenter name to ID in the database.
	datacenters map[string]int64
	// current is the slice of racks in the database.
	current []*Rack

	// additions is a slice of racks pending addition to the database.
	additions []*Rack
	// removals is a slice of racks pending removal from the database.
	removals []*Rack
	// updates is a slice of racks pending update in the database.
	updates []*Rack
}

// fetch fetches the racks from the database.
func (t *RacksTable) fetch(c context.Context) error {
	db := database.Get(c)
	rows, err := db.QueryContext(c, `
		SELECT id, name, description, state, datacenter_id
		FROM racks
	`)
	if err != nil {
		return errors.Annotate(err, "failed to select racks").Err()
	}
	defer rows.Close()
	for rows.Next() {
		rack := &Rack{}
		if err := rows.Scan(&rack.Id, &rack.Name, &rack.Description, &rack.State, &rack.DatacenterId); err != nil {
			return errors.Annotate(err, "failed to scan rack").Err()
		}
		t.current = append(t.current, rack)
	}
	return nil
}

// needsUpdate returns true if the given row needs to be updated to match the given config.
func (*RacksTable) needsUpdate(row, cfg *Rack) bool {
	return row.Description != cfg.Description || row.State != cfg.State || row.DatacenterId != cfg.DatacenterId
}

// computeChanges computes the changes that need to be made to the racks in the database.
func (t *RacksTable) computeChanges(c context.Context, datacenters []*config.Datacenter) error {
	cfgs := make(map[string]*Rack, len(datacenters))
	for _, dc := range datacenters {
		for _, cfg := range dc.Rack {
			id, ok := t.datacenters[dc.Name]
			if !ok {
				return errors.Reason("failed to determine datacenter ID for rack %q: datacenter %q does not exist", cfg.Name, dc.Name).Err()
			}
			cfgs[cfg.Name] = &Rack{
				Rack: config.Rack{
					Name:        cfg.Name,
					Description: cfg.Description,
					State:       cfg.State,
				},
				DatacenterId: id,
			}
		}
	}

	for _, rack := range t.current {
		if cfg, ok := cfgs[rack.Name]; ok {
			// Rack found in the config.
			if t.needsUpdate(rack, cfg) {
				// Rack doesn't match the config.
				cfg.Id = rack.Id
				t.updates = append(t.updates, cfg)
			}
			// Record that the rack config has been seen.
			delete(cfgs, cfg.Name)
		} else {
			// Rack not found in the config.
			t.removals = append(t.removals, rack)
		}
	}

	// Racks remaining in the map are present in the config but not the database.
	// Iterate deterministically over the slices to determine which racks need to be added.
	for _, dc := range datacenters {
		for _, cfg := range dc.Rack {
			if rack, ok := cfgs[cfg.Name]; ok {
				t.additions = append(t.additions, rack)
			}
		}
	}
	return nil
}

// add adds all racks pending addition to the database, clearing pending additions.
// No-op unless computeChanges was called first. Idempotent until computeChanges is called again.
func (t *RacksTable) add(c context.Context) error {
	// Avoid using the database connection to prepare unnecessary statements.
	if len(t.additions) == 0 {
		return nil
	}

	db := database.Get(c)
	stmt, err := db.PrepareContext(c, `
		INSERT INTO racks (name, description, state, datacenter_id)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return errors.Annotate(err, "failed to prepare statement").Err()
	}
	defer stmt.Close()

	// Add each rack to the database, and update the slice of racks with each addition.
	for len(t.additions) > 0 {
		rack := t.additions[0]
		result, err := stmt.ExecContext(c, rack.Name, rack.Description, rack.State, rack.DatacenterId)
		if err != nil {
			return errors.Annotate(err, "failed to add rack %q", rack.Name).Err()
		}
		t.current = append(t.current, rack)
		t.additions = t.additions[1:]
		logging.Infof(c, "Added rack %q", rack.Name)
		rack.Id, err = result.LastInsertId()
		if err != nil {
			return errors.Annotate(err, "failed to get rack ID %q", rack.Name).Err()
		}
	}
	return nil
}

// remove removes all racks pending removal from the database, clearing pending removals.
// No-op unless computeChanges was called first. Idempotent until computeChanges is called again.
func (t *RacksTable) remove(c context.Context) error {
	// Avoid using the database connection to prepare unnecessary statements.
	if len(t.removals) == 0 {
		return nil
	}

	db := database.Get(c)
	stmt, err := db.PrepareContext(c, `
		DELETE FROM racks
		WHERE id = ?
	`)
	if err != nil {
		return errors.Annotate(err, "failed to prepare statement").Err()
	}
	defer stmt.Close()

	// Remove each rack from the table. It's more efficient to update the slice of
	// racks once at the end rather than for each removal, so use a defer.
	removed := make(map[int64]struct{}, len(t.removals))
	defer func() {
		var racks []*Rack
		for _, rack := range t.current {
			if _, ok := removed[rack.Id]; !ok {
				racks = append(racks, rack)
			}
		}
		t.current = racks
	}()
	for len(t.removals) > 0 {
		rack := t.removals[0]
		if _, err := stmt.ExecContext(c, rack.Id); err != nil {
			// Defer ensures the slice of racks is updated even if we exit early.
			return errors.Annotate(err, "failed to remove rack %q", rack.Name).Err()
		}
		removed[rack.Id] = struct{}{}
		t.removals = t.removals[1:]
		logging.Infof(c, "Removed rack %q", rack.Name)
	}
	return nil
}

// update updates all racks pending update in the database, clearing pending updates.
// No-op unless computeChanges was called first. Idempotent until computeChanges is called again.
func (t *RacksTable) update(c context.Context) error {
	// Avoid using the database connection to prepare unnecessary statements.
	if len(t.updates) == 0 {
		return nil
	}

	db := database.Get(c)
	stmt, err := db.PrepareContext(c, `
		UPDATE racks
		SET description = ?, state = ?, datacenter_id = ?
		WHERE id = ?
	`)
	if err != nil {
		return errors.Annotate(err, "failed to prepare statement").Err()
	}
	defer stmt.Close()

	// Update each rack in the table. It's more efficient to update the slice of
	// racks once at the end rather than for each update, so use a defer.
	updated := make(map[int64]*Rack, len(t.updates))
	defer func() {
		for _, rack := range t.current {
			if u, ok := updated[rack.Id]; ok {
				rack.Description = u.Description
				rack.State = u.State
				rack.DatacenterId = u.DatacenterId
			}
		}
	}()
	for len(t.updates) > 0 {
		rack := t.updates[0]
		if _, err := stmt.ExecContext(c, rack.Description, rack.State, rack.DatacenterId, rack.Id); err != nil {
			return errors.Annotate(err, "failed to update rack %q", rack.Name).Err()
		}
		updated[rack.Id] = rack
		t.updates = t.updates[1:]
		logging.Infof(c, "Updated rack %q", rack.Name)
	}
	return nil
}

// ids returns a map of rack names to IDs.
func (t *RacksTable) ids(c context.Context) map[string]int64 {
	racks := make(map[string]int64, len(t.current))
	for _, rack := range t.current {
		racks[rack.Name] = rack.Id
	}
	return racks
}

// EnsureRacks ensures the database contains exactly the given racks.
// Returns a map of rack names to IDs in the database.
func EnsureRacks(c context.Context, cfgs []*config.Datacenter, datacenterIds map[string]int64) (map[string]int64, error) {
	t := &RacksTable{}
	t.datacenters = datacenterIds
	if err := t.fetch(c); err != nil {
		return nil, errors.Annotate(err, "failed to fetch racks").Err()
	}
	if err := t.computeChanges(c, cfgs); err != nil {
		return nil, errors.Annotate(err, "failed to compute changes").Err()
	}
	if err := t.add(c); err != nil {
		return nil, errors.Annotate(err, "failed to add racks").Err()
	}
	if err := t.remove(c); err != nil {
		return nil, errors.Annotate(err, "failed to remove racks").Err()
	}
	if err := t.update(c); err != nil {
		return nil, errors.Annotate(err, "failed to update racks").Err()
	}
	return t.ids(c), nil
}
