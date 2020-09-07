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

package auth

import (
	"context"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
)

var (
	authenticateDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/authenticate",
		"Distribution of 'Authenticate' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	mintDelegationTokenDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/mint_delegation_token",
		"Distribution of 'MintDelegationToken' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	mintAccessTokenDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/mint_access_token",
		"Distribution of 'MintAccessTokenForServiceAccount' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	mintIDTokenDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/mint_id_token",
		"Distribution of 'MintIDTokenForServiceAccount' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	mintProjectTokenDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/mint_project_token",
		"Distribution of 'MintProjectToken' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	tokenInfoCallDuration = metric.NewCumulativeDistribution(
		"luci/auth/rpcs/oauth_token_info",
		"Distribution of duration of Google OAuth2 /tokeninfo RPCs.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("outcome")) // OK, BAD_TOKEN, DEADLINE, ERROR
)

func durationReporter(c context.Context, m metric.CumulativeDistribution) func(error, string) {
	startTs := clock.Now(c)
	return func(err error, result string) {
		// We should report context errors as such. It doesn't matter at which stage
		// the deadline happens, thus we ignore 'result' if seeing a context error.
		switch errors.Unwrap(err) {
		case context.DeadlineExceeded:
			result = "CONTEXT_DEADLINE_EXCEEDED"
		case context.Canceled:
			result = "CONTEXT_CANCELED"
		}
		m.Add(c, float64(clock.Since(c, startTs).Nanoseconds()/1000), result)
	}
}
