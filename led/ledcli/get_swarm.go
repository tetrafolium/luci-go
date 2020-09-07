// Copyright 2020 The LUCI Authors.
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

package ledcli

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/led/job"
	"github.com/tetrafolium/luci-go/led/ledcmd"
)

func getSwarmCmd(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "get-swarm <swarm task id>",
		ShortDesc: "obtain a JobDefinition from a swarming task",
		LongDesc:  `Obtains the task definition from swarming and produce a JobDefinition.`,

		CommandRun: func() subcommands.CommandRun {
			ret := &cmdGetSwarm{}
			ret.initFlags(opts)
			return ret
		},
	}
}

type cmdGetSwarm struct {
	cmdBase

	taskID       string
	swarmingHost string
	pinBotID     bool
}

func (c *cmdGetSwarm) initFlags(opts cmdBaseOptions) {
	c.Flags.StringVar(&c.swarmingHost, "S", "chromium-swarm.appspot.com",
		"the swarming `host` to get the task from.")

	c.Flags.BoolVar(&c.pinBotID, "pin-bot-id", false,
		"Pin the bot id in the generated job Definition's dimensions.")

	c.cmdBase.initFlags(opts)
}

func (c *cmdGetSwarm) jobInput() bool                  { return false }
func (c *cmdGetSwarm) positionalRange() (min, max int) { return 1, 1 }

func (c *cmdGetSwarm) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	c.taskID = positionals[0]
	return errors.Annotate(pingHost(c.swarmingHost), "swarming host").Err()
}

func (c *cmdGetSwarm) execute(ctx context.Context, authClient *http.Client, inJob *job.Definition) (out interface{}, err error) {
	return ledcmd.GetFromSwarmingTask(ctx, authClient, ledcmd.GetFromSwarmingTaskOpts{
		Name:         fmt.Sprintf("led get-swarm %s", c.taskID),
		PinBotID:     c.pinBotID,
		SwarmingHost: c.swarmingHost,
		TaskID:       c.taskID,

		KitchenSupport: c.kitchenSupport,
	})
}

func (c *cmdGetSwarm) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return c.doContextExecute(a, c, args, env)
}
