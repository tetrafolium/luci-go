// Copyright 2020 The LUCI Authors.
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

package ledcmd

import (
	"context"
	"net/http"
	"time"

	"github.com/tetrafolium/luci-go/auth"
	swarming "github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/gcloud/googleoauth"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/led/job"
	"github.com/tetrafolium/luci-go/led/job/jobexport"
)

// LaunchSwarmingOpts are the options for LaunchSwarming.
type LaunchSwarmingOpts struct {
	// If true, just generates the NewTaskRequest but does not send it to swarming
	// (SwarmingRpcsTaskRequestMetadata will be nil).
	DryRun bool

	// Must be a unique user identity string and must not be empty.
	//
	// Picking a bad value here means that generated logdog prefixes will
	// possibly collide, and the swarming task's User field will be misreported.
	//
	// See GetUID to obtain a standardized value here.
	UserID string

	// If launched from within a swarming task, this will be the current swarming
	// task's task id to be attached as the parent of the launched task.
	ParentTaskId string

	// A path, relative to ${ISOLATED_OUTDIR} of where to place the final
	// build.proto from this build. If omitted, the build.proto will not be
	// dumped.
	FinalBuildProto string

	KitchenSupport job.KitchenSupport
}

// GetUID derives a user id string from the Authenticator for use with
// LaunchSwarming.
//
// If the given authenticator has the userinfo.email scope, this will be the
// email associated with the Authenticator. Otherwise, this will be
// 'uid:<opaque user id>'.
func GetUID(ctx context.Context, authenticator *auth.Authenticator) (string, error) {
	tok, err := authenticator.GetAccessToken(time.Minute)
	if err != nil {
		return "", errors.Annotate(err, "getting access token").Err()
	}
	info, err := googleoauth.GetTokenInfo(ctx, googleoauth.TokenInfoParams{
		AccessToken: tok.AccessToken,
	})
	if info.Email != "" {
		return info.Email, nil
	}
	return "uid:" + info.Sub, nil
}

// LaunchSwarming launches the given job Definition on swarming, returning the
// NewTaskRequest launched, as well as the launch metadata.
func LaunchSwarming(ctx context.Context, authClient *http.Client, jd *job.Definition, opts LaunchSwarmingOpts) (*swarming.SwarmingRpcsNewTaskRequest, *swarming.SwarmingRpcsTaskRequestMetadata, error) {
	if opts.KitchenSupport == nil {
		opts.KitchenSupport = job.NoKitchenSupport()
	}
	if opts.UserID == "" {
		return nil, nil, errors.New("opts.UserID is empty")
	}

	logging.Infof(ctx, "building swarming task")
	if err := jd.FlattenToSwarming(ctx, opts.UserID, opts.ParentTaskId, opts.KitchenSupport); err != nil {
		return nil, nil, errors.Annotate(err, "failed to flatten job definition to swarming").Err()
	}

	st, err := jobexport.ToSwarmingNewTask(jd.GetSwarming(), jd.UserPayload)
	if err != nil {
		return nil, nil, err
	}
	logging.Infof(ctx, "building swarming task: done")

	if opts.DryRun {
		return st, nil, nil
	}

	swarm := newSwarmClient(authClient, jd.Info().SwarmingHostname())

	logging.Infof(ctx, "launching swarming task")
	req, err := swarm.Tasks.New(st).Do()
	if err != nil {
		return nil, nil, err
	}
	logging.Infof(ctx, "launching swarming task: done")

	return st, req, nil
}
