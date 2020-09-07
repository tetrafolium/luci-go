// Copyright 2017 The LUCI Authors.
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

package access

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/tetrafolium/luci-go/common/errors"
)

// Action is a kind of RPC that can be ACLed.
type Action uint32

const (
	// AddBuild: Schedule a build.
	AddBuild Action = 1 << iota

	// ViewBuild: Get information about a build.
	ViewBuild

	// LeaseBuild: Lease a build for execution. Normally done by build systems.
	LeaseBuild

	// CancelBuild: Cancel an existing build. Does not require a lease key.
	CancelBuild

	// ResetBuild: Unlease and reset state of an existing build. Normally done by admins.
	ResetBuild

	// SearchBuilds: Search for builds or get a list of scheduled builds.
	SearchBuilds

	// ReadACL: View bucket ACLs.
	ReadACL

	// WriteACL: Change bucket ACLs.
	WriteACL

	// DeleteScheduledBuilds: Delete all scheduled builds from a bucket.
	DeleteScheduledBuilds

	// AccessBucket: Know about bucket existence and read its info.
	AccessBucket

	// PauseBucket: Pause builds for a given bucket.
	PauseBucket

	// SetNextNumber: Set the number for the next build in a builder.
	SetNextNumber
)

var nameToAction = map[string]Action{
	"ADD_BUILD":               AddBuild,
	"VIEW_BUILD":              ViewBuild,
	"LEASE_BUILD":             LeaseBuild,
	"CANCEL_BUILD":            CancelBuild,
	"RESET_BUILD":             ResetBuild,
	"SEARCH_BUILDS":           SearchBuilds,
	"READ_ACL":                ReadACL,
	"WRITE_ACL":               WriteACL,
	"DELETE_SCHEDULED_BUILDS": DeleteScheduledBuilds,
	"ACCESS_BUCKET":           AccessBucket,
	"PAUSE_BUCKET":            PauseBucket,
	"SET_NEXT_NUMBER":         SetNextNumber,
}

var actionToName = func() map[Action]string {
	result := make(map[Action]string, len(nameToAction))
	for k, v := range nameToAction {
		result[v] = k
	}
	return result
}()

// ParseAction parses the action name into an.
func ParseAction(action string) (Action, error) {
	if action, ok := nameToAction[action]; ok {
		return action, nil
	}
	return 0, fmt.Errorf("unexpected action %q", action)
}

// String returns the action name as a string.
func (a Action) String() string {
	// Fast path for only one action.
	if name, ok := actionToName[a]; ok {
		return name
	}
	// Slow path for many actions.
	var values []string
	for action, name := range actionToName {
		if action&a == action {
			values = append(values, name)
		}
	}
	return strings.Join(values, ", ")
}

// MarshalBinary encodes the Action as bytes.
func (a *Action) MarshalBinary() ([]byte, error) {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, uint32(*a))
	return bytes, nil
}

// UnmarshalBinary decodes an Action from bytes.
func (a *Action) UnmarshalBinary(blob []byte) error {
	if len(blob) != 4 {
		return errors.New("expected length of 4")
	}
	*a = Action(binary.LittleEndian.Uint32(blob))
	return nil
}
