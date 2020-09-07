// Copyright 2017 The LUCI Authors.
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

package revocation

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/gae/impl/memory"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGenerateTokenID(t *testing.T) {
	Convey("Works", t, func() {
		ctx := memory.Use(context.Background())

		id, err := GenerateTokenID(ctx, "zzz")
		So(err, ShouldBeNil)
		So(id, ShouldEqual, 1)

		id, err = GenerateTokenID(ctx, "zzz")
		So(err, ShouldBeNil)
		So(id, ShouldEqual, 2)
	})
}
