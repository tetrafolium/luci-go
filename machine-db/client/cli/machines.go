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

package cli

import (
	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/common/cli"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/flag"

	"github.com/tetrafolium/luci-go/machine-db/api/crimson/v1"
)

// printMachines prints machine data to stdout in tab-separated columns.
func printMachines(tsv bool, machines ...*crimson.Machine) {
	if len(machines) > 0 {
		p := newStdoutPrinter(tsv)
		defer p.Flush()
		if !tsv {
			p.Row("Name", "Platform", "Rack", "Datacenter", "Description", "Asset Tag", "Service Tag", "Deployment Ticket", "DRAC Password", "State")
		}
		for _, m := range machines {
			p.Row(m.Name, m.Platform, m.Rack, m.Datacenter, m.Description, m.AssetTag, m.ServiceTag, m.DeploymentTicket, m.DracPassword, m.State)
		}
	}
}

// AddMachineCmd is the command to add a machine.
type AddMachineCmd struct {
	commandBase
	machine crimson.Machine
}

// Run runs the command to add a machine.
func (c *AddMachineCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	req := &crimson.CreateMachineRequest{
		Machine: &c.machine,
	}
	client := getClient(ctx)
	resp, err := client.CreateMachine(ctx, req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printMachines(c.f.tsv, resp)
	return 0
}

// addMachineCmd returns a command to add a machine.
func addMachineCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "add-machine -name <name> -plat <platform> -rack <rack> -state <state> [-desc <description>] [-atag <asset tag>] [-stag <service tag>] [-tick <deployment ticket>] [-dracpass <DRAC password>]",
		ShortDesc: "adds a machine",
		LongDesc:  "Adds a machine to the database.\n\nExample:\ncrimson add-machine -name xx11-11-720 -plat 'Apple Mac Pro' -rack xx1 -state test -stag BC0001",
		CommandRun: func() subcommands.CommandRun {
			cmd := &AddMachineCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.machine.Name, "name", "", "The name of the machine. Required and must be unique within the database.")
			cmd.Flags.StringVar(&cmd.machine.Platform, "plat", "", "The platform type this machine is. Required and must be the name of a platform returned by get-platforms.")
			cmd.Flags.StringVar(&cmd.machine.Rack, "rack", "", "The rack this machine belongs to. Required and must be the name of a rack returned by get-racks.")
			cmd.Flags.Var(StateFlag(&cmd.machine.State), "state", "The state of this machine. Required and must be a state returned by get-states.")
			cmd.Flags.StringVar(&cmd.machine.Description, "desc", "", "A description of this machine.")
			cmd.Flags.StringVar(&cmd.machine.AssetTag, "atag", "", "The asset tag associated with this machine.")
			cmd.Flags.StringVar(&cmd.machine.ServiceTag, "stag", "", "The service tag associated with this machine.")
			cmd.Flags.StringVar(&cmd.machine.DeploymentTicket, "tick", "", "The deployment ticket associated with this machine.")
			cmd.Flags.StringVar(&cmd.machine.DracPassword, "dracpass", "", "The initial DRAC password associated with this machine.")
			return cmd
		},
	}
}

// DeleteMachineCmd is the command to delete a machine.
type DeleteMachineCmd struct {
	commandBase
	req crimson.DeleteMachineRequest
}

// Run runs the command to delete a machine.
func (c *DeleteMachineCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	client := getClient(ctx)
	_, err := client.DeleteMachine(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// deleteMachineCmd returns a command to delete a machine.
func deleteMachineCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "del-machine -name <name>",
		ShortDesc: "deletes a machine",
		LongDesc:  "Deletes a machine from the database.\n\nExample:\ncrimson del-machine -name xx11-11-720",
		CommandRun: func() subcommands.CommandRun {
			cmd := &DeleteMachineCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.req.Name, "name", "", "The name of the machine to delete.")
			return cmd
		},
	}
}

// EditMachineCmd is the command to edit a machine.
type EditMachineCmd struct {
	commandBase
	machine crimson.Machine
}

