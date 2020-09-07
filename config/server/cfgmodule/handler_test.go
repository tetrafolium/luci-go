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

package cfgmodule

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tetrafolium/luci-go/common/proto/config"
	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/server/router"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInstallHandlers(t *testing.T) {
	t.Parallel()

	Convey("Initialization of validator, validation routes and handlers", t, func() {
		rules := validation.NewRuleSet()

		r := router.New()
		rr := httptest.NewRecorder()
		host := "example.com"

		metaCall := func() *config.ServiceDynamicMetadata {
			req, err := http.NewRequest("GET", "https://"+host+metadataPath, nil)
			So(err, ShouldBeNil)
			r.ServeHTTP(rr, req)

			var resp config.ServiceDynamicMetadata
			err = json.NewDecoder(rr.Body).Decode(&resp)
			So(err, ShouldBeNil)
			return &resp
		}
		valCall := func(configSet, path, content string) *config.ValidationResponseMessage {
			respBodyJSON, err := json.Marshal(config.ValidationRequestMessage{
				ConfigSet: configSet,
				Path:      path,
				Content:   []byte(content),
			})
			So(err, ShouldBeNil)
			req, err := http.NewRequest("POST", validationPath, bytes.NewReader(respBodyJSON))
			So(err, ShouldBeNil)
			r.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				return nil
			}
			var resp config.ValidationResponseMessage
			err = json.NewDecoder(rr.Body).Decode(&resp)
			So(err, ShouldBeNil)
			return &resp
		}

		InstallHandlers(r, router.NewMiddlewareChain(), rules)

		Convey("Basic metadataHandler call", func() {
			So(rr.Code, ShouldEqual, http.StatusOK)
			So(metaCall(), ShouldResemble, &config.ServiceDynamicMetadata{
				Version: metaDataFormatVersion,
				Validation: &config.Validator{
					Url: fmt.Sprintf("https://%s%s", host, validationPath),
				},
			})
		})

		Convey("metadataHandler call with patterns", func() {
			rules.Add("configSet", "path", nil)
			meta := metaCall()
			So(rr.Code, ShouldEqual, http.StatusOK)
			So(meta, ShouldResemble, &config.ServiceDynamicMetadata{
				Version: metaDataFormatVersion,
				Validation: &config.Validator{
					Url: fmt.Sprintf("https://%s%s", host, validationPath),
					Patterns: []*config.ConfigPattern{
						{
							ConfigSet: "exact:configSet",
							Path:      "exact:path",
						},
					},
				},
			})
		})

		Convey("Basic validationHandler call", func() {
			rules.Add("dead", "beef", func(ctx *validation.Context, configSet, path string, content []byte) error {
				So(string(content), ShouldEqual, "content")
				ctx.Errorf("blocking error")
				ctx.Warningf("diagnostic warning")
				return nil
			})
			valResp := valCall("dead", "beef", "content")
			So(rr.Code, ShouldEqual, http.StatusOK)
			So(valResp, ShouldResemble, &config.ValidationResponseMessage{
				Messages: []*config.ValidationResponseMessage_Message{
					{
						Text:     "in \"beef\": blocking error",
						Severity: config.ValidationResponseMessage_ERROR,
					},
					{
						Text:     "in \"beef\": diagnostic warning",
						Severity: config.ValidationResponseMessage_WARNING,
					},
				},
			})
		})

		Convey("validationHandler call with no configSet or path", func() {
			valCall("", "", "")
			So(rr.Code, ShouldEqual, http.StatusBadRequest)
			So(rr.Body.String(), ShouldEqual, "Must specify the config_set of the file to validate")
		})

		Convey("validationHandler call with no path", func() {
			valCall("dead", "", "")
			So(rr.Code, ShouldEqual, http.StatusBadRequest)
			So(rr.Body.String(), ShouldEqual, "Must specify the path of the file to validate")
		})
	})
}
