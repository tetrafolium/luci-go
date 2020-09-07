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

package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/common/gcloud/pubsub"
	googlepb "github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/testing/assertions"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	config_mem "github.com/tetrafolium/luci-go/config/impl/memory"
	dm "github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/distributor"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/secrets/testsecrets"
	"github.com/tetrafolium/luci-go/tumble"

	"github.com/golang/protobuf/proto"
	"github.com/smartystreets/goconvey/convey"
)

// Setup creates a new combination of testing and context objects:
//   * ttest - a tumble.Testing to allow you to control tumble's processing
//     state
//   * c - a context which includes a testing distributor registry, testsecrets,
//     as well as everything that tumble.Testing.Context adds (datastore,
//     memcache, etc.)
//   * dist - a fake Distributor implementation with a RunTask method that
//     allows your test to 'run' a scheduled task with the Distributor. This
//     will automatically notify the deps service (by calling `fn`).
//
// You should pass mutate.FinishExecutionFn for fn. It's not done automatically
// in order to break an import cycle. You could provide your own, but YMMV.
//
// This sets the following configuration using the memory configuration mock:
//   services/app/acls.cfg:
//     readers: "reader_group"
//     writers: "writer_group"
//
// Usage:
//   ttest, c, dist := fake.Setup(mutate.FinishExecutionFn)
//   s := deps.NewDecoratedServer()
//   # your tests
func Setup(fn distributor.FinishExecutionFn) (ttest *tumble.Testing, c context.Context, dist *Distributor) {
	ttest = &tumble.Testing{}
	c = ttest.Context()
	c = testsecrets.Use(c)

	c = cfgclient.Use(c, config_mem.New(map[config.Set]config_mem.Files{
		"services/${appid}": {
			"acls.cfg": `
				readers: "reader_group"
				writers: "writer_group"
			`,
		},
	}))
	c = auth.WithState(c, &authtest.FakeState{
		Identity: identity.AnonymousIdentity,
	})
	dist = &Distributor{}
	reg := distributor.NewTestingRegistry(distributor.TestFactoryMap{
		"fakeDistributor": func(c context.Context, cfg *distributor.Config) distributor.D {
			return &BoundDistributor{dist, c, cfg}
		},
	}, fn)
	c = distributor.WithRegistry(c, reg)
	return
}

// DistributorData is the blob of data that the fake.Distributor keeps when DM
// calls its Run method. This is roughly equivalent to the state that
// a distributor (like swarming) would store in its own datastore about a job.
type DistributorData struct {
	NotifyTopic pubsub.Topic
	NotifyAuth  string

	Auth *dm.Execution_Auth
	Desc *dm.Quest_Desc

	State *dm.JsonResult

	done   bool
	abnorm *dm.AbnormalFinish
}

// Task is the detail that the distributor task would get. This is roughly
// equivalent to the input that the swarming task/recipe engine would get.
type Task struct {
	Auth *dm.Execution_Auth
	Desc *dm.Quest_Desc
	// State is read/writable.
	State *dm.JsonResult
}

// Activate does the activation handshake with the provided DepsServer and
// returns an ActivatedTask.
func (t *Task) Activate(c context.Context, s dm.DepsServer) (*ActivatedTask, error) {
	newTok := model.MakeRandomToken(c, 32)
	_, err := s.ActivateExecution(c, &dm.ActivateExecutionReq{
		Auth: t.Auth, ExecutionToken: newTok})
	if err != nil {
		return nil, err
	}

	return &ActivatedTask{
		s,
		c,
		&dm.Execution_Auth{Id: t.Auth.Id, Token: newTok},
		t.Desc,
		t.State,
	}, nil
}

// MustActivate does the same thing as Activate, but panics if err != nil.
func (t *Task) MustActivate(c context.Context, s dm.DepsServer) *ActivatedTask {
	ret, err := t.Activate(c, s)
	panicIf(err)
	return ret
}

