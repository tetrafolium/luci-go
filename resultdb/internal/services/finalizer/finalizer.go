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

package finalizer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/common/trace"
	"github.com/tetrafolium/luci-go/server"
	"github.com/tetrafolium/luci-go/server/span"
	"github.com/tetrafolium/luci-go/server/tq"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/tasks"
	"github.com/tetrafolium/luci-go/resultdb/internal/tasks/taskspb"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// Options is finalizer server configuration.
type Options struct {
	// How often to query for tasks.
	TaskQueryInterval time.Duration

	// How long to lease a task for.
	TaskLeaseDuration time.Duration

	// Number of tasks to process concurrently.
	TaskWorkers int
}

// DefaultOptions returns Options with default values.
func DefaultOptions() Options {
	return Options{
		TaskQueryInterval: 5 * time.Second,
		TaskLeaseDuration: time.Minute,
		TaskWorkers:       10,
	}
}

// InitServer initializes a finalizer server.
func InitServer(srv *server.Server, opts Options) {
	d := tasks.Dispatcher{
		QueryInterval: opts.TaskQueryInterval,
		LeaseDuration: opts.TaskLeaseDuration,
		Workers:       opts.TaskWorkers,
	}
	srv.RunInBackground("finalize", func(ctx context.Context) {
		d.Run(ctx, tasks.TryFinalizeInvocation, func(ctx context.Context, invID invocations.ID, payload []byte) error {
			return tryFinalizeInvocation(ctx, invID)
		})
	})
}

func init() {
	tasks.FinalizationTasks.AttachHandler(func(ctx context.Context, msg proto.Message) error {
		task := msg.(*taskspb.TryFinalizeInvocation)
		return tryFinalizeInvocation(ctx, invocations.ID(task.InvocationId))
	})
}

// Invocation finalization is asynchronous. First, an invocation transitions
// from ACTIVE to FINALIZING state and transactionally an invocation task is
// enqueued to try to transition it from FINALIZING to FINALIZED.
// Then the task tries to finalize the invocation:
// 1. Check if the invocation is ready to be finalized.
// 2. Finalize the invocation.
//
// The invocation is ready to be finalized iff it is in FINALIZING state and it
// does not include, directly or indirectly, an active invocation.
// The latter involves a graph traversal.
// Given that a client cannot mutate inclusions of a FINALIZING/FINALIZED
// invocation, this means that once an invocation is ready to be finalized,
// it cannot become un-ready. This is why the check is done in a ready-only
// transaction with minimal contention.
// If the invocation is not ready to finalize, the task is dropped.
// This check is implemented in readyToFinalize() function.
//
// The second part is actual finalization. It is done in a separate read-write
// transaction. First the task checks again if the invocation is still
// FINALIZING. If so, the task changes state to FINALIZED, enqueues BQExport
// tasks and tasks to try to finalize invocations that directly include the
// current one (more about this below).
// The finalization is implemented in finalizeInvocation() function.
//
// If we have a chain of inclusions A includes B, B includes C, where A and B
// are FINALIZING and C is active, then A and B are waiting for C to be
// finalized.
// In this state, tasks attempting to finalize A or B will conclude that they
// are not ready.
// Once C is finalized, a task to try to finalize B is enqueued.
// B gets finalized and it enqueues a task to try to finalize A.
// More generally speaking, whenever a node transitions from FINALIZING to
// FINALIZED, we ping incoming edges. This may cause a chain of pings along
// the edges.
//
// More specifically, given edge (A, B), when finalizing B, A is pinged only if
// it is FINALIZING. It does not make sense to do it if A is FINALIZED for
// obvious reasons; and there is no need to do it if A is ACTIVE because
// a transition ACTIVE->FINALIZING is always accompanied with enqueuing a task
// to try to finalize it.

// tryFinalizeInvocation finalizes the invocation unless it directly or
// indirectly includes an ACTIVE invocation.
// If the invocation is too early to finalize, logs the reason and returns nil.
// Idempotent.
func tryFinalizeInvocation(ctx context.Context, invID invocations.ID) error {
	// The check whether the invocation is ready to finalize involves traversing
	// the invocation graph and reading Invocations.State column. Doing so in a
	// RW transaction will cause contention. Fortunately, once an invocation
	// is ready to finalize, it cannot go back to being unready, so doing
	// check and finalization in separate transactions is fine.
	switch ready, err := readyToFinalize(ctx, invID); {
	case err != nil:
		return err

	case !ready:
		return nil

	default:
		logging.Infof(ctx, "decided to finalize %s...", invID.Name())
		return finalizeInvocation(ctx, invID)
	}
}

var errAlreadyFinalized = fmt.Errorf("the invocation is already finalized")

// notReadyToFinalize means the invocation is not ready to finalize.
// It is used exclusively inside readyToFinalize.
var notReadyToFinalize = errors.BoolTag{Key: errors.NewTagKey("not ready to get finalized")}

