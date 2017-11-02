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

package buildbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/buildbucket"
	bucketApi "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/milo/common"
	"go.chromium.org/luci/milo/common/model"
	"go.chromium.org/luci/server/router"
)

var (
	buildCounter = metric.NewCounter(
		"luci/milo/buildbucket_pubsub/builds",
		"The number of buildbucket builds received by Milo from PubSub",
		nil,
		field.String("bucket"),
		// True for luci build, False for non-luci (ie buildbot) build.
		field.Bool("luci"),
		// Status can be "COMPLETED", "SCHEDULED", or "STARTED"
		field.String("status"),
		// Action can be one of 3 options.
		//   * "Created" - This is the first time Milo heard about this build
		//   * "Modified" - Milo updated some information about this build vs. what
		//     it knew before.
		//   * "Rejected" - Milo was unable to accept this build.
		field.String("action"))
)

// PubSubHandler is a webhook that stores the builds coming in from pubsub.
func PubSubHandler(ctx *router.Context) {
	err := pubSubHandlerImpl(ctx.Context, ctx.Request)
	if err != nil {
		logging.Errorf(ctx.Context, "error while handling pubsub event")
		errors.Log(ctx.Context, err)
	}
	if transient.Tag.In(err) {
		// Transient errors are 500 so that PubSub retries them.
		ctx.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	// No errors or non-transient errors are 200s so that PubSub does not retry
	// them.
	ctx.Writer.WriteHeader(http.StatusOK)
}

// generateSummary takes a decoded buildbucket event and generates
// a model.BuildSummary from it.
//
// This is the portion of the summarization process which cannot fail (i.e. is
// pure-data).
func generateSummary(c context.Context, hostname string, build buildbucket.Build) (*model.BuildSummary, error) {
	// HACK(iannucci,nodir) - crbug.com/776300 - The project and annotation URL
	// should be directly represented on the Build. This is a leaky abstraction;
	// swarming isn't relevant to either value.
	swarmTags := strpair.ParseMap(build.Tags["swarming_tag"])
	project := swarmTags.Get("luci_project")

	ret := &model.BuildSummary{
		ProjectID:     project,
		AnnotationURL: swarmTags.Get("log_location"),
		BuildKey:      MakeBuildKey(c, hostname, build.ID),
		BuilderID:     fmt.Sprintf("buildbucket/%s/%s", build.Bucket, build.Builder),
		BuildID:       fmt.Sprintf("buildbucket/%s", build.Address()),
		BuildSet:      build.Tags[buildbucket.TagBuildSet],

		SelfLink: fmt.Sprintf("/p/%s/builds/b%d", project, build.ID),

		Created: build.CreationTime,
		Summary: model.Summary{
			Start:  build.StartTime,
			End:    build.CompletionTime,
			Status: parseStatus(build.Status),
		},

		Version: build.UpdateTime.UnixNano(),
	}

	if shost, sid := build.Tags.Get("swarming_host"), build.Tags.Get("swarming_task_id"); shost != "" && sid != "" {
		ret.ContextURI = append(ret.ContextURI, fmt.Sprintf("swarming://%s/task/%s", shost, sid))
	}
	// TODO(iannucci,nodir): get the bot context too

	// TODO(iannucci,nodir): support manifests/got_revision
	for _, bset := range build.BuildSets {
		if err := ret.AddManifestKeysFromBuildSet(c, bset); err != nil {
			return nil, errors.Annotate(err, "failed to add manifest keys for %q", bset).Err()
		}
	}

	return ret, nil
}

// pubSubHandlerImpl takes the http.Request, expects to find
// a common.PubSubSubscription JSON object in the Body, containing a bbPSEvent,
// and handles the contents with generateSummary.
func pubSubHandlerImpl(c context.Context, r *http.Request) error {
	// This is the default action. The code below will modify the values of some
	// or all of these parameters.
	isLUCI, bucket, status, action := false, "UNKNOWN", "UNKNOWN", "Rejected"

	defer func() {
		// closure for late binding
		buildCounter.Add(c, 1, bucket, isLUCI, status, action)
	}()

	msg := common.PubSubSubscription{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		// This might be a transient error, e.g. when the json format changes
		// and Milo isn't updated yet.
		return errors.Annotate(err, "could not decode message").Tag(transient.Tag).Err()
	}
	bData, err := msg.GetData()
	if err != nil {
		return errors.Annotate(err, "could not parse pubsub message string").Err()
	}

	event := struct {
		Build    bucketApi.ApiCommonBuildMessage `json:"build"`
		Hostname string                          `json:"hostname"`
	}{}
	if err := json.Unmarshal(bData, &event); err != nil {
		return errors.Annotate(err, "could not parse pubsub message data").Err()
	}

	build := buildbucket.Build{}
	if err := build.ParseMessage(&event.Build); err != nil {
		return errors.Annotate(err, "could not parse buildbucket.Build").Err()
	}

	bucket = build.Bucket
	status = build.Status.String()
	isLUCI = strings.HasPrefix(bucket, "luci.")

	logging.Debugf(c, "Received from %s: build %s/%s (%s)\n%v",
		event.Hostname, bucket, build.Builder, status, build)

	if !isLUCI || build.Builder == "" {
		logging.Infof(c, "This is not an ingestable build, ignoring")
		return nil
	}

	bs, err := generateSummary(c, event.Hostname, build)
	if err != nil {
		return err
	}

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		curBS := &model.BuildSummary{BuildKey: bs.BuildKey}
		switch err := datastore.Get(c, curBS); err {
		case datastore.ErrNoSuchEntity:
			action = "Created"
		case nil:
			action = "Modified"
		default:
			return errors.Annotate(err, "reading current BuildSummary").Err()
		}

		if build.UpdateTime.UnixNano() <= curBS.Version {
			logging.Warningf(c, "current BuildSummary is newer: %d <= %d",
				build.UpdateTime.UnixNano(), curBS.Version)
			return nil
		}

		if err := datastore.Put(c, bs); err != nil {
			return err
		}

		return model.UpdateBuilderForBuild(c, bs)
	}, &datastore.TransactionOptions{XG: true})

	return transient.Tag.Apply(err)
}

// MakeBuildKey returns a new datastore Key for a buildbucket.Build.
//
// There's currently no model associated with this key, but it's used as
// a parent for a model.BuildSummary.
func MakeBuildKey(c context.Context, host string, buildID int64) *datastore.Key {
	return datastore.MakeKey(c,
		"buildbucket.Build", fmt.Sprintf("%s:%d", host, buildID))
}
