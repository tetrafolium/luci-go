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

package config

import (
	"context"
	"net/http"

	"github.com/tetrafolium/luci-go/appengine/gaemiddleware"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/server/router"
)

// importHandler handles HTTP requests to reimport the config.
func importHandler(c *router.Context) {
	c.Writer.Header().Set("Content-Type", "text/plain")

	if err := Import(c.Context); err != nil {
		errors.Log(c.Context, err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

// Import fetches, validates, and applies configs from the config service.
func Import(c context.Context) error {
	const configSet = "services/${appid}"
	if err := importOSes(c, configSet); err != nil {
		return errors.Annotate(err, "failed to import operating systems").Err()
	}
	platformIds, err := importPlatforms(c, configSet)
	if err != nil {
		return errors.Annotate(err, "failed to import platforms").Err()
	}
	if err := importVLANs(c, configSet); err != nil {
		return errors.Annotate(err, "failed to import vlans").Err()
	}
	if err := importDatacenters(c, configSet, platformIds); err != nil {
		return errors.Annotate(err, "failed to import datacenters").Err()
	}
	return nil
}

// InstallHandlers installs handlers for HTTP requests pertaining to configs.
func InstallHandlers(r *router.Router, middleware router.MiddlewareChain) {
	cronMiddleware := middleware.Extend(gaemiddleware.RequireCron)
	r.GET("/internal/cron/import-config", cronMiddleware, importHandler)
}
