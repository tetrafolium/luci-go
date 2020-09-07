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

package lib

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"github.com/tetrafolium/luci-go/client/isolate"
	"github.com/tetrafolium/luci-go/common/errors"
)

// CmdCheck returns an object for the `check` subcommand.
func CmdCheck() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "check <options>",
		ShortDesc: "checks that all the inputs are present",
		LongDesc:  "",
		CommandRun: func() subcommands.CommandRun {
			c := checkRun{}
			c.commonFlags.Init()
			c.isolateFlags.Init(&c.Flags)
			return &c
		},
	}
}

type checkRun struct {
	commonFlags
	isolateFlags
}

func (c *checkRun) Parse(a subcommands.Application, args []string) error {
	if err := c.commonFlags.Parse(); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := c.isolateFlags.Parse(cwd, RequireIsolateFile); err != nil {
		return err
	}
	if len(args) != 0 {
		return errors.Reason("position arguments not expected").Err()
	}
	return nil
}

func (c *checkRun) main(a subcommands.Application, args []string) error {
	if !c.defaultFlags.Quiet {
		fmt.Printf("Isolate:   %s\n", c.Isolate)
		fmt.Printf("Isolated:  %s\n", c.Isolated)
		fmt.Printf("Config:    %s\n", c.ConfigVariables)
		fmt.Printf("Path:      %s\n", c.PathVariables)
	}

	deps, _, _, err := isolate.ProcessIsolate(&c.ArchiveOptions)
	if err != nil {
		return errors.Annotate(err, "failed to process isolate").Err()
	}

	for _, dep := range deps {
		_, err := os.Stat(dep)
		if err != nil {
			return errors.Annotate(err, "failed to call stat").Err()
		}
	}
	return nil
}

func (c *checkRun) Run(a subcommands.Application, args []string, _ subcommands.Env) int {
	if err := c.Parse(a, args); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	if err := c.main(a, args); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}
