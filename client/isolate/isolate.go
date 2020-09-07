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

package isolate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/tetrafolium/luci-go/client/archiver"
	"github.com/tetrafolium/luci-go/common/flag/stringmapflag"
	"github.com/tetrafolium/luci-go/common/isolated"
	"github.com/tetrafolium/luci-go/common/isolatedclient"
)

// IsolatedGenJSONVersion is used in the batcharchive json format.
//
// TODO(tandrii): Migrate to batch_archive.go.
const IsolatedGenJSONVersion = 1

// ValidVariable is the regexp of valid isolate variable name.
const ValidVariable = "[A-Za-z_][A-Za-z_0-9]*"

var validVariableMatcher = regexp.MustCompile(ValidVariable)
var variableSubstitutionMatcher = regexp.MustCompile("<\\(" + ValidVariable + "\\)")

// IsValidVariable returns true if the variable is a valid symbol name.
func IsValidVariable(variable string) bool {
	return validVariableMatcher.MatchString(variable)
}

// Tree to be isolated.
type Tree struct {
	Cwd  string
	Opts ArchiveOptions
}

// ArchiveOptions for archiving trees.
type ArchiveOptions struct {
	Isolate                    string              `json:"isolate"`
	Isolated                   string              `json:"isolated"`
	IgnoredPathFilterRe        string              `json:"ignored_path_filter_re"`
	PathVariables              stringmapflag.Value `json:"path_variables"`
	ConfigVariables            stringmapflag.Value `json:"config_variables"`
	AllowCommandAndRelativeCWD bool                `json:"allow_command_and_relative_cwd"`
	AllowMissingFileDir        bool                `json:"allow_missing_file_dir"`
}

// Init initializes with non-nil values.
func (a *ArchiveOptions) Init() {
	a.PathVariables = map[string]string{}
	if runtime.GOOS == "windows" {
		a.PathVariables["EXECUTABLE_SUFFIX"] = ".exe"
	} else {
		a.PathVariables["EXECUTABLE_SUFFIX"] = ""
	}
	a.ConfigVariables = map[string]string{}
}

func genExtensionsRegex(exts ...string) string {
	if len(exts) == 0 {
		return ""
	}
	var res []string
	for _, e := range exts {
		res = append(res, `(\.`+e+`)`)
	}
	return "((" + strings.Join(res, "|") + ")$)"
}

func genDirectoriesRegex(dirs ...string) string {
	if len(dirs) == 0 {
		return ""
	}
	var res []string
	for _, d := range dirs {
		res = append(res, "("+d+")")
	}
	// #Backslashes: https://stackoverflow.com/a/4025505/12003165
	return `((^|[\\/])(` + strings.Join(res, "|") + `)([\\/]|$))`
}

// PostProcess post-processes the flags to fix any compatibility issue.
func (a *ArchiveOptions) PostProcess(cwd string) {
	if a.IgnoredPathFilterRe == "" {
		// Set default ignored paths regexp
		// .swp are vim files
		a.IgnoredPathFilterRe = genExtensionsRegex("pyc", "swp") + "|" + genDirectoriesRegex(`\.git`, `\.hg`, `\.svn`)
	}
	if !filepath.IsAbs(a.Isolate) {
		a.Isolate = filepath.Join(cwd, a.Isolate)
	}
	a.Isolate = filepath.Clean(a.Isolate)

	if !filepath.IsAbs(a.Isolated) {
		a.Isolated = filepath.Join(cwd, a.Isolated)
	}
	a.Isolated = filepath.Clean(a.Isolated)

	for k, v := range a.PathVariables {
		// This is due to a Windows + GYP specific issue, where double-quoted paths
		// would get mangled in a way that cannot be resolved unless a space is
		// injected.
		a.PathVariables[k] = strings.TrimSpace(v)
	}
}

