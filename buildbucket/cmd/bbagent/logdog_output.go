// Copyright 2019 The LUCI Authors.
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

package main

import (
	"context"

	"github.com/tetrafolium/luci-go/auth"

	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/logdog/client/butler/output"
	"github.com/tetrafolium/luci-go/logdog/client/butler/output/logdog"
	"github.com/tetrafolium/luci-go/logdog/common/types"
)

func mkLogdogOutput(ctx context.Context, opts *bbpb.BuildInfra_LogDog) (output.Output, error) {
	return (&logdog.Config{
		Auth: auth.NewAuthenticator(ctx, auth.SilentLogin, auth.Options{
			Scopes: []string{
				auth.OAuthScopeEmail,
				"https://www.googleapis.com/auth/cloud-platform",
			},
			MonitorAs: "bbagent/logdog",
		}),
		Host:    opts.Hostname,
		Project: opts.Project,
		Prefix:  types.StreamName(opts.Prefix),
	}).Register(ctx)
}
