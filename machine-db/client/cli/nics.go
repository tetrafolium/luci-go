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

// printNICs prints network interface data to stdout in tab-separated columns.
func printNICs(tsv bool, nics ...*crimson.NIC) {
	if len(nics) > 0 {
		p := newStdoutPrinter(tsv)
		defer p.Flush()
		if !tsv {
			p.Row("Name", "Machine", "MAC Address", "Switch", "Port")
		}
		for _, n := range nics {
			p.Row(n.Name, n.Machine, n.MacAddress, n.Switch, n.Switchport)
		}
	}
}

// AddNICCmd is the command to add a network interface.
type AddNICCmd struct {
	commandBase
	nic crimson.NIC
}

// Run runs the command to add a network interface.
func (c *AddNICCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	req := &crimson.CreateNICRequest{
		Nic: &c.nic,
	}
	client := getClient(ctx)
	resp, err := client.CreateNIC(ctx, req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printNICs(c.f.tsv, resp)
	return 0
}

// addNICCmd returns a command to add a network interface.
func addNICCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "add-nic -name <name> -machine <machine> -mac <mac address> -switch <switch> [-port <switch port>] [-host <hostname>] [-ip <ip address>]",
		ShortDesc: "adds a NIC",
		LongDesc:  "Adds a network interface to the database.\n\nExample:\ncrimson add-nic -name eth0 -machine xx1-01-720 -mac 00:00:00:00:00:bc -switch switch1.lab -port 30",
		CommandRun: func() subcommands.CommandRun {
			cmd := &AddNICCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.nic.Name, "name", "", "The name of the NIC. Required and must be unique per machine within the database.")
			cmd.Flags.StringVar(&cmd.nic.Machine, "machine", "", "The machine this NIC belongs to. Required and must be the name of a machine returned by get-machines.")
			cmd.Flags.StringVar(&cmd.nic.MacAddress, "mac", "", "The MAC address of this NIC. Required and must be a valid MAC-48 address.")
			cmd.Flags.StringVar(&cmd.nic.Switch, "switch", "", "The switch this NIC is connected to. Required and must be the name of a switch returned by get-switches.")
			cmd.Flags.Var(flag.Int32(&cmd.nic.Switchport), "port", "The switchport this NIC is connected to.")
			cmd.Flags.StringVar(&cmd.nic.Hostname, "host", "", "The name of this NIC on the network.")
			cmd.Flags.StringVar(&cmd.nic.Ipv4, "ip", "", "The IPv4 address assigned to this NIC. Must be a free IP address returned by get-ips.")
			return cmd
		},
	}
}

// DeleteNICCmd is the command to delete a network interface.
type DeleteNICCmd struct {
	commandBase
	req crimson.DeleteNICRequest
}

// Run runs the command to delete a network interface.
func (c *DeleteNICCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	client := getClient(ctx)
	_, err := client.DeleteNIC(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	return 0
}

// deleteNICCmd returns a command to delete a network interface.
func deleteNICCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "del-nic -name <name> -machine <machine>",
		ShortDesc: "deletes a NIC",
		LongDesc:  "Deletes a network interface from the database.\n\nExample:\ncrimson del-nic -name eth1 -machine xx1-01-720",
		CommandRun: func() subcommands.CommandRun {
			cmd := &DeleteNICCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.req.Name, "name", "", "The name of the NIC to delete.")
			cmd.Flags.StringVar(&cmd.req.Machine, "machine", "", "The machine the NIC belongs to.")
			return cmd
		},
	}
}

// EditNICCmd is the command to edit a network interface.
type EditNICCmd struct {
	commandBase
	nic crimson.NIC
}

// Run runs the command to edit a network interface.
func (c *EditNICCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	// TODO(smut): Validate required fields client-side.
	req := &crimson.UpdateNICRequest{
		Nic: &c.nic,
		UpdateMask: getUpdateMask(&c.Flags, map[string]string{
			"mac":    "mac_address",
			"switch": "switch",
			"port":   "switchport",
		}),
	}
	client := getClient(ctx)
	resp, err := client.UpdateNIC(ctx, req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printNICs(c.f.tsv, resp)
	return 0
}

// editNICCmd returns a command to edit a network interface.
func editNICCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "edit-nic -name <name> -machine <machine> [-mac <mac address>] [-switch <switch>] [-port <switch port>]",
		ShortDesc: "edit a NIC",
		LongDesc:  "Edits a network interface in the database.\n\nExample to edit a NIC's MAC address:\ncrimson edit-nic -name eth0 -machine xx1-01-720 -mac 00:00:00:00:00:bc",
		CommandRun: func() subcommands.CommandRun {
			cmd := &EditNICCmd{}
			cmd.Initialize(params)
			cmd.Flags.StringVar(&cmd.nic.Name, "name", "", "The name of the NIC. Required and must be the name of a NIC returned by get-nics.")
			cmd.Flags.StringVar(&cmd.nic.Machine, "machine", "", "The machine this NIC belongs to. Required and must be the name of a machine returned by get-machines.")
			cmd.Flags.StringVar(&cmd.nic.MacAddress, "mac", "", "The MAC address of this NIC. Must be a valid MAC-48 address.")
			cmd.Flags.StringVar(&cmd.nic.Switch, "switch", "", "The switch this NIC is connected to. Must be the name of a switch returned by get-switches.")
			cmd.Flags.Var(flag.Int32(&cmd.nic.Switchport), "port", "The switchport this NIC is connected to.")
			return cmd
		},
	}
}

// GetNICsCmd is the command to get network interfaces.
type GetNICsCmd struct {
	commandBase
	req crimson.ListNICsRequest
}

// Run runs the command to get network interfaces.
func (c *GetNICsCmd) Run(app subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(app, c, env)
	client := getClient(ctx)
	resp, err := client.ListNICs(ctx, &c.req)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}
	printNICs(c.f.tsv, resp.Nics...)
	return 0
}

// getNICCmd returns a command to get network interfaces.
func getNICsCmd(params *Parameters) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "get-nics [-name <name>]... [-machine <machine>]...",
		ShortDesc: "retrieves NICs",
		LongDesc:  "Retrieves network interfaces matching the given names and machines, or all network interfaces if names and machines are omitted.\n\nExample to get all NICs:\ncrimson get-nics\nExample to get the NIC with a certain MAC address:\ncrimson get-nics -mac 00:00:00:00:00:bc",
		CommandRun: func() subcommands.CommandRun {
			cmd := &GetNICsCmd{}
			cmd.Initialize(params)
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Names), "name", "Name of a NIC to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Machines), "machine", "Name of a machine to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.MacAddresses), "mac", "MAC address to filter by. Can be specified multiple times.")
			cmd.Flags.Var(flag.StringSlice(&cmd.req.Switches), "switch", "Name of a switch to filter by. Can be specified multiple times.")
			return cmd
		},
	}
}