// ReplaceVariables replaces any occurrences of '<(FOO)' in 'str' with the
// corresponding variable from 'opts'.
//
// If any substitution refers to a variable that is missing, the returned error will
// refer to the first such variable. In the case of errors, the returned string will
// still contain a valid result for any non-missing substitutions.
func ReplaceVariables(str string, opts *ArchiveOptions) (string, error) {
	var err error
	subst := variableSubstitutionMatcher.ReplaceAllStringFunc(str,
		func(match string) string {
			varName := match[2 : len(match)-1]
			if v, ok := opts.PathVariables[varName]; ok {
				return v
			}
			if v, ok := opts.ConfigVariables[varName]; ok {
				return v
			}
			if err == nil {
				err = errors.New("no value for variable '" + varName + "'")
			}
			return match
		})
	return subst, err
}

// Archive processes a .isolate, generates a .isolated and archive it.
// Returns a *PendingItem to the .isolated.
func Archive(arch *archiver.Archiver, opts *ArchiveOptions) *archiver.PendingItem {
	displayName := filepath.Base(opts.Isolated)
	f, err := archive(arch, opts, displayName)
	if err != nil {
		i := &archiver.PendingItem{DisplayName: displayName}
		i.SetErr(err)
		return i
	}
	return f
}

func processDependencies(deps []string, isolateDir string, opts *ArchiveOptions) ([]string, string, error) {
	// Expand variables in the deps, and convert each path to an absolute form.
	for i := range deps {
		dep, err := ReplaceVariables(deps[i], opts)
		if err != nil {
			return nil, "", err
		}
		deps[i] = filepath.Join(isolateDir, dep)
	}

	// Find the root directory of all the files (the root might be above isolateDir).
	rootDir := isolateDir
	resultDeps := make([]string, 0, len(deps))
	for _, dep := range deps {
		// Check if the dep is outside isolateDir.
		info, err := os.Stat(dep)
		if err != nil {
			if !opts.AllowMissingFileDir {
				return nil, "", err
			}
			log.Printf("Ignore missing dep: %s, err: %v", dep, err)
			continue
		}
		base := filepath.Dir(dep)
		if info.IsDir() {
			base = dep
			// Downstream expects the dependency of a directory to always end
			// with '/', but filepath.Join() removes that, so we add it back.
			dep += osPathSeparator
		}
		resultDeps = append(resultDeps, dep)
		for {
			rel, err := filepath.Rel(rootDir, base)
			if err != nil {
				return nil, "", err
			}
			if !strings.HasPrefix(rel, "..") {
				break
			}
			newRootDir := filepath.Dir(rootDir)
			if newRootDir == rootDir {
				return nil, "", errors.New("failed to find root dir")
			}
			rootDir = newRootDir
		}
	}
	if rootDir != isolateDir {
		log.Printf("Root: %s", rootDir)
	}
	return resultDeps, rootDir, nil
}

// ProcessIsolate parses an isolate file, returning the list of dependencies
// (both files and directories), the root directory and the initial Isolated struct.
func ProcessIsolate(opts *ArchiveOptions) ([]string, string, *isolated.Isolated, error) {
	content, err := ioutil.ReadFile(opts.Isolate)
	if err != nil {
		return nil, "", nil, err
	}
	cmd, deps, isolateDir, err := LoadIsolateForConfig(filepath.Dir(opts.Isolate), content, opts.ConfigVariables)
	if err != nil {
		return nil, "", nil, err
	}

	// Expand variables in the commands.
	for i := range cmd {
		if cmd[i], err = ReplaceVariables(cmd[i], opts); err != nil {
			return nil, "", nil, err
		}
	}

	deps, rootDir, err := processDependencies(deps, isolateDir, opts)
	if err != nil {
		return nil, "", nil, err
	}
	// Prepare the .isolated struct.
	isol := &isolated.Isolated{
		Algo:    "sha-1",
		Files:   map[string]isolated.File{},
		Version: isolated.IsolatedFormatVersion,
	}
	if len(cmd) != 0 {
		if opts.AllowCommandAndRelativeCWD {
			os.Stderr.WriteString(`
WARNING: command / relative_cwd in isolate file will be deprecated around 2020 Q3 or later (crbug.com/1069704).
`)
		} else {
			return nil, "", nil, errors.New(`
ERROR: command / relative_cwd in isolate file will be deprecated around 2020 Q3 or later.
Please conntact the LUCI team in crbug.com/1069704 if you see this error.
Escape hatch is to specify -allow-command-and-relative-cwd flag.
`)
		}

		isol.Command = cmd
		// Only set RelativeCwd if a command was also specified. This reduce the
		// noise for Swarming tasks where the command is specified as part of the
		// Swarming task request and not through the isolated file.
		if rootDir != isolateDir {
			relPath, err := filepath.Rel(rootDir, isolateDir)
			if err != nil {
				return nil, "", nil, err
			}
			isol.RelativeCwd = relPath
		}
	}
	return deps, rootDir, isol, nil
}

