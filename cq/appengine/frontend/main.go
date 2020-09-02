// Copyright 2019 The LUCI Authors.
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

// Binary frontend is the main entry point for the CQ app.
package main

import (
	"github.com/tetrafolium/luci-go/config/server/cfgmodule"
	"github.com/tetrafolium/luci-go/server"
	"github.com/tetrafolium/luci-go/server/module"

	// Ensure registration of validation rules.
	_ "github.com/tetrafolium/luci-go/cq/appengine/config"
)

func main() {
	modules := []module.Module{
		cfgmodule.NewModuleFromFlags(),
	}
	server.Main(nil, modules, nil)
}
