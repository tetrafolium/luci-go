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

package lib

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/maruel/subcommands"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/auth/client/authcli"
	"github.com/tetrafolium/luci-go/client/downloader"
	"github.com/tetrafolium/luci-go/client/internal/common"
	"github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/isolated"
	"github.com/tetrafolium/luci-go/common/isolatedclient"
	"github.com/tetrafolium/luci-go/common/lhttp"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/system/signals"
)

// triggerResults is a set of results from using the trigger subcommand,
// describing all of the tasks that were triggered successfully.
type triggerResults struct {
	// Tasks is a list of successfully triggered tasks represented as
	// TriggerResult values.
	Tasks []*swarming.SwarmingRpcsTaskRequestMetadata `json:"tasks"`
}

// The swarming server has an internal 60-second deadline for responding to
// requests, so 90 seconds shouldn't cause any requests to fail that would
// otherwise succeed.
const swarmingRPCRequestTimeout = 90 * time.Second

const swarmingAPISuffix = "/_ah/api/swarming/v1/"

// swarmingService is an interface intended to stub out the swarming API
// bindings for testing.
type swarmingService interface {
	NewTask(ctx context.Context, req *swarming.SwarmingRpcsNewTaskRequest) (*swarming.SwarmingRpcsTaskRequestMetadata, error)
	CountTasks(ctx context.Context, start float64, tags ...string) (*swarming.SwarmingRpcsTasksCount, error)
	ListTasks(ctx context.Context, limit int64, state string, tags []string, fields []googleapi.Field) ([]*swarming.SwarmingRpcsTaskResult, error)
	CancelTask(ctx context.Context, taskID string, req *swarming.SwarmingRpcsTaskCancelRequest) (*swarming.SwarmingRpcsCancelResponse, error)
	GetTaskRequest(ctx context.Context, taskID string) (*swarming.SwarmingRpcsTaskRequest, error)
	GetTaskResult(ctx context.Context, taskID string, perf bool) (*swarming.SwarmingRpcsTaskResult, error)
	GetTaskOutput(ctx context.Context, taskID string) (*swarming.SwarmingRpcsTaskOutput, error)
	GetTaskOutputs(ctx context.Context, taskID, outputDir string, ref *swarming.SwarmingRpcsFilesRef) ([]string, error)
	ListBots(ctx context.Context, dimensions []string, fields []googleapi.Field) ([]*swarming.SwarmingRpcsBotInfo, error)
}

type swarmingServiceImpl struct {
	client  *http.Client
	service *swarming.Service
	worker  int
}

func (s *swarmingServiceImpl) NewTask(ctx context.Context, req *swarming.SwarmingRpcsNewTaskRequest) (res *swarming.SwarmingRpcsTaskRequestMetadata, err error) {
	err = retryGoogleRPC(ctx, "NewTask", func() (ierr error) {
		res, ierr = s.service.Tasks.New(req).Context(ctx).Do()
		return
	})
	return
}

func (s *swarmingServiceImpl) CountTasks(ctx context.Context, start float64, tags ...string) (res *swarming.SwarmingRpcsTasksCount, err error) {
	err = retryGoogleRPC(ctx, "CountTasks", func() (ierr error) {
		res, ierr = s.service.Tasks.Count().Context(ctx).Start(start).Tags(tags...).Do()
		return
	})
	return
}

func (s *swarmingServiceImpl) ListTasks(ctx context.Context, limit int64, state string, tags []string, fields []googleapi.Field) ([]*swarming.SwarmingRpcsTaskResult, error) {
	// Create an empty array so that if serialized to JSON it's an empty list,
	// not null.
	tasks := []*swarming.SwarmingRpcsTaskResult{}
	// If no fields are specified, all fields will be returned. If any fields are
	// specified, ensure the cursor is specified so we can get subsequent pages.
	if len(fields) > 0 {
		fields = append(fields, "cursor")
	}
	call := s.service.Tasks.List().Context(ctx).Limit(limit).State(state).Tags(tags...).Fields(fields...)
	// Keep calling as long as there's a cursor indicating more bots to list.
	for {
		var res *swarming.SwarmingRpcsTaskList
		err := retryGoogleRPC(ctx, "ListTasks", func() (ierr error) {
			res, ierr = call.Do()
			return
		})
		if err != nil {
			return tasks, err
		}

		tasks = append(tasks, res.Items...)
		if res.Cursor == "" || int64(len(tasks)) >= limit || len(res.Items) == 0 {
			break
		}
		call.Cursor(res.Cursor)
	}

	if int64(len(tasks)) > limit {
		tasks = tasks[0:limit]
	}

	return tasks, nil
}

