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

package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/golang/protobuf/ptypes"

	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/gae/service/datastore"

	buildbucketpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	bbv1 "github.com/tetrafolium/luci-go/common/api/buildbucket/buildbucket/v1"
	"github.com/tetrafolium/luci-go/common/api/gitiles"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/router"

	notifypb "github.com/tetrafolium/luci-go/luci_notify/api/config"
	"github.com/tetrafolium/luci-go/luci_notify/config"
)

func getBuilderID(b *buildbucketpb.Build) string {
	return fmt.Sprintf("%s/%s", b.Builder.Bucket, b.Builder.Builder)
}

// EmailNotify contains information for delivery and personalization of notification emails.
type EmailNotify struct {
	Email         string `json:"email"`
	Template      string `json:"template"`
	MatchingSteps []*buildbucketpb.Step
}

// sortEmailNotify sorts a list of EmailNotify by Email, then Template.
func sortEmailNotify(en []EmailNotify) {
	sort.Slice(en, func(i, j int) bool {
		first := en[i]
		second := en[j]
		emailResult := strings.Compare(first.Email, second.Email)
		if emailResult == 0 {
			return strings.Compare(first.Template, second.Template) < 0
		}
		return emailResult < 0
	})
}

// extractEmailNotifyValues extracts EmailNotify slice from the build.
// TODO(nodir): remove parametersJSON once clients move to properties.
func extractEmailNotifyValues(build *buildbucketpb.Build, parametersJSON string) ([]EmailNotify, error) {
	const propertyName = "email_notify"
	value := build.GetOutput().GetProperties().GetFields()[propertyName]
	if value == nil {
		value = build.GetInput().GetProperties().GetFields()[propertyName]
	}
	if value != nil {
		notifiesPB := value.GetListValue().GetValues()
		ret := make([]EmailNotify, len(notifiesPB))
		for i, notifyPB := range notifiesPB {
			notifyFields := notifyPB.GetStructValue().GetFields()
			ret[i] = EmailNotify{
				Email:    notifyFields["email"].GetStringValue(),
				Template: notifyFields["template"].GetStringValue(),
				// MatchingSteps is left blank, as it is only available for recipients
				// derived from Notifications with step filters.
			}
		}
		return ret, nil
	}

	if parametersJSON == "" {
		return nil, nil
	}
	// json equivalent: {"email_notify": [{"email": "<address>"}, ...]}
	var output struct {
		EmailNotify []EmailNotify `json:"email_notify"`
	}

	if err := json.NewDecoder(strings.NewReader(parametersJSON)).Decode(&output); err != nil {
		return nil, errors.Annotate(err, "invalid msg.ParametersJson").Err()
	}
	return output.EmailNotify, nil
}

