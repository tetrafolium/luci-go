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

package server

import (
	"context"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/server/auth"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWorks(t *testing.T) {
	Convey("Works", t, func() {
		c := memory.Use(context.Background())
		c, _ = testclock.UseTime(c, time.Unix(1442540000, 0))
		s := SessionStore{Prefix: "ns"}
		u := auth.User{
			Identity: "user:abc@example.com",
			Email:    "abc@example.com",
			Name:     "Name",
			Picture:  "picture",
		}

		sid, err := s.OpenSession(c, "uid", &u, clock.Now(c).Add(time.Hour))
		So(err, ShouldBeNil)
		So(sid, ShouldEqual, "ns/uid/1")

		session, err := s.GetSession(c, sid)
		So(err, ShouldBeNil)
		So(session, ShouldResemble, &auth.Session{
			SessionID: "ns/uid/1",
			UserID:    "uid",
			User:      u,
			Exp:       clock.Now(c).Add(time.Hour).UTC(),
		})

		So(s.CloseSession(c, sid), ShouldBeNil)

		session, err = s.GetSession(c, sid)
		So(session, ShouldBeNil)
		So(err, ShouldBeNil)

		// Closed closed session is fine.
		So(s.CloseSession(c, sid), ShouldBeNil)
	})

	Convey("Test expiration", t, func() {
		c := memory.Use(context.Background())
		c, tc := testclock.UseTime(c, time.Unix(1442540000, 0))
		s := SessionStore{Prefix: "ns"}
		u := auth.User{Identity: "user:abc@example.com"}

		sid, err := s.OpenSession(c, "uid", &u, clock.Now(c).Add(time.Hour))
		So(err, ShouldBeNil)
		So(sid, ShouldEqual, "ns/uid/1")

		session, err := s.GetSession(c, sid)
		So(err, ShouldBeNil)
		So(session, ShouldNotBeNil)

		tc.Add(2 * time.Hour)

		session, err = s.GetSession(c, sid)
		So(err, ShouldBeNil)
		So(session, ShouldBeNil)
	})

	Convey("Test bad params in OpenSession", t, func() {
		c := memory.Use(context.Background())
		u := auth.User{Identity: "user:abc@example.com"}
		exp := time.Unix(1442540000, 0)

		s := SessionStore{Prefix: "/"}
		_, err := s.OpenSession(c, "uid", &u, exp)
		So(err, ShouldNotBeNil)

		s = SessionStore{Prefix: "ns"}
		_, err = s.OpenSession(c, "u/i/d", &u, exp)
		So(err, ShouldNotBeNil)

		_, err = s.OpenSession(c, "uid", &auth.User{Identity: "bad"}, exp)
		So(err, ShouldNotBeNil)
	})

	Convey("Test bad session ID", t, func() {
		c := memory.Use(context.Background())
		s := SessionStore{Prefix: "ns"}

		session, err := s.GetSession(c, "ns/uid")
		So(session, ShouldBeNil)
		So(err, ShouldBeNil)

		session, err = s.GetSession(c, "badns/uid/1")
		So(session, ShouldBeNil)
		So(err, ShouldBeNil)

		session, err = s.GetSession(c, "ns/uid/notint")
		So(session, ShouldBeNil)
		So(err, ShouldBeNil)

		session, err = s.GetSession(c, "ns/missing/1")
		So(session, ShouldBeNil)
		So(err, ShouldBeNil)
	})
}
