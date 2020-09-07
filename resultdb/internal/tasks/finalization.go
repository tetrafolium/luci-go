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

package tasks

import (
	"context"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/server/experiments"
	"github.com/tetrafolium/luci-go/server/span"
	"github.com/tetrafolium/luci-go/server/tq"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
	"github.com/tetrafolium/luci-go/resultdb/internal/spanutil"
	"github.com/tetrafolium/luci-go/resultdb/internal/tasks/taskspb"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"

	// Add support for Spanner transactions in TQ.
	_ "github.com/tetrafolium/luci-go/server/tq/txn/spanner"
)

// FinalizationTasks describes how to route finalization tasks.
//
// The handler is implemented in internal/services/finalizer.
var FinalizationTasks = tq.RegisterTaskClass(tq.TaskClass{
	ID:                  "try-finalize-inv",
	Prototype:           &taskspb.TryFinalizeInvocation{},
	Kind:                tq.Transactional,
	InheritTraceContext: true,
	Queue:               "finalizer",                 // use a dedicated queue
	RoutingPrefix:       "/internal/tasks/finalizer", // for routing to "finalizer" service
})

// UseFinalizationTQ experiment enables using server/tq for finalization tasks.
var UseFinalizationTQ = experiments.Register("rdb-use-tq-finalization")

// StartInvocationFinalization changes invocation state to FINALIZING
// and enqueues a TryFinalizeInvocation task.
//
// The caller is responsible for ensuring that the invocation is active.
//
// TODO(nodir): this package is not a great place for this function, but there
// is no better package at the moment. Keep it here for now, but consider a
// new package as the code base grows.
func StartInvocationFinalization(ctx context.Context, id invocations.ID) {
	span.BufferWrite(ctx, spanutil.UpdateMap("Invocations", map[string]interface{}{
		"InvocationId": id,
		"State":        pb.Invocation_FINALIZING,
	}))
	if UseFinalizationTQ.Enabled(ctx) {
		tq.MustAddTask(ctx, &tq.Task{
			Payload: &taskspb.TryFinalizeInvocation{InvocationId: string(id)},
			Title:   string(id),
		})
	} else {
		span.BufferWrite(ctx, Enqueue(TryFinalizeInvocation, "finalize/"+id.RowID(), id, nil, clock.Now(ctx).UTC()))
	}
}
