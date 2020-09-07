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

package tq

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/data/rand/cryptorand"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/trace"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/openid"
	"github.com/tetrafolium/luci-go/server/router"

	"github.com/tetrafolium/luci-go/server/tq/internal"
	"github.com/tetrafolium/luci-go/server/tq/internal/db"
	"github.com/tetrafolium/luci-go/server/tq/internal/metrics"
	"github.com/tetrafolium/luci-go/server/tq/internal/reminder"
)

// TraceContextHeader is name of a header that contains the trace context of
// a span that produced the task.
//
// It is always set regardless of InheritTraceContext setting. This header
// is read only by Dispatcher itself and exists mostly for FYI purposes.
const TraceContextHeader = "X-Luci-Tq-Trace-Context"

// Dispatcher is a registry of task classes that knows how serialize and route
// them.
//
// There's rarely a need to manually create instances of Dispatcher outside of
// Dispatcher's own tests. You should generally use the global Default
// dispatcher which is configured by the tq server module. Methods of the
// default dispatcher (such as RegisterTaskClass and AddTask) are also available
// as lop-level functions, prefer to use them.
//
// The dispatcher needs a way to submit tasks to Cloud Tasks or Cloud PubSub.
// This is the job of Submitter. It lives in the context, so that it can be
// mocked in tests. In production contexts (setup when using the tq server
// module), the submitter is initialized to be CloudSubmitter. Tests will need
// to provide their own submitter (usually via TestingContext).
//
// TODO(vadimsh): Support consuming PubSub tasks, not just producing them.
type Dispatcher struct {
	// Sweeper knows how to sweep transactional tasks reminders.
	//
	// If not set, Sweep calls will fail.
	Sweeper Sweeper

	// Namespace is a namespace for tasks that use DeduplicationKey.
	//
	// This is needed if two otherwise independent deployments share a single
	// Cloud Tasks instance.
	//
	// Used only for Cloud Tasks tasks. Doesn't affect PubSub tasks.
	//
	// Must be valid per ValidateNamespace. Default is "".
	Namespace string

	// GAE is true when running on Appengine.
	//
	// It alters how tasks are submitted and how incoming HTTP requests are
	// authenticated.
	GAE bool

	// NoAuth can be used to disable authentication on HTTP endpoints.
	//
	// This is useful when running in development mode on localhost or in tests.
	NoAuth bool

	// CloudProject is ID of a project to use to construct full resource names.
	//
	// If not set, "default" will be used, which is pretty useless outside of
	// tests.
	CloudProject string

	// CloudRegion is a ID of a region to use to construct full resource names.
	//
	// If not set, "default" will be used, which is pretty useless outside of
	// tests.
	CloudRegion string

	// DefaultRoutingPrefix is a URL prefix for produced Cloud Tasks.
	//
	// Used only for Cloud Tasks tasks whose TaskClass doesn't provide some custom
	// RoutingPrefix. Doesn't affect PubSub tasks.
	//
	// Default is "/internal/tasks/t/". It means generated Cloud Tasks by will
	// have target URL "/internal/tasks/t/<generated-per-task-suffix>".
	//
	// A non-default value may be valuable if you host multiple dispatchers in
	// a single process. This is a niche use case.
	DefaultRoutingPrefix string

	// DefaultTargetHost is a hostname to dispatch Cloud Tasks to by default.
	//
	// Individual Cloud Tasks task classes may override it with their own specific
	// host. Doesn't affect PubSub tasks.
	//
	// On GAE defaults to the GAE application itself. Elsewhere defaults to
	// "127.0.0.1", which is pretty useless outside of tests.
	DefaultTargetHost string

	// PushAs is a service account email to be used for generating OIDC tokens.
	//
	// Used only for Cloud Tasks tasks. Doesn't affect PubSub tasks.
	//
	// The service account must be within the same project. The server account
	// must have "iam.serviceAccounts.actAs" permission for PushAs account.
	//
	// Optional on GAE when submitting tasks targeting GAE. Elsewhere defaults to
	// "default@example.com", which is pretty useless outside of tests.
	PushAs string

	// AuthorizedPushers is a list of service account emails to accept pushes from
	// in addition to PushAs.
	//
	// This is handy when migrating from one PushAs account to another, or when
	// submitting tasks from one service, but handing them in another.
	//
	// Optional.
	AuthorizedPushers []string

	// SweepInitiationLaunchers is a list of service account emails authorized to
	// launch sweeps via the exposed HTTP endpoint.
	SweepInitiationLaunchers []string

	mu       sync.RWMutex
	clsByID  map[string]*taskClassImpl
	clsByTyp map[protoreflect.MessageType]*taskClassImpl
}

// Sweeper knows how sweep transaction tasks reminders.
type Sweeper interface {
	// sweep either performs the full sweep itself or schedules a task to do it.
	sweep(ctx context.Context, s Submitter, reminderKeySpaceBytes int) error
}

// TaskKind describes how a task class interoperates with transactions.
type TaskKind int

const (
	// NonTransactional is a task kind for tasks that must be enqueued outside
	// of a transaction.
	NonTransactional TaskKind = 0

	// Transactional is a task kind for tasks that must be enqueued only from
	// a transaction.
	//
	// Using transactional tasks requires setting up a sweeper first, see
	// ModuleOptions.
	Transactional TaskKind = 1

	// FollowsContext is a task kind for tasks that are enqueue transactionally
	// if the context is transactional or non-transactionally otherwise.
	//
	// Using transactional tasks requires setting up a sweeper first, see
	// ModuleOptions.
	FollowsContext TaskKind = 2
)

