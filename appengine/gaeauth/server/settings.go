// Copyright 2016 The LUCI Authors.
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

package server

import (
	"context"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"strings"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server/internal/authdbimpl"
	"github.com/tetrafolium/luci-go/gae/service/info"
	"github.com/tetrafolium/luci-go/server/portal"
)

type settingsPage struct {
	portal.BasePage
}

func (settingsPage) Title(ctx context.Context) (string, error) {
	return "Authorization settings", nil
}

func (settingsPage) Overview(ctx context.Context) (template.HTML, error) {
	serviceAcc, err := info.ServiceAccount(ctx)
	if err != nil {
		return "", err
	}
	return template.HTML(fmt.Sprintf(`<p>LUCI apps should be configured with an
URL to some existing <a href="https://github.com/luci/luci-py/blob/master/appengine/auth_service/README.md">LUCI Auth Service</a>.</p>

<p>This service distributes various authorization related configuration (like
the list of user groups, IP whitelists, OAuth client IDs, etc), which is
required to handle incoming requests. There's usually one instance of this
service per LUCI deployment.</p>

<p>To connect this app to LUCI Auth Service:</p>
<ul>
  <li>
    Figure out what instance of LUCI Auth Service to use. Use a development
    instance of LUCI Auth Service (*-dev) when running code locally or deploying
    to a staging instance.
  </li>
  <li>
    Make sure Google Cloud Pub/Sub API is enabled in the Cloud Console project
    of your app. LUCI Auth Service uses Pub/Sub to propagate changes.
  </li>
  <li>
    Add the service account belonging to your app (<b>%s</b>) to
    <b>auth-trusted-services</b> group on LUCI Auth Service. This authorizes
    your app to receive updates from LUCI Auth Service.
  </li>
  <li>
    Enter the hostname of LUCI Auth Service in the field below and hit
    "Save Settings". It will verify everything is properly configured (or return
    an error message with some clues if not).
  </li>
</ul>`, html.EscapeString(serviceAcc))), nil
}

func (settingsPage) Fields(ctx context.Context) ([]portal.Field, error) {
	return []portal.Field{
		{
			ID:    "AuthServiceURL",
			Title: "Auth service URL",
			Type:  portal.FieldText,
			Validator: func(authServiceURL string) (err error) {
				if authServiceURL != "" {
					_, err = normalizeAuthServiceURL(authServiceURL)
				}
				return err
			},
		},
		{
			ID:    "LatestRev",
			Title: "Latest fetched revision",
			Type:  portal.FieldStatic,
		},
	}, nil
}

func (settingsPage) ReadSettings(ctx context.Context) (map[string]string, error) {
	switch info, err := authdbimpl.GetLatestSnapshotInfo(ctx); {
	case err != nil:
		return nil, err
	case info == nil:
		return map[string]string{
			"AuthServiceURL": "",
			"LatestRev":      "unknown (not configured)",
		}, nil
	default:
		return map[string]string{
			"AuthServiceURL": info.AuthServiceURL,
			"LatestRev":      fmt.Sprintf("%d", info.Rev),
		}, nil
	}
}

func (settingsPage) WriteSettings(ctx context.Context, values map[string]string, who, why string) error {
	authServiceURL := values["AuthServiceURL"]
	if authServiceURL != "" {
		var err error
		authServiceURL, err = normalizeAuthServiceURL(authServiceURL)
		if err != nil {
			return err
		}
	}
	baseURL := "https://" + info.DefaultVersionHostname(ctx)
	return authdbimpl.ConfigureAuthService(ctx, baseURL, authServiceURL)
}

func normalizeAuthServiceURL(authServiceURL string) (string, error) {
	if !strings.Contains(authServiceURL, "://") {
		authServiceURL = "https://" + authServiceURL
	}
	parsed, err := url.Parse(authServiceURL)
	if err != nil {
		return "", fmt.Errorf("bad URL %q - %s", authServiceURL, err)
	}
	if !parsed.IsAbs() || parsed.Path != "" {
		return "", fmt.Errorf("bad URL %q - must be host root URL", authServiceURL)
	}
	return parsed.String(), nil
}

func init() {
	portal.RegisterPage("auth_service", settingsPage{})
}
