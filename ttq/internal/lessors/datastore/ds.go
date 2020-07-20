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

// package datastore implements leasing partitions for processing using Cloud
// Datastore.
package datastore

import (
	"context"
	"time"

	ds "go.chromium.org/gae/service/datastore"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/ttq/internal"
	"go.chromium.org/luci/ttq/internal/partition"
)

// Lessor implements internal.Lessor on top of Cloud Datastore.
type Lessor struct {
}

// WithLease acquires the lease and executes WithLeaseClbk.
// The obtained lease duration may be shorter than requested.
// The obtained lease may be only for some parts of the desired Partition.
func (l *Lessor) WithLease(ctx context.Context, shard int, part *partition.Partition, dur time.Duration, clbk internal.WithLeaseClbk) error {
	expiresAt := clock.Now(ctx).Add(dur)
	if d, ok := ctx.Deadline(); ok && expiresAt.After(d) {
		expiresAt = d
	}
	expiresAt = ds.RoundTime(expiresAt)

	lease, err := l.acquire(ctx, shard, part, expiresAt)
	if err != nil {
		return err
	}
	defer lease.delete(ctx) // failure to delete is logged & ignored.

	lctx, cancel := clock.WithDeadline(ctx, lease.ExpiresAt)
	defer cancel()
	clbk(lctx, lease.parts)
	return nil
}

var _ internal.Lessor = (*Lessor)(nil)

func (*Lessor) acquire(ctx context.Context, shard int, desired *partition.Partition, expiresAt time.Time) (*lease, error) {
	var acquired *lease
	deletedExpired := 0
	err := ds.RunInTransaction(ctx, func(ctx context.Context) error {
		deletedExpired = 0 // reset in case of retries.
		active, expired, err := loadAll(ctx, shard)
		if err != nil {
			return err
		}
		if len(expired) > 0 {
			// Deleting >= 1 lease every time a new one is created suffices to avoid
			// accumulating garbage above O(active leases).
			if len(expired) > 50 {
				expired = expired[:50]
			}
			if err = ds.Delete(ctx, expired); err != nil {
				return errors.Annotate(err, "failed to delete %d expired leases", len(expired)).Err()
			}
			deletedExpired = len(expired)
		}
		parts, err := availableForLease(desired, active)
		if err != nil {
			return errors.Annotate(err, "failed to decode available leases").Err()
		}
		acquired, err = save(ctx, shard, expiresAt, parts)
		return err
	}, &ds.TransactionOptions{Attempts: 5})
	if err != nil {
		return nil, errors.Annotate(err, "failed to transact a lease").Tag(transient.Tag).Err()
	}
	if deletedExpired > 0 {
		// If this is logged frequently, something is wrong either with the leasing
		// process or the lessees are holding to lease longer than they should.
		logging.Warningf(ctx, "deleted %d expired leases", deletedExpired)
	}
	return acquired, nil
}

func leasesRootKey(ctx context.Context, shard int) *ds.Key {
	// Integer ID of 0 are not allowed, so add 1000000 s.t. it's clear which shard
	// an entity belongs to visually.
	return ds.NewKey(ctx, "ttq.leasesRoot", "", 1000000+int64(shard), nil)
}

type lease struct {
	_kind string `gae:"$kind,ttq.lease"`

	Id              int64     `gae:"$id"`     // autoassigned. If not set, implies a noop lease.
	Parent          *ds.Key   `gae:"$parent"` // ttq.leasesRoot entity.
	SerializedParts []string  `gae:",noindex"`
	ExpiresAt       time.Time `gae:",noindex"` // precision up to microseconds.

	// Set only when lease object is created in save().
	parts partition.SortedPartitions `gae:"-"`
}

func save(ctx context.Context, shard int, expiresAt time.Time, parts partition.SortedPartitions) (*lease, error) {
	if len(parts) == 0 {
		return &lease{
			ExpiresAt: expiresAt,
			parts:     parts,
		}, nil // no need to save noop lease.
	}

	l := &lease{
		// ID will be autoassgined.
		Parent:          leasesRootKey(ctx, shard),
		SerializedParts: make([]string, len(parts)),
		ExpiresAt:       expiresAt.UTC(),
		parts:           parts,
	}
	for i, p := range parts {
		l.SerializedParts[i] = p.String()
	}
	if err := ds.Put(ctx, l); err != nil {
		return nil, errors.Annotate(err, "failed to save a new lease").Tag(transient.Tag).Err()
	}
	return l, nil
}

func (l *lease) delete(ctx context.Context) {
	if l.Id == 0 {
		return // noop leases are not saved.
	}
	if err := ds.Delete(ctx, l); err != nil {
		// Log only. Once lease expires, it'll garbage-collected next time a new
		// lease is acquired for the same shard.
		logging.Warningf(ctx, "failed to delete lease %v", l)
	}
}

func loadAll(ctx context.Context, shard int) (active, expired []*lease, err error) {
	var all []*lease
	q := ds.NewQuery("ttq.lease").Ancestor(leasesRootKey(ctx, shard))
	if err := ds.GetAll(ctx, q, &all); err != nil {
		return nil, nil, errors.Annotate(err, "failed to fetch leases").Tag(transient.Tag).Err()
	}
	// Partition active leases in the front and expired at the end of the slice.
	i, j := 0, len(all)
	now := clock.Now(ctx)
	for i < j {
		if all[i].ExpiresAt.After(now) {
			i++
			continue
		}
		j--
		all[i], all[j] = all[j], all[i]
	}
	return all[:i], all[i:], nil
}

func availableForLease(desired *partition.Partition, active []*lease) (partition.SortedPartitions, error) {
	builder := partition.NewSortedPartitionsBuilder(desired)
	// Exclude from desired all partitions under currently active leases.
	// TODO(tandrii): constrain number of partitions per lease to avoid excessive
	// runtime here.
	for _, l := range active {
		for _, s := range l.SerializedParts {
			p, err := partition.FromString(s)
			if err != nil {
				return nil, err
			}
			builder.Exclude(p)
			if builder.IsEmpty() {
				break
			}
		}
	}
	return builder.Result(), nil
}
