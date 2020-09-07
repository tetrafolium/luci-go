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
	"testing"

	"github.com/tetrafolium/luci-go/config/validation"

	"github.com/tetrafolium/luci-go/scheduler/appengine/messages"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidation(t *testing.T) {
	t.Parallel()

	run := func(p messages.TriggeringPolicy) error {
		ctx := validation.Context{}
		ValidateDefinition(&ctx, &p)
		return ctx.Finalize()
	}

	Convey("Works", t, func() {
		So(run(messages.TriggeringPolicy{}), ShouldBeNil)
		So(run(messages.TriggeringPolicy{Kind: 123}),
			ShouldErrLike, "unrecognized policy kind 123")
		So(run(messages.TriggeringPolicy{MaxConcurrentInvocations: -1}),
			ShouldErrLike, "max_concurrent_invocations should be positive, got -1")
		So(run(messages.TriggeringPolicy{MaxBatchSize: -1}),
			ShouldErrLike, "max_batch_size should be positive, got -1")
		So(run(messages.TriggeringPolicy{
			Kind: messages.TriggeringPolicy_LOGARITHMIC_BATCHING, LogBase: 0.5}),
			ShouldErrLike, "log_base should be larger or equal 1.0001, got 0.5")
	})
}
