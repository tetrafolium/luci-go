// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"github.com/luci/luci-go/client/internal/logdog/butler/output"
	logOutput "github.com/luci/luci-go/client/internal/logdog/butler/output/log"
	"github.com/luci/luci-go/common/flag/multiflag"
)

type logOutputFactory struct {
	bundleSize int
}

var _ outputFactory = (*logOutputFactory)(nil)

func (f *logOutputFactory) option() multiflag.Option {
	opt := newOutputOption("log", "Debug output that writes to STDOUT.", f)

	flags := opt.Flags()
	flags.IntVar(&f.bundleSize, "bundle-size", 1024*1024,
		"Maximum bundle size.")

	return opt
}

func (f *logOutputFactory) configOutput(a *application) (output.Output, error) {
	return logOutput.New(a, f.bundleSize), nil
}

func (f *logOutputFactory) scopes() []string { return nil }