// ActivatedTask is like a Task, but exists after calling Task.MustActivate, and
// contains an activated authentication token. This may be used to either add
// new dependencies or to provide a finished result.
//
// The implementation of DepsServer also automatically populates all outgoing
// RPCs with the activated Auth value.
type ActivatedTask struct {
	s dm.DepsServer
	c context.Context

	Auth *dm.Execution_Auth
	Desc *dm.Quest_Desc
	// State is read/writable.
	State *dm.JsonResult
}

// WalkGraph calls the bound DepsServer's WalkGraph method with the activated
// Auth field.
func (t *ActivatedTask) WalkGraph(req *dm.WalkGraphReq) (*dm.GraphData, error) {
	newReq := *req
	newReq.Auth = t.Auth
	return t.s.WalkGraph(t.c, &newReq)
}

// EnsureGraphData calls the bound DepsServer's EnsureGraphData method with the
// activated Auth field in ForExecution.
func (t *ActivatedTask) EnsureGraphData(req *dm.EnsureGraphDataReq) (*dm.EnsureGraphDataRsp, error) {
	newReq := *req
	newReq.ForExecution = t.Auth
	return t.s.EnsureGraphData(t.c, &newReq)
}

// DepOn is a shorthand for EnsureGraphData which allows you to depend on
// multiple existing quests by attempt id. The definitions for these quests must
// already have been added to the deps server (probably with an EnsureGraphData
// call).
func (t *ActivatedTask) DepOn(to ...*dm.Attempt_ID) (bool, error) {
	req := &dm.EnsureGraphDataReq{RawAttempts: dm.NewAttemptList(nil)}
	req.RawAttempts.AddAIDs(to...)

	rsp, err := t.EnsureGraphData(req)
	return rsp.ShouldHalt, err
}

// MustDepOn is the same as DepOn but will panic if DepOn would have returned
// a non-nil error.
func (t *ActivatedTask) MustDepOn(to ...*dm.Attempt_ID) (halt bool) {
	halt, err := t.DepOn(to...)
	panicIf(err)
	return
}

// Finish calls FinishAttempt with the provided JSON body and optional
// expiration time.
//
// This will panic if you provide more than one expiration time (so don't do
// that).
func (t *ActivatedTask) Finish(resultJSON string, expire ...time.Time) {
	req := &dm.FinishAttemptReq{
		Auth: t.Auth,
		Data: dm.NewJsonResult(resultJSON),
	}
	switch len(expire) {
	case 0:
	case 1:
		req.Data.Expiration = googlepb.NewTimestamp(expire[0])
	default:
		panic("may only specify 0 or 1 expire values")
	}

	_, err := t.s.FinishAttempt(t.c, req)
	panicIf(err)
}

// WalkShouldReturn is a shorthand for the package-level WalkShouldReturn which
// binds the activated auth to the WalkGraph request, but otherwise behaves
// identically.
//
// Use this method like:
//   req := &dm.WalkGraphReq{...}
//   So(req, activated.WalkShouldReturn, &dm.GraphData{
//     ...
//   })
func (t *ActivatedTask) WalkShouldReturn(request interface{}, expect ...interface{}) string {
	r := *request.(*dm.WalkGraphReq)
	r.Auth = t.Auth
	return WalkShouldReturn(t.c, t.s)(&r, expect...)
}

// Distributor implements distributor.D, and provides a method (RunTask) to
// allow a test to actually run a task which has been scheduled on this
// Distributor, and correctly notify the deps server that the execution is
// complete.
type Distributor struct {
	// RunError can be set to make Run return this error when it's invoked.
	RunError error
	// This can be set to turn the distributor into a polling-based distributor.
	PollbackTime time.Duration

	sync.Mutex
	tasks map[distributor.Token]*DistributorData
}

// BoundDistributor binds the fake.Distributor to a Context and
// a distributor.Config. It implements distributor.D.
type BoundDistributor struct {
	*Distributor
	c   context.Context
	cfg *distributor.Config
}

// MakeToken makes a distributor Token out of an Execution_ID. In this
// implementation of a Distributor there's a 1:1 mapping between Execution_ID
// and distributor task. This is not always the case for real distributor
// implementations.
func MakeToken(eid *dm.Execution_ID) distributor.Token {
	return distributor.Token(fmt.Sprintf("fakeDistributor:%s|%d|%d", eid.Quest,
		eid.Attempt, eid.Id))
}

