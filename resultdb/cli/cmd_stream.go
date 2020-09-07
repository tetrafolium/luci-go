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

package cli

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"google.golang.org/grpc/metadata"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/cli"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/text"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/flag/stringmapflag"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/system/exitcode"
	"github.com/tetrafolium/luci-go/common/system/signals"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/lucictx"
	"github.com/tetrafolium/luci-go/server/auth/realms"

	"github.com/tetrafolium/luci-go/resultdb/internal/services/recorder"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
	"github.com/tetrafolium/luci-go/resultdb/sink"
)

var matchInvalidInvocationIDChars = regexp.MustCompile(`[^a-z0-9_\-:.]`)

func cmdStream(p Params) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `stream [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Run a given test command and upload the results to ResultDB",
		// TODO(crbug.com/1017288): add a link to ResultSink protocol doc
		LongDesc: text.Doc(`
			Run a given test command, continuously collect the results over IPC, and
			upload them to ResultDB. Either use the current invocation from
			LUCI_CONTEXT or create/finalize a new one. Example:
				rdb stream -new -realm chromium:public ./out/chrome/test/browser_tests
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &streamRun{vars: make(stringmapflag.Value)}
			r.baseCommandRun.RegisterGlobalFlags(p)
			r.Flags.BoolVar(&r.isNew, "new", false, text.Doc(`
				If true, create and use a new invocation for the test command.
				If false, use the current invocation, set in LUCI_CONTEXT.
			`))
			r.Flags.StringVar(&r.realm, "realm", "", text.Doc(`
				Realm to create the new invocation in. Required if -new is set,
				ignored otherwise.
				e.g. "chromium:public"
			`))
			r.Flags.StringVar(&r.testIDPrefix, "test-id-prefix", "", text.Doc(`
				Prefix to prepend to the test ID of every test result.
			`))
			r.Flags.Var(&r.vars, "var", text.Doc(`
				Variant to add to every test result in "key=value" format.
				If the test command adds a variant with the same key, the value given by
				this flag will get overridden.
			`))
			r.Flags.UintVar(&r.artChannelMaxLeases, "max-concurrent-artifact-uploads",
				sink.DefaultArtChannelMaxLeases, text.Doc(`
				The maximum number of goroutines uploading artifacts.
			`))
			r.Flags.UintVar(&r.trChannelMaxLeases, "max-concurrent-test-result-uploads",
				sink.DefaultTestResultChannelMaxLeases, text.Doc(`
				The maximum number of goroutines uploading test results.
			`))

			return r
		},
	}
}

type streamRun struct {
	baseCommandRun

	// flags
	isNew               bool
	realm               string
	testIDPrefix        string
	vars                stringmapflag.Value
	artChannelMaxLeases uint
	trChannelMaxLeases  uint

	// TODO(ddoman): add flags
	// - tag (invocation-tag)
	// - log-file

	invocation lucictx.ResultDBInvocation
}

func (r *streamRun) validate(ctx context.Context, args []string) (err error) {
	if len(args) == 0 {
		return errors.Reason("missing a test command to run").Err()
	}
	if err := pbutil.ValidateVariant(&pb.Variant{Def: r.vars}); err != nil {
		return errors.Annotate(err, "invalid variant").Err()
	}
	if r.realm != "" {
		if err := realms.ValidateRealmName(r.realm, realms.GlobalScope); err != nil {
			return errors.Annotate(err, "invalid realm").Err()
		}
	}
	return nil
}

func (r *streamRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	ctx := cli.GetContext(a, r, env)

	if err := r.validate(ctx, args); err != nil {
		return r.done(err)
	}

	loginMode := auth.OptionalLogin
	// login is required only if it creates a new invocation.
	if r.isNew {
		if r.realm == "" {
			return r.done(errors.Reason("-realm is required for new invocations").Err())
		}
		loginMode = auth.SilentLogin
	}
	if err := r.initClients(ctx, loginMode); err != nil {
		return r.done(err)
	}

	// if -new is passed, create a new invocation. If not, use the existing one set in
	// lucictx.
	if r.isNew {
		ninv, err := r.createInvocation(ctx, r.realm)
		if err != nil {
			return r.done(err)
		}
		r.invocation = ninv

		// Update lucictx with the new invocation.
		ctx = lucictx.SetResultDB(ctx, &lucictx.ResultDB{
			Hostname:          r.host,
			CurrentInvocation: &r.invocation,
		})
	} else {
		if r.resultdbCtx == nil {
			return r.done(errors.Reason("the environment does not have an existing invocation; use -new to create a new one").Err())
		}
		if err := r.validateCurrentInvocation(); err != nil {
			return r.done(err)
		}
		r.invocation = *r.resultdbCtx.CurrentInvocation
	}

	defer func() {
		// Finalize the invocation if it was created by -new.
		if r.isNew {
			if err := r.finalizeInvocation(ctx); err != nil {
				logging.Errorf(ctx, "failed to finalize the invocation: %s", err)
				ret = r.done(err)
			}
		}
	}()

	err := r.runTestCmd(ctx, args)
	ec, ok := exitcode.Get(err)
	if !ok {
		logging.Errorf(ctx, "rdb-stream: failed to run the test command: %s", err)
		return r.done(err)
	}
	logging.Infof(ctx, "rdb-stream: exiting with %d", ec)
	return ec
}

