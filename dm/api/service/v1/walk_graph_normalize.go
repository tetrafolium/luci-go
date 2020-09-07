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

package dm

import (
	"math"
	"sort"

	"github.com/xtgo/set"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/proto/google"
)

const (
	// DefaultLimitMaxDataSize is the default MaxDataSize value (16MB).
	DefaultLimitMaxDataSize = 16 * 1024 * 1024

	// MaxLimitMaxDataSize is the maximum MaxDataSize value (30MB).
	MaxLimitMaxDataSize = 30 * 1024 * 1024
)

// MakeWalkGraphIncludeAll makes a new WalkGraphReq_Include which has all the
// boxes ticked. This should only be used when your application plans to dump
// the resulting graph query data to some logging/debugging trace for humans.
//
// If you don't plan on dumping it for humans, please set the Include options
// appropriately in order to avoid wasting bandwidth/cpu/datastore query time on
// the server (and draining your DM quotas unnecessarially).
func MakeWalkGraphIncludeAll() *WalkGraphReq_Include {
	return &WalkGraphReq_Include{
		Quest: &WalkGraphReq_Include_Options{
			Ids:  true,
			Data: true,
		},
		Attempt: &WalkGraphReq_Include_Options{
			Ids:      true,
			Data:     true,
			Result:   true,
			Abnormal: true,
			Expired:  true,
		},
		Execution: &WalkGraphReq_Include_Options{
			Ids:      true,
			Data:     true,
			Result:   true,
			Abnormal: true,
			Expired:  true,
		},
		NumExecutions: math.MaxUint32,
		FwdDeps:       true,
		BackDeps:      true,
	}
}

// Normalize returns an error iff the WalkGraphReq_Exclude is invalid.
func (e *WalkGraphReq_Exclude) Normalize() error {
	if len(e.Quests) > 0 {
		e.Quests = e.Quests[:set.Uniq(sort.StringSlice(e.Quests))]
	}
	return e.Attempts.Normalize()
}

// Normalize returns an error iff the WalkGraphReq is invalid.
func (w *WalkGraphReq) Normalize() error {
	if w.Auth != nil {
		if err := w.Auth.Normalize(); err != nil {
			return err
		}
	}

	if w.Query == nil {
		return errors.New("must specify a Query")
	}
	if err := w.Query.Normalize(); err != nil {
		return err
	}

	if w.Mode == nil {
		w.Mode = &WalkGraphReq_Mode{}
	}

	if w.Limit != nil {
		if w.Limit.MaxDepth < -1 {
			return errors.New("limit.max_depth must be >= -1")
		}
		if google.DurationFromProto(w.Limit.GetMaxTime()) < 0 {
			return errors.New("limit.max_time must be positive")
		}
	} else {
		w.Limit = &WalkGraphReq_Limit{}
	}
	if w.Limit.MaxDataSize == 0 {
		w.Limit.MaxDataSize = DefaultLimitMaxDataSize
	}
	if w.Limit.MaxDataSize > MaxLimitMaxDataSize {
		w.Limit.MaxDataSize = MaxLimitMaxDataSize
	}

	if w.Include == nil {
		w.Include = &WalkGraphReq_Include{
			Quest:     &WalkGraphReq_Include_Options{},
			Attempt:   &WalkGraphReq_Include_Options{},
			Execution: &WalkGraphReq_Include_Options{},
		}
	} else {
		if w.Include.Quest == nil {
			w.Include.Quest = &WalkGraphReq_Include_Options{}
		} else if w.Include.Quest.Result || w.Include.Quest.Abnormal || w.Include.Quest.Expired {
			return errors.New("include.quest does not support result, abnormal or expired")
		}

		if w.Include.Attempt == nil {
			w.Include.Attempt = &WalkGraphReq_Include_Options{}
		} else {
			if w.Include.Attempt.Result {
				w.Include.Attempt.Data = true
			}
		}

		if w.Include.Execution == nil {
			w.Include.Execution = &WalkGraphReq_Include_Options{}
		}
	}

	if w.Exclude == nil {
		w.Exclude = &WalkGraphReq_Exclude{}
	}
	return w.Exclude.Normalize()
}
