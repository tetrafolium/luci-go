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

// Package task defines interface between Scheduler engine and implementations
// of particular tasks (such as URL fetch tasks, Swarming tasks, DM tasks, etc).
//
// Its subpackages contain concrete realizations of Manager interface.
package task

import (
	"context"
	"net/http"
	"time"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/server/auth"

	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"

	"google.golang.org/api/pubsub/v1"

	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
)

// Status is status of a single job invocation.
type Status string

const (
	// StatusStarting means the task is about to start.
	StatusStarting Status = "STARTING"
	// StatusRetrying means the task was starting, but the launch failed in some
	// transient way. The start attempt is retried in this case a bunch of times,
	// until eventually the task moves into either StatusRunning or one of the
	// final states. The only possible transition into StatusRetrying is from
	// StatusStarting. A running task can only succeed or fail.
	StatusRetrying Status = "RETRYING"
	// StatusRunning means the task has started and is running now.
	StatusRunning Status = "RUNNING"
	// StatusSucceeded means the task finished with success.
	StatusSucceeded Status = "SUCCEEDED"
	// StatusFailed means the task finished with error or failed to start.
	StatusFailed Status = "FAILED"
	// StatusOverrun means the task should have been started, but previous one is
	// still running.
	StatusOverrun Status = "OVERRUN"
	// StatusAborted means the task was forcefully aborted (manually or due to
	// hard deadline).
	StatusAborted Status = "ABORTED"
)

// Initial returns true if Status is Starting or Retrying.
//
// These statuses indicate an invocation before LaunchTask (perhaps, a retry of
// it) is finished with the invocation.
func (s Status) Initial() bool {
	return s == StatusStarting || s == StatusRetrying
}

// Final returns true if Status represents some final status.
func (s Status) Final() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusOverrun, StatusAborted:
		return true
	default:
		return false
	}
}

// Traits describes properties that influence how the scheduler engine manages
// tasks handled by this Manager.
type Traits struct {
	// Multistage is true if Manager uses Starting -> Running -> Finished
	// state chain for all invocations (instead of just Starting -> Finished).
	//
	// This is the case for "heavy" tasks that can run for undetermined amount
	// of time (e.g. Swarming and Buildbucket tasks). By switching invocation
	// state to Running, the Manager acknowledges that it takes responsibility for
	// eventually moving the invocation to Finished state (perhaps in response to
	// a PubSub notification or a timer tick). In other words, once an invocation
	// is in Running state, the schedule engine will not automatically keep track
	// of it's healthiness (it's the responsibility of the Manager now).
	//
	// For smaller tasks (that finish in seconds, e.g. gitiles poller tasks) it is
	// simpler and more efficient just to do everything in LaunchTask and then
	// move the invocation to Finished state. By doing so, the Manager avoids
	// implementing healthiness checks, piggybacking on LaunchTask retries
	// automatically performed by the scheduler engine.
	//
	// Currently this trait only influences the UI. Invocations with
	// Multistage == false don't show up as "Starting" in the UI (they are
	// displayed as "Running" instead, since it makes more sense from end-user
	// perspective).
	Multistage bool
}

