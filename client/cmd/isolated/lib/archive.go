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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/chunker"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/command"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/tree"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/client/archiver"
	"go.chromium.org/luci/client/isolated"
	"go.chromium.org/luci/common/data/text/units"
	"go.chromium.org/luci/common/errors"
	isol "go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/system/signals"
)

// CmdArchive returns an object for the `archive` subcommand.
func CmdArchive(options CommandOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "archive <options>...",
		ShortDesc: "creates a .isolated file and uploads the tree to an isolate server",
		LongDesc: `Given a list of files and directories, creates a .isolated file and uploads the
tree to to an isolate server.

When specifying directories and files, you must also specify a current working
directory for that file or directory. The current working directory will not
be included in the archived path. For example, to isolate './usr/foo/bar' and
have it appear as 'foo/bar' in the .isolated, specify '-files ./usr:foo/bar' or
'-files usr:foo/bar'. When the .isolated is then downloaded, it will then appear
under 'foo/bar' in the desired directory.

Note that '.' may be omitted in general, so to upload 'foo' from the current
working directory, '-files :foo' is sufficient.`,
		CommandRun: func() subcommands.CommandRun {
			c := archiveRun{
				CommandOptions: options,
			}
			c.commonFlags.Init(options.DefaultAuthOpts)
			c.Flags.Var(&c.dirs, "dirs", "Directory(ies) to archive. Specify as <working directory>:<relative path to dir>")
			c.Flags.Var(&c.files, "files", "Individual file(s) to archive. Specify as <working directory>:<relative path to file>")
			c.Flags.StringVar(&c.dumpHash, "dump-hash", "",
				"Write the composite isolated hash to a file")
			c.Flags.StringVar(&c.isolated, "isolated", "",
				"Write the composite isolated to a file")
			c.Flags.StringVar(&c.dumpStatsJSON, "dump-stats-json", "",
				"Write the upload stats to this file as JSON")
			return &c
		},
	}
}

type archiveRun struct {
	commonFlags
	CommandOptions
	dirs          isolated.ScatterGather
	files         isolated.ScatterGather
	dumpHash      string
	isolated      string
	dumpStatsJSON string
}

func (c *archiveRun) Parse(a subcommands.Application, args []string) error {
	if err := c.commonFlags.Parse(); err != nil {
		return err
	}
	if len(args) != 0 {
		return errors.Reason("position arguments not expected").Err()
	}
	return nil
}

// getRoot returns root directory if there is only one working directory.
func getRoot(dirs, files isolated.ScatterGather) (string, error) {
	var rel0, wd0 string
	pickedOne := false
	for rel, wd := range dirs {
		if !pickedOne {
			rel0 = rel
			wd0 = wd
			pickedOne = true
			continue
		}

		if wd0 != wd {
			return "", errors.Reason("different root (working) directory is not supported: %s:%s vs %s:%s", wd0, rel0, wd, rel).Err()
		}
	}

	for rel, wd := range files {
		if !pickedOne {
			rel0 = rel
			wd0 = wd
			pickedOne = true
			continue
		}

		if wd0 != wd {
			return "", errors.Reason("different root (working) directory is not supported: %s:%s vs %s:%s", wd0, rel0, wd, rel).Err()
		}
	}

	if !pickedOne {
		return "", errors.Reason("-dirs or -files should be specified at least once").Err()
	}

	return wd0, nil
}

