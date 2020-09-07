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
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/maruel/subcommands"

	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	gerritapi "github.com/tetrafolium/luci-go/common/api/gerrit"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/proto/gerrit"
	"github.com/tetrafolium/luci-go/led/job"
)

func editCrCLCmd(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "edit-cr-cl [-remove|-no-implicit-clear] URL_TO_CHANGELIST",
		ShortDesc: "sets Chromium CL-related properties on this JobDefinition (for experimenting with tryjob recipes)",
		LongDesc: `This allows you to edit a JobDefinition for some tryjob recipe
(e.g. chromium_tryjob), and associate a changelist with it, as if the recipe
was triggered via Gerrit.

Recognized URLs:
	https://<gerrit_host>/c/<path/to/project>/+/<change>
	https://<gerrit_host>/c/<path/to/project>/+/<change>/<patchset>

If you provide a CL missing <patchset> AND <gerrit_host> has public read access,
this will fill in the patchset from the latest version of the change. Otherwise
this will fail and ask you to provide the full CL/patchset url.

By default, when adding a CL, this will clear all existing CLs on the job, unless
you pass -no-implicit-clear. Most jobs (as of 2020Q2) only expect one CL, so we
did this implicit clearing behavior for CLI ergonomic reasons.
`,

		CommandRun: func() subcommands.CommandRun {
			ret := &cmdEditCl{}
			ret.initFlags(opts)
			return ret
		},
	}
}

type cmdEditCl struct {
	cmdBase

	gerritChange    *bbpb.GerritChange
	remove          bool
	noImplicitClear bool
}

func (c *cmdEditCl) initFlags(opts cmdBaseOptions) {
	c.Flags.BoolVar(&c.remove, "remove", false, "If provided, will remove the given CL instead of adding it.")
	c.Flags.BoolVar(&c.noImplicitClear, "no-implicit-clear", false,
		"If provided, will not clear existing CLs when adding a new one.")
	c.cmdBase.initFlags(opts)
}

func (c *cmdEditCl) jobInput() bool                  { return true }
func (c *cmdEditCl) positionalRange() (min, max int) { return 1, 1 }

type patchsetResolver func(host string, change int64) (ps int64, err error)

func parseCrChangeListURL(clURL string, resolvePatchset patchsetResolver) (*bbpb.GerritChange, error) {
	p, err := url.Parse(clURL)
	if err != nil {
		return nil, errors.Annotate(err, "URL_TO_CHANGELIST").Err()
	}
	if !strings.HasSuffix(p.Hostname(), "-review.googlesource.com") {
		return nil, errors.New("only *-review.googlesource.com URLs are supported")
	}

	var toks []string
	if trimPath := strings.Trim(p.Path, "/"); len(trimPath) > 0 {
		toks = strings.Split(trimPath, "/")
	}

	if len(toks) == 0 {
		// https://<gerrit_host>/#/c/<change>
		// https://<gerrit_host>/#/c/<change>/<patchset>
		return nil, errors.Reason("old/empty gerrit URL: %q", clURL).Err()
	} else if toks[0] != "c" {
		return nil, errors.Reason("Unknown changelist URL format: %q", clURL).Err()
	}
	toks = toks[1:] // remove "c"

	// toks ==                 v --------------------------------v
	// https://<gerrit_host>/c/<change>
	// https://<gerrit_host>/c/<change>/<patchset>
	// https://<gerrit_host>/c/<project/path>/+/<change>
	// https://<gerrit_host>/c/<project/path>/+/<change>/<patchset>

	var projectToks []string
	var changePatchsetToks []string
	for i, tok := range toks {
		if tok == "+" {
			projectToks, changePatchsetToks = toks[:i], toks[i+1:]
			break
		}
	}

	if len(projectToks) == 0 {
		return nil, errors.Reason("gerrit URL missing project: %q", clURL).Err()
	}
	if len(changePatchsetToks) == 0 {
		return nil, errors.Reason("gerrit URL missing change/patchset: %q", clURL).Err()
	}

	ret := &bbpb.GerritChange{
		Host:    p.Hostname(),
		Project: strings.Join(projectToks, "/"),
	}
	ret.Change, err = strconv.ParseInt(changePatchsetToks[0], 10, 64)
	if err != nil {
		return nil, errors.Reason("gerrit URL parsing change %q from %q", changePatchsetToks[0], clURL).Err()
	}
	if len(changePatchsetToks) > 1 {
		ret.Patchset, err = strconv.ParseInt(changePatchsetToks[1], 10, 64)
		if err != nil {
			return nil, errors.Reason("gerrit URL parsing patchset %q from %q", changePatchsetToks[1], clURL).Err()
		}
	} else {
		ret.Patchset, err = resolvePatchset(ret.Host, ret.Change)
		if err != nil {
			return nil, errors.Annotate(
				err, "resolving patchset from Gerrit Url %q", clURL).Err()
		}
	}

	return ret, nil
}

func gerritResolver(ctx context.Context) patchsetResolver {
	return func(host string, change int64) (int64, error) {
		// TODO(iannucci): allow authentication for internal hosts.
		gc, err := gerritapi.NewRESTClient(http.DefaultClient, host, false)
		if err != nil {
			return 0, errors.Annotate(err, "creating new gerrit client").Err()
		}
		ci, err := gc.GetChange(ctx, &gerrit.GetChangeRequest{
			Number: change,
			Options: []gerrit.QueryOption{
				gerrit.QueryOption_CURRENT_REVISION,
			},
		})
		if grpc.Code(err) == codes.Unauthenticated {
			return 0, errors.Annotate(err,
				"Gerrit host %q requires authentication and no patchset was provided. "+
					"Please include the patchset you want in your URL (or add a patchset "+
					"`0` to ignore this).", host,
			).Err()
		}
		if err != nil {
			return 0, errors.Annotate(err, "GetChange").Err()
		}

		// There's only one.
		for _, rd := range ci.Revisions {
			return int64(rd.Number), nil
		}
		panic("impossible")
	}
}

func (c *cmdEditCl) validateFlags(ctx context.Context, positionals []string, _ subcommands.Env) (err error) {
	if c.remove && c.noImplicitClear {
		return errors.New("cannot specify both -remove and -no-implicit-clear")
	}

	c.gerritChange, err = parseCrChangeListURL(positionals[0], gerritResolver(ctx))
	return errors.Annotate(err, "invalid URL_TO_CHANGESET").Err()
}

func (c *cmdEditCl) execute(ctx context.Context, _ *http.Client, inJob *job.Definition) (out interface{}, err error) {
	return inJob, inJob.HighLevelEdit(func(je job.HighLevelEditor) {
		if c.remove {
			je.RemoveGerritChange(c.gerritChange)
		} else {
			if !c.noImplicitClear {
				je.ClearGerritChanges()
			}
			je.AddGerritChange(c.gerritChange)
		}

		// wipe out all the old properties
		je.Properties(map[string]string{
			"blamelist":            "",
			"buildbucket":          "",
			"issue":                "",
			"patch_gerrit_url":     "",
			"patch_issue":          "",
			"patch_project":        "",
			"patch_ref":            "",
			"patch_repository_url": "",
			"patch_set":            "",
			"patch_storage":        "",
			"patchset":             "",
			"repository":           "",
			"rietveld":             "",
		}, true)
	})
}

func (c *cmdEditCl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return c.doContextExecute(a, c, args, env)
}