// TaskClass defines how to treat tasks of a specific proto message type.
//
// It assigns some stable ID to a proto message kind and also defines how tasks
// of this kind should be submitted and routed.
//
// The are two backends for tasks: Cloud Tasks and Cloud PubSub. Which one to
// use for a particular task class is defined via mutually exclusive Queue and
// Topic fields.
//
// Refer to Google Cloud documentation for all semantic differences between
// Cloud Tasks and Cloud PubSub. One important difference is that Cloud PubSub
// tasks can't be deduplicated and thus the handler must expect to receive
// duplicates.
type TaskClass struct {
	// ID is unique identifier of this class of tasks.
	//
	// Must match `[a-zA-Z0-9_\-.]{1,100}`.
	//
	// It is used to decide how to deserialize and route the task. Changing IDs of
	// existing task classes is a disruptive operation, make sure the queue is
	// drained first. The dispatcher will permanently fail all Cloud Tasks with
	// unrecognized class IDs.
	//
	// Required.
	ID string

	// Prototype identifies a proto message type of a task payload.
	//
	// Used for its type information only. In particular it is used by AddTask
	// to discover what TaskClass matches the added task. There should be
	// one-to-one correspondence between proto message types and task classes.
	//
	// It is safe to arbitrarily change this type as long as JSONPB encoding of
	// the previous type can be decoded using the new type. The dispatcher will
	// permanently fail Cloud Tasks with bodies it can't deserialize.
	//
	// Required.
	Prototype proto.Message

	// Kind indicates whether the task requires a transaction to be enqueued.
	//
	// Note that using transactional tasks requires setting up a sweeper first,
	// see ModuleOptions.
	//
	// Default is NonTransactional which means that tasks can be enqueued only
	// outside of transactions.
	Kind TaskKind

	// Queue is a name of Cloud Tasks queue to use for the tasks.
	//
	// If set, indicates the task should be submitted through Cloud Tasks API.
	// The queue must exist already in this case. Can't be set together with
	// Topic.
	//
	// It can either be a short name like "default" or a full name like
	// "projects/<project>/locations/<region>/queues/<name>". If it is a full
	// name, it must have the above format or RegisterTaskClass would panic.
	//
	// If it is a short queue name, the full queue name will be constructed using
	// dispatcher's CloudProject and CloudRegion if they are set.
	Queue string

	// Topic is a name of PubSub topic to use for the tasks.
	//
	// If set, indicates the task should be submitted through Cloud PubSub API.
	// The topic must exist already in this case. Can't be set together with
	// Queue.
	//
	// It can either be a short name like "tasks" or a full name like
	// "projects/<project>/topics/<name>". If it is a full name, it must have the
	// above format or RegisterTaskClass would panic.
	Topic string

	// RoutingPrefix is a URL prefix for produced Cloud Tasks.
	//
	// Can only be used for Cloud Tasks task (i.e. only if Queue is also set).
	//
	// Default is dispatcher's DefaultRoutingPrefix which itself defaults to
	// "/internal/tasks/t/". It means generated Cloud Tasks by default will have
	// target URL "/internal/tasks/t/<generated-per-task-suffix>".
	//
	// A non-default value can be used to route Cloud Tasks tasks of a particular
	// class to particular processes, assuming the load balancer is configured
	// accordingly.
	RoutingPrefix string

	// TargetHost is a hostname to dispatch Cloud Tasks to.
	//
	// Can only be used for Cloud Tasks task (i.e. only if Queue is also set).
	//
	// If unset, will use dispatcher's DefaultTargetHost.
	TargetHost string

	// Quiet, if set, instructs the dispatcher not to log bodies of tasks.
	Quiet bool

	// InheritTraceContext, if set, makes the task handler trace span be a child
	// of the span that called AddTask.
	//
	// Ignored for PubSub tasks currently, since there's no easy way to put
	// the trace context header into PubSub request headers.
	//
	// Use it only for "one-off" tasks. Using it for deep chains of tasks usually
	// leads to messy complicated traces.
	InheritTraceContext bool

	// Custom, if given, will be called to generate a custom payload from the
	// task's proto payload.
	//
	// Useful for interoperability with existing code that doesn't use dispatcher
	// or if the tasks are meant to be consumed in some custom way. You'll need to
	// setup the consumer manually, the Dispatcher doesn't know how to handle
	// tasks with custom payload.
	//
	// For Cloud Tasks tasks it is possible to customize HTTP method, relative
	// URI, headers and the request body this way. Other properties of the task
	// (such as the target host, the queue, the task name, authentication headers)
	// are not customizable.
	//
	// For PubSub tasks it is possible to customize only task's body and
	// attributes (via CustomPayload.Meta). Other fields in CustomPayload are
	// ignored.
	//
	// Receives the exact same context as passed to AddTask. If returns nil
	// result, the task will be submitted as usual.
	Custom func(ctx context.Context, m proto.Message) (*CustomPayload, error)

	// Handler will be called by the dispatcher to execute the tasks.
	//
	// The handler will receive the task's payload as a proto message of the exact
	// same type as the type of Prototype. See Handler doc for more info.
	//
	// Populating this field is equivalent to calling AttachHandler after
	// registering the class. It may be left nil if the current process just wants
	// to submit tasks, but not handle them. Some other process would need to
	// attach the handler then to be able to process tasks.
	//
	// The dispatcher will permanently fail tasks if it can't find a handler for
	// them.
	Handler Handler
}

// CustomPayload is returned by TaskClass's Custom, see its doc.
type CustomPayload struct {
	Method      string            // e.g. "GET" or "POST", Cloud Tasks only
	RelativeURI string            // an URI relative to the task's target host, Cloud Tasks only
	Meta        map[string]string // HTTP headers or message attributes to attach
	Body        []byte            // serialized body of the request
}

// TaskClassRef represents a TaskClass registered in a Dispatcher.
type TaskClassRef interface {
	// AttachHandler sets a handler which will be called by the dispatcher to
	// execute the tasks.
	//
	// The handler will receive the task's payload as a proto message of the exact
	// same type as the type of TaskClass's Prototype. See Handler doc for more
	// info.
	//
	// Panics if the class has already a handler attached.
	AttachHandler(h Handler)
}

