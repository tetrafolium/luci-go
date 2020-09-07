// Copyright 2014 The LUCI Authors.
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

// Package main contains CIPD CLI implementation that uses Chrome Infrastructure
// defaults.
//
// It hardcodes default CIPD backend URL, OAuth client ID, location of the token
// cache, etc.
//
// See github.com/tetrafolium/luci-go/cipd/client/cli if you want to build your own
// version with different defaults.
package main

import (
	"os"

	"github.com/tetrafolium/luci-go/cipd/client/cli"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra"
)

func main() {
	mathrand.SeedRandomly()
	params := cli.Parameters{
		DefaultAuthOptions: chromeinfra.DefaultAuthOptions(),
		ServiceURL:         chromeinfra.CIPDServiceURL,
	}
	os.Exit(cli.Main(params, os.Args[1:]))
}
