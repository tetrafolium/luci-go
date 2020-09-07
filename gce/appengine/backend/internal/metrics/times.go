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

package metrics

import (
	"context"

	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
)

var (
	creationTime = metric.NewCumulativeDistribution(
		"gce/instances/times/creation",
		"Time between GCE Provider configuring an instance and GCE reporting it as created.",
		&types.MetricMetadata{Units: types.Seconds},
		distribution.GeometricBucketer(1.1, 100),
		field.String("prefix"),
		field.String("project"),
		field.String("zone"),
	)

	connectionTime = metric.NewCumulativeDistribution(
		"gce/instances/times/connection",
		"Time between GCE reporting an instance as created and Swarming reporting it as connected.",
		&types.MetricMetadata{Units: types.Seconds},
		distribution.GeometricBucketer(1.1, 100),
		field.String("prefix"),
		field.String("project"),
		field.String("server"),
		field.String("zone"),
	)
)

// ReportCreationTime sets GCE instance creation time metric.
func ReportCreationTime(c context.Context, secs float64, prefix, project, zone string) {
	creationTime.Add(c, secs, prefix, project, zone)
}

// ReportConnectionTime sets GCE instance creation time metric.
func ReportConnectionTime(c context.Context, secs float64, prefix, project, server, zone string) {
	connectionTime.Add(c, secs, prefix, project, server, zone)
}
