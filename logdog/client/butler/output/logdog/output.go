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

package logdog

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/data/rand/cryptorand"
	"github.com/tetrafolium/luci-go/common/errors"
	ps "github.com/tetrafolium/luci-go/common/gcloud/pubsub"
	"github.com/tetrafolium/luci-go/common/lhttp"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/grpc/prpc"
	api "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/registration/v1"
	"github.com/tetrafolium/luci-go/logdog/client/butler/output"
	"github.com/tetrafolium/luci-go/logdog/common/types"

	"cloud.google.com/go/pubsub"

	"google.golang.org/api/option"
)

// Scopes returns the set of OAuth scopes required for this Output.
func Scopes() []string {
	// E-mail scope needed for Coordinator authentication.
	scopes := []string{auth.OAuthScopeEmail}
	// Publisher scope needed to publish to Pub/Sub transport.
	scopes = append(scopes, ps.PublisherScopes...)

	return scopes
}

// Config is the set of configuration parameters for this Output instance.
type Config struct {
	// Auth is the Authenticator to use for registration and publishing. It should
	// be configured to hold the scopes returned by Scopes.
	Auth *auth.Authenticator

	// Host is the name of the LogDog Host to connect to.
	Host string

	// Project is the project that this stream belongs to.
	Project string
	// Prefix is the stream prefix to register.
	Prefix types.StreamName
	// PrefixExpiration is the prefix expiration to use when registering.
	// If zero, no expiration will be expressed to the Coordinator, and it will
	// choose based on its configuration.
	PrefixExpiration time.Duration

	// SourceInfo, if not empty, is auxiliary source information to register
	// alongside the stream.
	SourceInfo []string

	// PublishContext is the special Context to use for publishing messages. If
	// nil, the Context supplied to Register will be used.
	//
	// This is useful when the Context supplied to Register responds to
	// cancellation (e.g., user sends SIGTERM), but we might not want to
	// immediately cancel pending publishes due to flushing.
	PublishContext context.Context

	// RPCTimeout, if > 0, is the timeout to apply to an individual RPC.
	RPCTimeout time.Duration
}

