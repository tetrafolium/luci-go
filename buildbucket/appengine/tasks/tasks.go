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

// Package tasks contains task queue implementations.
package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/server/tq"

	taskdefs "github.com/tetrafolium/luci-go/buildbucket/appengine/tasks/defs"

	// Enable datastore transactional tasks support.
	_ "github.com/tetrafolium/luci-go/server/tq/txn/datastore"
)

// rejectionHandler returns a tq.Handler which rejects the given task.
// Used by tasks which are handled in Python.
// TODO(crbug/1042991): Remove once all handlers are implemented in Go.
func rejectionHandler(tq string) tq.Handler {
	return func(ctx context.Context, payload proto.Message) error {
		logging.Errorf(ctx, "tried to handle %s: %q", tq, payload)
		return errors.Reason("handler called").Err()
	}
}

func init() {
	tq.RegisterTaskClass(tq.TaskClass{
		ID: "cancel-swarming-task",
		Custom: func(ctx context.Context, m proto.Message) (*tq.CustomPayload, error) {
			task := m.(*taskdefs.CancelSwarmingTask)
			body, err := json.Marshal(map[string]interface{}{
				"hostname": task.Hostname,
				"task_id":  task.TaskId,
				"realm":    task.Realm,
			})
			if err != nil {
				return nil, errors.Annotate(err, "error marshaling payload").Err()
			}
			return &tq.CustomPayload{
				Body:        body,
				Method:      "POST",
				RelativeURI: fmt.Sprintf("/internal/task/buildbucket/cancel_swarming_task/%s/%s", task.Hostname, task.TaskId),
			}, nil
		},
		Handler:   rejectionHandler("cancel-swarming-task"),
		Kind:      tq.Transactional,
		Prototype: (*taskdefs.CancelSwarmingTask)(nil),
		Queue:     "backend-default",
	})

	tq.RegisterTaskClass(tq.TaskClass{
		ID: "notify-pubsub",
		Custom: func(ctx context.Context, m proto.Message) (*tq.CustomPayload, error) {
			task := m.(*taskdefs.NotifyPubSub)
			mode := "global"
			if task.Callback {
				mode = "callback"
			}
			body, err := json.Marshal(map[string]interface{}{
				"id":   task.BuildId,
				"mode": mode,
			})
			if err != nil {
				return nil, errors.Annotate(err, "error marshaling payload").Err()
			}
			return &tq.CustomPayload{
				Body:        body,
				Method:      "POST",
				RelativeURI: fmt.Sprintf("/internal/task/buildbucket/notify/%d", task.BuildId),
			}, nil
		},
		Handler:   rejectionHandler("notify-pubsub"),
		Kind:      tq.Transactional,
		Prototype: (*taskdefs.NotifyPubSub)(nil),
		Queue:     "backend-default",
	})
}

// CancelSwarmingTask enqueues a task queue task to cancel the given Swarming
// task.
func CancelSwarmingTask(ctx context.Context, task *taskdefs.CancelSwarmingTask) error {
	switch {
	case task.GetHostname() == "":
		return errors.Reason("hostname is required").Err()
	case task.TaskId == "":
		return errors.Reason("task_id is required").Err()
	}
	return tq.AddTask(ctx, &tq.Task{
		Payload: task,
	})
}

// NotifyPubSub enqueues a task to publish a Pub/Sub notification for the given
// build.
func NotifyPubSub(ctx context.Context, task *taskdefs.NotifyPubSub) error {
	if task.GetBuildId() == 0 {
		return errors.Reason("build_id is required").Err()
	}
	return tq.AddTask(ctx, &tq.Task{
		Payload: task,
	})
}
