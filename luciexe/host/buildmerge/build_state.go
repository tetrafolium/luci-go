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

package buildmerge

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/golang/protobuf/proto"
	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/sync/dispatcher"
	"github.com/tetrafolium/luci-go/common/sync/dispatcher/buffer"
	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/common/types"
)

// buildState represets the current state of a single build.proto stream.
type buildState struct {
	// build holds the most recently processed Build state. This message should be
	// treated as immutable (i.e. proto.Clone before modifying it).
	//
	// This may be `nil` until the first user-supplied build.proto is processed,
	// or until the buildStateTracker closes.
	build *bbpb.Build

	// closed is set to true when the build state is terminated and will receive
	// no more user updates (but may still need to be finalized()).
	closed bool

	// final is set to true when the build state is closed and all final
	// processing has occurred on the build state.
	final bool

	// invalid is set to true when the interior structure (i.e. Steps) of latest
	// contains invalid data and shouldn't be inspected.
	invalid bool
}

// buildStateTracker manages the state of a single build.proto datagram stream.
type buildStateTracker struct {
	ctx context.Context

	// The Agent that this buildStateTracker belongs to. Used to access:
	//   * clockNow
	//   * calculateURLs
	//   * informNewData
	merger *Agent

	ldNamespace types.StreamName

	// True iff we should expect zlib-compressed datagrams.
	zlib bool

	// We use this mutex to synchronize closure and sending operations on the work
	// channel; `work` is configured, if it's running, to immediately accept any
	// items pushed to it, so it's safe to hold this while sending on work.C.
	workMu sync.Mutex

	// The work channel is configured to only keep the latest incoming datagram.
	// It's send function parses and interprets the Build message.
	// Errors are not reported to the dispatcher.Channel, but are instead recorded
	// in the parsed Build state.
	work       dispatcher.Channel
	workClosed bool // true if we've closed work.C, protected by workMu

	latestStateMu sync.Mutex
	latestState   *buildState
}

// processDataUnlocked updates `state` with the Build.proto message contained as
// binary-encoded proto in `data`.
//
// If there's an error parsing `data`, or an error in the decoded message's
// contents, `state.invalid` and `state.closed` will be set to true, and
// `state.build` will be updated with the error message.
func (t *buildStateTracker) processDataUnlocked(state *buildState, data []byte) {
	var parsedBuild *bbpb.Build
	err := func() error {
		if t.zlib {
			z, err := zlib.NewReader(bytes.NewBuffer(data))
			if err != nil {
				return errors.Annotate(err, "constructing decompressor for Build").Err()
			}
			data, err = ioutil.ReadAll(z)
			if err != nil {
				return errors.Annotate(err, "decompressing Build").Err()
			}
		}

		build := &bbpb.Build{}
		if err := proto.Unmarshal(data, build); err != nil {
			return errors.Annotate(err, "parsing Build").Err()
		}
		parsedBuild = build

		for _, step := range parsedBuild.Steps {
			for _, log := range step.Logs {
				url := types.StreamName(log.Url)
				if err := url.Validate(); err != nil {
					step.Status = bbpb.Status_INFRA_FAILURE
					step.SummaryMarkdown += fmt.Sprintf("bad log url: %q", log.Url)
					return errors.Annotate(
						err, "step[%q].logs[%q].Url = %q", step.Name, log.Name, log.Url).Err()
				}

				log.Url, log.ViewUrl = t.merger.calculateURLs(t.ldNamespace, url)
			}
		}
		return nil
	}()
	if err != nil {
		if parsedBuild == nil {
			if state.build == nil {
				parsedBuild = &bbpb.Build{}
			} else {
				// make a shallow copy of the latest build
				buildVal := *state.build
				parsedBuild = &buildVal
			}
		}
		setErrorOnBuild(parsedBuild, err)
		state.closed = true
		state.invalid = true
	}

	state.build = parsedBuild
}

// newBuildStateTracker produces a new buildStateTracker in the given logdog
// namespace.
//
// `ctx` is used for cancellation/logging.
//
// `merger` is the Agent that this buildStateTracker belongs to. See the comment
// in buildStateTracker for its use of this.
//
// `namespace` is the logdog namespace under which this build.proto is being
// streamed from. e.g. if the updates to handleNewData are coming from a logdog
// stream "a/b/c/build.proto", then `namespace` here should be "a/b/c". This is
// used verbatim as the namespace argument to merger.calculateURLs.
//
// if `err` is provided, the buildStateTracker tracker is created in an errored
// (closed) state where getLatest always returns a fixed Build in the
// INFRA_FAILURE state with `err` reflected in the build's SummaryMarkdown
// field.
func newBuildStateTracker(ctx context.Context, merger *Agent, namespace types.StreamName, zlib bool, err error) *buildStateTracker {
	ret := &buildStateTracker{
		ctx:         ctx,
		merger:      merger,
		zlib:        zlib,
		ldNamespace: namespace.AsNamespace(),
		latestState: &buildState{},
	}

	if err != nil {
		ret.latestState.build = &bbpb.Build{}
		setErrorOnBuild(ret.latestState.build, err)
		ret.finalize()
	} else {
		ret.work, err = dispatcher.NewChannel(ctx, &dispatcher.Options{
			Buffer: buffer.Options{
				MaxLeases:    1,
				BatchSize:    1,
				FullBehavior: &buffer.DropOldestBatch{},
			},
			DropFn:    dispatcher.DropFnQuiet,
			DrainedFn: ret.finalize,
		}, ret.parseAndSend)
		if err != nil {
			panic(err) // creating dispatcher with static config should never fail
		}
		// Attach the cancelation of the context to the closure of work.C.
		go func() {
			select {
			case <-ctx.Done():
				ret.Close()
			case <-ret.work.DrainC:
				// already shut down w/o cancelation
			}
		}()
	}

	return ret
}