// Manager knows how to work with a particular kind of tasks (e.g URL fetch
// tasks, Swarming tasks, etc): how to deserialize, validate and execute them.
//
// Manager uses Controller to talk back to the scheduler engine.
type Manager interface {
	// Name returns task manager name. It identifies the corresponding kind
	// of tasks and used in various resource names (e.g. PubSub topic names).
	Name() string

	// ProtoMessageType returns a pointer to protobuf message struct that
	// describes config for the task kind, e.g. &UrlFetchTask{}. Will be used
	// only for its type signature.
	ProtoMessageType() proto.Message

	// Traits returns properties that influence how the scheduler engine manages
	// tasks handled by this Manager.
	//
	// See Traits struct for more details.
	Traits() Traits

	// ValidateProtoMessage verifies task definition proto message makes sense.
	// msg must have same underlying type as ProtoMessageType() return value.
	//
	// Errors are returned via validation.Context.
	ValidateProtoMessage(c *validation.Context, msg proto.Message)

	// LaunchTask starts (or starts and finishes in one go) the task.
	//
	// Manager's responsibilities:
	//  * To move the task to some state other than StatusStarting
	//    (by changing ctl.State().Status). If at some point the task has moved
	//    to StatusRunning, the manager MUST setup some way to track the task's
	//    progress to eventually move it to some final state. It can be a status
	//    check via a timer (see `AddTimer` below), or a PubSub callback (see
	//    `PrepareTopic` below).
	//  * Be idempotent, if possible, using ctl.InvocationID() as an operation
	//    key.
	//  * Not to use supplied controller outside of LaunchTask call.
	//  * Not to use supplied controller concurrently without synchronization.
	//
	// If `LaunchTask` crashes before returning or returns a transient error, it
	// will be called again later, receiving exact same ctl.InvocationID().
	//
	// TaskManager may optionally use ctl.Save() to checkpoint progress and save
	// debug log. ctl.Save() is also implicitly called by the engine when
	// `LaunchTask` returns.
	LaunchTask(c context.Context, ctl Controller) error

	// AbortTask is called to opportunistically abort launched task.
	//
	// It is called right before the job is forcefully switched to a failed state.
	// The engine does not wait for the task runner to acknowledge this action.
	//
	// AbortTask must be idempotent since it may be called multiple times in case
	// of errors.
	AbortTask(c context.Context, ctl Controller) error

	// HandleNotification is called whenever engine receives a PubSub message sent
	// to a topic created with Controller.PrepareTopic. Expect duplicated and
	// out-of-order messages here. HandleNotification must be idempotent.
	//
	// Returns transient error to trigger a redeliver of the message, no error to
	// to acknowledge the message and fatal error to move the invocation to failed
	// state.
	//
	// Any modifications made to the invocation state will be saved regardless of
	// the return value (to save the debug log).
	HandleNotification(c context.Context, ctl Controller, msg *pubsub.PubsubMessage) error

	// HandleTimer is called to process timers set up by Controller.AddTimer.
	//
	// Expect duplicated or delayed events here. HandleTimer must be idempotent.
	//
	// Returns a transient error to trigger a redelivery of the event (the
	// invocation state won't be saved in this case), no error to acknowledge the
	// event and a fatal error to move the invocation to failed state.
	HandleTimer(c context.Context, ctl Controller, name string, payload []byte) error

	// GetDebugState returns debug info about the state persisted by the manager.
	GetDebugState(c context.Context, ctl ControllerReadOnly) (*internal.DebugManagerState, error)
}

// Controller is passed to LaunchTask by the scheduler engine. It gives Manager
// control over one job invocation. Manager must not use it outside of
// LaunchTask. Controller implementation is generally not thread safe (but it's
// fine to use it from multiple goroutines if access is protected by a lock).
//
// All methods that accept context.Context expect contexts derived from ones
// passed to 'Manager' methods. A derived context can be used to set custom
// deadlines for some potentially expensive methods like 'PrepareTopic'.
type Controller interface {
	ControllerReadOnly

	// State returns a mutable portion of task invocation state.
	//
	// TaskManager can modify it in-place and then call Controller.Save to persist
	// the changes. The state will also be saved by the engine automatically if
	// Manager doesn't call Save.
	State() *State

	// AddTimer sets up a new delayed call to Manager.HandleTimer.
	//
	// Timers are active as long as the invocation is not in one of the final
	// states. There is no way to cancel a timer (ignore HandleTimer call
	// instead).
	//
	// 'title' will be visible in logs, it should convey a purpose for this timer.
	// It doesn't have to be unique.
	//
	// 'payload' is any byte blob carried verbatim to Manager.HandleTimer.
	//
	// All timers are actually enabled in Save(), in the same transaction that
	// updates the job state.
	AddTimer(c context.Context, delay time.Duration, title string, payload []byte)

	// PrepareTopic create PubSub topic for notifications related to the task and
	// adds given publisher to its ACL.
	//
	// It returns full name of the topic and a token that will be used to route
	// PubSub messages back to the Manager. Topic name and its configuration are
	// controlled by the Engine. The publisher to the topic must be instructed to
	// put the token into 'auth_token' attribute of PubSub messages. The engine
	// will know how to route such messages to Manager.HandleNotification.
	//
	// 'publisher' can be a service account email, or an URL to some luci service.
	// If URL is given, its /auth/api/v1/server/info endpoint will be used to
	// grab a corresponding service account name. All service that use luci auth
	// component expose this endpoint.
	PrepareTopic(c context.Context, publisher string) (topic string, token string, err error)

	// EmitTrigger delivers a given trigger to all jobs which are triggered by
	// current one.
	EmitTrigger(ctx context.Context, trigger *internal.Trigger)

	// Save updates the state of the task in the persistent store.
	//
	// It also schedules all pending timer ticks added via AddTimer.
	//
	// Will be called by the engine after it launches the task. May also be called
	// by the Manager itself, even multiple times (e.g. once to notify that the
	// task has started, a second time to notify it has finished).
	//
	// Returns error if it couldn't save the invocation state. It is fine to
	// ignore it. The engine will attempt to Save the invocation at the end anyway
	// and it will properly handle the error if it happens again.
	Save(c context.Context) error
}

