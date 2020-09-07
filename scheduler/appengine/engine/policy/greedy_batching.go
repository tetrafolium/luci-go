// Copyright 2018 The LUCI Authors.
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

package policy

import (
	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
)

// GreedyBatchingPolicy instantiates new GREEDY_BATCHING policy function.
//
// It takes all pending triggers and collapses them into one new invocation,
// deriving its properties from the most recent trigger alone.
func GreedyBatchingPolicy(maxConcurrentInvs, maxBatchSize int) (Func, error) {
	return basePolicy(maxConcurrentInvs, maxBatchSize, func(triggers []*internal.Trigger) int {
		return len(triggers)
	})
}
