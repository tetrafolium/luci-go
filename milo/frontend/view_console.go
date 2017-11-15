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

package frontend

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/context"

	"go.chromium.org/gae/service/memcache"
	"go.chromium.org/gae/service/urlfetch"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/google"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/templates"

	"go.chromium.org/luci/milo/api/config"
	"go.chromium.org/luci/milo/buildsource"
	"go.chromium.org/luci/milo/common"
	"go.chromium.org/luci/milo/common/model"
	"go.chromium.org/luci/milo/frontend/ui"
	"go.chromium.org/luci/milo/git"
)

// getConsoleDef finds the console definition as defined by any project.
// If the user is not a reader of the project, this will return a 404.
// TODO(hinoka): If the user is not a reader of any of of the builders returned,
// that builder will be removed from list of results.
func getConsoleDef(c context.Context, project, name string) (*config.Console, error) {
	cs, err := common.GetConsole(c, project, name)
	if err != nil {
		return nil, err
	}
	// TODO(hinoka): Remove builders that the user does not have access to.
	return &cs.Def, nil
}

// validateFaviconURL checks to see if the URL is well-formed and the host is in
// the whitelist.
func validateFaviconURL(faviconURL string) error {
	parsedFaviconURL, err := url.Parse(faviconURL)
	if err != nil {
		return err
	}
	host := strings.SplitN(parsedFaviconURL.Host, ":", 2)[0]
	if host != "storage.googleapis.com" {
		return fmt.Errorf("%q is not a valid FaviconURL hostname", host)
	}
	return nil
}

func getFaviconURL(c context.Context, def *config.Console) string {
	faviconURL := def.FaviconUrl
	if err := validateFaviconURL(faviconURL); err != nil {
		logging.WithError(err).Warningf(c, "invalid favicon URL")
		faviconURL = ""
	}
	return faviconURL
}

// validateConsoleID checks to see whether a console ID has both a project and
// name, and that the project is the same as the current project being checked.
func validateConsoleID(consoleID, project string) error {
	components := strings.SplitN(consoleID, "/", 2)
	if len(components) != 2 {
		return errors.Reason("console ID must have format <project>/<name>: %s", consoleID).Err()
	}
	if components[0] != project {
		// TODO(hinoka): Support cross-project consoles.
		return errors.Reason("console ID is from a different project").Err()
	}
	return nil
}

type builderRefFactory func(name, shortname string) *ui.BuilderRef

func buildTreeFromDef(def *config.Console, factory builderRefFactory) (*ui.Category, int) {
	// Build console table tree from builders.
	categoryTree := ui.NewCategory("")
	depth := 0
	for _, b := range def.Builders {
		builderRef := factory(b.Name[0], b.CalculateShortname())
		categories := b.ParseCategory()
		if len(categories) > depth {
			depth = len(categories)
		}
		categoryTree.AddBuilder(categories, builderRef)
	}
	return categoryTree, depth
}

func getBuilderSummaries(c context.Context, project, name string) (map[string]*model.BuilderSummary, error) {
	consoleID := project + "/" + name
	consoleSummary, err := buildsource.GetConsoleSummary(c, consoleID)
	if err != nil {
		return nil, err
	}
	builderSummaries := map[string]*model.BuilderSummary{}
	for _, builder := range consoleSummary.Builders {
		builderSummaries[builder.BuilderID] = builder
	}
	return builderSummaries, nil
}