// Run runs the command to edit a machine.
func (c *EditMachineCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	req := &crimson.UpdateMachineRequest{
		Machine: &c.machine,
		UpdateMask: getUpdateMask(&c.Flags, map[string]string{
			"plat":     "platform",
			"rack":     "rack",
			"state":    "state",
			"desc":     "description",
			"atag":     "asset_tag",
			"stag":     "service_tag",
			"tick":     "deployment_ticket",
			"dracpass": "drac_password",
		}),
	}
	client := getClient(ctx)
	resp, err := client.UpdateMachine(ctx, req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printMachines(c.f.tsv, resp)
	return 0
}

// editMachineCmd returns a command to edit a machine.
func editMachineCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "edit-machine -name <name> [-plat <platform>] [-rack <rack>] [-state <state>] [-desc <description>] [-atag <asset tag>] [-stag <service tag>] [-tick <deployment ticket>] [-dracpass <DRAC password>]",
		ShortDesc: "edits a machine",
		LongDesc:  "Edits a machine in the database.\n\nExample to change a machine to repair state and add ticket for repairs:\ncrimson edit-machine -name server01 -state repair -tick tick/1111",
		CommandRun: func() subcommands.CommandRun {
			cmd := &EditMachineCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.machine.Name, "name", "", "The name of the machine. Required and must be the name of a machine returned by get-machines.")
			cmd.Flags.StringVar(&cmd.machine.Platform, "plat", "", "The platform type this machine is. Must be the name of a platform returned by get-platforms.")
			cmd.Flags.StringVar(&cmd.machine.Rack, "rack", "", "The rack this machine belongs to. Must be the name of a rack returned by get-racks.")
			cmd.Flags.Var(StateFlag(&cmd.machine.State), "state", "The state of this machine. Must be a state returned by get-states.")
			cmd.Flags.StringVar(&cmd.machine.Description, "desc", "", "A description of this machine.")
			cmd.Flags.StringVar(&cmd.machine.AssetTag, "atag", "", "The asset tag associated with this machine.")
			cmd.Flags.StringVar(&cmd.machine.ServiceTag, "stag", "", "The service tag associated with this machine.")
			cmd.Flags.StringVar(&cmd.machine.DeploymentTicket, "tick", "", "The deployment ticket associated with this machine.")
			cmd.Flags.StringVar(&cmd.machine.DracPassword, "dracpass", "", "The initial DRAC password associated with this machine.")
			return cmd
		},
	}
}

// GetMachinesCmd is the command to get machines.
type GetMachinesCmd struct {
	commandBase
	req crimson.ListMachinesRequest
}

// Run runs the command to get machines.
func (c *GetMachinesCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	client := getClient(ctx)
	resp, err := client.ListMachines(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printMachines(c.f.tsv, resp.Machines...)
	return 0
}

// getMachinesCmd returns a command to get machines.
func getMachinesCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "get-machines [-name <name>]... [-plat <plat>]... [-rack <rack>]... [-dc <dc>]... [-state <state>]...",
		ShortDesc: "retrieves machines",
		LongDesc:  "Retrieves machines matching the given names, platforms, racks, and states, or all machines if names are omitted.\n\nExample to get all machines:\ncrimson get-machines\nExample to get all machines in rack xx1 that are in repair state:\ncrimson get-machines -rack xx1 -state repair",
		CommandRun: func() subcommands.CommandRun {
			cmd := &GetMachinesCmd{}
			cmd.Initialize(params)
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Names), "name", "Name of a machine to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Platforms), "plat", "Name of a platform to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Racks), "rack", "Name of a rack to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Datacenters), "dc", "Name of a datacenter to filter by. Can be specified multiple times.")
			cmd.Flags.Var(StateSliceFlag(&cmd.req.States), "state", "State to filter by. Can be specified multiple times.")
			return cmd
		},
	}
}

// RenameMachineCmd is the command to rename a machine.
type RenameMachineCmd struct {
	commandBase
	req crimson.RenameMachineRequest
}

// Run runs the command to rename a machine.
func (c *RenameMachineCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	client := getClient(ctx)
	resp, err := client.RenameMachine(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printMachines(c.f.tsv, resp)
	return 0
}

// renameMachineCmd returns a command to rename a machine.
func renameMachineCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "name-machine -old <name> -new <name>",
		ShortDesc: "renames a machine",
		LongDesc:  "Renames a machine in the database.\n\nExample:\ncrimson name-machine -old xx01-07-720 -new yy01-07-720",
		CommandRun: func() subcommands.CommandRun {
			cmd := &RenameMachineCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.req.Name, "old", "", "The name of the machine. Required and must be the name of a machine returned by get-machines.")
			cmd.Flags.StringVar(&cmd.req.NewName, "new", "", "The new name of the machine. Required and must be unique within the database.")
			return cmd
		},
	}
}
