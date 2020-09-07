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

package ledcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/led/job"
)

// EditRecipeBundleOpts are user-provided options for the recipe bundling
// process.
type EditRecipeBundleOpts struct {
	// Path on disk to the repo to extract the recipes from. May be a subdirectory
	// of the repo, as long as `git rev-parse --show-toplevel` can find the root
	// of the repository.
	//
	// If empty, uses the current working directory.
	RepoDir string

	// Overrides is a mapping of recipe project id (e.g. "recipe_engine") to
	// a local path to a checkout of that repo (e.g. "/path/to/recipes-py.git").
	//
	// When the bundle is created, this local repo will be used instead of the
	// pinned version of this recipe project id. This is helpful for preparing
	// bundles which have code changes in multiple recipe repos.
	Overrides map[string]string

	// DebugSleep is the amount of time to wait after the recipe completes
	// execution (either success or failure). This is injected into the generated
	// recipe bundle as a 'sleep X' command after the invocation of the recipe
	// itself.
	DebugSleep time.Duration
}

// RecipeDirectory is a very unfortunate constant which is here for
// a combination of reasons:
//   1) swarming doesn't allow you to 'checkout' an isolate relative to any path
//      in the task (other than the task root). This means that whatever value
//      we pick for EditRecipeBundle must be used EVERYWHERE the isolated hash
//      is used.
//   2) Currently the 'recipe_engine/led' module will blindly take the isolated
//      input and 'inject' it into further uses of led. This module currently
//      doesn't specify the checkout dir, relying on kitchen's default value of
//      (you guessed it) "kitchen-checkout".
//
// In order to fix this (and it will need to be fixed for bbagent support):
//   * The 'recipe_engine/led' module needs to accept 'checkout-dir' as
//     a parameter in its input properties.
//   * led needs to start passing the checkout dir to the led module's input
//     properties.
//   * `led edit` needs a way to manipulate the checkout directory in a job
//   * The 'recipe_engine/led' module needs to set this in the job
//     alongside the isolate hash when it's doing the injection.
//
// For now, we just hard-code it.
//
// TODO(crbug.com/1072117): Fix this, it's weird.
const RecipeDirectory = "kitchen-checkout"

// EditRecipeBundle overrides the recipe bundle in the given job with one
// located on disk.
//
// It isolates the recipes from the repository in the given working directory
// into the UserPayload under the directory "kitchen-checkout/". If there's an
// existing directory in the UserPayload at that location, it will be removed.
func EditRecipeBundle(ctx context.Context, authClient *http.Client, jd *job.Definition, opts *EditRecipeBundleOpts) error {
	if jd.GetSwarming() != nil {
		return errors.New("ledcmd.EditRecipeBundle is only available for Buildbucket tasks")
	}

	if opts == nil {
		opts = &EditRecipeBundleOpts{}
	}

	recipesPy, err := findRecipesPy(ctx, opts.RepoDir)
	if err != nil {
		return err
	}
	logging.Debugf(ctx, "using recipes.py: %q", recipesPy)

	err = EditIsolated(ctx, authClient, jd, func(ctx context.Context, dir string) error {
		logging.Infof(ctx, "bundling recipes")
		bundlePath := filepath.Join(dir, RecipeDirectory)
		// Remove existing bundled recipes, if any. Ignore the error.
		os.RemoveAll(bundlePath)
		if err := opts.prepBundle(ctx, opts.RepoDir, recipesPy, bundlePath); err != nil {
			return err
		}
		logging.Infof(ctx, "isolating recipes")
		return nil
	})
	if err != nil {
		return err
	}

	return jd.HighLevelEdit(func(je job.HighLevelEditor) {
		je.TaskPayloadSource("", "")
		je.TaskPayloadPath(RecipeDirectory)
	})
}

func logCmd(ctx context.Context, inDir string, arg0 string, args ...string) *exec.Cmd {
	ret := exec.CommandContext(ctx, arg0, args...)
	ret.Dir = inDir
	logging.Debugf(ctx, "Running (from %q) - %s %v", inDir, arg0, args)
	return ret
}

func cmdErr(err error, reason string) error {
	if err != nil {
		ee, _ := err.(*exec.ExitError)
		outErr := ""
		if ee != nil {
			outErr = strings.TrimSpace(string(ee.Stderr))
			if len(outErr) > 128 {
				outErr = outErr[:128] + "..."
			}
		}
		err = errors.Annotate(err, reason+": %s", outErr).Err()
	}
	return err
}

func appendText(path, fmtStr string, items ...interface{}) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, fmtStr, items...)
	return err
}

func (opts *EditRecipeBundleOpts) prepBundle(ctx context.Context, inDir, recipesPy, toDirectory string) (err error) {
	args := []string{
		recipesPy,
	}
	if logging.GetLevel(ctx) < logging.Info {
		args = append(args, "-v")
	}
	for projID, path := range opts.Overrides {
		args = append(args, "-O", fmt.Sprintf("%s=%s", projID, path))
	}
	args = append(args, "bundle", "--destination", filepath.Join(toDirectory))
	cmd := logCmd(ctx, inDir, "python", args...)
	if logging.GetLevel(ctx) < logging.Info {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err = cmdErr(cmd.Run(), "creating bundle"); err != nil {
		return
	}
	if opts.DebugSleep != 0 {
		fname := filepath.Join(toDirectory, "recipes")
		seconds := opts.DebugSleep / time.Second
		msg := "echo ENTERING DEBUG SLEEP. SSH to the bot to debug."

		if err = appendText(fname, "\n%s\nsleep %d\n", msg, seconds); err != nil {
			return
		}
		// Wait for a bogus event that won't occur... Windows sucks, amirite?
		if err = appendText(fname+".bat", "\r\n%s\r\nwaitfor /t %d DebugSessionEnd\r\n", msg, seconds); err != nil {
			return
		}
	}

	return
}

// findRecipesPy locates the current repo's `recipes.py`. It does this by:
//   * invoking git to find the repo root
//   * loading the recipes.cfg at infra/config/recipes.cfg
//   * stat'ing the recipes.py implied by the recipes_path in that cfg file.
//
// Failure will return an error.
//
// On success, the absolute path to recipes.py is returned.
func findRecipesPy(ctx context.Context, inDir string) (string, error) {
	cmd := logCmd(ctx, inDir, "git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err = cmdErr(err, "finding git repo"); err != nil {
		return "", err
	}

	repoRoot := strings.TrimSpace(string(out))

	pth := filepath.Join(repoRoot, "infra", "config", "recipes.cfg")
	switch st, err := os.Stat(pth); {
	case err != nil:
		return "", errors.Annotate(err, "reading recipes.cfg").Err()

	case !st.Mode().IsRegular():
		return "", errors.Reason("%q is not a regular file", pth).Err()
	}

	type recipesJSON struct {
		RecipesPath string `json:"recipes_path"`
	}
	rj := &recipesJSON{}

	f, err := os.Open(pth)
	if err != nil {
		return "", errors.Reason("reading recipes.cfg: %q", pth).Err()
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(rj); err != nil {
		return "", errors.Reason("parsing recipes.cfg: %q", pth).Err()
	}

	return filepath.Join(
		repoRoot, filepath.FromSlash(rj.RecipesPath), "recipes.py"), nil
}
