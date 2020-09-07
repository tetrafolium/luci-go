// Copyright 2016 The LUCI Authors.
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

package frontend

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/auth/identity"
	buildbucketpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/config/impl/memory"
	"github.com/tetrafolium/luci-go/milo/common"
	"github.com/tetrafolium/luci-go/milo/git"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/router"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFuncs(t *testing.T) {
	t.Parallel()

	Convey("Middleware Tests", t, func() {
		Convey("Format Commit Description", func() {
			Convey("linkify https://", func() {
				So(formatCommitDesc("https://foo.com"),
					ShouldEqual,
					"<a href=\"https://foo.com\">https://foo.com</a>")
				Convey("but not http://", func() {
					So(formatCommitDesc("http://foo.com"), ShouldEqual, "http://foo.com")
				})
			})
			Convey("linkify b/ and crbug/", func() {
				So(formatCommitDesc("blah blah b/123456 blah"), ShouldEqual, "blah blah <a href=\"http://b/123456\">b/123456</a> blah")
				So(formatCommitDesc("crbug:foo/123456"), ShouldEqual, "<a href=\"https://crbug.com/foo/123456\">crbug:foo/123456</a>")
			})
			Convey("linkify Bug: lines", func() {
				So(formatCommitDesc("\nBug: 12345\n"), ShouldEqual, "\nBug: <a href=\"https://crbug.com/12345\">12345</a>\n")
				So(
					formatCommitDesc(" > > BugS=  12345, butter:12345"),
					ShouldEqual,
					" &gt; &gt; BugS=  <a href=\"https://crbug.com/12345\">12345</a>, "+
						"<a href=\"https://crbug.com/butter/12345\">butter:12345</a>")
			})
			Convey("linkify rules should not collide", func() {
				So(
					formatCommitDesc("I \"fixed\" https://crbug.com/123456 <today>"),
					ShouldEqual,
					"I &#34;fixed&#34; <a href=\"https://crbug.com/123456\">https://crbug.com/123456</a> &lt;today&gt;")
				So(
					formatCommitDesc("Bug: 12, crbug/34, https://crbug.com/56, 78"),
					ShouldEqual,
					"Bug: <a href=\"https://crbug.com/12\">12</a>, <a href=\"https://crbug.com/34\">crbug/34</a>, <a href=\"https://crbug.com/56\">https://crbug.com/56</a>, <a href=\"https://crbug.com/78\">78</a>")
			})
			Convey("linkify rules interact correctly with escaping", func() {
				So(
					formatCommitDesc("\"https://example.com\""),
					ShouldEqual,
					"&#34;<a href=\"https://example.com\">https://example.com</a>&#34;")
				So(
					formatCommitDesc("Bug: <not a bug number, sorry>"),
					ShouldEqual,
					"Bug: &lt;not a bug number, sorry&gt;")
				// This is not remotely valid of a URL, but exists to test that
				// the linking template correctly escapes the URL, both as an
				// attribute and as a value.
				So(
					formatCommitDesc("https://foo&bar<baz\"aaa>bbb"),
					ShouldEqual,
					"<a href=\"https://foo&amp;bar%3cbaz%22aaa%3ebbb\">https://foo&amp;bar&lt;baz&#34;aaa&gt;bbb</a>")
			})

			Convey("trimLongString", func() {
				Convey("short", func() {
					So(trimLongString(4, "😀😀😀😀"), ShouldEqual, "😀😀😀😀")
				})
				Convey("long", func() {
					So(trimLongString(4, "😀😀😀😀😀"), ShouldEqual, "😀😀😀…")
				})
			})
		})

		Convey("Redirect unauthorized users to login page for projects with access restrictions", func() {
			projectACLMiddleware := buildProjectACLMiddleware(false)
			r := httptest.NewRecorder()
			c := gaetesting.TestingContextWithAppID("luci-milo-dev")

			// Fake user to be anonymous.
			c = auth.WithState(c, &authtest.FakeState{Identity: identity.AnonymousIdentity})

			// Create fake internal project named "secret".
			c = cfgclient.Use(c, memory.New(map[config.Set]memory.Files{
				"projects/secret": {
					"project.cfg": "name: \"secret\"\naccess: \"group:googlers\"",
				},
			}))
			So(common.UpdateProjects(c), ShouldBeNil)

			ctx := &router.Context{
				Context: c,
				Writer:  r,
				Request: httptest.NewRequest("GET", "/p/secret", bytes.NewReader(nil)),
				Params:  httprouter.Params{{Key: "project", Value: "secret"}},
			}
			projectACLMiddleware(ctx, nil)
			project, ok := git.ProjectFromContext(ctx.Context)
			So(ok, ShouldBeFalse)
			So(project, ShouldEqual, "")
			So(r.Code, ShouldEqual, 302)
			So(r.Result().Header["Location"], ShouldResemble, []string{"http://fake.example.com/login?dest=%2Fp%2Fsecret"})
		})

		Convey("Install git project to context when the user has access to the project", func() {
			optionalProjectACLMiddleware := buildProjectACLMiddleware(true)
			r := httptest.NewRecorder()
			c := gaetesting.TestingContextWithAppID("luci-milo-dev")

			// Fake user to be anonymous.
			c = auth.WithState(c, &authtest.FakeState{Identity: identity.AnonymousIdentity, IdentityGroups: []string{"all"}})

			// Create fake public project named "public".
			c = cfgclient.Use(c, memory.New(map[config.Set]memory.Files{
				"projects/public": {
					"project.cfg": "name: \"public\"\naccess: \"group:all\"",
				},
			}))
			So(common.UpdateProjects(c), ShouldBeNil)

			ctx := &router.Context{
				Context: c,
				Writer:  r,
				Request: httptest.NewRequest("GET", "/p/public", bytes.NewReader(nil)),
				Params:  httprouter.Params{{Key: "project", Value: "public"}},
			}
			nextCalled := false
			next := func(*router.Context) {
				nextCalled = true
			}
			optionalProjectACLMiddleware(ctx, next)
			project, ok := git.ProjectFromContext(ctx.Context)
			So(project, ShouldEqual, "public")
			So(ok, ShouldBeTrue)
			So(nextCalled, ShouldBeTrue)
			So(r.Code, ShouldEqual, 200)
		})

		Convey("Don't install git project to context when the user doesn't have access to the project", func() {
			optionalProjectACLMiddleware := buildProjectACLMiddleware(true)
			r := httptest.NewRecorder()
			c := gaetesting.TestingContextWithAppID("luci-milo-dev")

			// Fake user to be anonymous.
			c = auth.WithState(c, &authtest.FakeState{Identity: identity.AnonymousIdentity})

			// Create fake internal project named "secret".
			c = cfgclient.Use(c, memory.New(map[config.Set]memory.Files{
				"projects/secret": {
					"project.cfg": "name: \"secret\"\naccess: \"group:googlers\"",
				},
			}))
			So(common.UpdateProjects(c), ShouldBeNil)

			ctx := &router.Context{
				Context: c,
				Writer:  r,
				Request: httptest.NewRequest("GET", "/p/secret", bytes.NewReader(nil)),
				Params:  httprouter.Params{{Key: "project", Value: "secret"}},
			}
			nextCalled := false
			next := func(*router.Context) {
				nextCalled = true
			}
			optionalProjectACLMiddleware(ctx, next)
			project, ok := git.ProjectFromContext(ctx.Context)
			So(ok, ShouldBeFalse)
			So(project, ShouldEqual, "")
			So(nextCalled, ShouldBeTrue)
			So(r.Code, ShouldEqual, 200)
		})

		Convey("Convert LogDog URLs", func() {
			So(
				logdogLink(buildbucketpb.Log{Name: "foo", Url: "logdog://www.example.com:1234/foo/bar/+/baz"}, true),
				ShouldEqual,
				`<a href="https://www.example.com:1234/logs/foo/bar/&#43;/baz?format=raw" aria-label="raw log foo">raw</a>`)
			So(
				logdogLink(buildbucketpb.Log{Name: "foo", Url: "%zzzzz"}, true),
				ShouldEqual,
				`<a href="#invalid-logdog-link" aria-label="raw log foo">raw</a>`)
			So(
				logdogLink(buildbucketpb.Log{Name: "foo", Url: "logdog://logs.chromium.org/foo/bar/+/baz"}, false),
				ShouldEqual,
				`<a href="https://logs.chromium.org/logs/foo/bar/&#43;/baz" aria-label="raw log foo">foo</a>`)

		})
	})
}