// finalized is called exactly once when either:
//
//  * newBuildStateTracker is called with err != nil
//  * buildStateTracker.work is fully shut down (this is installed as
//    dispatcher.Options.DrainedFn)
func (t *buildStateTracker) finalize() {
	t.latestStateMu.Lock()
	defer t.latestStateMu.Unlock()

	state := *t.latestState
	if state.final {
		panic("impossible; finalize called twice?")
	}

	state.closed = true
	state.final = true
	if state.build == nil {
		state.build = &bbpb.Build{
			SummaryMarkdown: "Never received any build data.",
			Status:          bbpb.Status_INFRA_FAILURE,
		}
	} else {
		buildVal := *state.build
		state.build = &buildVal
	}
	processFinalBuild(t.merger.clockNow(), state.build)

	t.latestState = &state
	t.merger.informNewData()
}

func (t *buildStateTracker) parseAndSend(data *buffer.Batch) error {
	t.latestStateMu.Lock()
	state := *t.latestState
	t.latestStateMu.Unlock()

	// already closed
	if state.closed {
		return nil
	}

	oldBuild := state.build

	// may set state.closed on an error
	t.processDataUnlocked(&state, data.Data[0].([]byte))

	// if we didn't update state.build, make a shallow copy.
	if oldBuild == state.build {
		buildVal := *state.build
		state.build = &buildVal
	}

	if state.closed {
		t.Close()
	} else {
		state.build.UpdateTime = t.merger.clockNow()
	}

	t.latestStateMu.Lock()
	t.latestState = &state
	t.latestStateMu.Unlock()
	t.merger.informNewData()
	return nil
}

// getLatest returns the current state of the Build. See `buildState`.
//
// This always returns a non-nil buildState.build to make the calling code
// simpler.
func (t *buildStateTracker) getLatest() *buildState {
	t.latestStateMu.Lock()
	defer t.latestStateMu.Unlock()

	state := *t.latestState
	if state.build == nil {
		state.build = &bbpb.Build{
			SummaryMarkdown: "build.proto not found",
			Status:          bbpb.Status_SCHEDULED,
		}
	}
	return &state
}

// GetFinal waits for the build state to finalize then returns the final state
// of the Build.
//
// This always returns a non-nil buildState.build to make the calling code
// simpler.
//
// The returned buildState will always have `buildState.final == true`.
func (t *buildStateTracker) GetFinal() *buildState {
	if t.work.DrainC != nil {
		<-t.work.DrainC
	}
	return t.getLatest()
}

// This implements the bundler.StreamChunkCallback callback function.
//
// Each call to `handleNewData` expects `entry` to have a complete (non-Partial)
// datagram containing a single Build message. The message will (eventually) be
// parsed and fixed up (e.g. fixing Log Url/ViewUrl), and become this
// buildStateTracker's new state.
//
// This method does not block; Data here is submitted to the buildStateTracker's
// internal worker, which processes state updates as quickly as it can, skipping
// state updates which are submitted too rapidly.
//
// This method has no effect if the buildStateTracker is 'closed'.
//
// When this is called with `nil` as an argument (when the attached logdog
// stream is closed), it will start the closure process on this
// buildStateTracker. The final build state can be obtained synchronously by
// calling GetFinal().
func (t *buildStateTracker) handleNewData(entry *logpb.LogEntry) {
	t.workMu.Lock()
	defer t.workMu.Unlock()

	if entry == nil {
		t.closeWorkLocked()
	} else if !t.workClosed {
		select {
		case t.work.C <- entry.GetDatagram().Data:
		case <-t.ctx.Done():
			t.closeWorkLocked()
		}
	}
}

func (t *buildStateTracker) closeWorkLocked() {
	if !t.workClosed && t.work.C != nil {
		close(t.work.C)
		t.workClosed = true
	}
}

func (t *buildStateTracker) Close() {
	t.workMu.Lock()
	defer t.workMu.Unlock()
	t.closeWorkLocked()
}
