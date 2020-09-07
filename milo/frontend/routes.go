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

package frontend

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server"
	"github.com/tetrafolium/luci-go/appengine/gaemiddleware"
	"github.com/tetrafolium/luci-go/appengine/gaemiddleware/standard"
	"github.com/tetrafolium/luci-go/buildbucket/deprecated"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/grpc/discovery"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	milopb "github.com/tetrafolium/luci-go/milo/api/service/v1"
	"github.com/tetrafolium/luci-go/milo/backend"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/xsrf"
	"github.com/tetrafolium/luci-go/server/middleware"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/templates"
	"github.com/tetrafolium/luci-go/web/gowrappers/rpcexplorer"

	"github.com/tetrafolium/luci-go/milo/buildsource/buildbucket"
	"github.com/tetrafolium/luci-go/milo/buildsource/swarming"
	"github.com/tetrafolium/luci-go/milo/common"
)

// Run sets up all the routes and runs the server.
func Run(templatePath string) {
	// Register plain ol' http handlers.
	r := router.New()
	standard.InstallHandlers(r)

	baseMW := standard.Base()
	htmlMW := baseMW.Extend(
		middleware.WithContextTimeout(time.Minute),
		auth.Authenticate(server.CookieAuth, &server.OAuth2Method{Scopes: []string{server.EmailScope}}),
		withAccessClientMiddleware, // This must be called after the auth.Authenticate middleware.
		withGitMiddleware,
		withBuildbucketBuildsClient,
		withBuildbucketBuildersClient,
		templates.WithTemplates(getTemplateBundle(templatePath)),
	)
	xsrfMW := htmlMW.Extend(xsrf.WithTokenCheck)
	projectMW := htmlMW.Extend(buildProjectACLMiddleware(false))
	optionalProjectMW := htmlMW.Extend(buildProjectACLMiddleware(true))
	backendMW := baseMW.Extend(
		middleware.WithContextTimeout(10*time.Minute),
		withBuildbucketBuildsClient)
	cronMW := backendMW.Extend(gaemiddleware.RequireCron, withBuildbucketBuildsClient)

	r.GET("/", htmlMW, frontpageHandler)
	r.GET("/p", baseMW, movedPermanently("/"))
	r.GET("/search", htmlMW, searchHandler)
	r.GET("/opensearch.xml", baseMW, searchXMLHandler)

	// Admin and cron endpoints.
	r.GET("/admin/configs", htmlMW, ConfigsHandler)

	// Cron endpoints
	r.GET("/internal/cron/update-config", cronMW, UpdateConfigHandler)
	r.GET("/internal/cron/update-pools", cronMW, cronHandler(buildbucket.UpdatePools))

	// Builds.
	r.GET("/b/:id", htmlMW, handleError(redirectLUCIBuild))
	r.GET("/p/:project/builds/b:id", baseMW, movedPermanently("/b/:id"))
	r.GET("/p/:project/builders/:bucket/:builder/:numberOrId", optionalProjectMW, handleError(handleLUCIBuild))
	r.GET("/p/:project/builders/:bucket/:builder/:numberOrId/data", optionalProjectMW, handleError(handleLUCIBuildData))
	r.GET("/related_builds/:id", htmlMW, handleError(handleLUCIRelatedBuilds))

	// Console
	r.GET("/p/:project", projectMW, handleError(func(c *router.Context) error {
		return ConsolesHandler(c, c.Params.ByName("project"))
	}))
	r.GET("/p/:project/", baseMW, movedPermanently("/p/:project"))
	r.GET("/p/:project/g", baseMW, movedPermanently("/p/:project"))
	r.GET("/p/:project/g/:group/console", projectMW, handleError(ConsoleHandler))
	r.GET("/p/:project/g/:group", projectMW, redirect("/p/:project/g/:group/console", http.StatusFound))
	r.GET("/p/:project/g/:group/", baseMW, movedPermanently("/p/:project/g/:group"))

	// Builder list
	r.GET("/p/:project/builders", projectMW, handleError(func(c *router.Context) error {
		return BuildersRelativeHandler(c, c.Params.ByName("project"), "")
	}))
	r.GET("/p/:project/g/:group/builders", projectMW, handleError(func(c *router.Context) error {
		return BuildersRelativeHandler(c, c.Params.ByName("project"), c.Params.ByName("group"))
	}))

	// Swarming
	r.GET(swarming.URLBase+"/:id/steps/*logname", htmlMW, handleError(HandleSwarmingLog))
	r.GET(swarming.URLBase+"/:id", htmlMW, handleError(handleSwarmingBuild))
	// Backward-compatible URLs for Swarming:
	r.GET("/swarming/prod/:id/steps/*logname", htmlMW, handleError(HandleSwarmingLog))
	r.GET("/swarming/prod/:id", htmlMW, handleError(handleSwarmingBuild))

	// Buildbucket
	// If these routes change, also change links in common/model/build_summary.go:getLinkFromBuildID
	// and common/model/builder_summary.go:SelfLink.
	r.GET("/p/:project/builders/:bucket/:builder", optionalProjectMW, handleError(BuilderHandler))

	r.GET("/buildbucket/:bucket/:builder", baseMW, redirectFromProjectlessBuilder)

	// LogDog Milo Annotation Streams.
	// This mimics the `logdog://logdog_host/project/*path` url scheme seen on
	// swarming tasks.
	r.GET("/raw/build/:logdog_host/:project/*path", htmlMW, handleError(handleRawPresentationBuild))

	// PubSub subscription endpoints.
	r.POST("/_ah/push-handlers/buildbucket", backendMW, buildbucket.PubSubHandler)

	r.POST("/actions/cancel_build", xsrfMW, handleError(cancelBuildHandler))
	r.POST("/actions/retry_build", xsrfMW, handleError(retryBuildHandler))

	r.GET("/internal_widgets/related_builds/:id", htmlMW, handleError(handleGetRelatedBuildsTable))

	// Config for ResultUI frontend.
	r.GET("/configs.js", baseMW, handleError(configsJSHandler))

	apiMW := baseMW.Extend(withGitMiddleware)
	installAPIRoutes(r, apiMW)

	http.DefaultServeMux.Handle("/", r)
}

