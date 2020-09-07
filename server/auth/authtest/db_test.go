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

package authtest

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/tetrafolium/luci-go/server/auth/realms"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFakeDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testPerm1 := realms.RegisterPermission("testing.tests.perm1")
	testPerm2 := realms.RegisterPermission("testing.tests.perm2")
	testPerm3 := realms.RegisterPermission("testing.tests.perm3")

	dataRoot := &protocol.RealmData{EnforceInService: []string{"A"}}
	dataSome := &protocol.RealmData{EnforceInService: []string{"B"}}

	Convey("With FakeDB", t, func() {
		db := NewFakeDB(
			MockMembership("user:abc@def.com", "group-a"),
			MockMembership("user:abc@def.com", "group-b"),
			MockPermission("user:abc@def.com", "proj:realm", testPerm1),
			MockPermission("user:abc@def.com", "proj:realm", testPerm2),
			MockRealmData("proj:@root", dataRoot),
			MockRealmData("proj:some", dataSome),
			MockIPWhitelist("127.0.0.42", "wl"),
		)

		Convey("Membership checks work", func() {
			out, err := db.CheckMembership(ctx, "user:abc@def.com", []string{"group-a", "group-b", "group-c"})
			So(err, ShouldBeNil)
			So(out, ShouldResemble, []string{"group-a", "group-b"})

			resp, err := db.IsMember(ctx, "user:abc@def.com", nil)
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)

			resp, err = db.IsMember(ctx, "user:abc@def.com", []string{"group-b"})
			So(err, ShouldBeNil)
			So(resp, ShouldBeTrue)

			resp, err = db.IsMember(ctx, "user:abc@def.com", []string{"another", "group-b"})
			So(err, ShouldBeNil)
			So(resp, ShouldBeTrue)

			resp, err = db.IsMember(ctx, "user:another@def.com", []string{"group-b"})
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)

			resp, err = db.IsMember(ctx, "user:another@def.com", []string{"another", "group-b"})
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)

			resp, err = db.IsMember(ctx, "user:abc@def.com", []string{"another"})
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)
		})

		Convey("Permission checks work", func() {
			resp, err := db.HasPermission(ctx, "user:abc@def.com", testPerm1, "proj:realm")
			So(err, ShouldBeNil)
			So(resp, ShouldBeTrue)

			resp, err = db.HasPermission(ctx, "user:abc@def.com", testPerm2, "proj:realm")
			So(err, ShouldBeNil)
			So(resp, ShouldBeTrue)

			resp, err = db.HasPermission(ctx, "user:abc@def.com", testPerm3, "proj:realm")
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)

			resp, err = db.HasPermission(ctx, "user:abc@def.com", testPerm1, "proj:unknown")
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)
		})

		Convey("GetRealmData works", func() {
			data, err := db.GetRealmData(ctx, "proj:some")
			So(err, ShouldBeNil)
			So(data, ShouldEqual, dataSome)

			// No automatic fallback to root happens, mock it yourself.
			data, err = db.GetRealmData(ctx, "proj:zzz")
			So(err, ShouldBeNil)
			So(data, ShouldBeNil)
		})

		Convey("IP whitelist checks work", func() {
			resp, err := db.IsInWhitelist(ctx, net.ParseIP("127.0.0.42"), "wl")
			So(err, ShouldBeNil)
			So(resp, ShouldBeTrue)

			resp, err = db.IsInWhitelist(ctx, net.ParseIP("127.0.0.42"), "another")
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)

			resp, err = db.IsInWhitelist(ctx, net.ParseIP("192.0.0.1"), "wl")
			So(err, ShouldBeNil)
			So(resp, ShouldBeFalse)
		})

		Convey("Error works", func() {
			mockedErr := errors.New("boom")
			db.AddMocks(MockError(mockedErr))

			_, err := db.IsMember(ctx, "user:abc@def.com", []string{"group-a"})
			So(err, ShouldEqual, mockedErr)

			_, err = db.HasPermission(ctx, "user:abc@def.com", testPerm1, "proj:realm")
			So(err, ShouldEqual, mockedErr)

			_, err = db.IsInWhitelist(ctx, net.ParseIP("127.0.0.42"), "wl")
			So(err, ShouldEqual, mockedErr)
		})
	})
}
