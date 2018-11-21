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

// Package lucicfg contains LUCI config generator.
package lucicfg

import (
	"context"
	"fmt"
	"strings"

	"go.starlark.net/starlark"

	"go.chromium.org/luci/starlark/interpreter"
	"go.chromium.org/luci/starlark/starlarkproto"

	generated "go.chromium.org/luci/lucicfg/starlark"
)

// Inputs define all inputs for the config generator.
type Inputs struct {
	Main interpreter.Loader // a package with the entry LUCI.star file
}

// State is mutated throughout execution of the script and at the end contains
// the final execution result.
//
// It is available in the implementation of native functions exposed to the
// Starlark side. Starlark code operates with the state exclusively through
// these functions.
type State struct {
	Inputs Inputs // all inputs, exactly as passed to Generate.

	Greetings []string // this is just for demo purposes
}

// Generate interprets the high-level config.
func Generate(ctx context.Context, in Inputs) (*State, error) {
	state := &State{Inputs: in}

	// Expose two predeclared symbols: 'proto' with utilities for manipulating
	// proto messages and '__native__' with all hooks into Go code. 'proto' is
	// part of public API. '__native__' is NOT, it should be used only through
	// public @stdlib functions.
	predeclared := starlark.StringDict{
		"proto":      starlarkproto.ProtoLib()["proto"],
		"__native__": native(ctx, state),
	}

	// Expose @stdlib, @proto and __main__ package. All have no externally
	// observable state of their own, but they call low-level __native__.*
	// functions that manipulate 'state'.
	pkgs := embeddedPackages()
	pkgs[interpreter.MainPkg] = in.Main
	pkgs["proto"] = interpreter.ProtoLoader()

	// Execute LUCI.star in this environment. Return errors unwrapped so that
	// callers can sniff out various sorts of Starlark errors.
	intr := interpreter.Interpreter{
		Predeclared: predeclared,
		Packages:    pkgs,
	}
	if err := intr.Init(); err != nil {
		return nil, err
	}
	if _, err := intr.LoadModule(interpreter.MainPkg, "LUCI.star"); err != nil {
		return nil, err
	}

	// TODO(vadimsh): There'll likely be more stages of the execution. LoadModule
	// above only loads all starlark code, we may want to call some of callbacks
	// it has registered.

	return state, nil
}

// embeddedPackages makes a map of loaders for embedded Starlark packages.
//
// Each directory directly under go.chromium.org/luci/lucicfg/starlark/...
// represents a corresponding starlark package. E.g. files in 'stdlib' directory
// are loadable via load("@stdlib//<path>", ...).
func embeddedPackages() map[string]interpreter.Loader {
	perRoot := map[string]map[string]string{}

	for path, data := range generated.Assets() {
		chunks := strings.SplitN(path, "/", 2)
		if len(chunks) != 2 {
			panic(fmt.Sprintf("forbidden *.star outside the package dir: %s", path))
		}
		root, rel := chunks[0], chunks[1]
		m := perRoot[root]
		if m == nil {
			m = make(map[string]string, 1)
			perRoot[root] = m
		}
		m[rel] = data
	}

	loaders := make(map[string]interpreter.Loader, len(perRoot))
	for pkg, files := range perRoot {
		loaders[pkg] = interpreter.MemoryLoader(files)
	}
	return loaders
}
