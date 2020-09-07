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

package tsmon

import (
	"context"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/gae/service/module"
)

var (
	defaultVersion = metric.NewString(
		"appengine/default_version",
		"Name of the version currently marked as default.",
		nil)
)

// collectGlobalMetrics populates service-global metrics.
//
// Called by tsmon from inside /housekeeping cron handler. Metrics reported must
// not depend on the state of the particular process that happens to report
// them.
func collectGlobalMetrics(ctx context.Context) {
	version, err := module.DefaultVersion(ctx, "")
	if err != nil {
		logging.Errorf(ctx, "Error getting default appengine version: %s", err)
		defaultVersion.Set(ctx, "(unknown)")
	} else {
		defaultVersion.Set(ctx, version)
	}
}

func init() {
	tsmon.RegisterGlobalCallback(collectGlobalMetrics, defaultVersion)
}
