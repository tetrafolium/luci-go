// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package frontend

import (
	"fmt"
	"net/http"

	"github.com/tetrafolium/luci-go/milo/buildsource/buildbucket"

	"github.com/tetrafolium/luci-go/buildbucket/deprecated"
	buildbucketpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/templates"
)

// BuilderHandler renders the builder page.
// Does not control access directly because this endpoint delegates all access
// control to Buildbucket with the RPC calls.
func BuilderHandler(c *router.Context) error {
	bid := &buildbucketpb.BuilderID{
		Project: c.Params.ByName("project"),
		Bucket:  c.Params.ByName("bucket"),
		Builder: c.Params.ByName("builder"),
	}
	pageSize := GetLimit(c.Request, -1)

	// TODO(iannucci): standardize to "page token" term instead of cursor.
	pageToken := c.Request.FormValue("cursor")

	// Redirect to short bucket names.
	if _, v2Bucket := deprecated.BucketNameToV2(bid.Bucket); v2Bucket != "" {
		// Parameter "bucket" is v1, e.g. "luci.chromium.try".
		u := *c.Request.URL
		u.Path = fmt.Sprintf("/p/%s/builders/%s/%s", bid.Project, v2Bucket, bid.Builder)
		http.Redirect(c.Writer, c.Request, u.String(), http.StatusMovedPermanently)
		return nil
	}

	if pageSize <= 0 {
		pageSize = 25
	}

	page, err := buildbucket.GetBuilderPage(c.Context, bid, pageSize, pageToken)
	if err != nil {
		return err
	}

	templates.MustRender(c.Context, c.Writer, "pages/builder.html", templates.Args{
		"BuilderPage": page,
	})
	return nil
}
