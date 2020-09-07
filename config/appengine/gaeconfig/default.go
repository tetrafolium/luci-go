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

package gaeconfig

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/config/impl/erroring"
	"github.com/tetrafolium/luci-go/config/vars"
	"github.com/tetrafolium/luci-go/server/auth"

	"github.com/tetrafolium/luci-go/gae/service/info"
)

// devCfgDir is a name of the directory with config files when running in
// local dev appserver model. See Use for details.
const devCfgDir = "devcfg"

// Use installs the default luci-config client.
//
// The client is configured to use luci-config URL specified in the settings,
// using GAE app service account for authentication.
//
// If running in prod, and the settings don't specify luci-config URL, produces
// an implementation that returns a "not configured" error from all methods.
//
// If running on devserver, and the settings don't specify luci-config URL,
// returns a filesystem-based implementation that reads configs from a directory
// (or a symlink) named 'devcfg' located in the GAE module directory (where
// app.yaml is) or its immediate parent directory.
//
// If such directory can not be located, produces an implementation of that
// returns errors from all methods.
//
// Panics if it can't load the settings (should not happen since they are in
// the local memory cache usually).
func Use(c context.Context) context.Context {
	return cfgclient.Use(c, newClientFromSettings(c, mustFetchCachedSettings(c)))
}

// devServerConfigsDir finds a directory with configs to use on the dev server.
func devServerConfigsDir() (string, error) {
	pwd := os.Getenv("PWD") // os.Getwd works funny with symlinks, use PWD
	candidates := []string{
		filepath.Join(pwd, devCfgDir),
		filepath.Join(filepath.Dir(pwd), devCfgDir),
	}
	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		}
	}
	return "", fmt.Errorf("luci-config: could not find local configs in any of %s", candidates)
}

// newClientFromSettings instantiates a LUCI Config client based on settings.
func newClientFromSettings(c context.Context, s *Settings) config.Interface {
	var configsDir string
	if s.ConfigServiceHost == "" && info.IsDevAppServer(c) {
		var err error
		if configsDir, err = devServerConfigsDir(); err != nil {
			return erroring.New(err)
		}
	}
	client, err := cfgclient.New(cfgclient.Options{
		Vars:        &vars.Vars,
		ServiceHost: s.ConfigServiceHost,
		ConfigsDir:  configsDir,
		ClientFactory: func(ctx context.Context) (*http.Client, error) {
			t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
			if err != nil {
				return nil, err
			}
			return &http.Client{Transport: t}, nil
		},
	})
	if err != nil {
		return erroring.New(err)
	}
	return client
}
