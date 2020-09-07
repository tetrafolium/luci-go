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
	"net"
	"testing"

	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/realms"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFakeState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	testPerm := realms.RegisterPermission("testing.tests.perm")

	Convey("Default FakeState works", t, func() {
		state := FakeState{}
		So(state.DB(), ShouldResemble, &FakeDB{})
		So(state.Method(), ShouldNotBeNil)
		So(state.User(), ShouldResemble, &auth.User{Identity: "anonymous:anonymous"})
		So(state.PeerIdentity(), ShouldEqual, "anonymous:anonymous")
		So(state.PeerIP().String(), ShouldEqual, "127.0.0.1")
	})

	Convey("Non-default FakeState works", t, func() {
		state := FakeState{
			Identity:       "user:abc@def.com",
			IdentityGroups: []string{"abc"},
			IdentityPermissions: []RealmPermission{
				{"proj:realm1", testPerm},
			},
			PeerIPWhitelists:     []string{"wl"},
			PeerIdentityOverride: "bot:blah",
			PeerIPOverride:       net.ParseIP("192.192.192.192"),
		}

		So(state.Method(), ShouldNotBeNil)
		So(state.User(), ShouldResemble, &auth.User{
			Identity: "user:abc@def.com",
			Email:    "abc@def.com",
		})
		So(state.PeerIdentity(), ShouldEqual, "bot:blah")
		So(state.PeerIP().String(), ShouldEqual, "192.192.192.192")

		db := state.DB()

		yes, err := db.IsMember(ctx, "user:abc@def.com", []string{"abc"})
		So(err, ShouldBeNil)
		So(yes, ShouldBeTrue)

		yes, err = db.HasPermission(ctx, "user:abc@def.com", testPerm, "proj:realm1")
		So(err, ShouldBeNil)
		So(yes, ShouldBeTrue)

		yes, err = db.IsInWhitelist(ctx, net.ParseIP("192.192.192.192"), "wl")
		So(err, ShouldBeNil)
		So(yes, ShouldBeTrue)
	})
}
