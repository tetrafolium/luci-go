// Copyright 2019 The LUCI Authors.
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

package invoke

import (
	"bytes"
	"context"
	"os/exec"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/luciexe"

	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"
)

// Subprocess represents a running luciexe.
type Subprocess struct {
	Step        *bbpb.Step
	collectPath string

	cmd *exec.Cmd

	closeChannels chan<- struct{}
	allClosed     <-chan error

	waitOnce sync.Once
	build    *bbpb.Build
	err      error
}

// Start launches a binary implementing the luciexe protocol and returns
// immediately with a *Subprocess.
//
// Args:
//  * ctx will be used for deadlines/cancellation of the started luciexe.
//  * luciexeArgs[0] must be the full absolute path to the luciexe binary.
//  * input must be the Build message you wish to pass to the luciexe binary.
//  * opts is optional (may be nil to take all defaults)
//
// Callers MUST call Wait and/or cancel the context or this will leak handles
// for the process' stdout/stderr.
//
// This assumes that the current process is already operating within a "host
// application" environment. See "github.com/tetrafolium/luci-go/luciexe" for details.
//
// The caller SHOULD immediately take Subprocess.Step, append it to the current
// Build state, and send that (e.g. using `exe.BuildSender`). Otherwise this
// luciexe's steps will not show up in the Build.
func Start(ctx context.Context, luciexeArgs []string, input *bbpb.Build, opts *Options) (*Subprocess, error) {
	inputData, err := proto.Marshal(input)
	if err != nil {
		return nil, errors.Annotate(err, "marshalling input Build").Err()
	}

	launchOpts, _, err := opts.rationalize(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "normalizing options").Err()
	}

	closeChannels := make(chan struct{})
	allClosed := make(chan error)
	go func() {
		select {
		case <-ctx.Done():
		case <-closeChannels:
		}
		err := errors.NewLazyMultiError(2)
		err.Assign(0, errors.Annotate(launchOpts.stdout.Close(), "closing stdout").Err())
		err.Assign(1, errors.Annotate(launchOpts.stderr.Close(), "closing stderr").Err())
		allClosed <- err.Get()
	}()

	args := make([]string, 0, len(luciexeArgs)+len(launchOpts.args)-1)
	args = append(args, luciexeArgs[1:]...)
	args = append(args, launchOpts.args...)

	cmd := exec.CommandContext(ctx, luciexeArgs[0], args...)
	cmd.Env = launchOpts.env.Sorted()
	cmd.Dir = launchOpts.workDir
	cmd.Stdin = bytes.NewBuffer(inputData)
	cmd.Stdout = launchOpts.stdout
	cmd.Stderr = launchOpts.stderr
	if err := cmd.Start(); err != nil {
		// clean up stdout/stderr
		close(closeChannels)
		<-allClosed
		return nil, errors.Annotate(err, "launching luciexe").Err()
	}

	return &Subprocess{
		Step:        launchOpts.step,
		collectPath: launchOpts.collectPath,
		cmd:         cmd,

		closeChannels: closeChannels,
		allClosed:     allClosed,
	}, nil
}

// Wait waits for the subprocess to terminate.
//
// If Options.CollectOutput (default: false) was specified, this will return the
// final Build message, as reported by the luciexe.
//
// If you wish to cancel the subprocess (e.g. due to a timeout or deadline),
// make sure to pass a cancelable/deadline context to Start().
//
// Calling this multiple times is OK; it will return the same values every time.
func (s *Subprocess) Wait() (*bbpb.Build, error) {
	s.waitOnce.Do(func() {
		// No matter what, we want to close stdout/stderr; if none of the other
		// return values have set `err`, it will be set to the result of closing
		// stdout/stderr.
		defer func() {
			close(s.closeChannels)
			if closeErr := <-s.allClosed; s.err == nil {
				s.err = closeErr
			}
		}()

		if s.err = s.cmd.Wait(); s.err != nil {
			s.err = errors.Annotate(s.err, "waiting for luciexe").Err()
			return
		}
		s.build, s.err = luciexe.ReadBuildFile(s.collectPath)
	})
	return s.build, s.err
}