// Run implements distributor.D
func (d *BoundDistributor) Run(desc *dm.Quest_Desc, exAuth *dm.Execution_Auth, prev *dm.JsonResult) (tok distributor.Token, pollbackTime time.Duration, err error) {
	if err = d.RunError; err != nil {
		return
	}
	pollbackTime = d.PollbackTime

	tok = MakeToken(exAuth.Id)

	tsk := &DistributorData{
		Auth:  exAuth,
		Desc:  desc,
		State: prev,
	}
	tsk.NotifyTopic, tsk.NotifyAuth, err = d.cfg.PrepareTopic(d.c, exAuth.Id)
	panicIf(err)

	d.Lock()
	defer d.Unlock()
	if d.tasks == nil {
		d.tasks = map[distributor.Token]*DistributorData{}
	}
	d.tasks[tok] = tsk
	return
}

// Cancel implements distributor.D
func (d *BoundDistributor) Cancel(_ *dm.Quest_Desc, tok distributor.Token) (err error) {
	d.Lock()
	defer d.Unlock()
	if tsk, ok := d.tasks[tok]; ok {
		tsk.done = true
		tsk.abnorm = &dm.AbnormalFinish{
			Status: dm.AbnormalFinish_CANCELLED,
			Reason: "cancelled via Cancel()"}
	} else {
		err = fmt.Errorf("MISSING task %q", tok)
	}
	return
}

// GetStatus implements distributor.D
func (d *BoundDistributor) GetStatus(_ *dm.Quest_Desc, tok distributor.Token) (rslt *dm.Result, err error) {
	d.Lock()
	defer d.Unlock()
	if tsk, ok := d.tasks[tok]; ok {
		if tsk.done {
			if tsk.abnorm != nil {
				rslt = &dm.Result{AbnormalFinish: tsk.abnorm}
			} else {
				rslt = &dm.Result{Data: tsk.State}
			}
		}
	} else {
		rslt = &dm.Result{
			AbnormalFinish: &dm.AbnormalFinish{
				Status: dm.AbnormalFinish_MISSING,
				Reason: fmt.Sprintf("unknown token: %s", tok)},
		}
	}
	return
}

// FakeURLPrefix is the url that all fake InfoURLs are prefixed with.
const FakeURLPrefix = "https://info.example.com/"

// InfoURL builds a fake InfoURL for the given Execution_ID
func InfoURL(e *dm.Execution_ID) string {
	return FakeURLPrefix + string(MakeToken(e))
}

// InfoURL implements distributor.D
func (d *BoundDistributor) InfoURL(tok distributor.Token) string {
	return FakeURLPrefix + string(tok)
}

// HandleNotification implements distributor.D
func (d *BoundDistributor) HandleNotification(q *dm.Quest_Desc, n *distributor.Notification) (rslt *dm.Result, err error) {
	return d.GetStatus(q, distributor.Token(n.Attrs["token"]))
}

// HandleTaskQueueTask is not implemented, and shouldn't be needed for most
// tests. It could be implemented if some new test required it, however.
func (d *BoundDistributor) HandleTaskQueueTask(r *http.Request) ([]*distributor.Notification, error) {
	panic("not implemented")
}

// Validate implements distributor.D (by returning a nil error for every
// payload).
func (d *BoundDistributor) Validate(payload string) error {
	return nil
}

