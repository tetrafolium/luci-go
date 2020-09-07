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

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/auth/client/authcli"
	"github.com/tetrafolium/luci-go/client/versioncli"
	"github.com/tetrafolium/luci-go/common/cli"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/logging/gologger"

	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra"
)

// version must be updated whenever functional change (behavior, arguments,
// supported commands) is done.
const version = "0.1"

func getApplication(defaultAuthOpts auth.Options) *cli.Application {
	defaultAuthOpts.Scopes = []string{auth.OAuthScopeEmail, "https://www.googleapis.com/auth/cloud-platform"}
	return &cli.Application{
		Name:  "cloudkms",
		Title: "Client for interfacing with Google Cloud Key Management Service",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		// Keep in alphabetical order of their name.
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			cmdDecrypt(defaultAuthOpts),
			cmdEncrypt(defaultAuthOpts),
			cmdSign(defaultAuthOpts),
			cmdVerify(defaultAuthOpts),
			cmdDownload(defaultAuthOpts),
			authcli.SubcommandInfo(defaultAuthOpts, "whoami", false),
			authcli.SubcommandLogin(defaultAuthOpts, "login", false),
			authcli.SubcommandLogout(defaultAuthOpts, "logout", false),
			versioncli.CmdVersion(version),
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	app := getApplication(chromeinfra.DefaultAuthOptions())
	os.Exit(subcommands.Run(app, nil))
}