func installAPIRoutes(r *router.Router, base router.MiddlewareChain) {
	server := &prpc.Server{}
	milopb.RegisterMiloInternalServer(server, &milopb.DecoratedMiloInternal{
		Service:  &backend.MiloInternalService{},
		Postlude: backend.CommonPostlude,
	})
	discovery.Enable(server)
	rpcexplorer.Install(r)
	server.InstallHandlers(r, base)
}

// handleError is a wrapper for a handler so that the handler can return an error
// rather than call ErrorHandler directly.
// This should be used for handlers that render webpages.
func handleError(handler func(c *router.Context) error) func(c *router.Context) {
	return func(c *router.Context) {
		if err := handler(c); err != nil {
			ErrorHandler(c, err)
		}
	}
}

// cronHandler is a wrapper for cron handlers which do not require template rendering.
func cronHandler(handler func(c context.Context) error) func(c *router.Context) {
	return func(ctx *router.Context) {
		if err := handler(ctx.Context); err != nil {
			logging.WithError(err).Errorf(ctx.Context, "failed to run")
			ctx.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		ctx.Writer.WriteHeader(http.StatusOK)
	}
}

// redirect returns a handler that responds with given HTTP status
// with a location specified by the pathTemplate.
func redirect(pathTemplate string, status int) router.Handler {
	if !strings.HasPrefix(pathTemplate, "/") {
		panic("pathTemplate must start with /")
	}

	return func(c *router.Context) {
		parts := strings.Split(pathTemplate, "/")
		for i, p := range parts {
			if strings.HasPrefix(p, ":") {
				parts[i] = c.Params.ByName(p[1:])
			}
		}
		u := *c.Request.URL
		u.Path = strings.Join(parts, "/")
		http.Redirect(c.Writer, c.Request, u.String(), status)
	}
}

// movedPermanently is a special instance of redirect, returning a handler
// that responds with HTTP 301 (Moved Permanently) with a location specified
// by the pathTemplate.
//
// TODO(nodir,iannucci): delete all usages.
func movedPermanently(pathTemplate string) router.Handler {
	return redirect(pathTemplate, http.StatusMovedPermanently)
}

func redirectFromProjectlessBuilder(c *router.Context) {
	bucket := c.Params.ByName("bucket")
	builder := c.Params.ByName("builder")

	project, _ := deprecated.BucketNameToV2(bucket)
	u := *c.Request.URL
	u.Path = fmt.Sprintf("/p/%s/builders/%s/%s", project, bucket, builder)
	http.Redirect(c.Writer, c.Request, u.String(), http.StatusMovedPermanently)
}

// configsJSHandler serves /configs.js used by ResultUI frontend code.
func configsJSHandler(c *router.Context) error {
	template, err := template.ParseFiles("templates/configs.template.js")
	if err != nil {
		logging.Errorf(c.Context, "Failed to load configs.template.js: %s", err)
		return err
	}

	settings := common.GetSettings(c.Context)

	clientID, err := auth.GetFrontendClientID(c.Context)
	if err != nil {
		return err
	}

	header := c.Writer.Header()
	header.Set("content-type", "application/javascript")

	// The configs file rarely changes, and may block other scripts from running.
	// Set max-age to one hour, stale-while-revalidate to 7 days to improve
	// performance.
	header.Set("cache-control", "max-age=3600,stale-while-revalidate=604800")
	err = template.Execute(c.Writer, map[string]interface{}{
		"ResultDB": map[string]string{
			"Host": settings.GetResultdb().GetHost(),
		},
		"Buildbucket": map[string]string{
			"Host": settings.GetBuildbucket().GetHost(),
		},
		"OAuth2": map[string]string{
			"ClientID": clientID,
		},
	})

	if err != nil {
		logging.Errorf(c.Context, "Failed to execute configs.template.js: %s", err)
		return err
	}

	return nil
}
