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

package openid

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/server/settings"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSettings(t *testing.T) {
	Convey("Works", t, func() {
		c := context.Background()
		c = settings.Use(c, settings.New(&settings.MemoryStorage{}))

		cfg, err := fetchCachedSettings(c)
		So(err, ShouldBeNil)
		So(cfg, ShouldResemble, &Settings{})
	})
}
