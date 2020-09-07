// Copyright 2018 The LUCI Authors.
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

package config

import (
	"testing"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/logging/gologger"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/impl/memory"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEmailTemplate(t *testing.T) {
	t.Parallel()

	Convey("fetchAllEmailTemplates", t, func() {
		c := gaetesting.TestingContextWithAppID("luci-notify")
		c = gologger.StdConfig.Use(c)
		c = logging.SetLevel(c, logging.Debug)

		cfgService := memory.New(map[config.Set]memory.Files{
			"projects/x": {
				"luci-notify/email-templates/a.template":            "aSubject\n\naBody",
				"luci-notify/email-templates/b.template":            "bSubject\n\nbBody",
				"luci-notify/email-templates/invalid name.template": "subject\n\nbody",
			},
			"projects/y": {
				"luci-notify/email-templates/c.template": "cSubject\n\ncBody",
			},
		})
		templates, err := fetchAllEmailTemplates(c, cfgService, "x")
		So(err, ShouldBeNil)

		So(templates, ShouldResemble, map[string]*EmailTemplate{
			"a": {
				Name:                "a",
				SubjectTextTemplate: "aSubject",
				BodyHTMLTemplate:    "aBody",
				DefinitionURL:       "https://example.com/view/here/luci-notify/email-templates/a.template",
			},
			"b": {
				Name:                "b",
				SubjectTextTemplate: "bSubject",
				BodyHTMLTemplate:    "bBody",
				DefinitionURL:       "https://example.com/view/here/luci-notify/email-templates/b.template",
			},
		})
	})
}
