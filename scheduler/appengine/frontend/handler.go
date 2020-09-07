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

// Binary frontend implements GAE web server for luci-scheduler service.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/appengine"

	"github.com/tetrafolium/luci-go/gae/service/info"

	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/grpcmon"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware"
	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/appengine/tq"

	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"

	"github.com/tetrafolium/luci-go/scheduler/api/scheduler/v1"

	"github.com/tetrafolium/luci-go/scheduler/appengine/apiservers"
	"github.com/tetrafolium/luci-go/scheduler/appengine/catalog"
	"github.com/tetrafolium/luci-go/scheduler/appengine/engine"
	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task/buildbucket"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task/gitiles"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task/noop"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task/urlfetch"
	"github.com/tetrafolium/luci-go/scheduler/appengine/ui"
)

//// Global state. See init().

const adminGroup = "administrators"

var (
	globalDispatcher = tq.Dispatcher{
		// Default "/internal/tasks/" is already used by the old-style task queue
		// router, so pick some other prefix to avoid collisions.
		BaseURL: "/internal/tq/",
	}
	globalCatalog catalog.Catalog
	globalEngine  engine.EngineInternal

	// Known kinds of tasks.
	managers = []task.Manager{
		&buildbucket.TaskManager{},
		&gitiles.TaskManager{},
		&noop.TaskManager{},
		&urlfetch.TaskManager{},
	}
)

//// Helpers.

// requestContext is used to add helper methods.
type requestContext router.Context

// fail writes error message to the log and the response and sets status code.
func (c *requestContext) fail(code int, msg string, args ...interface{}) {
	body := fmt.Sprintf(msg, args...)
	logging.Errorf(c.Context, "HTTP %d: %s", code, body)
	http.Error(c.Writer, body, code)
}

// err sets status to 409 on tq.Retry errors, 500 on transient errors and 202 on
// fatal ones. Returning status code in range [200–299] is the only way to tell
// PubSub to stop redelivering the task.
func (c *requestContext) err(e error, msg string, args ...interface{}) {
	code := 0
	switch {
	case e == nil:
		panic("nil")
	case tq.Retry.In(e):
		code = 409
	case transient.Tag.In(e):
		code = 500
	default:
		code = 202 // fatal error, don't need a redelivery
	}
	args = append(args, e)
	c.fail(code, msg+" - %s", args...)
}

// ok sets status to 200 and puts "OK" in response.
func (c *requestContext) ok() {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(200)
	fmt.Fprintln(c.Writer, "OK")
}

///

var globalInit sync.Once

// initializeGlobalState does one time initialization for stuff that needs
// active GAE context.
func initializeGlobalState(c context.Context) {
	if info.IsDevAppServer(c) {
		// Dev app server doesn't preserve the state of task queues across restarts,
		// need to reset datastore state accordingly, otherwise everything gets stuck.
		if err := globalEngine.ResetAllJobsOnDevServer(c); err != nil {
			logging.Errorf(c, "Failed to reset jobs: %s", err)
		}
	}
}

//// Routes.

func main() {
	// Dev server likes to restart a lot, and upon a restart math/rand seed is
	// always set to 1, resulting in lots of presumably "random" IDs not being
	// very random. Seed it with real randomness.
	mathrand.SeedRandomly()

	// Register tasks handled here. 'NewEngine' call below will register more.
	globalDispatcher.RegisterTask(&internal.ReadProjectConfigTask{}, readProjectConfig, "read-project-config", nil)

	// Setup global singletons.
	globalCatalog = catalog.New()
	for _, m := range managers {
		if err := globalCatalog.RegisterTaskManager(m); err != nil {
			panic(err)
		}
	}
	globalCatalog.RegisterConfigRules(&validation.Rules)

	globalEngine = engine.NewEngine(engine.Config{
		Catalog:        globalCatalog,
		Dispatcher:     &globalDispatcher,
		PubSubPushPath: "/pubsub",
	})

	// Do global init before handling requests.
	base := standard.Base().Extend(
		func(c *router.Context, next router.Handler) {
			globalInit.Do(func() { initializeGlobalState(c.Context) })
			next(c)
		},
	)

	// Setup HTTP routes.
	r := router.New()

	standard.InstallHandlersWithMiddleware(r, base)
	globalDispatcher.InstallRoutes(r, base)
	rpcexplorer.Install(r)

	ui.InstallHandlers(r, base, ui.Config{
		Engine:        globalEngine.PublicAPI(),
		Catalog:       globalCatalog,
		TemplatesPath: "templates",
	})

	r.POST("/pubsub", base, pubsubPushHandler) // auth is via custom tokens
	r.GET("/internal/cron/read-config", base.Extend(gaemiddleware.RequireCron), readConfigCron)

	// Devserver can't accept PubSub pushes, so allow manual pulls instead to
	// simplify local development.
	if appengine.IsDevAppServer() {
		r.GET("/pubsub/pull/:ManagerName/:Publisher", base, pubsubPullHandler)
	}

	// Install RPC servers.
	api := prpc.Server{
		UnaryServerInterceptor: grpcmon.UnaryServerInterceptor,
	}
	scheduler.RegisterSchedulerServer(&api, &apiservers.SchedulerServer{
		Engine:  globalEngine.PublicAPI(),
		Catalog: globalCatalog,
	})
	internal.RegisterAdminServer(&api, apiservers.AdminServerWithACL(globalEngine, globalCatalog, adminGroup))
	discovery.Enable(&api)
	api.InstallHandlers(r, base)

	http.DefaultServeMux.Handle("/", r)
	appengine.Main()
}