// Task contains task body and metadata.
type Task struct {
	// Payload is task's payload as well as indicator of its class.
	//
	// Its type will be used to find a matching registered TaskClass which defines
	// how to route and handle the task.
	Payload proto.Message

	// DeduplicationKey is optional unique key used to derive name of the task.
	//
	// If a task of a given class with a given key has already been enqueued
	// recently (within ~1h), this task will be silently ignored.
	//
	// Because there is an extra lookup cost to identify duplicate task names,
	// enqueues of named tasks have significantly increased latency.
	//
	// Can be used only with Cloud Tasks tasks, since PubSub doesn't support
	// deduplication during enqueuing.
	//
	// Named tasks can only be used outside of transactions.
	DeduplicationKey string

	// Title is optional string that identifies the task in server logs.
	//
	// For Cloud Tasks it will also show up as a suffix in task handler URL. It
	// exists exclusively to simplify reading server logs. It serves no other
	// purpose! In particular, it is *not* a task name.
	//
	// Handlers won't ever see it. Pass all information through the payload.
	Title string

	// Delay specifies the duration the Cloud Tasks service must wait before
	// attempting to execute the task.
	//
	// Can be used only with Cloud Tasks tasks. Either Delay or ETA may be set,
	// but not both.
	Delay time.Duration

	// ETA specifies the earliest time a task may be executed.
	//
	// Can be used only with Cloud Tasks tasks. Either Delay or ETA may be set,
	// but not both.
	ETA time.Time
}

// Retry is an error tag used to indicate that the handler wants the task to
// be redelivered later.
//
// See Handler doc for more details.
var Retry = errors.BoolTag{Key: errors.NewTagKey("the task should be retried")}

// Handler is called to handle one enqueued task.
//
// If the returned error is tagged with Retry tag, the request finishes with
// HTTP status 429, indicating to the backend that it should attempt to execute
// the task later (which it may or may not do, depending on retry config). Same
// happens if the error is transient (i.e. tagged with the transient.Tag),
// except the request finishes with HTTP status 500. This difference allows to
// distinguish "expected" retry requests (errors tagged with Retry) from
// "unexpected" ones (errors tagged with transient.Tag).
//
// Retry tag should be used **only** if the handler is fully aware of the retry
// semantics and it **explicitly** wants the task to be retried because it can't
// be processed right now and the handler expects that the retry may help.
//
// For a contrived example, if the handler can process the task only after 2 PM,
// but it is 01:55 PM now, the handler should return an error tagged with Retry
// to indicate this. On the other hand, if the handler failed to process the
// task due to an RPC timeout or some other exceptional transient situation, it
// should return an error tagged with transient.Tag.
//
// Note that it is OK (and often desirable) to tag an error with both Retry and
// transient.Tag. Such errors propagate through the call stack as transient,
// until they reach Dispatcher, which treats them as retriable.
//
// An untagged error (or success) marks the task as "done", it won't be retried.
type Handler func(ctx context.Context, payload proto.Message) error

// ExecutionInfo is parsed from incoming task's metadata.
//
// It is accessible from within task handlers via TaskExecutionInfo(ctx).
type ExecutionInfo struct {
	// ExecutionCount is 0 on a first delivery attempt and increased by 1 for each
	// failed attempt.
	ExecutionCount int

	taskRetryReason       string // X-CloudTasks-TaskRetryReason
	taskPreviousResponse  string // X-CloudTasks-TaskPreviousResponse
	submitterTraceContext string // see TraceContextHeader
}

var executionInfoKey = "github.com/tetrafolium/luci-go/server/tq.ExecutionInfo"

// TaskExecutionInfo returns information about the currently executing task.
//
// Returns nil if called not from a task handler.
func TaskExecutionInfo(ctx context.Context) *ExecutionInfo {
	info, _ := ctx.Value(&executionInfoKey).(*ExecutionInfo)
	return info
}

// ValidateNamespace returns an error if `n` is not a valid namespace name.
//
// An empty string is a valid namespace (denoting the default namespace). Other
// valid namespaces must start with an ASCII letter or '_', contain only
// ASCII letters, digits or '_', and be less than 50 chars in length.
func ValidateNamespace(n string) error {
	if n != "" && !namespaceRe.MatchString(n) {
		return errors.New("must start with a letter or '_' and contain only letters, numbers and '_'")
	}
	return nil
}

// RegisterTaskClass tells the dispatcher how to route and handle tasks of some
// particular type.
//
// Intended to be called during process startup. Panics if there's already
// a registered task class with the same ID or Prototype.
func (d *Dispatcher) RegisterTaskClass(cls TaskClass) TaskClassRef {
	if !taskClassIDRe.MatchString(cls.ID) {
		panic(fmt.Sprintf("bad TaskClass ID %q", cls.ID))
	}
	if cls.Prototype == nil {
		panic("TaskClass Prototype must be set")
	}
	if cls.RoutingPrefix != "" && !strings.HasPrefix(cls.RoutingPrefix, "/") {
		panic("TaskClass RoutingPrefix must start with /")
	}

	var backend taskBackend
	switch {
	case cls.Queue == "" && cls.Topic == "":
		panic("TaskClass must have either Queue or Topic set")
	case cls.Queue != "" && cls.Topic != "":
		panic("TaskClass must have either Queue or Topic set, not both")
	case cls.Queue != "":
		backend = backendCloudTasks
		if strings.ContainsRune(cls.Queue, '/') && !isValidQueue(cls.Queue) {
			panic(fmt.Sprintf("not a valid full queue name %q", cls.Queue))
		}
	case cls.Topic != "":
		backend = backendPubSub
		if strings.ContainsRune(cls.Topic, '/') && !isValidTopic(cls.Topic) {
			panic(fmt.Sprintf("not a valid full topic name %q", cls.Topic))
		}
		if cls.RoutingPrefix != "" {
			panic("PubSub tasks do not support RoutingPrefix")
		}
		if cls.TargetHost != "" {
			panic("PubSub tasks do not support TargetHost")
		}
	}

	typ := cls.Prototype.ProtoReflect().Type()

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.clsByID == nil {
		d.clsByID = make(map[string]*taskClassImpl, 1)
	}
	if d.clsByTyp == nil {
		d.clsByTyp = make(map[protoreflect.MessageType]*taskClassImpl, 1)
	}

	if _, ok := d.clsByID[cls.ID]; ok {
		panic(fmt.Sprintf("TaskClass with ID %q is already registered", cls.ID))
	}
	if _, ok := d.clsByTyp[typ]; ok {
		panic(fmt.Sprintf("TaskClass with Prototype %q is already registered", proto.MessageName(cls.Prototype)))
	}

	impl := &taskClassImpl{
		TaskClass: cls,
		disp:      d,
		protoType: typ,
		backend:   backend,
	}
	d.clsByID[cls.ID] = impl
	d.clsByTyp[typ] = impl
	return impl
}

