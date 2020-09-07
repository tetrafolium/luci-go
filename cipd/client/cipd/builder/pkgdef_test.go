// Copyright 2015 The LUCI Authors.
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

package builder

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tetrafolium/luci-go/cipd/client/cipd/fs"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadPackageDef(t *testing.T) {
	t.Parallel()

	Convey("LoadPackageDef empty works", t, func() {
		body := strings.NewReader(`{"package": "package/name"}`)
		def, err := LoadPackageDef(body, nil)
		So(err, ShouldBeNil)
		So(def, ShouldResemble, PackageDef{
			Package: "package/name",
			Root:    ".",
		})
	})

	Convey("LoadPackageDef works", t, func() {
		body := strings.NewReader(`{
			"package": "package/${var1}",
			"root": "../..",
			"install_mode": "copy",
			"data": [
				{
					"file": "some_file_${var1}"
				},
				{
					"file": "another_file_${var2}"
				},
				{
					"dir": "some/directory"
				},
				{
					"version_file": "some/path/version_${var1}.json"
				},
				{
					"dir": "another/${var2}",
					"exclude": [
						".*\\.pyc",
						"abc_${var2}_def"
					]
				}
			]
		}`)
		def, err := LoadPackageDef(body, map[string]string{
			"var1": "value1",
			"var2": "value2",
		})
		So(err, ShouldBeNil)
		So(def, ShouldResemble, PackageDef{
			Package:     "package/value1",
			Root:        "../..",
			InstallMode: "copy",
			Data: []PackageChunkDef{
				{
					File: "some_file_value1",
				},
				{
					File: "another_file_value2",
				},
				{
					Dir: "some/directory",
				},
				{
					VersionFile: "some/path/version_value1.json",
				},
				{
					Dir: "another/value2",
					Exclude: []string{
						".*\\.pyc",
						"abc_value2_def",
					},
				},
			},
		})
		So(def.VersionFile(), ShouldEqual, "some/path/version_value1.json")
	})

	Convey("LoadPackageDef not yaml", t, func() {
		body := strings.NewReader(`{ not yaml)`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef bad type", t, func() {
		body := strings.NewReader(`{"package": []}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef missing variable", t, func() {
		body := strings.NewReader(`{
			"package": "abd",
			"data": [{"file": "${missing_var}"}]
		}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef space in missing variable", t, func() {
		body := strings.NewReader(`{
			"package": "abd",
			"data": [{"file": "${missing var}"}]
		}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef bad package name", t, func() {
		body := strings.NewReader(`{"package": "not a valid name"}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef bad file section (no dir or file)", t, func() {
		body := strings.NewReader(`{
			"package": "package/name",
			"data": [
				{"exclude": []}
			]
		}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef bad file section (both dir and file)", t, func() {
		body := strings.NewReader(`{
			"package": "package/name",
			"data": [
				{"file": "abc", "dir": "def"}
			]
		}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef bad version_file", t, func() {
		body := strings.NewReader(`{
			"package": "package/name",
			"data": [
				{"version_file": "../some/path.json"}
			]
		}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})

	Convey("LoadPackageDef two version_file entries", t, func() {
		body := strings.NewReader(`{
			"package": "package/name",
			"data": [
				{"version_file": "some/path.json"},
				{"version_file": "some/path.json"}
			]
		}`)
		_, err := LoadPackageDef(body, nil)
		So(err, ShouldNotBeNil)
	})
}

func TestExclusion(t *testing.T) {
	t.Parallel()

	Convey("makeExclusionFilter works", t, func() {
		filter, err := makeExclusionFilter([]string{
			".*\\.pyc",
			".*/pip-.*-build/.*",
			"bin/activate",
			"lib/.*/site-packages/.*\\.dist-info/RECORD",
		})
		So(err, ShouldBeNil)
		So(filter, ShouldNotBeNil)

		// *.pyc filtering.
		So(filter(filepath.FromSlash("test.pyc")), ShouldBeTrue)
		So(filter(filepath.FromSlash("test.py")), ShouldBeFalse)
		So(filter(filepath.FromSlash("d/e/f/test.pyc")), ShouldBeTrue)
		So(filter(filepath.FromSlash("d/e/f/test.py")), ShouldBeFalse)

		// Subdir filtering.
		So(filter(filepath.FromSlash("x/pip-blah-build/d/e/f")), ShouldBeTrue)

		// Single file exclusion.
		So(filter(filepath.FromSlash("bin/activate")), ShouldBeTrue)
		So(filter(filepath.FromSlash("bin/activate2")), ShouldBeFalse)
		So(filter(filepath.FromSlash("d/bin/activate")), ShouldBeFalse)

		// More complicated regexp.
		p := "lib/python2.7/site-packages/coverage-3.7.1.dist-info/RECORD"
		So(filter(filepath.FromSlash(p)), ShouldBeTrue)
	})

	Convey("makeExclusionFilter bad regexp", t, func() {
		_, err := makeExclusionFilter([]string{"****"})
		So(err, ShouldNotBeNil)
	})
}

func TestFindFiles(t *testing.T) {
	t.Parallel()

	Convey("Given a temp directory", t, func() {
		tempDir := mkTempDir()

		mkF := func(path string) { writeFile(tempDir, path, "", 0666) }
		mkD := func(path string) { mkDir(tempDir, path) }
		mkL := func(path, target string) { writeSymlink(tempDir, path, target) }

		Convey("FindFiles works", func() {
			mkF("ENV/abc.py")
			mkF("ENV/abc.pyc") // excluded via "exclude: '.*\.pyc'"
			mkF("ENV/abc.pyo")
			mkF("ENV/dir/def.py")
			mkD("ENV/empty")      // will be skipped
			mkF("ENV/exclude_me") // excluded via "exclude: 'exclude_me'"

			// Symlinks do not work on Windows.
			if runtime.GOOS != "windows" {
				mkL("ENV/abs_link", filepath.Dir(tempDir))
				mkL("ENV/rel_link", "abc.py")
				mkL("ENV/abs_in_root", filepath.Join(tempDir, "ENV", "dir", "def.py"))
			}

			mkF("infra/xyz.py")
			mkF("infra/zzz.pyo")
			mkF("infra/excluded.py")
			mkF("infra/excluded_dir/a")
			mkF("infra/excluded_dir/b")

			mkF("file1.py")
			mkF("dir/file2.py")

			mkF("garbage/a")
			mkF("garbage/b")

			assertFiles := func(pkgDef PackageDef, cwd string) {
				files, err := pkgDef.FindFiles(cwd)
				So(err, ShouldBeNil)
				names := make([]string, len(files))
				byName := make(map[string]fs.File, len(files))
				for i, f := range files {
					names[i] = f.Name()
					byName[f.Name()] = f
				}

				if runtime.GOOS == "windows" {
					So(names, ShouldResemble, []string{
						"ENV/abc.py",
						"ENV/abc.pyo",
						"ENV/dir/def.py",
						"dir/file2.py",
						"file1.py",
						"infra/xyz.py",
					})
				} else {
					So(names, ShouldResemble, []string{
						"ENV/abc.py",
						"ENV/abc.pyo",
						"ENV/abs_in_root",
						"ENV/abs_link",
						"ENV/dir/def.py",
						"ENV/rel_link",
						"dir/file2.py",
						"file1.py",
						"infra/xyz.py",
					})
					// Separately check symlinks.
					ensureSymlinkTarget(byName["ENV/abs_in_root"], "dir/def.py")
					ensureSymlinkTarget(byName["ENV/abs_link"], filepath.ToSlash(filepath.Dir(tempDir)))
					ensureSymlinkTarget(byName["ENV/rel_link"], "abc.py")
				}
			}

			pkgDef := PackageDef{
				Package: "test",
				Data: []PackageChunkDef{
					{
						Dir:     "ENV",
						Exclude: []string{".*\\.pyc", "exclude_me"},
					},
					{
						Dir: "infra",
						Exclude: []string{
							".*\\.pyo",
							"excluded.py",
							"excluded_dir",
						},
					},
					{File: "file1.py"},
					{File: "dir/file2.py"},
					// Will be "deduplicated", because already matched by first entry.
					{File: "ENV/abc.py"},
				},
			}

			Convey("with relative root", func() {
				pkgDef.Root = "../../"

				assertFiles(pkgDef, filepath.Join(tempDir, "a", "b"))
			})

			Convey("with absolute root", func() {
				pkgDef.Root = tempDir

				someOtherTmpDir := mkTempDir()
				assertFiles(pkgDef, someOtherTmpDir)
			})

		})

	})
}

////////////////////////////////////////////////////////////////////////////////

func mkTempDir() string {
	tempDir, err := ioutil.TempDir("", "cipd_test")
	So(err, ShouldBeNil)
	Reset(func() { os.RemoveAll(tempDir) })
	return tempDir
}

func mkDir(root string, path string) {
	abs := filepath.Join(root, filepath.FromSlash(path))
	err := os.MkdirAll(abs, 0777)
	if err != nil {
		panic("Failed to create a directory under temp directory")
	}
}

func writeFile(root string, path string, data string, mode os.FileMode) {
	abs := filepath.Join(root, filepath.FromSlash(path))
	os.MkdirAll(filepath.Dir(abs), 0777)
	err := ioutil.WriteFile(abs, []byte(data), mode)
	if err != nil {
		panic("Failed to write a temp file")
	}
}

func writeSymlink(root string, path string, target string) {
	abs := filepath.Join(root, filepath.FromSlash(path))
	os.MkdirAll(filepath.Dir(abs), 0777)
	err := os.Symlink(target, abs)
	if err != nil {
		panic("Failed to create symlink")
	}
}

func ensureSymlinkTarget(file fs.File, target string) {
	So(file.Symlink(), ShouldBeTrue)
	discoveredTarget, err := file.SymlinkTarget()
	So(err, ShouldBeNil)
	So(discoveredTarget, ShouldEqual, target)
}
