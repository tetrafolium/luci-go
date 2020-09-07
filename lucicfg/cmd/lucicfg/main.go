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

// Command lucicfg is CLI for LUCI config generator.
package main

import (
	"os"

	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra"

	"github.com/tetrafolium/luci-go/lucicfg/cli"
	"github.com/tetrafolium/luci-go/lucicfg/cli/base"
)

func main() {
	mathrand.SeedRandomly()
	params := base.Parameters{
		AuthOptions:       chromeinfra.DefaultAuthOptions(),
		ConfigServiceHost: chromeinfra.ConfigServiceHost,
	}
	os.Exit(cli.Main(params, os.Args[1:]))
}
