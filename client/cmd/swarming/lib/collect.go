// Copyright 2017 The LUCI Authors.
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

package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/maruel/subcommands"
	"golang.org/x/sync/semaphore"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/system/signals"
)

type taskOutputOption int64

const (
	taskOutputNone    taskOutputOption = 0
	taskOutputConsole taskOutputOption = 1 << 0
	taskOutputJSON    taskOutputOption = 1 << 1
	taskOutputAll     taskOutputOption = taskOutputConsole | taskOutputJSON
)

func (t *taskOutputOption) String() string {
	switch *t {
	case taskOutputJSON:
		return "json"
	case taskOutputConsole:
		return "console"
	case taskOutputAll:
		return "all"
	case taskOutputNone:
		fallthrough
	default:
		return "none"
	}
}

func (t *taskOutputOption) Set(s string) error {
	switch s {
	case "json":
		*t = taskOutputJSON
	case "console":
		*t = taskOutputConsole
	case "all":
		*t = taskOutputAll
	case "", "none":
		*t = taskOutputNone
	default:
		return errors.Reason("invalid task output option: %s", s).Err()
	}
	return nil
}

func (t *taskOutputOption) includesJSON() bool {
	return (*t & taskOutputJSON) != 0
}

func (t *taskOutputOption) includesConsole() bool {
	return (*t & taskOutputConsole) != 0
}

// weightedSemaphore allows mocking semaphore.Weighted in tests.
type weightedSemaphore interface {
	Acquire(context.Context, int64) error
	TryAcquire(int64) bool
	Release(int64)
}

// taskResult is a consolidation of the results of packaging up swarming
// task results from collect.
type taskResult struct {
	// taskID is the ID of the swarming task for which this results were retrieved.
	taskID string

	// result is the raw result structure returned by a swarming RPC call.
	// result may be nil if err is non-nil.
	result *swarming.SwarmingRpcsTaskResult

	// output is the console output produced by the swarming task.
	// output will only be populated if requested.
	output string

	// outputs is a list of file outputs from a task, downloaded from an isolate server.
	// outputs will only be populated if requested.
	outputs []string

	// err is set if an operational error occurred while doing RPCs to gather the
	// task result, which includes errors received from the server.
	err error
}

func (t *taskResult) Print(w io.Writer) {
	if t.err != nil {
		fmt.Fprintf(w, "%s: %v\n", t.taskID, t.err)
	} else {
		fmt.Fprintf(w, "%s: exit %d\n", t.taskID, t.result.ExitCode)
		if t.output != "" {
			fmt.Fprintln(w, t.output)
		}
	}
}

// CmdCollect returns an object for the `collect` subcommand.
func CmdCollect(defaultAuthOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "collect <options> (-requests-json file | task_id...)",
		ShortDesc: "Waits on a set of Swarming tasks",
		LongDesc:  "Waits on a set of Swarming tasks.",
		CommandRun: func() subcommands.CommandRun {
			r := &collectRun{}
			r.Init(defaultAuthOpts)
			return r
		},
	}
}

type collectRun struct {
	commonFlags

	timeout           time.Duration
	taskSummaryJSON   string
	taskSummaryPython bool
	taskOutput        taskOutputOption
	outputDir         string
	eager             bool
	perf              bool
	jsonInput         string
}

func (c *collectRun) Init(defaultAuthOpts auth.Options) {
	c.commonFlags.Init(defaultAuthOpts)

	c.Flags.DurationVar(&c.timeout, "timeout", 0, "Timeout to wait for result. Set to 0 for no timeout.")
	c.Flags.StringVar(&c.taskSummaryJSON, "task-summary-json", "", "Dump a summary of task results to a file as json.")

	//TODO(tikuta): Remove this flag once crbug.com/894045 is fixed.
	c.Flags.BoolVar(&c.taskSummaryPython, "task-summary-python", false, "Generate python client compatible task summary json.")

	c.Flags.BoolVar(&c.eager, "eager", false, "Return after first task completion.")
	c.Flags.BoolVar(&c.perf, "perf", false, "Includes performance statistics.")
	c.Flags.Var(&c.taskOutput, "task-output-stdout", "Where to output each task's console output (stderr/stdout). (none|json|console|all)")
	c.Flags.StringVar(&c.outputDir, "output-dir", "", "Where to download isolated output to.")
	c.Flags.StringVar(&c.jsonInput, "requests-json", "", "Load the task IDs from a .json file as saved by \"trigger -dump-json\"")
}

