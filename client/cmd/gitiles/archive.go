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

package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/api/gitiles"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	gitilespb "github.com/tetrafolium/luci-go/common/proto/gitiles"
)

func cmdArchive(authOpts auth.Options) *subcommands.Command {

	return &subcommands.Command{
		UsageLine: "archive <options> repository-url committish",
		ShortDesc: "downloads an archive of a repo at committish",
		LongDesc: `Downloads an archive of a repo at given committish.

This tool does not stream the archive, so the full contents are stored in
memory before being written to disk.
		`,
		CommandRun: func() subcommands.CommandRun {
			c := archiveRun{}
			c.commonFlags.Init(authOpts)
			c.Flags.StringVar(&c.rawFormat, "format", "GZIP",
				fmt.Sprintf("Format of the archive requested. One of %s", formatChoices()))
			c.Flags.StringVar(&c.output, "output", "", "Path to write archive to.")
			return &c
		},
	}
}

type archiveRun struct {
	commonFlags
	format gitilespb.ArchiveRequest_Format
	output string

	rawFormat string
}

func (c *archiveRun) Parse(a subcommands.Application, args []string) error {
	if err := c.commonFlags.Parse(); err != nil {
		return err
	}
	if len(args) != 2 {
		return errors.New("exactly 2 position arguments are expected")
	}
	if c.format = parseFormat(c.rawFormat); c.format == gitilespb.ArchiveRequest_Invalid {
		return errors.New("invalid archive format requested")
	}
	return nil
}

func formatChoices() []string {
	cs := make([]string, 0, len(gitilespb.ArchiveRequest_Format_value))
	for k := range gitilespb.ArchiveRequest_Format_value {
		cs = append(cs, k)
	}
	sort.Strings(cs)
	return cs
}

func parseFormat(f string) gitilespb.ArchiveRequest_Format {
	return gitilespb.ArchiveRequest_Format(gitilespb.ArchiveRequest_Format_value[strings.ToUpper(f)])
}

func (c *archiveRun) main(a subcommands.Application, args []string) error {
	ctx := c.defaultFlags.MakeLoggingContext(os.Stderr)
	host, project, err := gitiles.ParseRepoURL(args[0])
	if err != nil {
		return errors.Annotate(err, "invalid repo URL %q", args[0]).Err()
	}
	ref := args[1]
	req := &gitilespb.ArchiveRequest{
		Format:  c.format,
		Project: project,
		Ref:     ref,
	}

	authCl, err := c.createAuthClient()
	if err != nil {
		return err
	}
	g, err := gitiles.NewRESTClient(authCl, host, true)
	if err != nil {
		return err
	}

	res, err := g.Archive(ctx, req)
	if err != nil {
		return err
	}

	return c.dumpArchive(ctx, res)
}

func (c *archiveRun) dumpArchive(ctx context.Context, res *gitilespb.ArchiveResponse) error {
	var oPath string
	switch {
	case c.output != "":
		oPath = c.output
	case res.Filename != "":
		oPath = res.Filename
	default:
		return errors.New("No output path specified and no suggested archive name from remote")
	}

	f, err := os.Create(oPath)
	if err != nil {
		return errors.Annotate(err, "failed to open file to write archive").Err()
	}
	defer f.Close()

	l, err := f.Write(res.Contents)
	logging.Infof(ctx, "Archive written to %s (size: %s)", oPath, humanize.Bytes(uint64(l)))
	return err
}

func (c *archiveRun) Run(a subcommands.Application, args []string, _ subcommands.Env) int {
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
