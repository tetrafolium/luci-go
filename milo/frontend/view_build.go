// Copyright 2018 The LUCI Authors.
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
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/tetrafolium/luci-go/buildbucket/deprecated"
	buildbucketpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/milo/buildsource/buildbucket"
	"github.com/tetrafolium/luci-go/milo/frontend/ui"
	"github.com/tetrafolium/luci-go/server/auth/xsrf"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/templates"
)

// handleLUCIBuild renders a LUCI build.
func handleLUCIBuild(c *router.Context) error {
	bid := &buildbucketpb.BuilderID{
		Project: c.Params.ByName("project"),
		Bucket:  c.Params.ByName("bucket"),
		Builder: c.Params.ByName("builder"),
	}
	numberOrID := c.Params.ByName("numberOrId")
	forceBlamelist := c.Request.FormValue("blamelist") != ""
	blamelistOpt := buildbucket.GetBlamelist
	if forceBlamelist {
		blamelistOpt = buildbucket.ForceBlamelist
	}

	// Redirect to short bucket names.
	if _, v2Bucket := deprecated.BucketNameToV2(bid.Bucket); v2Bucket != "" {
		// Parameter "bucket" is v1, e.g. "luci.chromium.try".
		u := *c.Request.URL
		u.Path = fmt.Sprintf("/p/%s/builders/%s/%s/%s", bid.Project, v2Bucket, bid.Builder, numberOrID)
		http.Redirect(c.Writer, c.Request, u.String(), http.StatusMovedPermanently)
	}

	br, err := prepareGetBuildRequest(bid, numberOrID)
	if err != nil {
		return err
	}

	bp, err := buildbucket.GetBuildPage(c, br, blamelistOpt)
	return renderBuild(c, bp, err)
}