// handleBuild processes a Build and sends appropriate notifications.
//
// This function should serve as documentation of the process of going from
// a Build to sent notifications. It also should explicitly handle ACLs and
// stop the process of handling notifications early to avoid wasting compute
// time.
//
// getCheckout produces the associated source checkout for a build, if available.
// It's passed in as a parameter in order to mock it for testing.
//
// history is a function that contacts gitiles to obtain the git history for
// revision ordering purposes. It's passed in as a parameter in order to mock it
// for testing.
func handleBuild(c context.Context, ct CloudTasksClient, build *Build, getCheckout CheckoutFunc, history HistoryFunc) error {
	gCommit := build.Input.GetGitilesCommit()
	if gCommit != nil && gCommit.Id == "" {
		// Ignore builds without an associated commit ID. We can't order them,
		// and otherwise we'll end up making invalid gitiles requests and
		// returning errors. These are usually manually-created builds.
		return nil
	}

	luciProject := build.Builder.Project
	project := &config.Project{Name: luciProject}
	switch ex, err := datastore.Exists(c, project); {
	case err != nil:
		return err
	case !ex.All():
		return nil // This project is not tracked by luci-notify
	}

	// checkout is only used to compute the blamelist
	// As blamelist is not a "critical" feature of luci-notify, if there is an
	// error getting the checkout (mostly because there is no source manifest)
	// we should not throw 500, but just log the error, and inform the builder
	// owner that they are missing source manifest in their builds
	logdogContext, _ := context.WithTimeout(c, LOGDOG_REQUEST_TIMEOUT)
	checkout, err := getCheckout(logdogContext, build)
	if err != nil {
		// TODO (crbug.com/1058190): log the error and let the owner know
		logging.Warningf(c, "Got error when getting source manifest for build %v", err)
	}

	// Get the Builder for the first time, and initialize if there's nothing there.
	builderID := getBuilderID(&build.Build)
	builder := config.Builder{
		ProjectKey: datastore.KeyForObj(c, project),
		ID:         builderID,
	}
	templateInput := &notifypb.TemplateInput{
		BuildbucketHostname: build.BuildbucketHostname,
		Build:               &build.Build,
	}

	// Set up the initial list of recipients, derived from the build.
	recipients := make([]EmailNotify, len(build.EmailNotify))
	copy(recipients, build.EmailNotify)

	// Helper functions for notifying and updating tree closer status.
	notifyNoBlame := func(c context.Context, b config.Builder, oldStatus buildbucketpb.Status) error {
		notifications := Filter(&b.Notifications, oldStatus, &build.Build)
		recipients = append(recipients, ComputeRecipients(c, notifications, nil, nil)...)
		templateInput.OldStatus = oldStatus
		return Notify(c, ct, recipients, templateInput)
	}
	notifyAndUpdateTrees := func(c context.Context, b config.Builder, oldStatus buildbucketpb.Status) error {
		return parallel.FanOutIn(func(ch chan<- func() error) {
			ch <- func() error { return notifyNoBlame(c, b, oldStatus) }
			ch <- func() error { return UpdateTreeClosers(c, build, oldStatus) }
		})
	}

	keepGoing := false
	buildCreateTime, _ := ptypes.Timestamp(build.CreateTime)
	err = datastore.RunInTransaction(c, func(c context.Context) error {
		switch err := datastore.Get(c, &builder); {
		case err == datastore.ErrNoSuchEntity:
			// Even if the builder isn't found, we may still want to notify if the build
			// specifies email addresses to notify.
			logging.Infof(c, "No builder %q found for project %q", builderID, luciProject)
			return Notify(c, ct, recipients, templateInput)
		case err != nil:
			return errors.Annotate(err, "failed to get builder").Tag(transient.Tag).Err()
		}

		// Create a new builder as a copy of the old, updated with build information.
		updatedBuilder := builder
		updatedBuilder.Status = build.Status
		updatedBuilder.BuildTime = buildCreateTime
		if len(checkout) > 0 {
			updatedBuilder.GitilesCommits = checkout.ToGitilesCommits()
		}

		switch {
		case builder.Repository == "":
			// Handle the case where there's no repository being tracked.
			if builder.BuildTime.Before(buildCreateTime) {
				// The build is in-order with respect to build time, so notify normally.
				if err := notifyAndUpdateTrees(c, builder, builder.Status); err != nil {
					return err
				}
				return datastore.Put(c, &updatedBuilder)
			}
			logging.Infof(c, "Found build with old time")

			// Don't update trees, since it's out of order.
			return notifyNoBlame(c, builder, 0)
		case gCommit == nil:
			// If there's no revision information, and the builder has a repository, ignore
			// the build.
			logging.Infof(c, "No revision information found for this build, ignoring...")
			return nil
		}

		// Update the new builder with revision information as we know it's now available.
		updatedBuilder.Revision = gCommit.Id

		// If there's no revision information on the Builder, this means the Builder
		// is uninitialized. Notify about the build as best as we can and then store
		// the updated builder.
		if builder.Revision == "" {
			if err := notifyAndUpdateTrees(c, builder, 0); err != nil {
				return err
			}
			return datastore.Put(c, &updatedBuilder)
		}
		keepGoing = true
		return nil
	}, nil)
	if err != nil || !keepGoing {
		return err
	}

	builderRepoHost, builderRepoProject, _ := gitiles.ParseRepoURL(builder.Repository)
	if builderRepoHost != gCommit.Host || builderRepoProject != gCommit.Project {
		logging.Infof(c, "Builder %s triggered by commit to https://%s/%s"+
			"instead of known https://%s, ignoring...",
			builderID, gCommit.Host, gCommit.Project, builder.Repository)
		return nil
	}

	// Get the revision history for the build-related commit.
	commits, err := history(c, luciProject, gCommit.Host, gCommit.Project, builder.Revision, gCommit.Id)
	if err != nil {
		return errors.Annotate(err, "failed to retrieve git history for input commit").Err()
	}
	if len(commits) == 0 {
		logging.Debugf(c, "Found build with old commit, not updating tree closers")
		return notifyNoBlame(c, builder, 0)
	}

	// Get the blamelist logs, if needed.
	var aggregateLogs Logs
	aggregateRepoWhiteset := BlamelistRepoWhiteset(builder.Notifications)
	if len(aggregateRepoWhiteset) > 0 && len(checkout) > 0 {
		oldCheckout := NewCheckout(builder.GitilesCommits)
		aggregateLogs, err = ComputeLogs(c, luciProject, oldCheckout, checkout.Filter(aggregateRepoWhiteset), history)
		if err != nil {
			return errors.Annotate(err, "failed to compute logs").Err()
		}
	}

	// Update `builder`, and check if we need to store a newer version, then store it.
	oldRepository := builder.Repository
	err = datastore.RunInTransaction(c, func(c context.Context) error {
		switch err := datastore.Get(c, &builder); {
		case err == datastore.ErrNoSuchEntity:
			return errors.New("builder deleted between datastore.Get calls")
		case err != nil:
			return err
		}

		// If the builder's repository got updated in the meanwhile, we need to throw a
		// transient error and retry this whole thing.
		if builder.Repository != oldRepository {
			return errors.Reason("failed to notify because builder repository updated").Tag(transient.Tag).Err()
		}

		// Create a new builder as a copy of the old, updated with build information.
		updatedBuilder := builder
		updatedBuilder.Status = build.Status
		updatedBuilder.BuildTime = buildCreateTime
		updatedBuilder.Revision = gCommit.Id
		if len(checkout) > 0 {
			updatedBuilder.GitilesCommits = checkout.ToGitilesCommits()
		}

		index := commitIndex(commits, builder.Revision)
		outOfOrder := false
		switch {
		// If the revision is not found, we can conclude that the Builder has
		// advanced beyond gCommit.Revision. This is because:
		//   1) builder.Revision only ever moves forward.
		//   2) commits contains the git history up to gCommit.Revision.
		case index < 0:
			logging.Debugf(c, "Found build with old commit during transaction.")
			outOfOrder = true

		// If the revision is current, check build creation time.
		case index == 0 && builder.BuildTime.After(buildCreateTime):
			logging.Debugf(c, "Found build with the same commit but an old time.")
			outOfOrder = true
		}

		if outOfOrder {
			// If the build is out-of-order, we want to ignore only on_change notifications,
			// and not update trees.
			return notifyNoBlame(c, builder, 0)
		}

		// Notify, and include the blamelist.
		n := Filter(&builder.Notifications, builder.Status, &build.Build)
		recipients = append(recipients, ComputeRecipients(c, n, commits[:index], aggregateLogs)...)
		templateInput.OldStatus = builder.Status

		return parallel.FanOutIn(func(ch chan<- func() error) {
			ch <- func() error { return Notify(c, ct, recipients, templateInput) }
			ch <- func() error { return datastore.Put(c, &updatedBuilder) }
			ch <- func() error { return UpdateTreeClosers(c, build, 0) }
		})
	}, nil)
	return errors.Annotate(err, "failed to save builder").Tag(transient.Tag).Err()
}

