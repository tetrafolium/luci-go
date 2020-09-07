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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/auth/client/authcli"
	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
	"github.com/tetrafolium/luci-go/cipd/version"
	"github.com/tetrafolium/luci-go/common/data/text"
	"github.com/tetrafolium/luci-go/common/lhttp"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/grpc/prpc"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
)

var expectedCodeRPCOption = prpc.ExpectedCode(
	codes.InvalidArgument, codes.NotFound, codes.PermissionDenied)

func doc(doc string) string {
	return text.Doc(doc)
}

// baseCommandRun provides common command run functionality.
// All bb subcommands must embed it directly or indirectly.
type baseCommandRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	host      string
	json      bool
	noColor   bool

	httpClient *http.Client
	client     pb.BuildsClient
}

func (r *baseCommandRun) RegisterDefaultFlags(p Params) {
	r.Flags.StringVar(&r.host, "host", p.DefaultBuildbucketHost, doc(`
		Host for the buildbucket service instance.
	`))
	r.Flags.BoolVar(&r.noColor, "nocolor", false, doc(`
		Strip ANSI color codes in the output.
	`))
	r.authFlags.Register(&r.Flags, p.Auth)
}

func (r *baseCommandRun) RegisterJSONFlag() {
	r.Flags.BoolVar(&r.json, "json", false, doc(`
		Print objects in JSON format, one after another (not an array).

		Intended for "jq" tool. If using bb from scripts, consider "batch" subcommand.
	`))
}

// initClients validates -host flag and initializes r.httpClient and r.client.
func (r *baseCommandRun) initClients(ctx context.Context) error {
	// Create HTTP Client.
	authOpts, err := r.authFlags.Options()
	if err != nil {
		return err
	}
	r.httpClient, err = auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	switch {
	case err == auth.ErrLoginRequired:
		return errors.New("Login required: run `bb auth-login`")
	case err != nil:
		return err
	}

	// Validate -host
	if r.host == "" {
		return fmt.Errorf("a host for the buildbucket service must be provided")
	}
	if strings.ContainsRune(r.host, '/') {
		return fmt.Errorf("invalid host %q", r.host)
	}

	// Create Buildbucket client.
	rpcOpts := prpc.DefaultOptions()
	rpcOpts.Insecure = lhttp.IsLocalHost(r.host)
	info, err := version.GetCurrentVersion()
	if err != nil {
		return err
	}
	rpcOpts.UserAgent = fmt.Sprintf("buildbucket CLI, instanceID=%q", info.InstanceID)
	// TODO(crbug/1016443): remove AcceptContentSubtype defaulting into binary
	// protobuf encoding once Buildbucket server becomes faster.
	rpcOpts.AcceptContentSubtype = "json"
	// Per crbug/1030156, default of 5 retries isn't good enough.
	rpcOpts.Retry = func() retry.Iterator {
		return &retry.ExponentialBackoff{
			Limited: retry.Limited{
				Delay:   time.Second,
				Retries: 10,
			},
			Multiplier: 2.0,
			MaxDelay:   5 * time.Minute,
		}
	}
	r.client = pb.NewBuildsPRPCClient(&prpc.Client{
		C:       r.httpClient,
		Host:    r.host,
		Options: rpcOpts,
	})
	return nil
}

func (r *baseCommandRun) done(ctx context.Context, err error) int {
	if err != nil {
		logging.Errorf(ctx, "%s", err)
		return 1
	}
	return 0
}

// retrieveBuildID converts a build string into a build id.
// May make a GetBuild RPC.
func (r *baseCommandRun) retrieveBuildID(ctx context.Context, build string) (int64, error) {
	getBuild, err := protoutil.ParseGetBuildRequest(build)
	if err != nil {
		return 0, err
	}

	if getBuild.Id != 0 {
		return getBuild.Id, nil
	}

	res, err := r.client.GetBuild(ctx, getBuild)
	if err != nil {
		return 0, err
	}
	return res.Id, nil
}
