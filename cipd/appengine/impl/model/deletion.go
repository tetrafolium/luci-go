// Copyright 2018 The LUCI Authors.
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
	"strings"

	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/sync/parallel"

	api "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"
)

// DeletePackage deletes all entities associated with a package.
//
// Deleting everything under a single transaction is generally not feasible,
// since deleting packages with lots of metadata exceeds transaction limits.
//
// Instead we delete most of the stuff non-transactionally first, and then
// delete the rest inside the transaction (to make sure nothing stays, even if
// it was created while we were deleting stuff).
//
// Returns grpc-tagged errors (in particular NotFound if there's no such
// package).
func DeletePackage(c context.Context, pkg string) error {
	if err := CheckPackageExists(c, pkg); err != nil {
		return err
	}

	// Note that to maintain data model consistency during the deletion, we delete
	// various metadata first, and PackageInstance entities second (to make sure
	// we don't have metadata with dangling pointers to deleted instances).
	err := deleteEntityKinds(c, pkg, []string{
		"InstanceTag",
		"PackageRef",
		"ProcessingResult",
	})
	if err != nil {
		return err
	}
	if err := deleteEntityKinds(c, pkg, []string{"PackageInstance"}); err != nil {
		return nil
	}

	// Cleanup whatever remains. It contains entities that were created while we
	// were deleting stuff above. There should be very few (usually 0) such
	// entities, so it's OK to delete them transactionally.
	return Txn(c, "DeletePackage", func(c context.Context) error {
		err := deleteEntityKinds(c, pkg, []string{
			"InstanceTag",
			"PackageRef",
			"ProcessingResult",
			"PackageInstance",
		})
		if err != nil {
			return err
		}
		if err := datastore.Delete(c, PackageKey(c, pkg)); err != nil {
			return transient.Tag.Apply(err)
		}
		return EmitEvent(c, &api.Event{
			Kind:    api.EventKind_PACKAGE_DELETED,
			Package: pkg,
		})
	})
}

var (
	// Number of keys to delete at once in deleteKinds. Replaced in tests.
	deletionBatchSize = 256
)

// deleteEntityKinds deletes all entities of given kinds under given root.
func deleteEntityKinds(c context.Context, pkg string, kindsToDelete []string) error {
	logging.Infof(c, "Deleting %s...", strings.Join(kindsToDelete, ", "))
	return transient.Tag.Apply(parallel.WorkPool(len(kindsToDelete)+1, func(tasks chan<- func() error) {
		// A channel that receives keys to delete. Set some arbitrary buffer size to
		// parallelize work a bit better.
		keys := make(chan *datastore.Key, 1024)

		// Launch queries that fetch keys to delete, and feed them to the channel.
		// Each query enqueues nil when it is done to let the consumer know.
		for _, kind := range kindsToDelete {
			kind := kind
			tasks <- func() error {
				q := datastore.NewQuery(kind).
					Ancestor(PackageKey(c, pkg)).
					KeysOnly(true)

				count := 0
				err := datastore.Run(c, q, func(k *datastore.Key, _ datastore.CursorCB) error {
					count++
					keys <- k
					return nil
				})

				keys <- nil // put "we are done" signal
				if err == nil {
					logging.Infof(c, "Found %d %q entities to be deleted", count, kind)
				} else {
					logging.WithError(err).Errorf(c, "Found %d %q entities and then failed", count, kind)
				}
				return err
			}
		}

		// Delete keys in batches. Stop when we get all "we are done" signals from
		// all len(kindsToDelete) queries. Carry on on errors, to make sure to
		// drain the channel completely (otherwise enqueue ops will get blocked).
		tasks <- func() error {
			var batch []*datastore.Key
			var errs errors.MultiError

			// flush deletes all keys recorded in 'batch'.
			flush := func() {
				logging.Infof(c, "Deleting %d entities...", len(batch))
				if err := datastore.Delete(c, batch); err != nil {
					logging.WithError(err).Errorf(c, "Failed to delete %d entities", len(batch))
					errs = append(errs, err)
				}
				batch = batch[:0]
			}

			stillRunning := len(kindsToDelete)
			for k := range keys {
				if k != nil {
					if batch = append(batch, k); len(batch) >= deletionBatchSize {
						flush()
					}
					continue
				}

				// Got "we are done" signal from some query.
				if stillRunning--; stillRunning == 0 {
					break // all queries are done
				}
			}

			// Flush the last incomplete batch.
			if len(batch) > 0 {
				flush()
			}
			if len(errs) != 0 {
				return errs
			}
			return nil
		}
	}))
}