func newBuildsClient(c context.Context, host, project string) (buildbucketpb.BuildsClient, error) {
	t, err := auth.GetRPCTransport(c, auth.AsProject, auth.WithProject(project))
	if err != nil {
		return nil, err
	}
	opts := prpc.DefaultOptions()
	// TODO(crbug/1016443): remove AcceptContentSubtype defaulting into binary
	// protobuf encoding once Buildbucket server becomes faster.
	opts.AcceptContentSubtype = "json"
	return buildbucketpb.NewBuildsPRPCClient(&prpc.Client{
		C:       &http.Client{Transport: t},
		Host:    host,
		Options: opts,
	}), nil
}

// BuildbucketPubSubHandler is the main entrypoint for a new update from buildbucket's pubsub.
//
// This handler delegates the actual processing of the build to handleBuild.
// Its primary purpose is to unwrap context boilerplate and deal with progress-stopping errors.
func BuildbucketPubSubHandler(ctx *router.Context, ct CloudTasksClient) error {
	c := ctx.Context
	build, err := extractBuild(c, ctx.Request)
	switch {
	case err != nil:
		return errors.Annotate(err, "failed to extract build").Err()

	case build == nil:
		// Ignore.
		return nil

	default:
		return handleBuild(c, ct, build, srcmanCheckout, gitilesHistory)
	}
}

