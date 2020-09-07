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

package gaesecrets

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/server/caching"
	"github.com/tetrafolium/luci-go/server/secrets"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWorks(t *testing.T) {
	Convey("gaesecrets.Store works", t, func() {
		c := Use(memory.Use(context.Background()), nil)
		c = caching.WithEmptyProcessCache(c)

		// Autogenerates one.
		s1, err := secrets.GetSecret(c, "key1")
		So(err, ShouldBeNil)
		So(len(s1.Current), ShouldEqual, 32)

		// Returns same one.
		s2, err := secrets.GetSecret(c, "key1")
		So(err, ShouldBeNil)
		So(s2, ShouldResemble, s1)
	})
}