// pubsubPushHandler handles incoming PubSub messages.
func pubsubPushHandler(c *router.Context) {
	rc := requestContext(*c)
	body, err := ioutil.ReadAll(rc.Request.Body)
	if err != nil {
		rc.fail(500, "Failed to read the request: %s", err)
		return
	}
	if err = globalEngine.ProcessPubSubPush(rc.Context, body); err != nil {
		rc.err(err, "Failed to process incoming PubSub push")
		return
	}
	rc.ok()
}

// pubsubPullHandler is called on dev server by developer to pull pubsub
// messages from a topic created for a publisher.
func pubsubPullHandler(c *router.Context) {
	rc := requestContext(*c)
	if !appengine.IsDevAppServer() {
		rc.fail(403, "Not a dev server")
		return
	}
	err := globalEngine.PullPubSubOnDevServer(
		rc.Context, rc.Params.ByName("ManagerName"), rc.Params.ByName("Publisher"))
	if err != nil {
		rc.err(err, "Failed to pull PubSub messages")
	} else {
		rc.ok()
	}
}

// readConfigCron grabs a list of projects from the catalog and datastore and
// dispatches task queue tasks to update each project's cron jobs.
func readConfigCron(c *router.Context) {
	rc := requestContext(*c)
	projectsToVisit := map[string]bool{}

	// Visit all projects in the catalog.
	ctx, cancel := context.WithTimeout(rc.Context, 150*time.Second)
	defer cancel()
	projects, err := globalCatalog.GetAllProjects(ctx)
	if err != nil {
		rc.err(err, "Failed to grab a list of project IDs from catalog")
		return
	}
	for _, id := range projects {
		projectsToVisit[id] = true
	}

	// Also visit all registered projects that do not show up in the catalog
	// listing anymore. It will unregister all jobs belonging to them.
	existing, err := globalEngine.GetAllProjects(rc.Context)
	if err != nil {
		rc.err(err, "Failed to grab a list of project IDs from datastore")
		return
	}
	for _, id := range existing {
		projectsToVisit[id] = true
	}

	// Handle each project in its own task to avoid "bad" projects (e.g. ones with
	// lots of jobs) to slow down "good" ones.
	tasks := make([]*tq.Task, 0, len(projectsToVisit))
	for projectID := range projectsToVisit {
		tasks = append(tasks, &tq.Task{
			Payload: &internal.ReadProjectConfigTask{ProjectId: projectID},
		})
	}
	if err = globalDispatcher.AddTask(rc.Context, tasks...); err != nil {
		rc.err(err, "Failed to add tasks to task queue")
	} else {
		rc.ok()
	}
}

// readProjectConfig grabs a list of jobs in a project from catalog, updates
// all changed jobs, adds new ones, disables old ones.
func readProjectConfig(c context.Context, task proto.Message) error {
	projectID := task.(*internal.ReadProjectConfigTask).ProjectId

	ctx, cancel := context.WithTimeout(c, 150*time.Second)
	defer cancel()

	jobs, err := globalCatalog.GetProjectJobs(ctx, projectID)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to query for a list of jobs")
		return err
	}

	if err := globalEngine.UpdateProjectJobs(ctx, projectID, jobs); err != nil {
		logging.WithError(err).Errorf(c, "Failed to update some jobs")
		return err
	}

	return nil
}
