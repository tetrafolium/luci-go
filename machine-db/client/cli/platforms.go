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

package cli

import (
	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/common/cli"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/flag"

	"github.com/tetrafolium/luci-go/machine-db/api/crimson/v1"
)

// printPlatforms prints platform data to stdout in tab-separated columns.
func printPlatforms(tsv bool, platforms ...*crimson.Platform) {
	if len(platforms) > 0 {
		p := newStdoutPrinter(tsv)
		defer p.Flush()
		if !tsv {
			p.Row("Name", "Description", "Manufacturer")
		}
		for _, plat := range platforms {
			p.Row(plat.Name, plat.Description, plat.Manufacturer)
		}
	}
}

// GetPlatformsCmd is the command to get platforms.
type GetPlatformsCmd struct {
	commandBase
	req crimson.ListPlatformsRequest
}

// Run runs the command to get platforms.
func (c *GetPlatformsCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	client := getClient(ctx)
	resp, err := client.ListPlatforms(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printPlatforms(c.f.tsv, resp.Platforms...)
	return 0
}

// getPlatformsCmd returns a command to get platforms.
func getPlatformsCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "get-platforms [-name <name>]... [-man <manufacturer>]...",
		ShortDesc: "retrieves platforms",
		LongDesc:  "Retrieves platforms matching the given names, or all platforms if names and manufacturers are omitted.\n\nExample to get all platforms:\ncrimson get-platforms\nExample to get all Apple platforms:\ncrimson get-platforms -man apple",
		CommandRun: func() subcommands.CommandRun {
			cmd := &GetPlatformsCmd{}
			cmd.Initialize(params)
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Names), "name", "Name of a platform to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Manufacturers), "man", "Manufacturer to filter by. Can be specified multiple times.")
			return cmd
		},
	}
}