// AddTask submits a task for later execution.
//
// The task payload type should match some registered TaskClass. Its ID will
// be used to identify the task class in the serialized Cloud Tasks task body.
//
// At some later time, in some other process, the dispatcher will invoke
// a handler attached to the corresponding TaskClass, based on its ID extracted
// from the task body.
//
// If the given context is transactional, inherits the transaction if allowed
// according to the TaskClass's Kind. A transactional task will eventually be
// submitted to Cloud Tasks if and only if the transaction successfully commits.
// This requires a sweeper instance to be running somewhere, see ModuleOptions.
// Note that a failure to submit the task to Cloud Tasks will not abort
// the transaction.
//
// If the task has a DeduplicationKey and there already was a recent task with
// the same TaskClass ID and DeduplicationKey, silently ignores the added task.
// This works only outside of transactions. Using DeduplicationKey with
// transactional tasks results in an error.
//
// Annotates retriable errors with transient.Tag.
func (d *Dispatcher) AddTask(ctx context.Context, task *Task) (err error) {
	sub, err := currentSubmitter(ctx)
	if err != nil {
		return err
	}

	// Start a span annotated with the task's class.
	cls, _, err := d.classByMsg(task.Payload)
	if err != nil {
		return err
	}
	ctx, span := startSpan(ctx, "github.com/tetrafolium/luci-go/server/tq.AddTask", logging.Fields{
		"cr.dev/class": cls.ID,
		"cr.dev/title": task.Title,
	})
	defer func() { span.End(err) }()

	// Prepare a raw request. We'll either submit it right away (for non-tx
	// tasks), or attach it to a reminder and store in the DB for later handling.
	payload, err := d.prepPayload(ctx, cls, task)
	if err != nil {
		return err
	}

	// Examine the context to see if we are inside a transaction.
	db := db.TxnDB(ctx)
	switch cls.Kind {
	case FollowsContext:
		// do nothing, will use `db` if it is non-nil
	case Transactional:
		if db == nil {
			return errors.Reason("enqueuing of tasks %q must be done from inside a transaction", cls.ID).Err()
		}
	case NonTransactional:
		if db != nil {
			return errors.Reason("enqueuing of tasks %q must be done outside of a transaction", cls.ID).Err()
		}
	default:
		panic(fmt.Sprintf("unrecognized TaskKind %v", cls.Kind))
	}

	// If not inside a transaction, submit the task right away.
	if db == nil {
		return internal.Submit(ctx, sub, payload, internal.TxnPathNone)
	}

	// Named transactional tasks are not supported.
	if task.DeduplicationKey != "" {
		return errors.Reason("when enqueuing %q: can't use DeduplicationKey for a transactional task", cls.ID).Err()
	}

	// Otherwise transactionally commit a reminder and schedule a best-effort
	// post-transaction enqueuing of the actual task. If it fails, the sweeper
	// will eventually discover the reminder and enqueue the task. Note that this
	// modifies `payload` with the reminder's ID.
	r, err := d.attachToReminder(ctx, payload)
	if err != nil {
		return errors.Annotate(err, "failed to prepare a reminder").Err()
	}
	span.Attribute("cr.dev/reminder", r.ID)
	if err := db.SaveReminder(ctx, r); err != nil {
		return errors.Annotate(err, "failed to store a transactional enqueue reminder").Err()
	}

	once := int32(0)
	db.Defer(ctx, func(ctx context.Context) {
		if count := atomic.AddInt32(&once, 1); count > 1 {
			panic("transaction defer has already been called")
		}

		// `ctx` here is an outer non-transactional context.
		var err error
		ctx, span := startSpan(ctx, "github.com/tetrafolium/luci-go/server/tq.PostTxn", logging.Fields{
			"cr.dev/class":    cls.ID,
			"cr.dev/title":    task.Title,
			"cr.dev/reminder": r.ID,
		})
		defer func() { span.End(err) }()

		// Attempt to submit the task right away if the reminder is still fresh.
		err = internal.ProcessReminderPostTxn(ctx, sub, db, r)
	})

	return nil
}

// Sweep initiates a sweep of transactional tasks reminders.
//
// It must be called periodically (e.g. once per minute) somewhere in the fleet.
func (d *Dispatcher) Sweep(ctx context.Context) error {
	if d.Sweeper == nil {
		return errors.New("can't sweep: the Sweeper is not set")
	}
	sub, err := currentSubmitter(ctx)
	if err != nil {
		return err
	}
	return d.Sweeper.sweep(ctx, sub, reminderKeySpaceBytes)
}