// RunTask allows you to run the task associated with the provided execution id.
//
// If the task corresponding to `eid` returns an error, or if the distributor
// itself actually has an error, this method will return an error. Notably, if
// `cb` returns an error, it will simply mark the corresponding task as FAILED,
// but will return nil here.
//
// If the task exists and hasn't been run yet, cb will be called, and can do
// anything that you may want to a test to do. Think of the callback as the
// recipe engine; it has the opportunity to do anything it wants to, interact
// with the deps server (or not), succeed (or not), etc.
//
// If the callback needs to maintain state between executions, Task.State is
// read+write; when the callback exits, the final value of Task.State will be
// passed back to the DM instance under test. A re-execution of the attempt will
// start with the new value.
func (d *Distributor) RunTask(c context.Context, eid *dm.Execution_ID, cb func(*Task) error) (err error) {
	tok := MakeToken(eid)

	d.Lock()
	tsk := d.tasks[tok]
	if tsk == nil {
		err = fmt.Errorf("cannot RunTask(%q): doesn't exist", tok)
	} else {
		if tsk.done {
			err = fmt.Errorf("cannot RunTask(%q): running twice", tok)
		} else {
			tsk.done = true
		}
	}
	d.Unlock()

	if err != nil {
		return
	}

	abnorm := (*dm.AbnormalFinish)(nil)

	usrTsk := &Task{
		tsk.Auth,
		tsk.Desc,
		tsk.State,
	}

	defer func() {
		d.Lock()
		{
			tsk.abnorm = abnorm
			tsk.State = usrTsk.State

			if r := recover(); r != nil {
				tsk.abnorm = &dm.AbnormalFinish{
					Status: dm.AbnormalFinish_CRASHED,
					Reason: fmt.Sprintf("caught panic: %q", r),
				}
			}
		}
		d.Unlock()

		err = tumble.RunMutation(c, &distributor.NotifyExecution{
			CfgName: "fakeDistributor",
			Notification: &distributor.Notification{
				ID:    tsk.Auth.Id,
				Attrs: map[string]string{"token": string(tok)}},
		})
	}()

	err = cb(usrTsk)
	if err != nil {
		err = nil
		abnorm = &dm.AbnormalFinish{
			Status: dm.AbnormalFinish_FAILED,
			Reason: fmt.Sprintf("cb error: %q", err),
		}
	}
	return
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

var _ distributor.D = (*BoundDistributor)(nil)

// QuestDesc generates a normalized generic QuestDesc of the form:
//   Quest_Desc{
//     DistributorConfigName: "fakeDistributor",
//     Parameters:            `{"name":"$name"}`,
//     DistributorParameters: "{}",
//   }
func QuestDesc(name string) *dm.Quest_Desc {
	params, err := json.Marshal(struct {
		Name string `json:"name"`
	}{name})
	panicIf(err)
	desc := &dm.Quest_Desc{
		DistributorConfigName: "fakeDistributor",
		Parameters:            string(params),
		DistributorParameters: "{}",
	}
	panicIf(desc.Normalize())
	return desc
}

// WalkShouldReturn is a convey-style assertion factory to assert that a given
// WalkGraph request object results in the provided GraphData.
//
// If keepTimestamps (a singular, optional boolean) is provided and true,
// WalkShouldReturn will not remove timestamps from the compared GraphData. If
// it is absent or false, GraphData.PurgeTimestamps will be called on the
// returned GraphData before comparing it to the expected value.
//
// Use this function like:
//   req := &dm.WalkGraphReq{...}
//   So(req, WalkShouldReturn(c, s), &dm.GraphData{
//     ...
//   })
func WalkShouldReturn(c context.Context, s dm.DepsServer, keepTimestamps ...bool) func(request interface{}, expect ...interface{}) string {
	kt := len(keepTimestamps) > 0 && keepTimestamps[0]
	if len(keepTimestamps) > 1 {
		panic("may only specify 0 or 1 keepTimestamps values")
	}

	normalize := func(gd *dm.GraphData) *dm.GraphData {
		data, err := proto.Marshal(gd)
		panicIf(err)
		ret := &dm.GraphData{}

		panicIf(proto.Unmarshal(data, ret))

		if !kt {
			ret.PurgeTimestamps()
		}
		return ret
	}

	return func(request interface{}, expect ...interface{}) string {
		r := request.(*dm.WalkGraphReq)
		if len(expect) != 1 {
			panic(fmt.Errorf("expected 1 arg on rhs, got %d", len(expect)))
		}
		e := expect[0].(*dm.GraphData)
		ret, err := s.WalkGraph(c, r)
		if nilExpect := assertions.ShouldErrLike(err, nil); nilExpect != "" {
			return nilExpect
		}
		return convey.ShouldResemble(normalize(ret), e)
	}
}