// readyToFinalize returns true if the invocation should be finalized.
// An invocation is ready to be finalized if no ACTIVE invocation is reachable
// from it.
func readyToFinalize(ctx context.Context, invID invocations.ID) (ready bool, err error) {
	ctx, ts := trace.StartSpan(ctx, "resultdb.readyToFinalize")
	defer func() { ts.End(err) }()

	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()

	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	// Ensure the root invocation is in FINALIZING state.
	eg.Go(func() error {
		return ensureFinalizing(ctx, invID)
	})

	// Walk the graph of invocations, starting from the root, along the inclusion
	// edges.
	// Stop walking as soon as we encounter an active invocation.
	seen := make(invocations.IDSet, 1)
	var mu sync.Mutex

	// Limit the number of concurrent queries.
	sem := semaphore.NewWeighted(64)

	var visit func(id invocations.ID)
	visit = func(id invocations.ID) {
		// Do not visit same node twice.
		mu.Lock()
		if seen.Has(id) {
			mu.Unlock()
			return
		}
		seen.Add(id)
		mu.Unlock()

		// Concurrently fetch inclusions without a lock.
		eg.Go(func() error {
			// Limit concurrent Spanner queries.
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			// Ignore inclusions of FINALIZED invocations. An ACTIVE invocation is
			// certainly not reachable from those.
			st := spanner.NewStatement(`
				SELECT included.InvocationId, included.State
				FROM IncludedInvocations incl
				JOIN Invocations included on incl.IncludedInvocationId = included.InvocationId
				WHERE incl.InvocationId = @invID AND included.State != @finalized
			`)
			st.Params = spanutil.ToSpannerMap(map[string]interface{}{
				"finalized": pb.Invocation_FINALIZED,
				"invID":     id,
			})
			var b spanutil.Buffer
			return span.Query(ctx, st).Do(func(row *spanner.Row) error {
				var includedID invocations.ID
				var includedState pb.Invocation_State
				switch err := b.FromSpanner(row, &includedID, &includedState); {
				case err != nil:
					return err

				case includedState == pb.Invocation_ACTIVE:
					return errors.Reason("%s is still ACTIVE", includedID.Name()).Tag(notReadyToFinalize).Err()

				case includedState != pb.Invocation_FINALIZING:
					return errors.Reason("%s has unexpected state %s", includedID.Name(), includedState).Err()

				default:
					// The included invocation is FINALIZING and MAY include other
					// still-active invocations. We must go deeper.
					visit(includedID)
					return nil
				}
			})
		})
	}

	visit(invID)

	switch err := eg.Wait(); {
	case errors.Unwrap(err) == errAlreadyFinalized:
		// The invocation is already finalized.
		return false, nil

	case notReadyToFinalize.In(err):
		logging.Infof(ctx, "not ready to finalize: %s", err.Error())
		return false, nil

	default:
		return err == nil, err
	}
}

func ensureFinalizing(ctx context.Context, invID invocations.ID) error {
	switch state, err := invocations.ReadState(ctx, invID); {
	case err != nil:
		return err
	case state == pb.Invocation_FINALIZED:
		return errAlreadyFinalized
	case state != pb.Invocation_FINALIZING:
		return errors.Reason("expected %s to be FINALIZING, but it is %s", invID.Name(), state).Err()
	default:
		return nil
	}
}

// finalizeInvocation updates the invocation state to FINALIZED.
// Enqueues BigQuery export tasks.
// For each FINALIZING invocation that includes the given one, enqueues
// a finalization task.
func finalizeInvocation(ctx context.Context, invID invocations.ID) error {
	var reach invocations.IDSet
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		// Check the state before proceeding, so that if the invocation already
		// finalized, we return errAlreadyFinalized.
		if err := ensureFinalizing(ctx, invID); err != nil {
			return err
		}

		return parallel.FanOutIn(func(work chan<- func() error) {
			// Read all reachable invocations to cache them after the transaction.
			work <- func() (err error) {
				reach, err = invocations.ReachableSkipRootCache(ctx, invocations.NewIDSet(invID))
				return
			}

			// Enqueue tasks to try to finalize invocations that include ours.
			work <- func() error {
				if tasks.UseFinalizationTQ.Enabled(ctx) {
					parentInvs, err := parentsInFinalizingState(ctx, invID)
					if err != nil {
						return err
					}
					// Note that AddTask in a Spanner transaction is essentially
					// a BufferWrite (no RPCs inside), it's fine to call it sequentially
					// and panic on errors.
					for _, id := range parentInvs {
						tq.MustAddTask(ctx, &tq.Task{
							Payload: &taskspb.TryFinalizeInvocation{InvocationId: string(id)},
							Title:   string(id),
						})
					}
				} else {
					if err := insertNextFinalizationTasks(ctx, invID); err != nil {
						return err
					}
				}
				// Enqueue tasks to export the invocation to BigQuery.
				// Note: this cannot be done in parallel with insertNextFinalizationTasks
				// because a Spanner session can process only one DML query at a time.
				return insertBigQueryTasks(ctx, invID)
			}

			// Update the invocation state.
			work <- func() error {
				span.BufferWrite(ctx, spanutil.UpdateMap("Invocations", map[string]interface{}{
					"InvocationId": invID,
					"State":        pb.Invocation_FINALIZED,
					"FinalizeTime": spanner.CommitTimestamp,
				}))
				return nil
			}
		})
	})
	switch {
	case err == errAlreadyFinalized:
		return nil
	case err != nil:
		return err
	default:
		// Cache the reachable invocations.
		invocations.ReachCache(invID).TryWrite(ctx, reach)
		return nil
	}
}