func (c *archiveRun) doCASAarchive(ctx context.Context) error {
	root, err := getRoot(c.dirs, c.files)
	if err != nil {
		return err
	}

	is := command.InputSpec{}
	for dir := range c.dirs {
		is.Inputs = append(is.Inputs, dir)
	}
	for file := range c.files {
		is.Inputs = append(is.Inputs, file)
	}

	rootDg, chunkers, _, err := tree.ComputeMerkleTree(root, &is, chunker.DefaultChunkSize, filemetadata.NewNoopCache())
	if err != nil {
		return errors.Annotate(err, "failed to call ComputeMerkleTree").Err()
	}

	client, err := c.casFlags.NewClient(ctx)
	if err != nil {
		return errors.Annotate(err, "failed to create cas client").Err()
	}
	defer client.Close()

	if err := client.UploadIfMissing(ctx, chunkers...); err != nil {
		return errors.Annotate(err, "failed to call UploadIfMissing").Err()
	}

	if c.dumpHash != "" {
		if err := ioutil.WriteFile(c.dumpHash, []byte(rootDg.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (c *archiveRun) doIsolatedArchive(ctx context.Context) (stats *archiver.Stats, err error) {
	isolatedClient, isolErr := c.createIsolatedClient(ctx, c.CommandOptions)
	if isolErr != nil {
		err = errors.Annotate(isolErr, "failed to create isolated client").Err()
		return
	}
	var out io.Writer = os.Stdout
	if c.defaultFlags.Quiet {
		out = ioutil.Discard
	}
	arch := archiver.New(ctx, isolatedClient, out)
	defer func() {
		// This waits for all uploads.
		if cerr := arch.Close(); err == nil {
			err = cerr
		}
		// We must take the stats until after all the uploads have finished
		if err == nil {
			stats = arch.Stats()
		}
	}()

	opts := isolated.ArchiveOptions{
		Files:    c.files,
		Dirs:     c.dirs,
		Isolated: c.isolated,
	}
	if len(c.isolated) != 0 {
		var dumpIsolated *os.File
		dumpIsolated, err = os.Create(c.isolated)
		if err != nil {
			return
		}
		// This is OK to close before arch because isolated.Archive
		// does the writing (it's not handed off elsewhere).
		defer dumpIsolated.Close()
		opts.LeakIsolated = dumpIsolated
	}
	item := isolated.Archive(ctx, arch, &opts)
	if err = item.Error(); err != nil {
		return
	}

	item.WaitForHashed()
	if len(c.dumpHash) != 0 {
		if err = ioutil.WriteFile(c.dumpHash, []byte(item.Digest()), 0644); err != nil {
			return
		}
	}
	return
}

// Does the archive by uploading to isolate-server, then return the archive stats and error.
func (c *archiveRun) doArchive(a subcommands.Application, args []string) (stats *archiver.Stats, err error) {
	ctx, cancel := context.WithCancel(c.defaultFlags.MakeLoggingContext(os.Stderr))
	signals.HandleInterrupt(cancel)

	if c.casFlags.Instance != "" {
		// TODO(crbug.com/1110569): get stats
		return &archiver.Stats{}, c.doCASAarchive(ctx)
	}

	return c.doIsolatedArchive(ctx)
}

func (c *archiveRun) postprocessStats(stats *archiver.Stats, start time.Time) error {
	if !c.defaultFlags.Quiet {
		duration := time.Since(start)
		fmt.Fprintf(os.Stderr, "Hits    : %5d (%s)\n", stats.TotalHits(), stats.TotalBytesHits())
		fmt.Fprintf(os.Stderr, "Misses  : %5d (%s)\n", stats.TotalMisses(), stats.TotalBytesPushed())
		fmt.Fprintf(os.Stderr, "Duration: %s\n", units.Round(duration, time.Millisecond))
	}
	if c.dumpStatsJSON != "" {
		return dumpStatsJSON(c.dumpStatsJSON, stats)
	}
	return nil
}

func (c *archiveRun) Run(a subcommands.Application, args []string, _ subcommands.Env) int {
	if err := c.Parse(a, args); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	cl, err := c.defaultFlags.StartTracing()
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	defer cl.Close()
	defer c.profilerFlags.Stop()
	start := time.Now()
	stats, err := c.doArchive(a, args)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	if err := c.postprocessStats(stats, start); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func dumpStatsJSON(jsonPath string, stats *archiver.Stats) error {
	hits := make([]int64, len(stats.Hits))
	for i, h := range stats.Hits {
		hits[i] = int64(h)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i] < hits[j] })
	itemsHot, err := isol.Pack(hits)
	if err != nil {
		return errors.Annotate(err, "failed to pack itemsHot").Err()
	}

	pushed := make([]int64, len(stats.Pushed))
	for i, p := range stats.Pushed {
		pushed[i] = int64(p.Size)
	}
	sort.Slice(pushed, func(i, j int) bool { return pushed[i] < pushed[j] })
	itemsCold, err := isol.Pack(pushed)
	if err != nil {
		return errors.Annotate(err, "failed to pack itemsCold").Err()
	}

	statsJSON, err := json.Marshal(struct {
		ItemsCold []byte `json:"items_cold"`
		ItemsHot  []byte `json:"items_hot"`
	}{
		ItemsCold: itemsCold,
		ItemsHot:  itemsHot,
	})
	if err != nil {
		return errors.Annotate(err, "failed to marshal result json").Err()
	}
	if err := ioutil.WriteFile(jsonPath, statsJSON, 0664); err != nil {
		return errors.Annotate(err, "failed to write stats json to %s", jsonPath).Err()
	}
	return nil
}
