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

package authdbimpl

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/server/auth/service"
	"github.com/tetrafolium/luci-go/server/router"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPubSubHandlers(t *testing.T) {
	Convey("pubSubPush skips when not configured", t, func() {
		c := gaetesting.TestingContext()
		rec := httptest.NewRecorder()
		pubSubPush(&router.Context{
			Context: c,
			Writer:  rec,
			Request: makePostRequest(),
		})
		So(rec.Code, ShouldEqual, 200)
		So(rec.Body.String(), ShouldEqual, "Auth Service URL is not configured, skipping the message")
	})

	Convey("pubSubPush works when no notification", t, func() {
		c, _ := setupCtx()
		rec := httptest.NewRecorder()
		pubSubPush(&router.Context{
			Context: c,
			Writer:  rec,
			Request: makePostRequest(),
		})
		So(rec.Code, ShouldEqual, 200)
		So(rec.Body.String(), ShouldEqual, "No new valid AuthDB change notifications")
	})

	Convey("pubSubPush old notification", t, func() {
		c, srv := setupCtx()
		srv.Notification = &service.Notification{Revision: 122} // older than 123
		rec := httptest.NewRecorder()
		pubSubPush(&router.Context{
			Context: c,
			Writer:  rec,
			Request: makePostRequest(),
		})
		So(rec.Code, ShouldEqual, 200)
		So(rec.Body.String(), ShouldEqual, "Processed PubSub notification for rev 122: 123 -> 123")
	})

	Convey("pubSubPush fresh notification", t, func() {
		c, srv := setupCtx()
		srv.LatestRev = 130
		srv.Notification = &service.Notification{Revision: 124}
		rec := httptest.NewRecorder()
		pubSubPush(&router.Context{
			Context: c,
			Writer:  rec,
			Request: makePostRequest(),
		})
		So(rec.Code, ShouldEqual, 200)
		So(rec.Body.String(), ShouldEqual, "Processed PubSub notification for rev 124: 123 -> 130")
	})
}

func setupCtx() (context.Context, *fakeAuthService) {
	srv := &fakeAuthService{LatestRev: 123}
	c := setAuthService(gaetesting.TestingContext(), srv)
	So(ConfigureAuthService(c, "http://base_url", "http://auth-service"), ShouldBeNil)
	return c, srv
}

func makePostRequest() *http.Request {
	req, _ := http.NewRequest("POST", "/doesntmatter", bytes.NewReader(nil))
	return req
}