// ControllerReadOnly is a subset of Controller interface with methods that do
// not mutate the job's state.
type ControllerReadOnly interface {
	// JobID returns full job ID the controller is operating on.
	JobID() string

	// InvocationID returns unique identifier of this particular invocation.
	InvocationID() int64

	// Request contains parameters of the invocation supplied when it was created.
	Request() Request

	// Task is proto message with task definition.
	//
	// It is guaranteed to have same underlying type as manager.ProtoMessageType()
	// return value.
	Task() proto.Message

	// DebugLog appends a line to the free form text log of the task.
	DebugLog(format string, args ...interface{})

	// GetClient returns http.Client that is configured to use job's service
	// account credentials to talk to other services.
	GetClient(c context.Context, opts ...auth.RPCOption) (*http.Client, error)
}

// State is mutable portion of the task invocation state.
//
// It can be mutated by TaskManager directly.
type State struct {
	Status   Status // overall status of the invocation, see the enum
	TaskData []byte // storage for TaskManager-specific task data
	ViewURL  string // URL to human readable task page, shows in UI
}

// Request contains parameters of the invocation supplied when it was created.
//
// They are calculated from the pending triggers when the invocation is
// initiated.
type Request struct {
	// TriggeredBy contains ID of an end user that triggered this invocation (e.g
	// through UI or API) or an empty string if it was triggered by the engine or
	// it is a result of a multiple different triggers.
	//
	// Mostly FYI.
	TriggeredBy identity.Identity

	// IncomingTriggers is a list of all triggers consumed by this invocation.
	//
	// Task managers may use them, but they are encouraged to accept task
	// parameters through Properties and Tags instead.
	//
	// Already sorted by time they were emitted (oldest first).
	IncomingTriggers []*internal.Trigger

	// Properties are arbitrary key-value pairs derived from the triggers by the
	// triggering policy function and interpreted by the triggered task manager.
	//
	// TODO(vadimsh): Needs more structure :(
	Properties *structpb.Struct

	// Tags are arbitrary "<key>:<value>" pairs derived from the triggers by the
	// triggering policy function.
	//
	// Primarily used for indexing and correlation of jobs/invocations with
	// each other (including across different services). Task managers can pass
	// them down the stack.
	//
	// TODO(vadimsh): Needs more structure :(
	Tags []string

	// DebugLog is optional multi-line string to put in the invocation debug log
	// when it starts.
	//
	// It is used to report debug information (produced by the engine triggering
	// guts) to the invocation debug log (visible via UI).
	//
	// This field is used internally by the engine. Task managers will never see
	// it set.
	DebugLog string
}

// TriggerIDs extracts list of IDs from IncomingTriggers.
//
// This is useful in tests for asserts.
func (r *Request) TriggerIDs() []string {
	ids := make([]string, len(r.IncomingTriggers))
	for i, t := range r.IncomingTriggers {
		ids[i] = t.Id
	}
	return ids
}

// StringProperty returns a value of string property or "" if no such property
// or it has a different type.
//
// This is useful in tests for asserts.
func (r *Request) StringProperty(k string) string {
	if r.Properties == nil {
		return ""
	}
	prop := r.Properties.Fields[k]
	if prop == nil {
		return ""
	}
	return prop.GetStringValue()
}
