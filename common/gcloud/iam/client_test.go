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

package iam

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock/testclock"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClient(t *testing.T) {
	Convey("SignBlob works", t, func(c C) {
		bodies := make(chan []byte, 1)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.RequestURI != "/v1/projects/-/serviceAccounts/abc@example.com:signBlob?alt=json" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			bodies <- body

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(
				fmt.Sprintf(`{"keyId":"key_id","signature":"%s"}`,
					base64.StdEncoding.EncodeToString([]byte("signature")))))
		}))
		defer ts.Close()

		cl := Client{
			Client:   http.DefaultClient,
			BasePath: ts.URL,
		}

		keyID, sig, err := cl.SignBlob(context.Background(), "abc@example.com", []byte("blob"))
		So(err, ShouldBeNil)
		So(keyID, ShouldEqual, "key_id")
		So(string(sig), ShouldEqual, "signature")

		// The request body looks sane too.
		body := <-bodies
		So(string(body), ShouldEqual, `{"bytesToSign":"YmxvYg=="}`)
	})

	Convey("SignJWT works", t, func(c C) {
		bodies := make(chan []byte, 1)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.RequestURI != "/v1/projects/-/serviceAccounts/abc@example.com:signJwt?alt=json" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			bodies <- body

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"keyId":"key_id","signedJwt":"signed_jwt"}`))
		}))
		defer ts.Close()

		cl := Client{
			Client:   http.DefaultClient,
			BasePath: ts.URL,
		}

		keyID, jwt, err := cl.SignJWT(context.Background(), "abc@example.com", &ClaimSet{Exp: 123})
		So(err, ShouldBeNil)
		So(keyID, ShouldEqual, "key_id")
		So(jwt, ShouldEqual, "signed_jwt")

		// The request body looks sane too.
		body := <-bodies
		So(string(body), ShouldEqual, `{"payload":"{\"iss\":\"\",\"aud\":\"\",\"exp\":123,\"iat\":0}"}`)
	})

	Convey("ModifyIAMPolicy works", t, func(c C) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}

			switch r.URL.Path {
			case "/v1/project/1/resource/2:getIamPolicy":
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				w.Write([]byte(`{"etag":"blah"}`))

			case "/v1/project/1/resource/2:setIamPolicy":
				c.So(string(body), ShouldEqual,
					`{"policy":{"bindings":[{"role":"role","members":["principal"]}],"etag":"blah"}}`)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				w.Write([]byte(`{"bindings":[{"role":"role","members":["principal"]}],"etag":"blah"}}`))

			default:
				c.Printf("Unknown URL: %q\n", r.URL.Path)
				w.WriteHeader(404)
			}
		}))
		defer ts.Close()

		cl := Client{
			Client:   http.DefaultClient,
			BasePath: ts.URL,
		}

		err := cl.ModifyIAMPolicy(context.Background(), "project/1/resource/2", func(p *Policy) error {
			p.GrantRole("role", "principal")
			return nil
		})
		So(err, ShouldBeNil)
	})

	Convey("GenerateAccessToken works", t, func(c C) {
		expireTime := testclock.TestRecentTimeUTC.Round(time.Second)

		var body map[string]interface{}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			switch r.URL.Path {
			case "/v1/projects/-/serviceAccounts/abc@example.com:generateAccessToken":
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					c.Printf("Bad body: %s\n", err)
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				resp := fmt.Sprintf(`{"accessToken":"%s","expireTime":"%s"}`, "token1", expireTime.Format(time.RFC3339))
				w.Write([]byte(resp))

			default:
				c.Printf("Unknown URL: %q\n", r.URL.Path)
				w.WriteHeader(404)
			}
		}))
		defer ts.Close()

		cl := Client{
			Client:   http.DefaultClient,
			BasePath: ts.URL,
		}

		token, err := cl.GenerateAccessToken(context.Background(),
			"abc@example.com", []string{"a", "b"}, []string{"deleg"}, 30*time.Minute)
		So(err, ShouldBeNil)
		So(token.AccessToken, ShouldEqual, "token1")
		So(token.Expiry, ShouldResemble, expireTime)

		So(body, ShouldResemble, map[string]interface{}{
			"delegates": []interface{}{"deleg"},
			"scope":     []interface{}{"a", "b"},
			"lifetime":  "30m0s",
		})
	})

	Convey("GenerateIDToken works", t, func(c C) {
		var body map[string]interface{}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			switch r.URL.Path {
			case "/v1/projects/-/serviceAccounts/abc@example.com:generateIdToken":
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					c.Printf("Bad body: %s\n", err)
					w.WriteHeader(500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				w.Write([]byte(`{"token":"fake_id_token"}`))

			default:
				c.Printf("Unknown URL: %q\n", r.URL.Path)
				w.WriteHeader(404)
			}
		}))
		defer ts.Close()

		cl := Client{
			Client:   http.DefaultClient,
			BasePath: ts.URL,
		}

		token, err := cl.GenerateIDToken(context.Background(),
			"abc@example.com", "aud", true, []string{"deleg"})
		So(err, ShouldBeNil)
		So(token, ShouldEqual, "fake_id_token")

		So(body, ShouldResemble, map[string]interface{}{
			"delegates":    []interface{}{"deleg"},
			"audience":     "aud",
			"includeEmail": true,
		})
	})
}
