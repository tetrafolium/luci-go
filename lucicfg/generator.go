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
//
// All Starlark code is executed sequentially in a single goroutine from inside
// Generate function, thus this package doesn't used any mutexes or other
// synchronization primitives. It is safe to call Generate concurrently though,
// since there's no global shared state, each Generate call operates on its
// own state.
package lucicfg

import (
	"context"
	"fmt"
	"strings"

	"go.starlark.net/starlark"

	"github.com/tetrafolium/luci-go/starlark/builtins"
	"github.com/tetrafolium/luci-go/starlark/interpreter"
	"github.com/tetrafolium/luci-go/starlark/starlarkproto"

	generated "github.com/tetrafolium/luci-go/lucicfg/starlark"
)

// Inputs define all inputs for the config generator.
type Inputs struct {
	Code  interpreter.Loader // a package with the user supplied code
	Entry string             // a name of the entry point script in this package
	Vars  map[string]string  // var values passed via `-var key=value` flags

	// Used to setup additional facilities for unit tests.
	testOmitHeader              bool
	testPredeclared             starlark.StringDict
	testThreadModifier          func(th *starlark.Thread)
	testDisableFailureCollector bool
}

// Generate interprets the high-level config.
//
// Returns a multi-error with all captured errors. Some of them may implement
// BacktracableError interface.
func Generate(ctx context.Context, in Inputs) (*State, error) {
	state := &State{Inputs: in}
	ctx = withState(ctx, state)

	// All available symbols implemented in go.
	predeclared := starlark.StringDict{
		// Part of public API of the generator.
		"fail":       builtins.Fail,
		"proto":      starlarkproto.ProtoLib()["proto"],
		"stacktrace": builtins.Stacktrace,
		"struct":     builtins.Struct,
		"to_json":    builtins.ToJSON,

		// '__native__' is NOT public API. It should be used only through public
		// @stdlib functions.
		"__native__": native(starlark.StringDict{
			"ctor":             builtins.Ctor,
			"genstruct":        builtins.GenStruct,
			"re_submatches":    builtins.RegexpMatcher("submatches"),
			"wellknown_descpb": wellKnownDescSet,
			"googtypes_descpb": googTypesDescSet,
			"lucitypes_descpb": luciTypesDescSet,
		}),
	}
	for k, v := range in.testPredeclared {
		predeclared[k] = v
	}

	// Expose @stdlib and __main__ package. They have no externally observable
	// state of their own, but they call low-level __native__.* functions that
	// manipulate 'state' by getting it through the context.
	pkgs := embeddedPackages()
	pkgs[interpreter.MainPkg] = in.Code

	// Create a proto loader, hook up load("@proto//<path>", ...) to load proto
	// modules through it. See ThreadModifier below where it is set as default in
	// the thread. This exposes it to Starlark code, so it can register descriptor
	// sets in it.
	ploader := starlarkproto.NewLoader()
	pkgs["proto"] = func(path string) (dict starlark.StringDict, _ string, err error) {
		mod, err := ploader.Module(path)
		if err != nil {
			return nil, "", err
		}
		return starlark.StringDict{mod.Name: mod}, "", nil
	}

	// Capture details of fail(...) calls happening inside Starlark code.
	failures := builtins.FailureCollector{}

	// Execute the config script in this environment. Return errors unwrapped so
	// that callers can sniff out various sorts of Starlark errors.
	intr := interpreter.Interpreter{
		Predeclared: predeclared,
		Packages:    pkgs,

		PreExec:  func(th *starlark.Thread, _ interpreter.ModuleKey) { state.vars.OpenScope(th) },
		PostExec: func(th *starlark.Thread, _ interpreter.ModuleKey) { state.vars.CloseScope(th) },

		ThreadModifier: func(th *starlark.Thread) {
			starlarkproto.SetDefaultLoader(th, ploader)
			if !in.testDisableFailureCollector {
				failures.Install(th)
			}
			if in.testThreadModifier != nil {
				in.testThreadModifier(th)
			}
		},
	}

	// Load builtins.star, and then execute the user-supplied script.
	var err error
	if err = intr.Init(ctx); err == nil {
		_, err = intr.ExecModule(ctx, interpreter.MainPkg, in.Entry)
	}
	if err != nil {
		if f := failures.LatestFailure(); f != nil {
			err = f // prefer this error, it has custom stack trace
		}
		return nil, state.err(err)
	}

	// Verify all var values provided via Inputs.Vars were actually used by
	// lucicfg.var(expose_as='...') definitions.
	if errs := state.checkUncosumedVars(); len(errs) != 0 {
		return nil, state.err(errs...)
	}

	// Executing the script (with all its dependencies) populated the graph.
	// Finalize it. This checks there are no dangling edges, freezes the graph,
	// and makes it queryable, so generator callbacks can traverse it.
	if errs := state.graph.Finalize(); len(errs) != 0 {
		return nil, state.err(errs...)
	}

	// The script registered a bunch of callbacks that take the graph and
	// transform it into actual output config files. Run these callbacks now.
	genCtx := newGenCtx()
	if errs := state.generators.call(intr.Thread(ctx), genCtx); len(errs) != 0 {
		return nil, state.err(errs...)
	}
	output, err := genCtx.assembleOutput(!in.testOmitHeader)
	if err != nil {
		return nil, state.err(err)
	}
	state.Output = output

	if len(state.errors) != 0 {
		return nil, state.errors
	}

	// Discover what main package modules we actually executed.
	for _, key := range intr.Visited() {
		if key.Package == interpreter.MainPkg {
			state.Visited = append(state.Visited, key.Path)
		}
	}

	return state, nil
}

// embeddedPackages makes a map of loaders for embedded Starlark packages.
//
// Each directory directly under github.com/tetrafolium/luci-go/lucicfg/starlark/...
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