func console(c context.Context, project, name string, limit int) (*ui.Console, error) {
	tStart := clock.Now(c)
	def, err := getConsoleDef(c, project, name)
	if err != nil {
		return nil, err
	}
	commitInfo, err := git.GetHistory(c, def.RepoUrl, def.Ref, limit)
	if err != nil {
		return nil, err
	}
	tGitiles := clock.Now(c)
	logging.Debugf(c, "Loading commits took %s.", tGitiles.Sub(tStart))

	commitNames := make([]string, len(commitInfo.Commits))
	for i, commit := range commitInfo.Commits {
		commitNames[i] = hex.EncodeToString(commit.Hash)
	}

	rows, err := buildsource.GetConsoleRows(c, project, def, commitNames)
	tConsole := clock.Now(c)
	logging.Debugf(c, "Loading the console took a total of %s.", tConsole.Sub(tGitiles))
	if err != nil {
		return nil, err
	}

	// Build list of commits.
	commits := make([]ui.Commit, len(commitInfo.Commits))
	for row, commit := range commitInfo.Commits {
		commits[row] = ui.Commit{
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
			CommitTime:  google.TimeFromProto(commit.CommitTime),
			Repo:        def.RepoUrl,
			Branch:      def.Ref, // TODO(hinoka): Actually this doesn't match, change branch to ref.
			Description: commit.Msg,
			Revision:    ui.NewLink(commitNames[row], def.RepoUrl+"/+/"+commitNames[row], fmt.Sprintf("commit by %s", commit.AuthorEmail)),
		}
	}

	builderSummaries, err := getBuilderSummaries(c, project, name)
	if err != nil {
		return nil, err
	}

	categoryTree, depth := buildTreeFromDef(def, func(name, shortname string) *ui.BuilderRef {
		// Group together all builds for this builder.
		builds := make([]*model.BuildSummary, len(commits))
		id := buildsource.BuilderID(name)
		for row := 0; row < len(commits); row++ {
			if summaries := rows[row].Builds[id]; len(summaries) > 0 {
				builds[row] = summaries[0]
			}
		}
		builder, ok := builderSummaries[name]
		if !ok {
			logging.Warningf(c, "could not find builder summary for %s", name)
		}
		return &ui.BuilderRef{
			Name:      name,
			ShortName: shortname,
			Build:     builds,
			Builder:   builder,
		}
	})

	header, err := consoleHeader(c, project, def.GetHeader())
	if err != nil {
		return nil, err
	}

	return &ui.Console{
		Name:       def.Name,
		Project:    project,
		Header:     header,
		Commit:     commits,
		Table:      *categoryTree,
		MaxDepth:   depth + 1,
		FaviconURL: getFaviconURL(c, def),
	}, nil
}

func consolePreview(c context.Context, project string, def *config.Console) (*ui.Console, error) {
	builderSummaries, err := getBuilderSummaries(c, project, def.Id)
	if err != nil {
		return nil, err
	}
	categoryTree, depth := buildTreeFromDef(def, func(name, shortname string) *ui.BuilderRef {
		builder, ok := builderSummaries[name]
		if !ok {
			logging.Warningf(c, "could not find builder summary for %s", name)
		}
		return &ui.BuilderRef{
			Name:      name,
			ShortName: shortname,
			Builder:   builder,
		}
	})
	return &ui.Console{
		Name:       def.Name,
		Table:      *categoryTree,
		MaxDepth:   depth + 1,
		FaviconURL: getFaviconURL(c, def),
	}, nil
}

// getJSONData fetches data from the given URL into the target, caching it
// for expiration seconds in memcache.
func getJSONData(c context.Context, url string, expiration time.Duration, target interface{}) error {
	// Try memcache first.
	item := memcache.NewItem(c, "url:"+url)
	if err := memcache.Get(c, item); err == nil {
		if err = json.Unmarshal(item.Value(), target); err == nil {
			return nil
		}
		logging.WithError(err).Warningf(c, "couldn't load stored item from datastore")
	} else if err != memcache.ErrCacheMiss {
		logging.WithError(err).Errorf(c, "memcache is having issues")
	}

	// Fetch it from the app.
	transport := urlfetch.Get(c)
	response, err := (&http.Client{Transport: transport}).Get(url)
	if err != nil {
		return errors.Annotate(err, "failed to get data from %q", url).Err()
	}
	defer response.Body.Close()

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Annotate(err, "failed to read response body from %q", url).Err()
	}

	// Parse JSON into target.
	if err := json.Unmarshal(bytes, target); err != nil {
		return errors.Annotate(
			err, "failed to decode JSON %q from %q", bytes, url).Err()
	}

	// Save data into memcache on a best-effort basis.
	item.SetValue(bytes).SetExpiration(expiration)
	if err = memcache.Set(c, item); err != nil {
		logging.WithError(err).Warningf(c, "could not save url data %q into memcache", url)
	}
	return nil
}

// getTreeStatus returns the current tree status from the chromium-status app.
// This never errors, instead it constructs a fake purple TreeStatus
func getTreeStatus(c context.Context, host string) *ui.TreeStatus {
	q := url.Values{}
	q.Add("format", "json")
	url := url.URL{
		Scheme:   "https",
		Host:     host,
		Path:     "current",
		RawQuery: q.Encode(),
	}

	status := &ui.TreeStatus{}
	if err := getJSONData(c, url.String(), 30, status); err != nil {
		// Generate a fake tree status.
		logging.WithError(err).Errorf(c, "loading tree status")
		status = &ui.TreeStatus{
			GeneralState: "maintenance",
			Message:      "could not load tree status",
		}
	}

	return status
}

