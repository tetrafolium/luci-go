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
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/luci/client/isolated"
)

func TestGetRoot(t *testing.T) {
	t.Parallel()

	Convey(`Basic`, t, func() {
		dirs := make(isolated.ScatterGather)
		So(dirs.Add("wd1", "rel"), ShouldBeNil)
		So(dirs.Add("wd1", "rel2"), ShouldBeNil)

		wd, err := getRoot(dirs, nil)
		So(err, ShouldBeNil)
		So(wd, ShouldEqual, "wd1")

		So(dirs.Add("wd2", "rel3"), ShouldBeNil)

		_, err = getRoot(dirs, nil)
		So(err, ShouldNotBeNil)
	})
}