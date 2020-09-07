// Copyright 2016 The LUCI Authors.
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
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"google.golang.org/genproto/protobuf/field_mask"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
	"github.com/tetrafolium/luci-go/common/cli"

	structpb "github.com/golang/protobuf/ptypes/struct"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
)

func cmdAdd(p Params) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `add [flags] [BUILDER [[BUILDER...]]`,
		ShortDesc: "add builds",
		LongDesc: doc(`
			Add a build for each BUILDER argument.

			A BUILDER must have format "<project>/<bucket>/<builder>", for
			example "chromium/try/linux-rel".
			If no builders were specified on the command line, they are read
			from stdin.

			Example: add linux-rel and mac-rel builds to chromium/ci bucket using Shell expansion.
				bb add chromium/ci/{linux-rel,mac-rel}
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &addRun{}
			r.RegisterDefaultFlags(p)

			r.clsFlag.Register(&r.Flags, doc(`
				CL URL as input for the builds. Can be specified multiple times.

				Example: add a linux-rel tryjob for CL 1539021
					bb add -cl https://chromium-review.googlesource.com/c/infra/luci/luci-go/+/1539021/1 chromium/try/linux-rel
			`))
			r.commitFlag.Register(&r.Flags, doc(`
				Commit URL as input to the builds.

				Example: build a specific revision
					bb add -commit https://chromium.googlesource.com/chromium/src/+/7dab11d0e282bfa1d6f65cc52195f9602921d5b9 chromium/ci/linux-rel

				Example: build latest chromium/src revision
					bb add -commit https://chromium.googlesource.com/chromium/src/+/master chromium/ci/linux-rel
			`))
			r.Flags.StringVar(&r.ref, "ref", "refs/heads/master", "Git ref for the -commit that specifies a commit hash.")
			r.tagsFlag.Register(&r.Flags, doc(`
				Build tags. Can be specified multiple times.

				Example: add a build with tags "a:1" and "b:2".
					bb add -t a:1 -t b:2 chromium/try/linux-rel
			`))
			r.Flags.BoolVar(&r.experimental, "exp", false, doc(`
				Mark the builds as experimental
			`))
			r.Flags.Var(PropertiesFlag(&r.properties), "p", doc(`
				Input properties for the build.

				If a flag value starts with @, properties are read from the JSON file at the
				path that follows @. Example:
					bb add -p @my_properties.json chromium/try/linux-rel
				This form can be used only in the first flag value.

				Otherwise, a flag value must have name=value form.
				If the property value is valid JSON, then it is parsed as JSON;
				otherwise treated as a string. Example:
					bb add -p foo=1 -p 'bar={"a": 2}' chromium/try/linux-rel
				Different property names can be specified multiple times.
			`))
			r.Flags.BoolVar(&r.canary, "canary", false, doc(`
				Force the build to use canary infrastructure.
			`))
			r.Flags.BoolVar(&r.noCanary, "nocanary", false, doc(`
				Force the build to NOT use canary infrastructure.
			`))
			r.Flags.StringVar(&r.swarmingParentRunID, "swarming-parent-run-id", "", doc(`
				Establish parent->child relationship between provided swarming task (parent)
				and the build to be triggered (child).

				Provided value must be an ID of the swarming task sharing the same
				swarming server as the build being created. If parent task completes
				before the newly created build does, then swarming server will
				forcefully terminate the build.

				This makes the child build lifetime bounded by the lifetime of the given swarming task.
			`))
			return r
		},
	}
}

type addRun struct {
	printRun
	clsFlag
	commitFlag
	tagsFlag

	ref                 string
	experimental        bool
	canary, noCanary    bool
	properties          structpb.Struct
	swarmingParentRunID string
}

func (r *addRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if r.canary && r.noCanary {
		fmt.Fprintf(os.Stderr, "-canary and -nocanary are mutually exclusive\n")
		return 1
	}

	ctx := cli.GetContext(a, r, env)
	if err := r.initClients(ctx); err != nil {
		return r.done(ctx, err)
	}

	baseReq, err := r.prepareBaseRequest(ctx)
	if err != nil {
		return r.done(ctx, err)
	}

	i := int32(0)
	return r.PrintAndDone(ctx, args, argOrder, func(ctx context.Context, builder string) (*pb.Build, error) {
		req := proto.Clone(baseReq).(*pb.ScheduleBuildRequest)

		// PrintAndDone callback is executed concurrently.
		req.RequestId += fmt.Sprintf("-%d", atomic.AddInt32(&i, 1))

		var err error
		req.Builder, err = protoutil.ParseBuilderID(builder)
		if err != nil {
			return nil, err
		}
		return r.client.ScheduleBuild(ctx, req, expectedCodeRPCOption)
	})
}

func (r *addRun) prepareBaseRequest(ctx context.Context) (*pb.ScheduleBuildRequest, error) {
	ret := &pb.ScheduleBuildRequest{
		RequestId:  uuid.New().String(),
		Tags:       r.Tags(),
		Fields:     &field_mask.FieldMask{Paths: []string{"*"}},
		Properties: &r.properties,
		Swarming:   &pb.ScheduleBuildRequest_Swarming{ParentRunId: r.swarmingParentRunID},
	}

	switch {
	case r.canary:
		ret.Canary = pb.Trinary_YES
	case r.noCanary:
		ret.Canary = pb.Trinary_NO
	}

	if r.experimental {
		ret.Experimental = pb.Trinary_YES
	}

	var err error
	if ret.GerritChanges, err = r.retrieveCLs(ctx, r.httpClient, !kRequirePatchset); err != nil {
		return nil, err
	}

	if ret.GitilesCommit, err = r.retrieveCommit(ctx, r.httpClient); err != nil {
		return nil, err
	}
	if ret.GitilesCommit != nil && ret.GitilesCommit.Ref == "" {
		ret.GitilesCommit.Ref = r.ref
	}

	return ret, nil
}