func (c *collectRun) Parse(args *[]string) error {
	var err error
	if err := c.commonFlags.Parse(); err != nil {
		return err
	}

	// Validate timeout duration.
	if c.timeout < 0 {
		return errors.Reason("negative timeout is not allowed").Err()
	}

	// Validate arguments.
	if c.jsonInput != "" {
		data, err := ioutil.ReadFile(c.jsonInput)
		if err != nil {
			return errors.Annotate(err, "reading json input").Err()
		}
		input := triggerResults{}
		if err := json.Unmarshal(data, &input); err != nil {
			return errors.Annotate(err, "unmarshalling json input").Err()
		}
		// Modify args to contain all the task IDs.
		for _, task := range input.Tasks {
			*args = append(*args, task.TaskId)
		}
	}
	for _, arg := range *args {
		if !regexp.MustCompile("^[a-z0-9]+$").MatchString(arg) {
			return errors.Reason("task ID %q must contain only [a-z0-9]", arg).Err()
		}
	}
	if len(*args) == 0 {
		return errors.Reason("must specify at least one task id, either directly or through -json").Err()
	}
	return err
}

func (c *collectRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.Parse(&args); err != nil {
		printError(a, err)
		return 1
	}
	if err := c.main(a, args); err != nil {
		printError(a, err)
		return 1
	}
	return 0
}

func (c *collectRun) fetchTaskResults(
	ctx context.Context,
	taskID string,
	service swarmingService,
	downloadSem weightedSemaphore,
) taskResult {
	defer logging.Debugf(ctx, "Finished fetching task result: %s", taskID)
	var result *swarming.SwarmingRpcsTaskResult
	var output string
	var outputs []string
	err := retry.Retry(ctx, transient.Only(retry.Default), func() error {
		var err error

		// Fetch the result details.
		logging.Debugf(ctx, "Fetching task result: %s", taskID)
		result, err = service.GetTaskResult(ctx, taskID, c.perf)
		if err != nil {
			return tagTransientGoogleAPIError(err)
		}

		// Signal that we want to start downloading outputs. We'll only proceed
		// to download them if another task has not already finished and
		// triggered an eager return.
		if !downloadSem.TryAcquire(1) {
			return errors.New("canceled by first task")
		}
		defer downloadSem.Release(1)

		// TODO(mknyszek): Fetch output and outputs in parallel.

		// If we got the result details, try to fetch stdout if the
		// user asked for it.
		if c.taskOutput != taskOutputNone {
			logging.Debugf(ctx, "Fetching task output: %s", taskID)
			taskOutput, err := service.GetTaskOutput(ctx, taskID)
			if err != nil {
				return tagTransientGoogleAPIError(err)
			}
			output = taskOutput.Output
		}

		// Download the result isolated if available and if we have a place to put it.
		if c.outputDir != "" {
			logging.Debugf(ctx, "Fetching task outputs: %s", taskID)
			outputs, err = service.GetTaskOutputs(ctx, taskID, c.outputDir, result.OutputsRef)
			if err != nil {
				return tagTransientGoogleAPIError(err)
			}
		}
		return nil
	}, func(err error, d time.Duration) {
		logging.WithError(err).Warningf(ctx, "Transient error while making request, retrying in %s...", d)
	})
	if err != nil {
		return taskResult{taskID: taskID, err: err}
	}

	return taskResult{
		taskID:  taskID,
		result:  result,
		output:  output,
		outputs: outputs,
	}
}

