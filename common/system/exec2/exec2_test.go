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

package exec2

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/tetrafolium/luci-go/common/system/environ"
)

func build(src, tmpdir string) (string, error) {
	binary := filepath.Join(tmpdir, "exe.exe")
	cmd := exec.Command("go", "build", "-o", binary, src)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return binary, nil
}

func TestExec(t *testing.T) {
	t.Parallel()

	Convey("TestExec", t, func() {
		ctx := context.Background()

		tmpdir, err := ioutil.TempDir("", "test")
		So(err, ShouldBeNil)
		defer func() {
			So(os.RemoveAll(tmpdir), ShouldBeNil)
		}()

		errCh := make(chan error, 1)

		Convey("exit", func() {
			testBinary, err := build(filepath.Join("testdata", "exit.go"), tmpdir)
			So(err, ShouldBeNil)

			Convey("exit 0", func() {
				cmd := CommandContext(ctx, testBinary)
				So(cmd.Start(), ShouldBeNil)

				So(cmd.Wait(), ShouldBeNil)

				So(cmd.ProcessState.ExitCode(), ShouldEqual, 0)
			})

			Convey("exit 42", func() {
				cmd := CommandContext(ctx, testBinary, "42")
				So(cmd.Start(), ShouldBeNil)

				So(cmd.Wait(), ShouldBeError, "exit status 42")

				So(cmd.ProcessState.ExitCode(), ShouldEqual, 42)
			})
		})

		Convey("timeout", func() {
			testBinary, err := build(filepath.Join("testdata", "timeout.go"), tmpdir)
			So(err, ShouldBeNil)

			cmd := CommandContext(ctx, testBinary)
			rc, err := cmd.StdoutPipe()
			So(err, ShouldBeNil)

			So(cmd.Start(), ShouldBeNil)

			expected := []byte("I'm alive!")
			buf := make([]byte, len(expected))
			n, err := rc.Read(buf)
			So(err, ShouldBeNil)
			So(n, ShouldEqual, len(expected))
			So(buf, ShouldResemble, expected)

			So(rc.Close(), ShouldBeNil)

			go func() {
				errCh <- cmd.Wait()
			}()

			select {
			case err := <-errCh:
				Print(err)
				So("should not reach here", ShouldBeNil)
			case <-time.After(time.Millisecond):
			}

			So(cmd.Terminate(), ShouldBeNil)

			select {
			case err := <-errCh:
				if runtime.GOOS == "windows" {
					So(err, ShouldBeError, "exit status 2")
				} else {
					So(err, ShouldBeError, "signal: terminated")
				}
			case <-time.After(time.Minute):
				Print(err)
				So("should not reach here", ShouldBeNil)
			}

			if runtime.GOOS == "windows" {
				So(cmd.ProcessState.ExitCode(), ShouldEqual, 2)
			} else {
				So(cmd.ProcessState.ExitCode(), ShouldEqual, -1)
			}
		})

		Convey("context timeout", func() {
			testBinary, err := build(filepath.Join("testdata", "timeout.go"), tmpdir)
			So(err, ShouldBeNil)

			if runtime.GOOS == "windows" {
				// TODO(tikuta): support context timeout on windows
				return
			}

			ctx, cancel := context.WithTimeout(ctx, time.Millisecond)
			defer cancel()

			cmd := CommandContext(ctx, testBinary)

			So(cmd.Start(), ShouldBeNil)

			So(cmd.Wait(), ShouldBeError, "signal: killed")

			So(cmd.ProcessState.ExitCode(), ShouldEqual, -1)
		})

	})
}

func TestSetEnv(t *testing.T) {
	t.Parallel()

	Convey("TestSetEnv", t, func() {
		ctx := context.Background()

		tmpdir, err := ioutil.TempDir("", "test")
		So(err, ShouldBeNil)
		defer func() {
			So(os.RemoveAll(tmpdir), ShouldBeNil)
		}()

		testBinary, err := build(filepath.Join("testdata", "env.go"), tmpdir)
		So(err, ShouldBeNil)

		cmd := CommandContext(ctx, testBinary)
		env := environ.System()
		env.Set("envvar", "envvar")
		cmd.Env = env.Sorted()

		So(cmd.Start(), ShouldBeNil)

		So(cmd.Wait(), ShouldBeNil)

		So(cmd.ProcessState.ExitCode(), ShouldEqual, 0)
	})
}
