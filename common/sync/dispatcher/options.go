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

package dispatcher

import (
	"context"

	"golang.org/x/time/rate"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/sync/dispatcher/buffer"
)

// ErrorFn is called to handle the error from SendFn.
//
// It executes in the main handler loop of the dispatcher so it can make
// synchronous decisions about the dispatcher state.
//
// Blocking in this function will block ALL dispatcher actions, so be quick
// :).
//
// DO NOT WRITE TO THE CHANNEL DIRECTLY FROM THIS FUNCTION. Doing so will very
// likely cause deadlocks.
//
// This may:
//   * inspect/log the error
//   * manipulate the contents of failedBatch
//   * return a boolean of whether this Batch should be retried or not. If
//     this is false then the Batch is dropped. If it's true, then it will be
//     re-queued as-is for transmission according to BufferFullBehavior.
//   * pass the Batch.Data to another goroutine (in a non-blocking way!) to be
//     re-queued through Channel.WriteChan.
//
// Args:
//   * failedBatch - The Batch for which SendFn produced a non-nil error.
//   * err - The error SendFn produced.
//
// Returns true iff the dispatcher should re-try sending this Batch, according
// to Buffer.Retry.
type ErrorFn func(failedBatch *buffer.Batch, err error) (retry bool)

// Options is the configuration options for NewChannel.
type Options struct {
	// [OPTIONAL] The ErrorFn to use (see ErrorFn docs for details).
	//
	// Default: Logs the error (at Info for retryable errors, and Error for
	// non-retryable errors) and returns true on a transient error.
	ErrorFn ErrorFn

	// [OPTIONAL] Called with the dropped batch any time the Channel drops a batch.
	//
	// This includes:
	//   * When FullBehavior==DropOldestBatch and we get new data.
	//   * When FullBehavior==DropOldestBatch and we attempt to retry old data.
	//   * When ErrorFn returns false for a batch.
	//
	// It executes in the main handler loop of the dispatcher so it can make
	// synchronous decisions about the dispatcher state.
	//
	// Blocking in this function will block ALL dispatcher actions, so be quick
	// :).
	//
	// DO NOT WRITE TO THE CHANNEL DIRECTLY FROM THIS FUNCTION. Doing so will very
	// likely cause deadlocks.
	//
	// When the channel is fully drained, this will be invoked exactly once with
	// `(nil, true)`. This will occur immediately before the DrainedFn is called.
	// Some drop functions buffer their information, and this gives them an
	// opportunity to flush out any buffered data.
	//
	// Default: logs (at Info level if FullBehavior==DropOldestBatch, or Warning
	// level otherwise) the number of data items in the Batch being dropped.
	DropFn func(b *buffer.Batch, flush bool)

	// [OPTIONAL] Called exactly once when the associated Channel is closed and
	// has fully drained its buffer, but before DrainC is closed.
	//
	// Note that this takes effect whether the Channel is shut down via Context
	// cancellation or explicitly by closing Channel.C.
	//
	// This is useful for performing final state synchronization tasks/metrics
	// finalization/helpful "everything is done!" messages/etc. without having to
	// poll the Channel to see if it's done and also maintain external
	// synchronization around the finalization action.
	//
	// Called in the main handler loop, but it's called after all other work is
	// done by the Channel, so the only thing it blocks is the closure of DrainC.
	//
	// Default: No action.
	DrainedFn func()

	// [OPTIONAL] A rate limiter for how frequently this will invoke SendFn.
	//
	// Default: No limit.
	QPSLimit *rate.Limiter

	Buffer buffer.Options

	// Debug output for tests.
	testingDbg func(string, ...interface{})
}

func defaultDropFnFactory(ctx context.Context, fullBehavior buffer.FullBehavior) func(*buffer.Batch, bool) {
	return func(dropped *buffer.Batch, flush bool) {
		if flush {
			return
		}
		logFn := logging.Warningf
		if _, ok := fullBehavior.(*buffer.DropOldestBatch); ok {
			logFn = logging.Infof
		}
		logFn(
			ctx,
			"dropping Batch(len(Data): %d, Meta: %+v)",
			len(dropped.Data), dropped.Meta)
	}
}

func defaultErrorFnFactory(ctx context.Context) ErrorFn {
	return func(failedBatch *buffer.Batch, err error) (retry bool) {
		retry = transient.Tag.In(err)
		logFn := logging.Errorf
		if retry {
			logFn = logging.Infof
		}
		logFn(
			ctx,
			"failed to send Batch(len(Data): %d, Meta: %+v): %s",
			len(failedBatch.Data), failedBatch.Meta, err)

		return
	}
}

// ErrorFnQuiet is an implementation of Options.ErrorFn which doesn't log the
// batch, but does check for `transient.Tag` to determine `retry`.
func ErrorFnQuiet(b *buffer.Batch, err error) (retry bool) {
	return transient.Tag.In(err)
}

// DropFnQuiet is an implementation of Options.DropFn which drops batches
// without logging anything.
func DropFnQuiet(*buffer.Batch, bool) {}

// DropFnSummarized returns an implementation of Options.DropFn which counts the
// number of dropped batches, and only reports it at the rate provided.
//
// Unlike the default log function, this only logs the number of dropped items
// and the duration that they were collected over.
func DropFnSummarized(ctx context.Context, lim *rate.Limiter) func(*buffer.Batch, bool) {
	durationStart := clock.Now(ctx)
	dropCount := 0
	return func(b *buffer.Batch, flush bool) {
		dataLen := 0
		if b != nil {
			dataLen = len(b.Data)
		}
		if lim.Allow() || flush {
			now := clock.Now(ctx)
			logging.Infof(
				ctx, "dropped %d items over %s", dropCount+dataLen, now.Sub(durationStart))
			durationStart = now
			dropCount = 0
		} else {
			dropCount += dataLen
		}
	}
}