func (c *collectRun) pollForTaskResult(
	ctx context.Context,
	taskID string,
	service swarmingService,
	downloadSem weightedSemaphore,
) taskResult {
	var result taskResult
	startedTime := clock.Now(ctx)
	for {
		result = c.fetchTaskResults(ctx, taskID, service, downloadSem)
		if result.err != nil {
			// If we received an error from fetchTaskResults, it either hit a fatal
			// failure, or it hit too many transient failures.
			return result
		}

		// Only stop if the swarming bot is "dead" (i.e. not running).
		state, err := parseTaskState(result.result.State)
		if err != nil {
			logging.Debugf(ctx, "Task %s failed with error: %v", taskID, err)
			return taskResult{taskID: taskID, err: err}
		}
		if !state.Alive() {
			logging.Debugf(ctx, "Task completed successfully: %s", taskID)
			return result
		}

		currentTime := clock.Now(ctx)

		// Start with a 1 second delay and for each 30 seconds of waiting
		// add another second until hitting a 15 second ceiling.
		delay := time.Second + (currentTime.Sub(startedTime) / 30)
		if delay >= 15*time.Second {
			delay = 15 * time.Second
		}

		logging.Debugf(ctx, "Waiting %s for task: %s", delay.Round(time.Millisecond), taskID)
		timerResult := <-clock.After(ctx, delay)

		// timerResult should have an error if the context's deadline was exceeded,
		// or if the context was cancelled.
		if timerResult.Err != nil {
			err := timerResult.Err
			if result.err != nil {
				result.err = errors.Annotate(result.err, "%v", timerResult.Err).Err()
			} else {
				result.err = err
			}
			return result
		}
	}
}

// summarizeResultsPython generates summary JSON file compatible with python's
// swarming client.
func summarizeResultsPython(results []taskResult) ([]byte, error) {
	var shards []map[string]interface{}

	for _, result := range results {
		buf, err := json.Marshal(result.result)
		if err != nil {
			return nil, err
		}

		var jsonResult map[string]interface{}
		if err := json.Unmarshal(buf, &jsonResult); err != nil {
			return nil, err
		}

		if jsonResult != nil {
			jsonResult["output"] = result.output
		}
		shards = append(shards, jsonResult)
	}

	return json.MarshalIndent(map[string]interface{}{
		"shards": shards,
	}, "", "  ")
}

// summarizeResults generate a marshalled JSON summary of the task results.
func (c *collectRun) summarizeResults(results []taskResult) ([]byte, error) {
	if c.taskSummaryPython {
		return summarizeResultsPython(results)
	}

	jsonResults := map[string]interface{}{}
	for _, result := range results {
		jsonResult := map[string]interface{}{}
		if result.err != nil {
			jsonResult["error"] = result.err.Error()
		}
		if result.result != nil {
			jsonResult["results"] = *result.result
			if c.taskOutput.includesJSON() {
				jsonResult["output"] = result.output
			}
			jsonResult["outputs"] = result.outputs
		}
		jsonResults[result.taskID] = jsonResult
	}
	return json.MarshalIndent(jsonResults, "", "  ")
}

func (c *collectRun) pollForTasks(
	ctx context.Context,
	taskIDs []string,
	service swarmingService,
	downloadSem weightedSemaphore,
) []taskResult {
	if len(taskIDs) == 0 {
		return nil
	}

	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = clock.WithTimeout(ctx, c.timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// Aggregate results by polling and fetching across multiple goroutines.
	results := make([]taskResult, len(taskIDs))
	var wg sync.WaitGroup
	wg.Add(len(taskIDs))
	taskFinished := make(chan int, len(taskIDs))
	for i := range taskIDs {
		go func(i int) {
			defer func() {
				taskFinished <- i
				wg.Done()
			}()
			results[i] = c.pollForTaskResult(ctx, taskIDs[i], service, downloadSem)
		}(i)
	}

	if c.eager {
		go func() {
			<-taskFinished
			// After the first task finishes, block any new tasks from starting
			// to download outputs, but let any in-progress downloads complete.
			downloadSem.Acquire(ctx, int64(len(taskIDs)))
			cancel()
		}()
	}

	wg.Wait()

	return results
}

func (c *collectRun) main(_ subcommands.Application, taskIDs []string) error {
	// Set up swarming service.
	ctx, cancel := context.WithCancel(c.defaultFlags.MakeLoggingContext(os.Stderr))
	signals.HandleInterrupt(func() {
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
		cancel()
	})
	service, err := c.createSwarmingClient(ctx)
	if err != nil {
		return err
	}

	downloadSem := semaphore.NewWeighted(int64(len(taskIDs)))
	results := c.pollForTasks(ctx, taskIDs, service, downloadSem)

	// Summarize and write summary json if applicable.
	if c.taskSummaryJSON != "" {
		jsonSummary, err := c.summarizeResults(results)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(c.taskSummaryJSON, jsonSummary, 0664); err != nil {
			return err
		}
	}
	for _, result := range results {
		if c.taskOutput.includesConsole() || result.err != nil {
			result.Print(os.Stdout)
		}
	}
	return nil
}
