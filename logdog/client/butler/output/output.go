// Copyright 2015 The LUCI Authors.
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

package output

import (
	"fmt"

	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/client/butler/bootstrap"
)

// Output is a sink endpoint for groups of messages.
//
// An Output's methods must be goroutine-safe.
//
// Note that there is no guarantee that any of the bundles passed through an
// Output are ordered.
type Output interface {
	// SendBundle sends a constructed ButlerLogBundle through the Output.
	//
	// If an error is returned, it indicates a failure to send the bundle.
	// If there is a data error or a message type is not supported by the
	// Output, it should log the error and return nil.
	SendBundle(*logpb.ButlerLogBundle) error

	// MaxSendBundles is the number of concurrent calls to SendBundle allowed.
	//
	// If <= 0, only one SendBundle will be called at a time.
	MaxSendBundles() int

	// URLConstructionEnv should return a bootstrap.Environment containing
	// any fields necessary for clients to construct a URL pointing to where this
	// Output is sending its data.
	//
	// Returning an empty Environment means that clients will not be able to
	// construct URLs to this data (which may be accurate, depending on the
	// Output implementation).
	//
	// StreamServerURI is ignored and should not be specified.
	//
	// NOTE: This is an awful encapsulation violation. We should change the butler
	// protocol so that opening a new stream has the butler immediately reply with
	// the externally-visible URL to the stream and stop exporting these envvars
	// entirely.
	URLConstructionEnv() bootstrap.Environment

	// MaxSize returns the maximum number of bytes that this Output can process
	// with a single send. A return value <=0 indicates that there is no fixed
	// maximum size for this Output.
	//
	// Since it is impossible for callers to know the actual size of the message
	// that is being submitted, and since message batching may cluster across
	// size boundaries, this should be a conservative estimate.
	MaxSize() int

	// Collect current Output stats.
	Stats() Stats

	// Close closes the Output, blocking until any buffered actions are flushed.
	Close()
}

// Stats is an interface to query Output statistics.
//
// An Output's ability to keep statistics varies with its implementation
// details. Currently, Stats are for debugging/information purposes only.
type Stats interface {
	fmt.Stringer

	// SentBytes returns the number of bytes
	SentBytes() int64
	// SentMessages returns the number of successfully transmitted messages.
	SentMessages() int64
	// DiscardedMessages returns the number of discarded messages.
	DiscardedMessages() int64
	// Errors returns the number of errors encountered during operation.
	Errors() int64
}

// StatsBase is a simple implementation of the Stats interface.
type StatsBase struct {
	F struct {
		SentBytes         int64 // The number of bytes sent.
		SentMessages      int64 // The number of messages sent.
		DiscardedMessages int64 // The number of messages that have been discarded.
		Errors            int64 // The number of errors encountered.
	}
}

var _ Stats = (*StatsBase)(nil)

func (s *StatsBase) String() string {
	return fmt.Sprintf("%+v", s.F)
}

// SentBytes implements Stats.
func (s *StatsBase) SentBytes() int64 {
	return s.F.SentBytes
}

// SentMessages implements Stats.
func (s *StatsBase) SentMessages() int64 {
	return s.F.SentMessages
}

// DiscardedMessages implements Stats.
func (s *StatsBase) DiscardedMessages() int64 {
	return s.F.DiscardedMessages
}

// Errors implements Stats.
func (s *StatsBase) Errors() int64 {
	return s.F.Errors
}

// Merge merges the values from one Stats block into another.
func (s *StatsBase) Merge(o Stats) {
	s.F.SentBytes += o.SentBytes()
	s.F.SentMessages += o.SentMessages()
	s.F.DiscardedMessages += o.DiscardedMessages()
	s.F.Errors += o.Errors()
}