// getOncallData fetches oncall data and caches it for 10 minutes.
func getOncallData(c context.Context, name, url string) (ui.Oncall, error) {
	result := ui.Oncall{}
	err := getJSONData(c, url, 600, &result)
	// Set the name, this is not loaded from the sheriff JSON.
	result.Name = name
	return result, err
}

func consoleHeader(c context.Context, project string, header *config.Header) (*ui.ConsoleHeader, error) {
	if header == nil {
		return nil, nil
	}

	// Return nil if the header is empty.
	switch {
	case len(header.Oncalls) != 0:
		// continue
	case len(header.Links) != 0:
		// continue
	case len(header.ConsoleGroups) != 0:
		// continue
	default:
		return nil, nil
	}

	// Get oncall data from URLs.
	oncalls := make([]ui.Oncall, len(header.Oncalls))
	err := parallel.FanOutIn(func(gen chan<- func() error) {
		for i, oc := range header.Oncalls {
			i := i
			oc := oc
			gen <- func() error {
				oncall, err := getOncallData(c, oc.Name, oc.Url)
				oncalls[i] = oncall
				return err
			}
		}
	})
	if err != nil {
		logging.WithError(err).Errorf(c, "getting oncalls")
	}

	// Restructure links as resp data structures.
	//
	// This should be a one-to-one transformation.
	links := make([]ui.LinkGroup, len(header.Links))
	for i, linkGroup := range header.Links {
		mlinks := make([]*ui.Link, len(linkGroup.Links))
		for j, link := range linkGroup.Links {
			ariaLabel := fmt.Sprintf("%s in %s", link.Text, linkGroup.Name)
			mlinks[j] = ui.NewLink(link.Text, link.Url, ariaLabel)
		}
		links[i] = ui.LinkGroup{
			Name:  linkGroup.Name,
			Links: mlinks,
		}
	}

	// Set up console summaries for the header.
	consoleGroups := make([]ui.ConsoleGroup, len(header.ConsoleGroups))
	for i, group := range header.ConsoleGroups {
		for _, id := range group.ConsoleIds {
			if err := validateConsoleID(id, project); err != nil {
				return nil, err
			}
		}
		summaries, err := buildsource.GetConsoleSummaries(c, group.ConsoleIds)
		if err != nil {
			return nil, err
		}
		if group.Title != nil {
			ariaLabel := fmt.Sprintf("console group %s", group.Title.Text)
			consoleGroups[i].Title = ui.NewLink(group.Title.Text, group.Title.Url, ariaLabel)
		}
		consoleGroups[i].Consoles = summaries
	}

	var treeStatus *ui.TreeStatus
	if header.TreeStatusHost != "" {
		treeStatus = getTreeStatus(c, header.TreeStatusHost)
		treeStatus.URL = &url.URL{Scheme: "https", Host: header.TreeStatusHost}
	}

	return &ui.ConsoleHeader{
		Oncalls:       oncalls,
		Links:         links,
		ConsoleGroups: consoleGroups,
		TreeStatus:    treeStatus,
	}, nil
}

// consoleRenderer is a wrapper around Console to provide additional methods.
type consoleRenderer struct {
	*ui.Console
}

// ConsoleTable generates the main console table html.
//
// This cannot be generated with templates due to the 'recursive' nature of
// this layout.
func (c consoleRenderer) ConsoleTable() template.HTML {
	var buffer bytes.Buffer
	// The first node is a dummy node
	for _, column := range c.Table.Children {
		column.RenderHTML(&buffer, 1, c.MaxDepth)
	}
	return template.HTML(buffer.String())
}

// ConsoleSummary generates the html for the console's builders in the console list.
//
// It is similar to ConsoleTable, but flattens the structure.
func (c consoleRenderer) ConsoleSummary() template.HTML {
	var buffer bytes.Buffer
	for _, column := range c.Table.Children {
		column.RenderHTML(&buffer, 1, -1)
	}
	return template.HTML(buffer.String())
}

func (c consoleRenderer) BuilderLink(bs *model.BuildSummary) (*ui.Link, error) {
	_, _, builderName, err := buildsource.BuilderID(bs.BuilderID).Split()
	if err != nil {
		return nil, err
	}
	return ui.NewLink(builderName, "/"+bs.BuilderID, fmt.Sprintf("builder %s", builderName)), nil
}

