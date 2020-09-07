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

// Package main is a .isolate compiler that compiles .isolate files into
// .isolated files and can also act as a client to an Isolate server.
//
// It is designed to be compatible with the reference python implementation at
// https://github.com/luci/luci-py/tree/master/client.
package main

import (
	"log"
	"os"

	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/auth/client/authcli"
	"github.com/tetrafolium/luci-go/client/cmd/isolate/lib"
	"github.com/tetrafolium/luci-go/client/versioncli"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"

	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra"
)

func getApplication(defaultAuthOpts auth.Options) *subcommands.DefaultApplication {
	return &subcommands.DefaultApplication{
		Name:  "isolate",
		Title: "isolate.py but faster",
		// Keep in alphabetical order of their name.
		Commands: []*subcommands.Command{
			lib.CmdArchive(defaultAuthOpts),
			lib.CmdBatchArchive(defaultAuthOpts),
			lib.CmdCheck(),
			lib.CmdRemap(),
			lib.CmdRun(),
			subcommands.CmdHelp,
			authcli.SubcommandInfo(defaultAuthOpts, "whoami", false),
			authcli.SubcommandLogin(defaultAuthOpts, "login", false),
			authcli.SubcommandLogout(defaultAuthOpts, "logout", false),
			versioncli.CmdVersion(lib.IsolateVersion),
		},
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	mathrand.SeedRandomly()
	app := getApplication(chromeinfra.DefaultAuthOptions())
	os.Exit(subcommands.Run(app, nil))
}
