// Copyright 2017 The LUCI Authors.
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

package vpython

import (
	"context"
	"os"
	"os/exec"
	"os/signal"

	"github.com/tetrafolium/luci-go/vpython/python"
	"github.com/tetrafolium/luci-go/vpython/venv"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/system/environ"
	"github.com/tetrafolium/luci-go/common/system/exitcode"
)

// systemSpecificLaunch launches the process described by "cmd" while ensuring
// that the VirtualEnv lock is held throughout its duration (best effort).
//
// On Windows, we don't forward signals. Forwarding signals on Windows is
// nuanced. For now, we won't, since sending them via Python is similarly
// nuanced and not commonly done.
//
// For more discussion, see:
// https://github.com/golang/go/issues/6720
//
// On Windows, we launch it as a child process and interpret any signal that we
// receive as terminal, cancelling the child.
func systemSpecificLaunch(c context.Context, ve *venv.Env, cl *python.CommandLine, env environ.Env, dir string) error {
	return Exec(c, ve.Interpreter(), cl, env, dir, nil)
}

func execImpl(c context.Context, argv []string, env environ.Env, dir string, setupFn func() error) error {
	cmd := exec.Cmd{
		Path:   argv[0],
		Args:   argv,
		Env:    env.Sorted(),
		Dir:    dir,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	// At this point, ANY ERROR will be fatal (panic). We assume that each
	// operation may permanently alter our runtime environment.
	if setupFn != nil {
		if err := setupFn(); err != nil {
			panic(err)
		}
	}

	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		logging.Debugf(c, "os.Interrupt recieved, restoring signal handler.")
		signal.Stop(ch)
		// Due to the nature of os.Interrupt (either CTRL_C_EVENT or
		// CTRL_BREAK_EVENT), they're sent to the entire process group. Since we
		// haven't created a separate group for `cmd`, we don't need to relay the
		// signal (since `cmd` would have gotten it as well).
	}()

	err := cmd.Run()
	if rc, has := exitcode.Get(err); has {
		// The process had an exit code (includes err==nil, 0).
		logging.Debugf(c, "Python subprocess has terminated: %v", err)
		os.Exit(rc)
		panic("must not return")
	}
	panic(err)
}
