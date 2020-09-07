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

// printSwitches prints switch data to stdout in tab-separated columns.
func printSwitches(tsv bool, switches ...*crimson.Switch) {
	if len(switches) > 0 {
		p := newStdoutPrinter(tsv)
		defer p.Flush()
		if !tsv {
			p.Row("Name", "Ports", "Rack", "Datacenter", "Description", "State")
		}
		for _, s := range switches {
			p.Row(s.Name, s.Ports, s.Rack, s.Datacenter, s.Description, s.State)
		}
	}
}

// GetSwitchesCmd is the command to get switches.
type GetSwitchesCmd struct {
	commandBase
	req crimson.ListSwitchesRequest
}

// Run runs the command to get switches.
func (c *GetSwitchesCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	client := getClient(ctx)
	resp, err := client.ListSwitches(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printSwitches(c.f.tsv, resp.Switches...)
	return 0
}

// getSwitchesCmd returns a command to get switches.
func getSwitchesCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "get-switches [-name <name>]... [-rack <rack>]... [-dc <datacenter>]...",
		ShortDesc: "retrieves switches",
		LongDesc:  "Retrieves switches matching the given names, racks and dcs, or all switches if names, racks, and dcs are omitted.\n\nExample to get all switches:\ncrimson get-switches\nExample to get the switch of rack xx1:\ncrimson get-switches -rack xx1",
		CommandRun: func() subcommands.CommandRun {
			cmd := &GetSwitchesCmd{}
			cmd.Initialize(params)
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Names), "name", "Name of a switch to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Racks), "rack", "Name of a rack to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Datacenters), "dc", "Name of a datacenter to filter by. Can be specified multiple times.")
			return cmd
		},
	}
}