// handleLUCIBuildData renders the data of the build needed to render a build
// page.
// TODO(crbug.com/1108200): once all the build page data fetching code are moved
// to ResultUI, delete this.
func handleLUCIBuildData(c *router.Context) error {
	bid := &buildbucketpb.BuilderID{
		Project: c.Params.ByName("project"),
		Bucket:  c.Params.ByName("bucket"),
		Builder: c.Params.ByName("builder"),
	}
	numberOrID := c.Params.ByName("numberOrId")
	br, err := prepareGetBuildRequest(bid, numberOrID)
	if err != nil {
		return err
	}
	bp, err := buildbucket.GetBuildPage(c, br, buildbucket.NoBlamelist)
	if err != nil {
		return err
	}
	bpd := ui.BuildPageData{
		BuildPage:               bp,
		CommitLinkHTML:          bp.CommitLinkHTML(),
		Summary:                 bp.Summary(),
		RecipeLink:              bp.RecipeLink(),
		BuildbucketLink:         bp.BuildbucketLink(),
		BuildSets:               bp.BuildSets(),
		BuildSetLinks:           bp.BuildSetLinks(),
		Steps:                   bp.Steps(),
		HumanStatus:             bp.HumanStatus(),
		ShouldShowCanaryWarning: bp.ShouldShowCanaryWarning(),
		InputProperties:         bp.InputProperties(),
		OutputProperties:        bp.OutputProperties(),
		BuilderLink:             bp.BuilderLink(),
		Link:                    bp.Link(),
		Banners:                 bp.Banners(),
		Timeline:                bp.Timeline(),
	}

	if err := json.NewEncoder(c.Writer).Encode(bpd); err != nil {
		logging.Errorf(c.Context, "Failed to JSON encode output - %s", err)
		return err
	}
	c.Writer.Header().Add("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	return nil
}

// handleLUCIRelatedBuilds returns related builds to a given build ID.
// TODO (crbug.com/1112224): convert this to pRPC.
func handleLUCIRelatedBuilds(c *router.Context) error {
	logging.Infof(c.Context, "Inside handleLUCIRelatedBuilds")
	rbt, err := getRelatedBuilds(c)
	if err != nil {
		return err
	}

	if err := json.NewEncoder(c.Writer).Encode(rbt); err != nil {
		logging.Errorf(c.Context, "Failed to JSON encode output - %s", err)
		return err
	}

	c.Writer.Header().Add("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	return nil
}

// renderBuild is a shortcut for rendering build or returning err if it is not nil.
func renderBuild(c *router.Context, bp *ui.BuildPage, err error) error {
	if err != nil {
		return err
	}

	bp.StepDisplayPref = getStepDisplayPrefCookie(c)
	bp.ShowDebugLogsPref = getShowDebugLogsPrefCookie(c)

	templates.MustRender(c.Context, c.Writer, "pages/build.html", templates.Args{
		"BuildPage":      bp,
		"RetryRequestID": rand.Int31(),
		"XsrfTokenField": xsrf.TokenField(c.Context),
		"BannerHTML":     bp.TestResultsOptInHTML(),
	})
	return nil
}

// redirectLUCIBuild redirects to a canonical build URL
// e.g. to /p/{project}/builders/{bucket}/{builder}/{number or id}.
func redirectLUCIBuild(c *router.Context) error {
	id, err := parseBuildID(c.Params.ByName("id"))
	if err != nil {
		return err
	}
	builder, number, err := buildbucket.GetBuilderID(c.Context, id)
	if err != nil {
		return err
	}
	numberOrID := fmt.Sprintf("%d", number)
	if number == 0 {
		numberOrID = fmt.Sprintf("b%d", id)
	}

	u := fmt.Sprintf("/p/%s/builders/%s/%s/%s?%s", builder.Project, builder.Bucket, builder.Builder, numberOrID, c.Request.URL.RawQuery)
	http.Redirect(c.Writer, c.Request, u, http.StatusMovedPermanently)
	return nil
}

func handleGetRelatedBuildsTable(c *router.Context) error {
	rbt, err := getRelatedBuilds(c)
	if err != nil {
		return err
	}
	templates.MustRender(c.Context, c.Writer, "widgets/related_builds_table.html", templates.Args{
		"RelatedBuildsTable": rbt,
	})
	return nil
}

func getRelatedBuilds(c *router.Context) (*ui.RelatedBuildsTable, error) {
	idInput := c.Params.ByName("id")

	id, err := strconv.ParseInt(idInput, 10, 64)
	if err != nil {
		return nil, errors.Annotate(err, "bad build id").Tag(grpcutil.InvalidArgumentTag).Err()
	}
	rbt, err := buildbucket.GetRelatedBuildsTable(c.Context, id)
	if err != nil {
		return nil, errors.Annotate(err, "error when getting related builds table").Err()
	}
	return rbt, nil
}

func getStepDisplayPrefCookie(c *router.Context) ui.StepDisplayPref {
	switch cookie, err := c.Request.Cookie("stepDisplayPref"); err {
	case nil:
		return ui.StepDisplayPref(cookie.Value)
	case http.ErrNoCookie:
		return ui.StepDisplayDefault
	default:
		logging.WithError(err).Errorf(c.Context, "failed to read stepDisplayPref cookie")
		return ui.StepDisplayDefault
	}
}

func getShowDebugLogsPrefCookie(c *router.Context) bool {
	switch cookie, err := c.Request.Cookie("showDebugLogsPref"); err {
	case nil:
		return cookie.Value == "true"
	case http.ErrNoCookie:
		return false
	default:
		logging.WithError(err).Errorf(c.Context, "failed to read showDebugLogsPref cookie")
		return false
	}
}

// parseBuildID parses build ID from string.
func parseBuildID(idStr string) (id int64, err error) {
	// Verify it is an int64.
	id, err = strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		err = errors.Annotate(err, "invalid id").Tag(grpcutil.InvalidArgumentTag).Err()
	}
	return
}

func prepareGetBuildRequest(builderID *buildbucketpb.BuilderID, numberOrID string) (*buildbucketpb.GetBuildRequest, error) {
	br := &buildbucketpb.GetBuildRequest{}
	if strings.HasPrefix(numberOrID, "b") {
		id, err := parseBuildID(numberOrID[1:])
		if err != nil {
			return nil, err
		}
		br.Id = id
	} else {
		number, err := strconv.ParseInt(numberOrID, 10, 32)
		if err != nil {
			return nil, errors.Annotate(err, "bad build number").Tag(grpcutil.InvalidArgumentTag).Err()
		}
		br.Builder = builderID
		br.BuildNumber = int32(number)
	}
	return br, nil
}