// ConsoleHandler renders the console page.
func ConsoleHandler(c *router.Context) {
	project := c.Params.ByName("project")
	if project == "" {
		ErrorHandler(c, errors.New("Missing Project", common.CodeParameterError))
		return
	}
	name := c.Params.ByName("name")
	const defaultLimit = 50
	const maxLimit = 1000
	limit := defaultLimit
	if tLimit := GetLimit(c.Request, -1); tLimit >= 0 {
		limit = tLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	result, err := console(c.Context, project, name, limit)
	if err != nil {
		ErrorHandler(c, err)
		return
	}

	var reload *int
	if tReload := GetReload(c.Request, -1); tReload >= 0 {
		reload = &tReload
	}

	templates.MustRender(c.Context, c.Writer, "pages/console.html", templates.Args{
		"Console": consoleRenderer{result},
		"Reload":  reload,
	})
}

// ConsolesHandler is responsible for taking a project name and rendering the
// console list page (defined in ./appengine/templates/pages/consoles.html).
func ConsolesHandler(c *router.Context, projectName string) {
	cons, err := common.GetProjectConsoles(c.Context, projectName)
	if err != nil {
		ErrorHandler(c, err)
		return
	}
	type fullConsole struct {
		ProjectID string
		Def       *config.Console
		Render    consoleRenderer
	}
	var consoles []fullConsole
	for _, con := range cons {
		respConsole, err := consolePreview(c.Context, projectName, &con.Def)
		if err != nil {
			logging.WithError(err).Errorf(c.Context, "failed to generate resp console")
			continue
		}
		full := fullConsole{
			ProjectID: con.GetProjectID(),
			Def:       &con.Def,
			Render:    consoleRenderer{respConsole},
		}
		consoles = append(consoles, full)
	}

	var reload *int
	if tReload := GetReload(c.Request, -1); tReload >= 0 {
		reload = &tReload
	}

	templates.MustRender(c.Context, c.Writer, "pages/consoles.html", templates.Args{
		"ProjectName": projectName,
		"Consoles":    consoles,
		"Reload":      reload,
	})
}

func consoleTestData() []common.TestBundle {
	builder := &ui.BuilderRef{
		Name:      "tester",
		ShortName: "tst",
		Build: []*model.BuildSummary{
			{
				Summary: model.Summary{
					Status: model.Success,
				},
			},
			nil,
		},
		Builder: &model.BuilderSummary{
			BuilderID:          "tester",
			LastFinishedStatus: model.InfraFailure,
		},
	}
	root := ui.NewCategory("Root")
	root.AddBuilder([]string{"cat1", "cat2"}, builder)
	return []common.TestBundle{
		{
			Description: "Full console with Header",
			Data: templates.Args{
				"Console": consoleRenderer{&ui.Console{
					Name:    "Test",
					Project: "Testing",
					Header: &ui.ConsoleHeader{
						Oncalls: []ui.Oncall{
							{
								Name: "Sheriff",
								Emails: []string{
									"test@example.com",
									"watcher@example.com",
								},
							},
						},
						Links: []ui.LinkGroup{
							{
								Name: "Some group",
								Links: []*ui.Link{
									ui.NewLink("LiNk", "something", ""),
									ui.NewLink("LiNk2", "something2", ""),
								},
							},
						},
						ConsoleGroups: []ui.ConsoleGroup{
							{
								Title: ui.NewLink("bah", "something2", ""),
								Consoles: []ui.ConsoleSummary{
									{
										Name: ui.NewLink("hurrah", "something2", ""),
										Builders: []*model.BuilderSummary{
											{
												LastFinishedStatus: model.Success,
											},
											{
												LastFinishedStatus: model.Success,
											},
											{
												LastFinishedStatus: model.Failure,
											},
										},
									},
								},
							},
							{
								Consoles: []ui.ConsoleSummary{
									{
										Name: ui.NewLink("hurrah", "something2", ""),
										Builders: []*model.BuilderSummary{
											{
												LastFinishedStatus: model.Success,
											},
										},
									},
								},
							},
						},
					},
					Commit: []ui.Commit{
						{
							AuthorEmail: "x@example.com",
							CommitTime:  time.Date(12, 12, 12, 12, 12, 12, 0, time.UTC),
							Revision:    ui.NewLink("12031802913871659324", "blah blah blah", ""),
							Description: "Me too.",
						},
						{
							AuthorEmail: "y@example.com",
							CommitTime:  time.Date(12, 12, 12, 12, 12, 11, 0, time.UTC),
							Revision:    ui.NewLink("120931820931802913", "blah blah blah 1", ""),
							Description: "I did something.",
						},
					},
					Table:    *root,
					MaxDepth: 3,
				}},
			},
		},
	}
}