// InstallTasksRoutes installs tasks HTTP routes under the given prefix.
//
// The exposed HTTP endpoints are called by Cloud Tasks service when it is time
// to execute a task.
func (d *Dispatcher) InstallTasksRoutes(r *router.Router, prefix string) {
	if prefix == "" {
		prefix = "/internal/tasks/"
	} else if !strings.HasPrefix(prefix, "/") {
		panic("the prefix should start with /")
	}

	var mw router.MiddlewareChain
	if !d.NoAuth {
		// Tasks are primarily submitted as `PushAs`, but we also accept all
		// `AuthorizedPushers`.
		pushers := append([]string{d.PushAs}, d.AuthorizedPushers...)
		// On GAE X-Appengine-* headers can be trusted. Check we are being called
		// by Cloud Tasks. We don't care by which queue exactly though. It is
		// easier to move tasks between queues that way.
		header := ""
		if d.GAE {
			header = "X-Appengine-Queuename"
		}
		mw = authMiddleware(pushers, header, func(ctx context.Context) {
			metrics.ServerRejectedCount.Add(ctx, 1, "auth")
		})
	}

	// We don't really care about the exact format of URLs. At the same time
	// accepting all requests under InternalRoutingPrefix is necessary for
	// compatibility with "appengine/tq" which used totally different URL format.
	prefix = strings.TrimRight(prefix, "/") + "/*path"
	r.POST(prefix, mw, func(c *router.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			httpReply(c, 500, "Failed to read the request", err)
			return
		}
		switch err := d.handlePush(c.Context, body, parseHeaders(c.Request.Header)); {
		case err == nil:
			httpReply(c, 200, "OK", nil)
		case Retry.In(err):
			httpReply(c, 429, "The handler asked for retry", err)
		case transient.Tag.In(err):
			httpReply(c, 500, "Transient error", err)
		default:
			httpReply(c, 202, "Fatal error", err)
		}
	})
}

// InstallSweepRoute installs a route that initiates a sweep.
//
// It may be called periodically (e.g. by Cloud Scheduler) to launch sweeps.
func (d *Dispatcher) InstallSweepRoute(r *router.Router, path string) {
	var mw router.MiddlewareChain
	if !d.NoAuth {
		// On GAE X-Appengine-* headers can be trusted. Check we are being called
		// by Cloud Scheduler.
		header := ""
		if d.GAE {
			header = "X-Appengine-Cron"
		}
		mw = authMiddleware(d.SweepInitiationLaunchers, header, nil)
	}

	r.GET(path, mw, func(c *router.Context) {
		switch err := d.Sweep(c.Context); {
		case err == nil:
			httpReply(c, 200, "OK", nil)
		case transient.Tag.In(err):
			httpReply(c, 500, "Transient error", err)
		default:
			httpReply(c, 202, "Fatal error", err)
		}
	})
}

////////////////////////////////////////////////////////////////////////////////

