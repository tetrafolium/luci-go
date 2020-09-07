// Copyright 2018 The LUCI Authors.
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

package interpreter

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

// runs a script in an environment where 'custom' package uses the given loader.
func runScriptWithLoader(body string, l Loader) (logs []string, err error) {
	_, logs, err = runIntr(intrParams{
		stdlib: map[string]string{"builtins.star": body},
		custom: l,
	})
	return logs, err
}

func TestLoaders(t *testing.T) {
	t.Parallel()

	Convey("FileSystemLoader", t, func() {
		tmp, err := ioutil.TempDir("", "starlark")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tmp)

		loader := FileSystemLoader(tmp)

		put := func(path, body string) {
			path = filepath.Join(tmp, filepath.FromSlash(path))
			So(os.MkdirAll(filepath.Dir(path), 0700), ShouldBeNil)
			So(ioutil.WriteFile(path, []byte(body), 0600), ShouldBeNil)
		}

		Convey("Works", func() {
			put("1.star", `load("//a/b/c/2.star", _sym="sym"); sym = _sym`)
			put("a/b/c/2.star", "print('Hi')\nsym = 1")

			logs, err := runScriptWithLoader(`load("@custom//1.star", "sym")`, loader)
			So(err, ShouldBeNil)
			So(logs, ShouldResemble, []string{
				"[@custom//a/b/c/2.star:1] Hi",
			})
		})

		Convey("Missing module", func() {
			put("1.star", `load("//a/b/c/2.star", "sym")`)

			_, err := runScriptWithLoader(`load("@custom//1.star", "sym")`, loader)
			So(err, ShouldErrLike, "cannot load //a/b/c/2.star: no such module")
		})

		Convey("Outside the root", func() {
			_, err := runScriptWithLoader(`load("@custom//../1.star", "sym")`, loader)
			So(err, ShouldErrLike, "cannot load @custom//../1.star: outside the package root")
		})
	})

	Convey("MemoryLoader", t, func() {
		Convey("Works", func() {
			loader := MemoryLoader(map[string]string{
				"1.star":       `load("//a/b/c/2.star", _sym="sym"); sym = _sym`,
				"a/b/c/2.star": "print('Hi')\nsym = 1",
			})

			logs, err := runScriptWithLoader(`load("@custom//1.star", "sym")`, loader)
			So(err, ShouldBeNil)
			So(logs, ShouldResemble, []string{
				"[@custom//a/b/c/2.star:1] Hi",
			})
		})

		Convey("Missing module", func() {
			loader := MemoryLoader(map[string]string{
				"1.star": `load("//a/b/c/2.star", "sym")`,
			})

			_, err := runScriptWithLoader(`load("@custom//1.star", "sym")`, loader)
			So(err, ShouldErrLike, "cannot load //a/b/c/2.star: no such module")
		})
	})
}