func (s *swarmingServiceImpl) CancelTask(ctx context.Context, taskID string, req *swarming.SwarmingRpcsTaskCancelRequest) (res *swarming.SwarmingRpcsCancelResponse, err error) {
	err = retryGoogleRPC(ctx, "CancelTask", func() (ierr error) {
		res, ierr = s.service.Task.Cancel(taskID, req).Context(ctx).Do()
		return
	})
	return
}

func (s *swarmingServiceImpl) GetTaskRequest(ctx context.Context, taskID string) (res *swarming.SwarmingRpcsTaskRequest, err error) {
	err = retryGoogleRPC(ctx, "GetTaskResult", func() (ierr error) {
		res, ierr = s.service.Task.Request(taskID).Context(ctx).Do()
		return
	})
	return
}

func (s *swarmingServiceImpl) GetTaskResult(ctx context.Context, taskID string, perf bool) (res *swarming.SwarmingRpcsTaskResult, err error) {
	err = retryGoogleRPC(ctx, "GetTaskResult", func() (ierr error) {
		res, ierr = s.service.Task.Result(taskID).IncludePerformanceStats(perf).Context(ctx).Do()
		return
	})
	return
}

func (s *swarmingServiceImpl) GetTaskOutput(ctx context.Context, taskID string) (res *swarming.SwarmingRpcsTaskOutput, err error) {
	err = retryGoogleRPC(ctx, "GetTaskOutput", func() (ierr error) {
		res, ierr = s.service.Task.Stdout(taskID).Context(ctx).Do()
		return
	})
	return
}

func (s *swarmingServiceImpl) GetTaskOutputs(ctx context.Context, taskID, outputDir string, ref *swarming.SwarmingRpcsFilesRef) ([]string, error) {
	// Create a task-id-based subdirectory to house the outputs.
	dir := filepath.Join(filepath.Clean(outputDir), taskID)

	// This function can be retried when the RPC returned an HTTP 500. In this case,
	// the directory will already exist and may contain partial results. Take no chance
	// and restart from scratch.
	if err := os.RemoveAll(dir); err != nil {
		return nil, errors.Annotate(err, "failed to remove directory: %s", dir).Err()
	}

	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		return nil, err
	}

	// If there is no file reference, then we short-circuit, as there are no
	// outputs to return. We do as after having created the directory for
	// uniform behavior, so that there is an ID-namespaced directory for each
	// task's outputs, with an empty directory signifying there having been no
	// outputs.
	if ref == nil {
		return nil, nil
	}

	isolatedClient := isolatedclient.NewClient(ref.Isolatedserver, isolatedclient.WithAuthClient(s.client), isolatedclient.WithNamespace(ref.Namespace), isolatedclient.WithUserAgent(SwarmingUserAgent))

	var filesMu sync.Mutex
	var files []string
	ctx, cancel := context.WithCancel(ctx)
	signals.HandleInterrupt(cancel)
	opts := &downloader.Options{
		MaxConcurrentJobs: s.worker,
		FileCallback: func(name string, _ *isolated.File) {
			filesMu.Lock()
			files = append(files, name)
			filesMu.Unlock()
		},
		FileStatsCallback: func(fileStats downloader.FileStats, _ time.Duration) {
			logging.Debugf(ctx, "Downloaded %d of %d bytes in %d of %d files for task: %s",
				fileStats.BytesCompleted, fileStats.BytesScheduled,
				fileStats.CountCompleted, fileStats.CountScheduled,
				taskID)
		},
	}
	dl := downloader.New(ctx, isolatedClient, isolated.HexDigest(ref.Isolated), dir, opts)
	return files, dl.Wait()
}

func (s *swarmingServiceImpl) ListBots(ctx context.Context, dimensions []string, fields []googleapi.Field) ([]*swarming.SwarmingRpcsBotInfo, error) {
	// Create an empty array so that if serialized to JSON it's an empty list,
	// not null.
	bots := []*swarming.SwarmingRpcsBotInfo{}
	// If no fields are specified, all fields will be returned. If any fields are
	// specified, ensure the cursor is specified so we can get subsequent pages.
	if len(fields) > 0 {
		fields = append(fields, "cursor")
	}
	call := s.service.Bots.List().Context(ctx).Dimensions(dimensions...).Fields(fields...)
	// Keep calling as long as there's a cursor indicating more bots to list.
	for {
		var res *swarming.SwarmingRpcsBotList
		err := retryGoogleRPC(ctx, "ListBots", func() (ierr error) {
			res, ierr = call.Do()
			return
		})
		if err != nil {
			return bots, err
		}

		bots = append(bots, res.Items...)
		if res.Cursor == "" {
			break
		}
		call.Cursor(res.Cursor)
	}
	return bots, nil
}

