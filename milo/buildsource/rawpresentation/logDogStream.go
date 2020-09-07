// Copyright 2016 The LUCI Authors.
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

package rawpresentation

import (
	annopb "github.com/tetrafolium/luci-go/luciexe/legacy/annotee/proto"
)

// Streams represents a group of LogDog Streams with a single entry point.
// Generally all of the streams are referenced by the entry point.
type Streams struct {
	// MainStream is a pointer to the primary stream for this group of streams.
	MainStream *Stream
	// Streams is the full map streamName->stream referenced by MainStream.
	// It includes MainStream.
	Streams map[string]*Stream
}

// Stream represents a single LogDog style stream, which can contain either
// annotations (assumed to be annopb.Step) or text.  Other types of annotations
// are not supported.
type Stream struct {
	// Server is the LogDog server this stream originated from.
	Server string
	// Prefix is the LogDog prefix for the Stream.
	Prefix string
	// Path is the final part of the LogDog path of the Stream.
	Path string
	// IsDatagram is true if this is an annopb.Step. False implies that this is a
	// text log.
	IsDatagram bool
	// Data is the annopb.Step of the Stream, if IsDatagram is true.  Otherwise
	// this is nil.
	Data *annopb.Step
	// Text is the text of the Stream, if IsDatagram is false.  Otherwise
	// this is an empty string.
	Text string

	// Closed specifies whether Text or Data may change in the future.
	// If Closed, they may not.
	Closed bool
}
