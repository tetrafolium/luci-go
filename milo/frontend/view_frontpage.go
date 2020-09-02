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
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/templates"

	"github.com/tetrafolium/luci-go/milo/common"
	"github.com/tetrafolium/luci-go/milo/frontend/ui"
)

func frontpageHandler(c *router.Context) {
	projs, err := common.GetVisibleProjects(c.Context)
	if err != nil {
		ErrorHandler(c, err)
		return
	}
	templates.MustRender(c.Context, c.Writer, "pages/frontpage.html", templates.Args{
		"frontpage": ui.Frontpage{Projects: projs},
	})
}
