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

package xsrf

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/secrets/testsecrets"

	. "github.com/smartystreets/goconvey/convey"
)

func TestXsrf(t *testing.T) {
	Convey("Token + Check", t, func() {
		c := makeContext()
		tok, err := Token(c)
		So(err, ShouldBeNil)
		So(Check(c, tok), ShouldBeNil)
		So(Check(c, tok+"abc"), ShouldNotBeNil)
	})

	Convey("TokenField works", t, func() {
		c := makeContext()
		So(TokenField(c), ShouldResemble,
			template.HTML("<input type=\"hidden\" name=\"xsrf_token\" "+
				"value=\"AXsiX2kiOiIxNDQyMjcwNTIwMDAwIn1ceiDv1yfNK9OHcdb209l3fM4p_gn-Uaembaa8gr3WXg\">"))
	})

	Convey("Middleware works", t, func() {
		c := makeContext()
		tok, _ := Token(c)

		h := func(c *router.Context) {
			c.Writer.Write([]byte("hi"))
		}
		mc := router.NewMiddlewareChain(WithTokenCheck)

		// Has token -> works.
		rec := httptest.NewRecorder()
		req := makeRequest(tok)
		router.RunMiddleware(&router.Context{
			Context: c,
			Writer:  rec,
			Request: req,
		}, mc, h)
		So(rec.Code, ShouldEqual, 200)

		// No token.
		rec = httptest.NewRecorder()
		req = makeRequest("")
		router.RunMiddleware(&router.Context{
			Context: c,
			Writer:  rec,
			Request: req,
		}, mc, h)
		So(rec.Code, ShouldEqual, 403)

		// Bad token.
		rec = httptest.NewRecorder()
		req = makeRequest("blah")
		router.RunMiddleware(&router.Context{
			Context: c,
			Writer:  rec,
			Request: req,
		}, mc, h)
		So(rec.Code, ShouldEqual, 403)
	})
}

func makeContext() context.Context {
	c := testsecrets.Use(context.Background())
	c, _ = testclock.UseTime(c, time.Unix(1442270520, 0))
	return c
}

func makeRequest(tok string) *http.Request {
	body := url.Values{}
	if tok != "" {
		body.Add("xsrf_token", tok)
	}
	req, _ := http.NewRequest("POST", "https://example.com", strings.NewReader(body.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return req
}