// Build is buildbucketpb.Build along with the parsed 'email_notify' values.
type Build struct {
	BuildbucketHostname string
	buildbucketpb.Build
	EmailNotify []EmailNotify
}

// extractBuild constructs a Build from the PubSub HTTP request.
func extractBuild(c context.Context, r *http.Request) (*Build, error) {
	// sent by pubsub.
	// This struct is just convenient for unwrapping the json message
	var msg struct {
		Message struct {
			Data []byte
		}
		Attributes map[string]interface{}
	}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return nil, errors.Annotate(err, "could not decode message").Err()
	}

	if v, ok := msg.Attributes["version"].(string); ok && v != "v1" {
		// Ignore v2 pubsub messages. TODO(nodir): use v2.
		return nil, nil
	}
	var message struct {
		Build    bbv1.LegacyApiCommonBuildMessage
		Hostname string
	}
	switch err := json.Unmarshal(msg.Message.Data, &message); {
	case err != nil:
		return nil, errors.Annotate(err, "could not parse pubsub message data").Err()
	case !strings.HasPrefix(message.Build.Bucket, "luci."):
		logging.Infof(c, "Received build that isn't part of LUCI, ignoring...")
		return nil, nil
	case message.Build.Status != bbv1.StatusCompleted:
		logging.Infof(c, "Received build that hasn't completed yet, ignoring...")
		return nil, nil
	}

	buildsClient, err := newBuildsClient(c, message.Hostname, message.Build.Project)
	if err != nil {
		return nil, err
	}

	logging.Infof(c, "fetching build %d", message.Build.Id)
	res, err := buildsClient.GetBuild(c, &buildbucketpb.GetBuildRequest{
		Id: message.Build.Id,
		Fields: &field_mask.FieldMask{
			Paths: []string{"*"},
		},
	})
	switch {
	case status.Code(err) == codes.NotFound:
		logging.Warningf(c, "no access to build %d", message.Build.Id)
		return nil, nil
	case err != nil:
		err = grpcutil.WrapIfTransient(err)
		err = errors.Annotate(err, "could not fetch buildbucket build %d", message.Build.Id).Err()
		return nil, err
	}

	emails, err := extractEmailNotifyValues(res, message.Build.ParametersJson)
	if err != nil {
		return nil, errors.Annotate(err, "could not decode email_notify").Err()
	}

	return &Build{
		BuildbucketHostname: message.Hostname,
		Build:               *res,
		EmailNotify:         emails,
	}, nil
}
