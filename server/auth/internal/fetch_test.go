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

package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func init() {
	RegisterClientFactory(func(c context.Context, scopes []string) (*http.Client, error) {
		return http.DefaultClient, nil
	})
}

func TestFetch(t *testing.T) {
	Convey("with test context", t, func(c C) {
		body := ""
		status := http.StatusOK
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(body))
		}))
		defer ts.Close()

		ctx := context.Background()

		Convey("fetch works", func() {
			var val struct {
				A string `json:"a"`
			}
			body = `{"a": "hello"}`
			req := Request{
				Method: "GET",
				URL:    ts.URL,
				Out:    &val,
			}
			So(req.Do(ctx), ShouldBeNil)
			So(val.A, ShouldEqual, "hello")
		})

		Convey("handles bad status code", func() {
			var val struct{}
			status = http.StatusNotFound
			req := Request{
				Method: "GET",
				URL:    ts.URL,
				Out:    &val,
			}
			So(req.Do(ctx), ShouldErrLike, "HTTP code (404)")
		})

		Convey("handles bad body", func() {
			var val struct{}
			body = "not json"
			req := Request{
				Method: "GET",
				URL:    ts.URL,
				Out:    &val,
			}
			So(req.Do(ctx), ShouldErrLike, "can't deserialize JSON")
		})

		Convey("handles connection error", func() {
			var val struct{}
			req := Request{
				Method: "GET",
				URL:    "http://localhost:12345678",
				Out:    &val,
			}
			So(req.Do(ctx), ShouldErrLike, "dial tcp")
		})
	})
}