func (r *streamRun) runTestCmd(ctx context.Context, args []string) error {
	// Kill the subprocess if rdb-stream is asked to stop.
	// Subprocess exiting will unblock rdb-stream and it will stop soon.
	cmdCtx, cancelCmd := context.WithCancel(ctx)
	defer cancelCmd()
	defer signals.HandleInterrupt(func() {
		logging.Warningf(ctx, "Interrupt signal received; killing the subprocess")
		cancelCmd()
	})()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// TODO(ddoman): send the logs of SinkServer to --log-file

	cfg := sink.ServerConfig{
		Recorder:                   r.recorder,
		Invocation:                 r.invocation.Name,
		UpdateToken:                r.invocation.UpdateToken,
		TestIDPrefix:               r.testIDPrefix,
		BaseVariant:                &pb.Variant{Def: r.vars},
		ArtifactUploader:           &sink.ArtifactUploader{Client: r.http, Host: r.host},
		ArtChannelMaxLeases:        r.artChannelMaxLeases,
		TestResultChannelMaxLeases: r.trChannelMaxLeases,
	}
	return sink.Run(ctx, cfg, func(ctx context.Context, cfg sink.ServerConfig) error {
		exported, err := lucictx.Export(ctx)
		if err != nil {
			return err
		}
		defer func() {
			logging.Infof(ctx, "rdb-stream: the test process terminated")
			exported.Close()
		}()
		exported.SetInCmd(cmd)
		logging.Infof(ctx, "rdb-stream: starting the test command - %q", cmd.Args)
		if err := cmd.Start(); err != nil {
			return errors.Annotate(err, "cmd.start").Err()
		}
		return cmd.Wait()
	})
}

func (r *streamRun) createInvocation(ctx context.Context, realm string) (ret lucictx.ResultDBInvocation, err error) {
	invID, err := genInvID(ctx)
	if err != nil {
		return
	}

	md := metadata.MD{}
	resp, err := r.recorder.CreateInvocation(ctx, &pb.CreateInvocationRequest{
		InvocationId: invID,
		Invocation: &pb.Invocation{
			Realm: realm,
		},
	}, prpc.Header(&md))
	if err != nil {
		err = errors.Annotate(err, "failed to create an invocation").Err()
		return
	}
	tks := md.Get(recorder.UpdateTokenMetadataKey)
	if len(tks) == 0 {
		err = errors.Reason("Missing header: update-token").Err()
		return
	}

	ret = lucictx.ResultDBInvocation{Name: resp.Name, UpdateToken: tks[0]}
	fmt.Fprintf(os.Stderr, "rdb-stream: created invocation - https://ci.chromium.org/ui/inv/%s\n", invID)
	return
}

// finalizeInvocation finalizes the invocation.
func (r *streamRun) finalizeInvocation(ctx context.Context) error {
	ctx = metadata.AppendToOutgoingContext(
		ctx, recorder.UpdateTokenMetadataKey, r.invocation.UpdateToken)
	_, err := r.recorder.FinalizeInvocation(ctx, &pb.FinalizeInvocationRequest{
		Name: r.invocation.Name,
	})
	return err
}

// genInvID generates an invocation ID, made of the username, the current timestamp
// in a human-friendly format, and a random suffix.
//
// This can be used to generate a random invocation ID, but the creator and creation time
// can be easily found.
func genInvID(ctx context.Context) (string, error) {
	whoami, err := user.Current()
	if err != nil {
		return "", err
	}
	bytes := make([]byte, 8)
	if _, err := mathrand.Read(ctx, bytes); err != nil {
		return "", err
	}

	username := strings.ToLower(whoami.Username)
	username = matchInvalidInvocationIDChars.ReplaceAllString(username, "")

	suffix := strings.ToLower(fmt.Sprintf(
		"%s-%s", time.Now().UTC().Format("2006-01-02-15-04-00"),
		// Note: cannot use base64 because not all of its characters are allowed
		// in invocation IDs.
		hex.EncodeToString(bytes)))

	// An invocation ID can contain up to 100 ascii characters that conform to the regex,
	return fmt.Sprintf("u-%.*s-%s", 100-len(suffix), username, suffix), nil
}