// parentsInFinalizingState returns IDs of invocations in FINALIZING state that
// directly include ours.
func parentsInFinalizingState(ctx context.Context, invID invocations.ID) (ids []invocations.ID, err error) {
	st := spanner.NewStatement(`
		SELECT including.InvocationId
		FROM IncludedInvocations@{FORCE_INDEX=ReversedIncludedInvocations} incl
		JOIN Invocations including ON incl.InvocationId = including.InvocationId
		WHERE IncludedInvocationId = @invID AND including.State = @finalizing
	`)
	st.Params = spanutil.ToSpannerMap(map[string]interface{}{
		"invID":      invID.RowID(),
		"finalizing": pb.Invocation_FINALIZING,
	})
	err = span.Query(ctx, st).Do(func(row *spanner.Row) error {
		var id invocations.ID
		if err := spanutil.FromSpanner(row, &id); err != nil {
			return err
		}
		ids = append(ids, id)
		return nil
	})
	return ids, err
}

// insertNextFinalizationTasks, for each FINALIZING invocation that directly
// includes ours, schedules a task to try to finalize it.
func insertNextFinalizationTasks(ctx context.Context, invID invocations.ID) error {
	// Note: its OK not to schedule a task for active invocations because
	// state transition ACTIVE->FINALIZING includes creating a finalization
	// task.
	// Note: Spanner currently does not support PENDING_COMMIT_TIMESTAMP()
	// in "INSERT INTO ... SELECT" queries.
	st := spanner.NewStatement(`
		INSERT INTO InvocationTasks (TaskType, TaskId, InvocationId, CreateTime, ProcessAfter)
		SELECT @taskType, FORMAT("%s/%s", @invID, including.InvocationId), including.InvocationId, CURRENT_TIMESTAMP(), CURRENT_TIMESTAMP()
		FROM IncludedInvocations@{FORCE_INDEX=ReversedIncludedInvocations} incl
		JOIN Invocations including ON incl.InvocationId = including.InvocationId
		WHERE IncludedInvocationId = @invID AND including.State = @finalizing
	`)
	st.Params = spanutil.ToSpannerMap(map[string]interface{}{
		"taskType":   string(tasks.TryFinalizeInvocation),
		"invID":      invID.RowID(),
		"finalizing": pb.Invocation_FINALIZING,
	})
	count, err := span.Update(ctx, st)
	if err != nil {
		return errors.Annotate(err, "failed to insert further finalizing tasks").Err()
	}
	logging.Infof(ctx, "Inserted %d %s tasks", count, tasks.TryFinalizeInvocation)
	return nil
}

// insertBigQueryTasks inserts a bq_export invocation task for each element
// of Invocations.BigQueryExports array in the specified invocation.
func insertBigQueryTasks(ctx context.Context, invID invocations.ID) error {
	// Note: Spanner currently does not support PENDING_COMMIT_TIMESTAMP()
	// in "INSERT INTO ... SELECT" queries.
	st := spanner.NewStatement(`
		INSERT INTO InvocationTasks (TaskType, TaskId, InvocationId, Payload, CreateTime, ProcessAfter)
		SELECT @taskType, FORMAT("%s:%d",  @invID, i), @invID, payload, CURRENT_TIMESTAMP(), CURRENT_TIMESTAMP()
		FROM Invocations inv, UNNEST(inv.BigQueryExports) payload WITH OFFSET AS i
		WHERE inv.InvocationId = @invID
	`)
	st.Params = spanutil.ToSpannerMap(map[string]interface{}{
		"taskType": string(tasks.BQExport),
		"invID":    invID.RowID(),
	})
	count, err := span.Update(ctx, st)
	if err != nil {
		return errors.Annotate(err, "failed to insert bq_export tasks").Err()
	}
	logging.Infof(ctx, "Inserted %d %s tasks", count, tasks.BQExport)
	return nil
}
