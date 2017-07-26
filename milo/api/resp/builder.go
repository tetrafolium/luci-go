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

//go:generate stringer -type=BotStatus

package resp

import (
	"time"

	"github.com/luci/luci-go/milo/common/model"
)

// Interval is a time interval which has a start, an end and a duration.
type Interval struct {
	Started  time.Time     // when did this interval start
	Finished time.Time     // when did this interval finish
	Duration time.Duration // length of the interval; may be non-zero if Finished is zero
}

// BuildSummary is a summary of a build, with just enough information for display
// on a builders page, with an optional field to return the whole build
// information if available.
type BuildSummary struct {
	// Link to the build.
	Link *Link

	// Status of the build.
	Status model.Status

	// Pending is time interval that this build was pending.
	PendingTime Interval

	// Execution is time interval that this build was executing.
	ExecutionTime Interval

	// Revision is the main revision of the build.
	// TODO(hinoka): Maybe use a commit object instead?
	Revision string

	// Arbitrary text to display below links.  One line per entry,
	// newlines are stripped.
	Text []string

	// Blame is for tracking whose change the build belongs to, if any.
	Blame []*Commit

	// Build is a reference to the full underlying MiloBuild, if it's available.
	// The only reason this would be calculated is if populating the BuildSummary
	// requires fetching the entire build anyways.  This is assumed to not
	// be available.
	Build *MiloBuild
}

// Builder denotes an ordered list of MiloBuilds
type Builder struct {
	// Name of the builder
	Name string

	// Warning text, if any.
	Warning string

	CurrentBuilds []*BuildSummary
	PendingBuilds []*BuildSummary
	// PendingBuildNum is the number of pending builds, since the slice above
	// may be a snapshot instead of the full set.
	PendingBuildNum int
	FinishedBuilds  []*BuildSummary

	// MachinePool is primarily used by buildbot builders to list the set of
	// machines that can run in a builder.  It has no meaning in buildbucket or dm
	// and is expected to be nil.
	MachinePool *MachinePool

	// PrevCursor is a cursor to the previous page.
	PrevCursor string `json:",omitempty"`
	// NextCursor is a cursor to the next page.
	NextCursor string `json:",omitempty"`
}

type BotStatus int

const (
	UnknownStatus BotStatus = iota
	Idle
	Busy
	Disconnected
)

// Bot represents a single bot.
type Bot struct {
	Name   Link
	Status BotStatus
}

// MachinePool represents the capacity and availability of a builder.
type MachinePool struct {
	Total        int
	Disconnected int
	Idle         int
	Busy         int
	Bots         []Bot
}
