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

package authdbimpl

import (
	"context"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
)

var (
	latestSnapshotInfoDuration = metric.NewCumulativeDistribution(
		"luci/authdb/methods/latest_snapshot_info",
		"Distribution of 'GetLatestSnapshotInfo' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	getSnapshotDuration = metric.NewCumulativeDistribution(
		"luci/authdb/methods/get_snapshot",
		"Distribution of 'GetAuthDBSnapshot' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	syncAuthDBDuration = metric.NewCumulativeDistribution(
		"luci/authdb/methods/sync_auth_db",
		"Distribution of 'syncAuthDB' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))
)

func durationReporter(ctx context.Context, m metric.CumulativeDistribution) func(...interface{}) {
	startTs := clock.Now(ctx)
	return func(fields ...interface{}) {
		m.Add(ctx, float64(clock.Since(ctx, startTs).Nanoseconds()/1000), fields...)
	}
}
