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

// Command run_annotations is a LUCI executable that wraps a command that
// produces @@@ANNOTATIONS@@@, and converts annotations to build message.
package main

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/logdog/client/butlerlib/bootstrap"
	"github.com/tetrafolium/luci-go/luciexe"
	"github.com/tetrafolium/luci-go/luciexe/exe"
	"github.com/tetrafolium/luci-go/luciexe/legacy/annotee"
	"github.com/tetrafolium/luci-go/luciexe/legacy/annotee/annotation"
	annopb "github.com/tetrafolium/luci-go/luciexe/legacy/annotee/proto"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	exe.Run(func(ctx context.Context, build *pb.Build, userArgs []string, sendBuild exe.BuildSender) error {
		if len(userArgs) == 0 {
			return errors.New("No arguments were provided")
		}

		cwd, err := os.Getwd()
		check(err)

		// Start the subprocess.
		cmd := exec.CommandContext(ctx, userArgs[0], userArgs[1:]...)
		stdout, err := cmd.StdoutPipe()
		check(err)
		stderr, err := cmd.StderrPipe()
		check(err)
		err = cmd.Start()
		check(errors.Annotate(err, "failed to start subprocess").Err())

		ldBootstrap, err := bootstrap.Get()
		check(err)

		var buildMU sync.Mutex
		sendAnnotations := func(ann *annopb.Step) {
			latest, err := annotee.ConvertRootStep(ctx, ann)
			check(errors.Annotate(err, "failed to convert an annotation root step to a build").Err())

			buildMU.Lock()
			defer buildMU.Unlock()
			updateBaseBuild(build, latest)
			sendBuild()
		}

		// Run STDOUT/STDERR streams through the processor.
		// Run the process' output streams through Annotee. This will block until
		// they are all consumed.
		processor := annotee.New(ctx, annotee.Options{
			Client:                 ldBootstrap.Client,
			Execution:              annotation.ProbeExecution(userArgs, os.Environ(), cwd),
			MetadataUpdateInterval: 30 * time.Second,
			Offline:                false,
			CloseSteps:             true,
			AnnotationUpdated: func(annBytes []byte) {
				ann := &annopb.Step{}
				check(errors.Annotate(proto.Unmarshal(annBytes, ann), "failed to parse annotation proto").Err())
				sendAnnotations(ann)
			},
		})
		streams := []*annotee.Stream{
			{
				Reader:   stdout,
				Name:     annotee.STDOUT,
				Annotate: true,
			},
			{
				Reader:   stderr,
				Name:     annotee.STDERR,
				Annotate: true,
			},
		}
		err = processor.RunStreams(streams)
		check(errors.Annotate(err, "failed to process annotations").Err())

		// Wait for the subprocess to exit.
		switch err := cmd.Wait().(type) {
		case *exec.ExitError:
		case nil:
		default:
			check(errors.Annotate(err, "failed to wait for the subprocess to exit").Err())
		}

		// Send the final state.
		sendAnnotations(processor.Finish().RootStep().Proto())
		return nil
	})
}

func updateBaseBuild(base, latest *pb.Build) {
	base.Status = latest.Status
	base.StartTime = latest.StartTime
	base.EndTime = latest.EndTime
	base.SummaryMarkdown = latest.SummaryMarkdown
	base.Output = latest.Output
	for _, log := range base.GetOutput().GetLogs() {
		// Rename the annotation's std logs so that it won't conflict with the
		// std log of this command/luciexe when merging. More specifically,
		// recipe engine will try to merge all logs in output into the step
		// logs which already opens up stdout and stderr log.
		if log.Name == "stdout" || log.Name == "stderr" {
			log.Name = "annotation." + log.Name
		}
	}
	base.Steps = latest.Steps
	for _, step := range base.Steps {
		if len(step.Logs) > 0 && step.Logs[0].Name == luciexe.BuildProtoLogName {
			step.Logs[0].Name = "disabled-" + step.Logs[0].Name
			if step.SummaryMarkdown != "" {
				step.SummaryMarkdown += "\n\n"
			}
			step.SummaryMarkdown += "Launching a merge step in legacy annotation " +
				"mode is not supported. Please migrate to bbagent first. " +
				" $build.proto log is currently renamed to disabled-$build.proto."
		}
	}
}
