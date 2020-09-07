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

package sweep

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/sync/dispatcher"
	"github.com/tetrafolium/luci-go/common/sync/dispatcher/buffer"

	"github.com/tetrafolium/luci-go/server/tq/internal"
	"github.com/tetrafolium/luci-go/server/tq/internal/db"
	"github.com/tetrafolium/luci-go/server/tq/internal/reminder"
)

// BatchProcessor handles reminders in batches.
type BatchProcessor struct {
	Context   context.Context    // the context to use for processing
	DB        db.DB              // DB to use to fetch reminders from
	Submitter internal.Submitter // knows how to submit tasks

	BatchSize         int // max size of a single reminder batch
	ConcurrentBatches int // how many concurrent batches to process

	ch        dispatcher.Channel
	processed int32 // total reminders successfully processed
}

// Start launches background processor goroutines.
func (p *BatchProcessor) Start() error {
	var err error
	p.ch, err = dispatcher.NewChannel(
		p.Context,
		&dispatcher.Options{
			Buffer: buffer.Options{
				MaxLeases: p.ConcurrentBatches,
				BatchSize: p.BatchSize,
				// Max waiting time to fill the batch.
				BatchDuration: 10 * time.Millisecond,
				FullBehavior: &buffer.BlockNewItems{
					// If all workers are busy, block Enqueue.
					MaxItems: p.ConcurrentBatches * p.BatchSize,
				},
			},
		},
		p.processBatch,
	)
	if err != nil {
		return errors.Annotate(err, "invalid sweeper configuration").Err()
	}
	return nil
}

// Stop waits until all enqueues reminders are processed and then stops the
// processor.
//
// Returns the total number of successfully processed reminders.
func (p *BatchProcessor) Stop() int {
	p.ch.Close()
	<-p.ch.DrainC
	return int(atomic.LoadInt32(&p.processed))
}

// Enqueue adds reminder to the to-be-processed queue.
//
// Must be called only between Start and Stop. Drops reminders on the floor if
// the context is canceled.
func (p *BatchProcessor) Enqueue(ctx context.Context, r []*reminder.Reminder) {
	for _, rem := range r {
		select {
		case p.ch.C <- rem:
		case <-ctx.Done():
			return
		}
	}
}

// processBatch called concurrently to handle a single batch of items.
//
// Logs errors inside, doesn't return them.
func (p *BatchProcessor) processBatch(data *buffer.Batch) error {
	batch := make([]*reminder.Reminder, len(data.Data))
	for i, d := range data.Data {
		batch[i] = d.(*reminder.Reminder)
	}
	count, err := internal.SubmitBatch(p.Context, p.Submitter, p.DB, batch)
	if err != nil {
		logging.Errorf(p.Context, "Processed only %d/%d reminders: %s", count, len(batch), err)
	}
	atomic.AddInt32(&p.processed, int32(count))
	return nil
}
