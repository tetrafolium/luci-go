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

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/sync/dispatcher/buffer"
)

// Channel holds a chan which you can push individual work items to.
type Channel struct {
	// C is an unbuffered channel which you can push single work items into.
	//
	// Close this to shut down the Channel.
	C chan<- interface{}

	// DrainC will unblock when this Channel is closed/canceled and fully drained.
	DrainC <-chan struct{}
}

// Close is a convenience function which closes C (and swallows panic if
// already closed).
func (c Channel) Close() {
	defer func() { recover() }()
	close(c.C)
}

// CloseAndDrain is a convenience function which closes C (and swallows panic if
// already closed) and then blocks on DrainC/ctx.Done().
func (c Channel) CloseAndDrain(ctx context.Context) {
	c.Close()
	select {
	case <-ctx.Done():
	case <-c.DrainC:
	}
}

// IsDrained returns true iff the Channel is closed and drained.
func (c Channel) IsDrained() bool {
	select {
	case <-c.DrainC:
		return true
	default:
		return false
	}
}

// SendFn is the function which does the work to actually transmit the Batch to
// the next stage of your processing pipeline (e.g. do an RPC to a remote
// service).
//
// The function may manipulate the Batch however it wants (see Batch).
//
// In particular, shrinking the size of Batch.Data for confirmed-sent items
// will allow the dispatcher to reduce its buffer count when SendFn returns,
// even if SendFn returns an error. Removing items from the Batch will not
// cause the remaining items to be coalesced into a different Batch.
//
// The SendFn MUST be bound to this Channel's Context; if the Channel's Context
// is Cancel'd, SendFn MUST terminate, or the Channel's DrainC will be blocked.
// We don't pass it as part of SendFn's signature in case SendFn needs to be
// bound to a derived Context.
//
// Non-nil errors returned by this function will be handled by ErrorFn.
type SendFn func(data *buffer.Batch) error

// NewChannel produces a new Channel ready to listen and send work items.
//
// Args:
//   * `ctx` will be used for cancellation and logging. When the `ctx` is
//     canceled, the Channel will:
//       * drop all incoming data on Channel.C; All new data will be dropped
//         (calling DropFn).
//       * drop all existing unleased batches (calling DropFn)
//       * ignore all errors from SendFn (i.e. even if ErrorFn returns
//         'retry=true', the batch will be dropped anyway)
//       If you want to gracefully drain the Channel, you must close the channel
//       and wait for DrainC before canceling the context.
//   * `send` is required, and defines the function to use to process Batches
//     of data. This function MUST respect `ctx.Done`, or the Channel cannot
//     drain properly.
//   * `opts` is optional (see Options for the defaults).
//
// The Channel MUST be Close()'d when you're done with it, or the Channel will
// not terminate. This applies even if you cancel it via ctx. The caller is
// responsible for this (as opposed to having Channel implement this internally)
// because there is no generally-safe way in Go to close a channel without
// coordinating that event with all senders on that channel. Because the caller
// of NewChannel is effectively the sender (owner) of Channel.C, they must
// coordinate closure of this channel with all their use of sends to this
// channel.
func NewChannel(ctx context.Context, opts *Options, send SendFn) (Channel, error) {
	if send == nil {
		return Channel{}, errors.New("send is required: got nil")
	}

	var optsCopy Options
	if opts != nil {
		optsCopy = *opts
	}

	buf, err := buffer.NewBuffer(&optsCopy.Buffer)
	if err != nil {
		return Channel{}, errors.Annotate(err, "allocating Buffer").Err()
	}

	if err = optsCopy.normalize(ctx); err != nil {
		return Channel{}, errors.Annotate(err, "normalizing dispatcher.Options").Err()
	}

	itemCh := make(chan interface{})
	drainCh := make(chan struct{})

	cstate := coordinatorState{
		opts:    optsCopy,
		buf:     buf,
		itemCh:  itemCh,
		drainCh: drainCh,

		resultCh: make(chan workerResult),

		timer: clock.NewTimer(clock.Tag(ctx, "coordinator")),
	}

	go cstate.run(ctx, send)
	return Channel{C: itemCh, DrainC: drainCh}, nil
}