type taskState int32

const (
	maskAlive                 = 1
	stateBotDied    taskState = 1 << 1
	stateCancelled  taskState = 1 << 2
	stateCompleted  taskState = 1 << 3
	stateExpired    taskState = 1 << 4
	statePending    taskState = 1<<5 | maskAlive
	stateRunning    taskState = 1<<6 | maskAlive
	stateTimedOut   taskState = 1 << 7
	stateNoResource taskState = 1 << 8
	stateKilled     taskState = 1 << 9
	stateUnknown    taskState = -1
)

func parseTaskState(state string) (taskState, error) {
	switch state {
	case "BOT_DIED":
		return stateBotDied, nil
	case "CANCELED":
		return stateCancelled, nil
	case "COMPLETED":
		return stateCompleted, nil
	case "EXPIRED":
		return stateExpired, nil
	case "PENDING":
		return statePending, nil
	case "RUNNING":
		return stateRunning, nil
	case "TIMED_OUT":
		return stateTimedOut, nil
	case "NO_RESOURCE":
		return stateNoResource, nil
	case "KILLED":
		return stateKilled, nil
	default:
		return stateUnknown, errors.Reason("unrecognized state: %q", state).Err()
	}
}

func (t taskState) Alive() bool {
	return (t & maskAlive) != 0
}

type commonFlags struct {
	subcommands.CommandRunBase
	defaultFlags common.Flags
	authFlags    authcli.Flags
	serverURL    string

	parsedAuthOpts auth.Options
	worker         int
}

// Init initializes common flags.
func (c *commonFlags) Init(authOpts auth.Options) {
	c.defaultFlags.Init(&c.Flags)
	c.authFlags.Register(&c.Flags, authOpts)
	c.Flags.StringVar(&c.serverURL, "server", os.Getenv("SWARMING_SERVER"), "Server URL; required. Set $SWARMING_SERVER to set a default.")
	c.Flags.StringVar(&c.serverURL, "S", os.Getenv("SWARMING_SERVER"), "Alias for -server.")
	c.Flags.IntVar(&c.worker, "worker", 8, "Number of workers used to download isolated files.")
}

// Parse parses the common flags.
func (c *commonFlags) Parse() error {
	if err := c.defaultFlags.Parse(); err != nil {
		return err
	}
	if c.serverURL == "" {
		return errors.Reason("must provide -server").Err()
	}
	s, err := lhttp.CheckURL(c.serverURL)
	if err != nil {
		return err
	}
	c.serverURL = s
	c.parsedAuthOpts, err = c.authFlags.Options()
	return err
}

func (c *commonFlags) createAuthClient(ctx context.Context) (*http.Client, error) {
	// Don't enforce authentication by using OptionalLogin mode. This is needed
	// for IP whitelisted bots: they have NO credentials to send.
	return auth.NewAuthenticator(ctx, auth.OptionalLogin, c.parsedAuthOpts).Client()
}

func (c *commonFlags) createSwarmingClient(ctx context.Context) (swarmingService, error) {
	client, err := c.createAuthClient(ctx)
	if err != nil {
		return nil, err
	}
	// Create a copy of the client so that the timeout only applies to Swarming
	// RPC requests, not to Isolate requests made by this service. A shallow
	// copy is ok because only the timeout needs to be different.
	rpcClient := *client
	rpcClient.Timeout = swarmingRPCRequestTimeout
	s, err := swarming.NewService(ctx, option.WithHTTPClient(&rpcClient))
	if err != nil {
		return nil, err
	}
	s.BasePath = c.serverURL + swarmingAPISuffix
	s.UserAgent = SwarmingUserAgent
	return &swarmingServiceImpl{client, s, c.worker}, nil
}

func tagTransientGoogleAPIError(err error) error {
	// Responses with HTTP codes < 500, if we got them, indicate fatal errors.
	if gerr, _ := err.(*googleapi.Error); gerr != nil && gerr.Code < 500 {
		return err
	}
	// Everything else (HTTP code >= 500, timeouts, DNS issues, etc) is considered
	// a transient error.
	return transient.Tag.Apply(err)
}

func printError(a subcommands.Application, err error) {
	fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
}

// retryGoogleRPC retries an RPC on transient errors, such as HTTP 500.
func retryGoogleRPC(ctx context.Context, rpcName string, rpc func() error) error {
	return retry.Retry(ctx, transient.Only(retry.Default), func() error {
		err := rpc()
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code >= 500 {
			err = transient.Tag.Apply(err)
		}
		return err
	}, retry.LogCallback(ctx, rpcName))
}
