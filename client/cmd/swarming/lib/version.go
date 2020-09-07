// Copyright 2020 The LUCI Authors.
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

package lib

import (
	"fmt"

	"github.com/tetrafolium/luci-go/cipd/version"
)

// SwarmingVersion must be updated whenever functional change (behavior,
// arguments, supported commands) is done.
const SwarmingVersion = "0.3"

// SwarmingUserAgent stores the user agent name for this CLI.
var SwarmingUserAgent = "swarming-go/" + SwarmingVersion

func init() {
	ver, err := version.GetStartupVersion()
	if err != nil || ver.InstanceID == "" {
		return
	}
	SwarmingUserAgent += fmt.Sprintf(" (%s@%s)", ver.PackageName, ver.InstanceID)
}
