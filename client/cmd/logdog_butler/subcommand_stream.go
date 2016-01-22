// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os"

	"github.com/luci/luci-go/client/internal/logdog/butler"
	"github.com/luci/luci-go/client/logdog/butlerlib/streamproto"
	"github.com/luci/luci-go/common/flag/nestedflagset"
	"github.com/luci/luci-go/common/logdog/types"
	log "github.com/luci/luci-go/common/logging"
	"github.com/maruel/subcommands"
)

var subcommandStream = &subcommands.Command{
	UsageLine: "stream",
	ShortDesc: "Serves a single input stream.",
	LongDesc:  "Configures and transmits a single stream from the command-line.",
	CommandRun: func() subcommands.CommandRun {
		cmd := &streamCommandRun{}

		cmd.Flags.StringVar(&cmd.path, "source", "-",
			"The path of the file to stream. If omitted, STDIN will be used")

		streamFlag := &nestedflagset.FlagSet{}
		cmd.stream.addFlags(&streamFlag.F)
		cmd.Flags.Var(streamFlag, "stream", "Output stream parameters.")
		return cmd
	},
}

type streamCommandRun struct {
	subcommands.CommandRunBase

	path   string       // The stream input path.
	stream streamConfig // Stream configuration parameters.
}

// subcommands.Run
func (cmd *streamCommandRun) Run(app subcommands.Application, args []string) int {
	a := app.(*application)

	streamFile := (*os.File)(nil)
	if cmd.path == "-" {
		cmd.stream.Name = "stdin"
		streamFile = os.Stdin
	} else {
		streamName, err := types.MakeStreamName("file:", cmd.path)
		if err != nil {
			log.Fields{
				log.ErrorKey: err,
			}.Errorf(a, "Failed to generate stream name.")
			return runtimeErrorReturnCode
		}
		cmd.stream.Name = streamproto.StreamNameFlag(streamName)

		file, err := os.Open(cmd.path)
		if err != nil {
			log.Fields{
				log.ErrorKey: err,
				"path":       cmd.path,
			}.Errorf(a, "Failed to open input stream file.")
			return runtimeErrorReturnCode
		}
		streamFile = file
	}

	// Instantiate our Processor.
	err := a.Main(func(b *butler.Butler) error {
		if err := b.AddStream(streamFile, cmd.stream.properties()); err != nil {
			return err
		}

		b.Activate()
		return nil
	})
	if err != nil {
		log.Fields{
			log.ErrorKey: err,
		}.Errorf(a, "Failed to stream file.")
		return runtimeErrorReturnCode
	}

	return 0
}
