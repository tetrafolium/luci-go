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

package results

import (
	"testing"

	resultspb "go.chromium.org/luci/results/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestVariantUtils(t *testing.T) {
	Convey(`Map conversion works`, t, func() {
		def := VariantDefMap{
			"k2": "v2",
			"k3": "v3",
			"k1": "v1",
		}

		varpb := def.Proto()
		So(varpb, ShouldResembleProto, &resultspb.VariantDef{
			Def: map[string]string{
				"k1": "v1",
				"k2": "v2",
				"k3": "v3",
			},
			Digest: "6b12ab2958d568f28ca588b136468adbc0ed3a9978a4192c961f754061dbbae0",
		})
	})
}