// ProcessIsolateForCAS works similarly to ProcessIsolate. However, it is
// simpler in that it returns a list of dependency *relative* paths and the
// root directory, which are the necessary input to upload to RBE-CAS.
func ProcessIsolateForCAS(opts *ArchiveOptions) ([]string, string, error) {
	content, err := ioutil.ReadFile(opts.Isolate)
	if err != nil {
		return nil, "", err
	}
	_, deps, isolateDir, err := LoadIsolateForConfig(filepath.Dir(opts.Isolate), content, opts.ConfigVariables)
	if err != nil {
		return nil, "", err
	}

	deps, rootDir, err := processDependencies(deps, isolateDir, opts)
	if err != nil {
		return nil, "", err
	}
	relDeps := make([]string, len(deps))
	for i, dep := range deps {
		rel, err := filepath.Rel(rootDir, dep)
		if err != nil {
			return nil, "", err
		}
		if strings.HasSuffix(dep, osPathSeparator) && !strings.HasSuffix(rel, osPathSeparator) {
			// Make it consistent with the isolated format such that directory paths must end with osPathSeparator.
			rel += osPathSeparator
		}
		relDeps[i] = rel
	}
	return relDeps, rootDir, err
}

func archive(arch *archiver.Archiver, opts *ArchiveOptions, displayName string) (*archiver.PendingItem, error) {
	deps, rootDir, i, err := ProcessIsolate(opts)
	if err != nil {
		return nil, err
	}
	// Handle each dependency, either a file or a directory.
	var fileItems []*archiver.PendingItem
	for _, dep := range deps {
		relPath, err := filepath.Rel(rootDir, dep)
		if err != nil {
			return nil, err
		}
		// Grab the stats right away; this can be used for both checking whether
		// it's a directory and checking whether it's a link.
		info, err := os.Lstat(dep)
		if err != nil {
			return nil, err
		}
		if mode := info.Mode(); mode.IsDir() {
			if relPath, err = filepath.Rel(rootDir, dep); err != nil {
				return nil, err
			}
			err, dirFItems, dirSymItems := archiver.PushDirectory(arch, dep, relPath)
			if err != nil {
				return nil, err
			}
			for pending, item := range dirFItems {
				i.Files[item.RelPath] = isolated.BasicFile("", int(item.Info.Mode()), item.Info.Size())
				fileItems = append(fileItems, pending)
			}
			for relPath, dstPath := range dirSymItems {
				i.Files[relPath] = isolated.SymLink(dstPath)
			}
		} else {
			if mode&os.ModeSymlink == os.ModeSymlink {
				l, err := os.Readlink(dep)
				if err != nil {
					// Kill the process: there's no reason to continue if a file is
					// unavailable.
					log.Fatalf("Unable to stat %q: %v", dep, err)
				}
				i.Files[relPath] = isolated.SymLink(l)
			} else {
				i.Files[relPath] = isolated.BasicFile("", int(mode.Perm()), info.Size())
				fileItems = append(fileItems, arch.PushFile(relPath, dep, -info.Size()))
			}
		}
	}

	for _, item := range fileItems {
		item.WaitForHashed()
		if err = item.Error(); err != nil {
			return nil, err
		}
		f := i.Files[item.DisplayName]
		f.Digest = item.Digest()
		i.Files[item.DisplayName] = f
	}

	raw := &bytes.Buffer{}
	if err = json.NewEncoder(raw).Encode(i); err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(opts.Isolated, raw.Bytes(), 0644); err != nil {
		return nil, err
	}
	return arch.Push(displayName, isolatedclient.NewBytesSource(raw.Bytes()), 0), nil
}