// Register registers the supplied Prefix with the Coordinator. Upon success,
// an Output instance bound to that stream will be returned.
func (cfg *Config) Register(c context.Context) (output.Output, error) {
	// Validate our configuration parameters.
	switch {
	case cfg.Auth == nil:
		return nil, errors.New("no authenticator supplied")
	case cfg.Host == "":
		return nil, errors.New("no host supplied")
	}
	if err := config.ValidateProjectName(cfg.Project); err != nil {
		return nil, errors.Annotate(err, "failed to validate project").
			InternalReason("project(%v)", cfg.Project).Err()
	}
	if err := cfg.Prefix.Validate(); err != nil {
		return nil, errors.Annotate(err, "failed to validate prefix").
			InternalReason("prefix(%v)", cfg.Prefix).Err()
	}

	// Open a pRPC client to our Coordinator instance.
	httpClient, err := cfg.Auth.Client()
	if err != nil {
		log.WithError(err).Errorf(c, "Failed to get authenticated HTTP client.")
		return nil, err
	}

	// Configure our pRPC client.
	clientOpts := prpc.DefaultOptions()
	clientOpts.PerRPCTimeout = cfg.RPCTimeout
	client := prpc.Client{
		C:       httpClient,
		Host:    cfg.Host,
		Options: clientOpts,
	}

	// If our host begins with "localhost", set insecure option automatically.
	if lhttp.IsLocalHost(cfg.Host) {
		log.Infof(c, "Detected localhost; enabling insecure RPC connection.")
		client.Options.Insecure = true
	}

	// Register our Prefix with the Coordinator.
	log.Fields{
		"prefix": cfg.Prefix,
		"host":   cfg.Host,
	}.Debugf(c, "Registering prefix space with Coordinator service.")

	// Build our source info.
	sourceInfo := make([]string, 0, len(cfg.SourceInfo)+2)
	sourceInfo = append(sourceInfo, cfg.SourceInfo...)
	sourceInfo = append(sourceInfo,
		fmt.Sprintf("GOARCH=%s", runtime.GOARCH),
		fmt.Sprintf("GOOS=%s", runtime.GOOS),
	)

	nonce := make([]byte, types.OpNonceLength)
	if _, err = cryptorand.Read(c, nonce); err != nil {
		log.WithError(err).Errorf(c, "Failed to generate RegisterPrefix nonce.")
		return nil, errors.Annotate(err, "generating nonce").Err()
	}
	req := &api.RegisterPrefixRequest{
		Project:    string(cfg.Project),
		Prefix:     string(cfg.Prefix),
		SourceInfo: sourceInfo,
		Expiration: google.NewDuration(cfg.PrefixExpiration),
		OpNonce:    nonce,
	}

	svc := api.NewRegistrationPRPCClient(&client)
	var resp *api.RegisterPrefixResponse
	err = retry.Retry(c, retry.Default, func() error {
		var err error
		resp, err = svc.RegisterPrefix(c, req)
		return grpcutil.WrapIfTransient(err)
	}, retry.LogCallback(c, "RegisterPrefix"))
	if err != nil {
		log.WithError(err).Errorf(c, "Failed to register prefix with Coordinator service.")
		return nil, err
	}
	log.Fields{
		"prefix":      cfg.Prefix,
		"bundleTopic": resp.LogBundleTopic,
	}.Debugf(c, "Successfully registered log stream prefix.")

	// Validate the response topic.
	fullTopic := ps.Topic(resp.LogBundleTopic)
	if err := fullTopic.Validate(); err != nil {
		log.Fields{
			log.ErrorKey: err,
			"fullTopic":  fullTopic,
		}.Errorf(c, "Coordinator returned invalid Pub/Sub topic.")
		return nil, err
	}

	// Split our topic into project and topic name. This must succeed, since we
	// just finished validating the topic.
	proj, topic := fullTopic.Split()

	// Instantiate our Pub/Sub instance.
	//
	// We will use the non-cancelling context, for all Pub/Sub calls, as we want
	// the Pub/Sub system to drain without interruption if the application is
	// otherwise canceled.
	pctx := cfg.PublishContext
	if pctx == nil {
		pctx = c
	}

	tokenSource, err := cfg.Auth.TokenSource()
	if err != nil {
		log.WithError(err).Errorf(c, "Failed to get TokenSource for Pub/Sub client.")
		return nil, err
	}

	psClient, err := pubsub.NewClient(pctx, proj, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Fields{
			log.ErrorKey: err,
			"project":    proj,
		}.Errorf(c, "Failed to create Pub/Sub client.")
		return nil, errors.New("failed to get Pub/Sub client")
	}
	psTopic := psClient.Topic(topic)

	// We own the prefix and all verifiable parameters have been validated.
	// Successfully return our Output instance.
	//
	// Note that we use our publishing context here.
	return newPubsub(pctx, pubsubConfig{
		Topic:      pubSubTopicWrapper{psTopic},
		Host:       cfg.Host,
		Project:    cfg.Project,
		Prefix:     string(cfg.Prefix),
		Secret:     resp.Secret,
		Compress:   true,
		RPCTimeout: cfg.RPCTimeout,
	}), nil
}

// pubSubTopicWrapper wraps a cloud pubsub package Topic and converts it into
// a Butler pubsub.Topic.
type pubSubTopicWrapper struct {
	t *pubsub.Topic
}

func (w pubSubTopicWrapper) String() string {
	return w.t.String()
}

func (w pubSubTopicWrapper) Publish(ctx context.Context, msg *pubsub.Message) (string, error) {
	return w.t.Publish(ctx, msg).Get(ctx)
}