var (
	// namespaceRe is used to validate Dispatcher.Namespace.
	namespaceRe = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]{0,49}$`)
	// taskClassIDRe is used to validate TaskClass.ID.
	taskClassIDRe = regexp.MustCompile(`^[a-zA-Z0-9_\-.]{1,100}$`)
)

const (
	// reminderKeySpaceBytes defines the space of the Reminder Ids.
	//
	// Because Reminder.ID is hex-encoded, actual length is doubled.
	//
	// 16 is chosen is big enough to avoid collisions in practice yet small enough
	// for easier human-debugging of key ranges in queries.
	reminderKeySpaceBytes = 16

	// happyPathMaxDuration caps how long the happy path will be waited for.
	happyPathMaxDuration = time.Minute
)

// defaultHeaders returns headers to add to all submitted tasks.
func defaultHeaders() map[string]string {
	return map[string]string{"Content-Type": "application/json"}
}

// startSpan starts a new span and puts `meta` into its attributes and into
// logger fields.
func startSpan(ctx context.Context, title string, meta logging.Fields) (context.Context, trace.Span) {
	ctx = logging.SetFields(ctx, meta)
	ctx, span := trace.StartSpan(ctx, title)
	for k, v := range meta {
		span.Attribute(k, v)
	}
	return ctx, span
}

// prepPayload converts a task into a reminder.Payload.
func (d *Dispatcher) prepPayload(ctx context.Context, cls *taskClassImpl, t *Task) (*reminder.Payload, error) {
	payload := &reminder.Payload{
		TaskClass: cls.ID,
		Created:   clock.Now(ctx),
		Raw:       t.Payload, // used on a happy path only (essentially only in tests)
	}
	var err error
	switch cls.backend {
	case backendCloudTasks:
		payload.CreateTaskRequest, err = d.prepCloudTasksRequest(ctx, cls, t)
	case backendPubSub:
		payload.PublishRequest, err = d.prepPubSubRequest(ctx, cls, t)
	default:
		panic("impossible")
	}
	return payload, err
}

// prepCloudTasksRequest prepares Cloud Tasks request based on a *Task.
func (d *Dispatcher) prepCloudTasksRequest(ctx context.Context, cls *taskClassImpl, t *Task) (*taskspb.CreateTaskRequest, error) {
	queueID, err := d.queueID(cls.Queue)
	if err != nil {
		return nil, err
	}

	taskID := ""
	if t.DeduplicationKey != "" {
		taskID = queueID + "/tasks/" + cls.taskName(t, d.Namespace)
	}

	var scheduleTime *timestamppb.Timestamp
	switch {
	case !t.ETA.IsZero():
		if t.Delay != 0 {
			return nil, errors.New("bad task: either ETA or Delay should be given, not both")
		}
		scheduleTime = timestamppb.New(t.ETA)
	case t.Delay > 0:
		scheduleTime = timestamppb.New(clock.Now(ctx).Add(t.Delay))
	}

	// E.g. ("example.com", "/internal/tasks/t/<class>[/<title>]").
	// Note: relativeURI is discarded when using custom payload.
	host, relativeURI, err := d.taskTarget(cls, t)
	if err != nil {
		return nil, err
	}

	var payload *CustomPayload
	if cls.Custom != nil {
		if payload, err = cls.Custom(ctx, t.Payload); err != nil {
			return nil, err
		}
	}
	if payload == nil {
		// This is not really a "custom" payload, we are just reusing the struct.
		payload = &CustomPayload{
			Method:      "POST",
			RelativeURI: relativeURI,
			Meta:        defaultHeaders(),
		}
		if payload.Body, err = cls.serialize(t); err != nil {
			return nil, err
		}
	} else {
		// We'll likely be mutating the headers below, make a copy.
		meta := make(map[string]string, len(payload.Meta))
		for k, v := range payload.Meta {
			meta[k] = v
		}
		payload.Meta = meta
	}

	// Inject tracing headers.
	if span := trace.SpanContext(ctx); span != "" {
		payload.Meta[TraceContextHeader] = span
		if cls.InheritTraceContext {
			payload.Meta["X-Cloud-Trace-Context"] = span
		}
	}

	method := taskspb.HttpMethod(taskspb.HttpMethod_value[payload.Method])
	if method == 0 {
		return nil, errors.Reason("bad HTTP method %q", payload.Method).Err()
	}
	if !strings.HasPrefix(payload.RelativeURI, "/") {
		return nil, errors.Reason("bad relative URI %q", payload.RelativeURI).Err()
	}

	// We need to populate one of Task.MessageType oneof alternatives. It has
	// unexported type, so we have to instantiate the message now and then mutate
	// it.
	req := &taskspb.CreateTaskRequest{
		Parent: queueID,
		Task: &taskspb.Task{
			Name:         taskID,
			ScheduleTime: scheduleTime,
			// TODO(vadimsh): Make DispatchDeadline configurable?
		},
	}

	// On GAE we by default push to the GAE itself.
	if host == "" && d.GAE {
		req.Task.MessageType = &taskspb.Task_AppEngineHttpRequest{
			AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
				HttpMethod:  method,
				RelativeUri: payload.RelativeURI,
				Headers:     payload.Meta,
				Body:        payload.Body,
			},
		}
		return req, nil
	}

	// Elsewhere pick up some defaults mostly used only in tests.
	if host == "" {
		host = "127.0.0.1"
	}
	pushAs := d.PushAs
	if d.PushAs == "" {
		pushAs = "default@example.com"
	}

	req.Task.MessageType = &taskspb.Task_HttpRequest{
		HttpRequest: &taskspb.HttpRequest{
			HttpMethod: method,
			Url:        "https://" + host + payload.RelativeURI,
			Headers:    payload.Meta,
			Body:       payload.Body,
			AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
				OidcToken: &taskspb.OidcToken{
					ServiceAccountEmail: pushAs,
				},
			},
		},
	}
	return req, nil
}

// queueID expands `id` into a full queue name if necessary.
func (d *Dispatcher) queueID(id string) (string, error) {
	if strings.HasPrefix(id, "projects/") {
		return id, nil // already full name
	}
	project := d.CloudProject
	if project == "" {
		project = "default"
	}
	region := d.CloudRegion
	if region == "" {
		region = "default"
	}
	return fmt.Sprintf("projects/%s/locations/%s/queues/%s", project, region, id), nil
}

// taskTarget constructs a target URL for a task.
//
// `host` will be "" if no explicit host is configured anywhere. On GAE this
// means "send the task back to the GAE app". On non-GAE this indicates to use
// default "127.0.0.1" which is really usable only in tests.
func (d *Dispatcher) taskTarget(cls *taskClassImpl, t *Task) (host string, relativeURI string, err error) {
	if cls.TargetHost != "" {
		host = cls.TargetHost
	} else {
		host = d.DefaultTargetHost
	}

	pfx := cls.RoutingPrefix
	if pfx == "" {
		pfx = d.DefaultRoutingPrefix
	}
	if pfx == "" {
		pfx = "/internal/tasks/t/"
	}

	if !strings.HasPrefix(pfx, "/") {
		return "", "", errors.Reason("bad routing prefix %q: must start with /", pfx).Err()
	}
	if !strings.HasSuffix(pfx, "/") {
		pfx += "/"
	}

	relativeURI = pfx + cls.ID
	if t.Title != "" {
		relativeURI += "/" + t.Title
	}
	return
}

// prepPubSubRequest prepares Cloud PubSub request based on a *Task.
func (d *Dispatcher) prepPubSubRequest(ctx context.Context, cls *taskClassImpl, t *Task) (*pubsubpb.PublishRequest, error) {
	if t.DeduplicationKey != "" {
		return nil, errors.New("can't use DeduplicationKey with PubSub tasks")
	}
	if t.Delay != 0 || !t.ETA.IsZero() {
		return nil, errors.New("can't use Delay or ETA with PubSub tasks")
	}

	topicID, err := d.topicID(cls.Topic)
	if err != nil {
		return nil, err
	}

	var payload *CustomPayload
	if cls.Custom != nil {
		if payload, err = cls.Custom(ctx, t.Payload); err != nil {
			return nil, err
		}
	}
	if payload == nil {
		// This is not really a "custom" payload, we are just reusing the struct.
		payload = &CustomPayload{}
		if payload.Body, err = cls.serialize(t); err != nil {
			return nil, err
		}
	}

	msg := &pubsubpb.PubsubMessage{
		Data:       payload.Body,
		Attributes: make(map[string]string, len(payload.Meta)+1),
	}
	for k, v := range payload.Meta {
		msg.Attributes[k] = v
	}
	if span := trace.SpanContext(ctx); span != "" {
		msg.Attributes[TraceContextHeader] = span
	}

	return &pubsubpb.PublishRequest{
		Topic:    topicID,
		Messages: []*pubsubpb.PubsubMessage{msg},
	}, nil
}

// topicID expands `id` into a full topic name if necessary.
func (d *Dispatcher) topicID(id string) (string, error) {
	if strings.HasPrefix(id, "projects/") {
		return id, nil // already full name
	}
	project := d.CloudProject
	if project == "" {
		project = "default"
	}
	return fmt.Sprintf("projects/%s/topics/%s", project, id), nil
}

// attachToReminder makes a reminder and attaches the payload to it, thus
// mutating the payload with reminder's ID.
//
// Returns the constructed reminder. It will eventually be stored in the
// database to remind the sweeper to submit the task if best-effort
// post-transactional submit fails.
func (d *Dispatcher) attachToReminder(ctx context.Context, payload *reminder.Payload) (*reminder.Reminder, error) {
	buf := make([]byte, reminderKeySpaceBytes)
	if _, err := io.ReadFull(cryptorand.Get(ctx), buf); err != nil {
		return nil, errors.Annotate(err, "failed to get random bytes").Tag(transient.Tag).Err()
	}

	// Note: length of the generate ID here is different from length of IDs
	// we generate when using DeduplicationKey, so there'll be no collisions
	// between two different sorts of named tasks.
	r := &reminder.Reminder{ID: hex.EncodeToString(buf)}

	// Bound FreshUntil to at most current context deadline.
	r.FreshUntil = clock.Now(ctx).Add(happyPathMaxDuration)
	if deadline, ok := ctx.Deadline(); ok && r.FreshUntil.After(deadline) {
		// TODO(tandrii): allow propagating custom deadline for the async happy
		// path which won't bind the context's deadline.
		r.FreshUntil = deadline
	}
	r.FreshUntil = r.FreshUntil.UTC().Truncate(reminder.FreshUntilPrecision)

	return r, r.AttachPayload(payload)
}

// isValidQueue is true if q looks like "projects/.../locations/.../queues/...".
func isValidQueue(q string) bool {
	chunks := strings.Split(q, "/")
	return len(chunks) == 6 &&
		chunks[0] == "projects" &&
		chunks[1] != "" &&
		chunks[2] == "locations" &&
		chunks[3] != "" &&
		chunks[4] == "queues" &&
		chunks[5] != ""
}

// isValidTopic is true if t looks like "projects/.../topics/...".
func isValidTopic(t string) bool {
	chunks := strings.Split(t, "/")
	return len(chunks) == 4 &&
		chunks[0] == "projects" &&
		chunks[1] != "" &&
		chunks[2] == "topics" &&
		chunks[3] != ""
}

// handlePush handles one incoming task.
//
// Returns errors annotated in the same style as errors from Handler, see its
// doc.
func (d *Dispatcher) handlePush(ctx context.Context, body []byte, info ExecutionInfo) error {
	// See taskClassImpl.serialize().
	env := envelope{}
	if err := json.Unmarshal(body, &env); err != nil {
		metrics.ServerRejectedCount.Add(ctx, 1, "bad_request")
		return errors.Annotate(err, "not a valid JSON body").Err()
	}

	// Find the matching registered task class. Newer tasks always have `class`
	// set. Older ones have `type` instead.
	var cls *taskClassImpl
	var h Handler
	var err error
	if env.Class != "" {
		cls, h, err = d.classByID(env.Class)
	} else if env.Type != "" {
		cls, h, err = d.classByTyp(env.Type)
	} else {
		err = errors.Reason("malformed task body, no class").Err()
	}
	if err != nil {
		logging.Debugf(ctx, "TQ: %s", body)
		metrics.ServerRejectedCount.Add(ctx, 1, "unknown_class")
		return err
	}

	if !cls.Quiet {
		logging.Debugf(ctx, "TQ: %s", body)
		if info.submitterTraceContext != "" {
			logging.Debugf(ctx, "TQ: submitted at %s", info.submitterTraceContext)
		}
		if info.ExecutionCount != 0 {
			logging.Debugf(ctx, "TQ: this is a retry: %d previous attempt(s) already failed", info.ExecutionCount)
			if info.taskRetryReason != "" || info.taskPreviousResponse != "" {
				logging.Debugf(ctx, "TQ: the previous attempt failed with %s: %s", info.taskPreviousResponse, info.taskRetryReason)
			}
		}
	}

	if h == nil {
		metrics.ServerRejectedCount.Add(ctx, 1, "no_handler")
		return errors.Reason("task class %q exists, but has no handler attached", cls.ID).Err()
	}

	msg, err := cls.deserialize(&env)
	if err != nil {
		metrics.ServerRejectedCount.Add(ctx, 1, "bad_payload")
		return errors.Annotate(err, "malformed body of task class %q", cls.ID).Err()
	}

	ctx = context.WithValue(ctx, &executionInfoKey, &info)

	start := clock.Now(ctx)
	err = h(ctx, msg)
	dur := clock.Now(ctx).Sub(start)

	result := "OK"
	switch {
	case Retry.In(err):
		result = "retry"
	case transient.Tag.In(err):
		result = "transient"
	case err != nil:
		result = "fatal"
	}

	metrics.ServerHandledCount.Add(ctx, 1, cls.ID, result)
	metrics.ServerDurationMS.Add(ctx, float64(dur.Milliseconds()), cls.ID, result)

	return err
}

// classByID returns a task class given its ID or an error if no such class.
//
// Reads cls.Handler while under the lock as well, since it may be concurrently
// modified by AttachHandler.
func (d *Dispatcher) classByID(id string) (*taskClassImpl, Handler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if cls := d.clsByID[id]; cls != nil {
		return cls, cls.Handler, nil
	}
	return nil, nil, errors.Reason("no task class with ID %q is registered", id).Err()
}

// classByMsg returns a task class given proto message or an error if no
// such class.
//
// Reads cls.Handler while under the lock as well, since it may be concurrently
// modified by AttachHandler.
func (d *Dispatcher) classByMsg(msg proto.Message) (*taskClassImpl, Handler, error) {
	typ := msg.ProtoReflect().Type()
	d.mu.RLock()
	defer d.mu.RUnlock()
	if cls := d.clsByTyp[typ]; cls != nil {
		return cls, cls.Handler, nil
	}
	return nil, nil, errors.Reason("no task class matching type %q is registered", typ.Descriptor().FullName()).Err()
}

// classByTyp returns a task class given proto message name or an error if no
// such class.
//
// Reads cls.Handler while under the lock as well, since it may be concurrently
// modified by AttachHandler.
func (d *Dispatcher) classByTyp(typ string) (*taskClassImpl, Handler, error) {
	msgTyp, _ := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typ))
	if msgTyp == nil {
		return nil, nil, errors.Reason("no proto message %q is registered", typ).Err()
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	if cls := d.clsByTyp[msgTyp]; cls != nil {
		return cls, cls.Handler, nil
	}
	return nil, nil, errors.Reason("no task class matching type %q is registered", typ).Err()
}

////////////////////////////////////////////////////////////////////////////////

type taskBackend int

const (
	backendCloudTasks taskBackend = 1
	backendPubSub     taskBackend = 2
)

// taskClassImpl knows how to prepare and handle tasks of a particular class.
type taskClassImpl struct {
	TaskClass
	disp      *Dispatcher
	protoType protoreflect.MessageType
	backend   taskBackend
}

// envelope is what we put into all Cloud Tasks.
type envelope struct {
	Class string           `json:"class,omitempty"` // ID of TaskClass
	Type  string           `json:"type,omitempty"`  // for compatibility with appengine/tq
	Body  *json.RawMessage `json:"body"`            // JSONPB-serialized Task.Payload
}

// AttachHandler implements TaskClassRef interface.
func (cls *taskClassImpl) AttachHandler(h Handler) {
	cls.disp.mu.Lock()
	defer cls.disp.mu.Unlock()
	if h == nil {
		panic("The handler must not be nil")
	}
	if cls.Handler != nil {
		panic("The task class has a handler attached already")
	}
	cls.Handler = h
}

// taskName returns a short ID for the task to use to dedup it.
func (cls *taskClassImpl) taskName(t *Task, namespace string) string {
	h := sha256.New()
	h.Write([]byte(namespace))
	h.Write([]byte{0})
	h.Write([]byte(cls.ID))
	h.Write([]byte{0})
	h.Write([]byte(t.DeduplicationKey))
	return hex.EncodeToString(h.Sum(nil))
}

// serialize serializes the task body into JSONPB.
func (cls *taskClassImpl) serialize(t *Task) ([]byte, error) {
	opts := protojson.MarshalOptions{
		Indent:         "\t",
		UseEnumNumbers: true,
	}
	blob, err := opts.Marshal(t.Payload)
	if err != nil {
		return nil, errors.Annotate(err, "failed to serialize %q", proto.MessageName(t.Payload)).Err()
	}
	raw := json.RawMessage(blob)
	return json.MarshalIndent(envelope{
		Class: cls.ID,
		Type:  string(proto.MessageName(t.Payload)),
		Body:  &raw,
	}, "", "\t")
}

// deserialize instantiates a proto message based on its serialized body.
func (cls *taskClassImpl) deserialize(env *envelope) (proto.Message, error) {
	if env.Body == nil {
		return nil, errors.Reason("no body").Err()
	}
	opts := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	msg := cls.protoType.New().Interface()
	if err := opts.Unmarshal(*env.Body, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

////////////////////////////////////////////////////////////////////////////////

// parseHeaders examines headers of the incoming Cloud Tasks push.
func parseHeaders(h http.Header) ExecutionInfo {
	magicHeader := func(key string) string {
		if val := h.Get("X-AppEngine-" + key); val != "" {
			return val
		}
		return h.Get("X-CloudTasks-" + key)
	}

	var execCount int64
	if count := magicHeader("TaskExecutionCount"); count != "" {
		execCount, _ = strconv.ParseInt(count, 10, 32)
	}

	return ExecutionInfo{
		ExecutionCount:        int(execCount),
		taskRetryReason:       magicHeader("TaskRetryReason"),
		taskPreviousResponse:  magicHeader("TaskPreviousResponse"),
		submitterTraceContext: h.Get(TraceContextHeader),
	}
}

// authMiddleware returns a middleware chain that authorizes requests from given
// callers.
//
// Checks OpenID Connect tokens have us in the audience, and the email in them
// is in `callers` list.
//
// If `header` is set, will also accept requests that have this header,
// regardless of its value. This is used to authorize GAE tasks and cron based
// on `X-AppEngine-*` headers.
func authMiddleware(callers []string, header string, rejected func(context.Context)) router.MiddlewareChain {
	oidc := auth.Authenticate(&openid.GoogleIDTokenAuthMethod{
		AudienceCheck: openid.AudienceMatchesHost,
	})
	return router.NewMiddlewareChain(oidc, func(c *router.Context, next router.Handler) {
		if header != "" && c.Request.Header.Get(header) != "" {
			next(c)
			return
		}

		if ident := auth.CurrentIdentity(c.Context); ident.Kind() != identity.Anonymous {
			if checkContainsIdent(callers, ident) {
				next(c)
			} else {
				if rejected != nil {
					rejected(c.Context)
				}
				httpReply(c, 403,
					fmt.Sprintf("Caller %q is not authorized", ident),
					errors.Reason("expecting any of %q", callers).Err(),
				)
			}
			return
		}

		var err error
		if header != "" {
			err = errors.Reason("no OIDC token and no %s header", header).Err()
		} else {
			err = errors.Reason("no OIDC token").Err()
		}
		if rejected != nil {
			rejected(c.Context)
		}
		httpReply(c, 403, "Authentication required", err)
	})
}

// checkContainsIdent is true if `ident` emails matches some of `callers`.
func checkContainsIdent(callers []string, ident identity.Identity) bool {
	if ident.Kind() != identity.User {
		return false // we want service accounts
	}
	email := ident.Email()
	for _, c := range callers {
		if email == c {
			return true
		}
	}
	return false
}

// httpReply writes and logs HTTP response.
//
// `msg` is sent to the caller as is. `err` is logged, but not sent.
func httpReply(c *router.Context, code int, msg string, err error) {
	if err != nil {
		logging.Errorf(c.Context, "%s: %s", msg, err)
	}
	http.Error(c.Writer, msg, code)
}
