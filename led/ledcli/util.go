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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/maruel/subcommands"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/auth/client/authcli"
	"github.com/tetrafolium/luci-go/common/cli"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	job "github.com/tetrafolium/luci-go/led/job"
)

// TODO(iannucci): the 'subcommands' library is a mess, use something better.

type command interface {
	subcommands.CommandRun

	initFlags(opts cmdBaseOptions)

	jobInput() bool
	positionalRange() (min, max int)

	validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error
	execute(ctx context.Context, authClient *http.Client, inJob *job.Definition) (output interface{}, err error)
}

type cmdBaseOptions struct {
	authOpts       auth.Options
	kitchenSupport job.KitchenSupport
}

type cmdBase struct {
	subcommands.CommandRunBase

	logFlags  logging.Config
	authFlags authcli.Flags

	kitchenSupport job.KitchenSupport

	authenticator *auth.Authenticator
}

func (c *cmdBase) initFlags(opts cmdBaseOptions) {
	c.kitchenSupport = opts.kitchenSupport
	c.logFlags.Level = logging.Info
	c.logFlags.AddFlags(&c.Flags)
	c.authFlags.Register(&c.Flags, opts.authOpts)
}

func readJobDefinition(ctx context.Context) (*job.Definition, error) {
	readErr := make(chan error)

	jd := &job.Definition{}
	go func() {
		defer close(readErr)
		readErr <- jsonpb.Unmarshal(os.Stdin, jd)
	}()

	var err error
	select {
	case err = <-readErr:
		// we read it before the timeout
	case <-clock.After(ctx, time.Second):
		logging.Warningf(ctx, "waiting for JobDefinition on stdin...")
		err = <-readErr
	}

	return jd, errors.Annotate(err, "decoding job Definition").Err()
}

func (c *cmdBase) doContextExecute(a subcommands.Application, cmd command, args []string, env subcommands.Env) int {
	ctx := c.logFlags.Set(cli.GetContext(a, cmd, env))
	authOpts, err := c.authFlags.Options()
	if err != nil {
		logging.Errorf(ctx, "bad auth arguments: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}
	c.authenticator = auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts)
	authClient, err := c.authenticator.Client()
	if err == auth.ErrLoginRequired {
		fmt.Fprintln(os.Stderr, "Login required: run `led auth-login`.")
		return 1
	}

	//positional
	min, max := cmd.positionalRange()
	if len(args) < min {
		logging.Errorf(ctx, "expected at least %d positional arguments, got %d", min, len(args))
		c.GetFlags().Usage()
		return 1
	}
	if len(args) > max {
		logging.Errorf(ctx, "expected at most %d positional arguments, got %d", max, len(args))
		c.GetFlags().Usage()
		return 1
	}

	if err = cmd.validateFlags(ctx, args, env); err != nil {
		logging.Errorf(ctx, "bad arguments: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}

	var inJob *job.Definition
	if cmd.jobInput() {
		if inJob, err = readJobDefinition(ctx); err != nil {
			errors.Log(ctx, err)
			return 1
		}
	}

	output, err := cmd.execute(ctx, authClient, inJob)
	if err != nil {
		errors.Log(ctx, err)
		return 1
	}

	if output != nil {
		switch x := output.(type) {
		case proto.Message:
			err = (&jsonpb.Marshaler{
				OrigName: true,
				Indent:   "  ",
			}).Marshal(os.Stdout, x)

		default:
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			err = enc.Encode(output)
		}
		if err != nil {
			errors.Log(ctx, errors.Annotate(err, "encoding output").Err())
			return 1
		}
	}

	return 0
}

func pingHost(host string) error {
	rsp, err := http.Get("https://" + host)
	if err != nil {
		return errors.Annotate(err, "%q", host).Err()
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		return errors.Reason("%q: bad status %d", host, rsp.StatusCode).Err()
	}
	return nil
}
