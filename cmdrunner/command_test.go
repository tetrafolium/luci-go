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

package cmdrunner

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/isolatedclient"
	"github.com/tetrafolium/luci-go/common/isolatedclient/isolatedfake"
	"github.com/tetrafolium/luci-go/common/system/environ"
	"github.com/tetrafolium/luci-go/common/testing/testfs"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReplaceParameters(t *testing.T) {
	t.Parallel()

	Convey("replaceParameters", t, func() {
		ctx := context.Background()

		Convey("test EXECUTABLE_SUFFIX", func() {
			arg, err := replaceParameters(ctx, "program${EXECUTABLE_SUFFIX}", "", "")
			So(err, ShouldBeNil)
			if runtime.GOOS == "windows" {
				So(arg, ShouldEqual, "program.exe")
			} else {
				So(arg, ShouldEqual, "program")
			}
		})

		Convey("test ISOLATED_OUTDIR", func() {
			arg, err := replaceParameters(ctx, "${ISOLATED_OUTDIR}/result.txt", "out", "")
			So(err, ShouldBeNil)
			So(arg, ShouldEqual, filepath.Join("out", "result.txt"))
		})

		Convey("test SWARMING_BOT_FILE", func() {
			arg, err := replaceParameters(ctx, "${SWARMING_BOT_FILE}/config", "", "cfgdir")
			So(err, ShouldBeNil)
			So(arg, ShouldEqual, filepath.Join("cfgdir", "config"))
		})
	})
}

func TestProcessCommand(t *testing.T) {
	t.Parallel()

	Convey("processCommand", t, func() {
		ctx := context.Background()
		args, err := processCommand(ctx, []string{
			"program${EXECUTABLE_SUFFIX}",
			"${ISOLATED_OUTDIR}/result.txt",
			"${SWARMING_BOT_FILE}/config",
		}, "out", "cfgdir")

		So(err, ShouldBeNil)

		executableSuffix := ""
		if runtime.GOOS == "windows" {
			executableSuffix = ".exe"
		}

		So(args, ShouldResemble, []string{
			"program" + executableSuffix,
			filepath.Join("out", "result.txt"),
			filepath.Join("cfgdir", "config"),
		})
	})
}

func TestGetCommandEnv(t *testing.T) {
	t.Parallel()
	originalEnvironSystem := environSystem
	defer func() {
		environSystem = originalEnvironSystem
	}()

	Convey("GetCommandEnv", t, func() {
		environSystem = func() environ.Env {
			return environ.New([]string{
				"C=foo",
				"D=bar",
				"E=baz",
				"PATH=/bin",
			})
		}

		Convey("simple case", func() {
			env, err := getCommandEnv(context.Background(), "/a", nil, "/b", environ.New([]string{
				"A=a",
				"B=",
				"C=",
				"E=${ISOLATED_OUTDIR}/eggs",
			}), map[string][]string{"D": {"foo"}}, "/spam", "")

			So(err, ShouldBeNil)

			_, ok := env.Get("B")
			So(ok, ShouldBeFalse)

			_, ok = env.Get("C")
			So(ok, ShouldBeFalse)

			if runtime.GOOS == "windows" {
				So(env.GetEmpty("D"), ShouldEqual, `\b\foo;bar`)
			} else {
				So(env.GetEmpty("D"), ShouldEqual, "/b/foo:bar")
			}

			So(env.GetEmpty("E"), ShouldEqual, string(filepath.Separator)+filepath.Join("spam", "eggs"))
		})

		Convey("cipdInfo", func() {
			env, err := getCommandEnv(context.Background(), "tmp", &cipdInfo{
				binaryPath: "cipddir/cipd",
				cacheDir:   ".cipd/cache",
			}, "", nil, nil, "", "")
			So(err, ShouldBeNil)

			expected := map[string]string{
				"C":              "foo",
				"CIPD_CACHE_DIR": ".cipd/cache",
				"D":              "bar",
				"E":              "baz",
				"PATH": strings.Join([]string{"cipddir", "/bin"},
					string(filepath.ListSeparator)),
				"TMPDIR": "tmp",
			}

			if runtime.GOOS == "windows" {
				expected["TMP"] = "tmp"
				expected["TEMP"] = "tmp"
			} else if runtime.GOOS == "darwin" {
				expected["MAC_CHROMIUM_TMPDIR"] = "tmp"
			}

			So(env.Map(), ShouldResemble, expected)
		})
	})
}

func TestRun(t *testing.T) {
	t.Parallel()
	Convey("TestRunCommand", t, func() {
		ctx := context.Background()

		Convey("simple", func() {
			exitcode, err := Run(ctx, []string{"go", "help"}, ".", environ.System(), time.Minute, time.Minute, false, false)

			So(exitcode, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		// TODO(tikuta): have test for error cases.
	})
}

func TestLinkOutputsToOutdir(t *testing.T) {
	t.Parallel()

	Convey("LinkOutputsToOutdir", t, func() {
		dir := t.TempDir()
		rundir := filepath.Join(dir, "rundir")

		So(testfs.Build(rundir, map[string]string{
			"a/b":   "ab",
			"a/c/d": "acd",
			"e":     "e",
		}), ShouldBeNil)

		outdir := filepath.Join(dir, "outdir")
		So(linkOutputsToOutdir(rundir, outdir, []string{
			filepath.Join("a", "b"),
			"e",
			// do not link a/c/d here.
		}), ShouldBeNil)

		layout, err := testfs.Collect(outdir)
		So(err, ShouldBeNil)
		So(layout, ShouldResemble, map[string]string{
			"a/b": "ab",
			"e":   "e",
		})
	})
}

func TestUploadThenDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey("UploadThenDelete", t, func() {
		dir := t.TempDir()
		server := isolatedfake.New()
		ts := httptest.NewServer(server)
		defer ts.Close()
		namespace := isolatedclient.DefaultNamespace
		client := isolatedclient.NewClient(ts.URL, isolatedclient.WithNamespace(namespace))

		So(testfs.Build(dir, map[string]string{
			"a/b":   "ab",
			"a/c/d": "acd",
		}), ShouldBeNil)

		Convey("UploadThenDelete", func() {
			digest, stats, err := uploadThenDelete(ctx, client, dir, "a")
			So(err, ShouldBeNil)
			So(digest, ShouldNotEqual, "")
			So(stats, ShouldNotBeNil)
			So(stats.Duration, ShouldNotBeNil)
			So(stats.ItemsCold, ShouldNotBeEmpty)
			So(stats.ItemsHot, ShouldBeEmpty)
		})

		Convey("Upload", func() {
			digest, stats, err := upload(ctx, client, dir, "a")
			So(err, ShouldBeNil)
			So(digest, ShouldNotEqual, "")
			So(stats, ShouldNotBeNil)
			So(stats.Duration, ShouldNotBeNil)
			So(stats.ItemsCold, ShouldNotBeEmpty)
			So(stats.ItemsHot, ShouldBeEmpty)
		})
	})
}
