// Copyright 2015 The LUCI Authors.
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

package services

import (
	"context"
	"crypto/subtle"
	"time"

	ds "github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/clock"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/endpoints"
	"github.com/tetrafolium/luci-go/logdog/common/types"
	"github.com/tetrafolium/luci-go/logdog/server/config"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"
)

var (
	registerStreamMetric = metric.NewCounter(
		"logdog/endpoints/register_stream",
		"Requests to register stream",
		nil,
		field.String("project"),
		field.Bool("terminate"))
)

func buildLogStreamState(ls *coordinator.LogStream, lst *coordinator.LogStreamState) *logdog.LogStreamState {
	st := logdog.LogStreamState{
		ProtoVersion:  ls.ProtoVersion,
		Secret:        lst.Secret,
		TerminalIndex: lst.TerminalIndex,
		Archived:      lst.ArchivalState().Archived(),
		Purged:        ls.Purged,
	}
	if !lst.Terminated() {
		st.TerminalIndex = -1
	}
	return &st
}

// RegisterStream is an idempotent stream state register operation.
//
// Successive operations will succeed if they have the correct secret for their
// registered stream, regardless of whether the contents of their request match
// the currently registered state.
func (s *server) RegisterStream(c context.Context, req *logdog.RegisterStreamRequest) (*logdog.RegisterStreamResponse, error) {
	var path types.StreamPath

	// Unmarshal the serialized protobuf.
	var desc logpb.LogStreamDescriptor
	switch req.ProtoVersion {
	case logpb.Version:
		if err := proto.Unmarshal(req.Desc, &desc); err != nil {
			log.Fields{
				log.ErrorKey:   err,
				"protoVersion": req.ProtoVersion,
			}.Errorf(c, "Failed to unmarshal descriptor protobuf.")
			return nil, grpcutil.Errf(codes.InvalidArgument, "Failed to unmarshal protobuf.")
		}

	default:
		log.Fields{
			"protoVersion": req.ProtoVersion,
		}.Errorf(c, "Unrecognized protobuf version.")
		return nil, grpcutil.Errf(codes.InvalidArgument, "Unrecognized protobuf version: %q", req.ProtoVersion)
	}

	path = desc.Path()
	logStreamID := coordinator.LogStreamID(path)
	c = log.SetFields(c, log.Fields{
		"project":       req.Project,
		"path":          path,
		"prospectiveID": logStreamID,
		"terminalIndex": req.TerminalIndex,
	})
	log.Infof(c, "Registration request for log stream.")

	if err := desc.Validate(true); err != nil {
		return nil, grpcutil.Errf(codes.InvalidArgument, "Invalid log stream descriptor: %s", err)
	}
	prefix, _ := path.Split()

	// Load our service and project configs.
	cfg, err := config.Config(c)
	if err != nil {
		log.WithError(err).Errorf(c, "Failed to load configuration.")
		return nil, grpcutil.Internal
	}

	pcfg, err := coordinator.CurrentProjectConfig(c)
	if err != nil {
		log.WithError(err).Errorf(c, "Failed to load current project configuration.")
		return nil, grpcutil.Internal
	}

	// Load our Prefix. It must be registered.
	pfx := &coordinator.LogPrefix{ID: coordinator.LogPrefixID(prefix)}
	c = log.SetFields(c, log.Fields{
		"id":     pfx.ID,
		"prefix": prefix,
	})
	if err := ds.Get(c, pfx); err != nil {
		log.WithError(err).Errorf(c, "Failed to load log stream prefix.")
		if err == ds.ErrNoSuchEntity {
			return nil, grpcutil.Errf(codes.FailedPrecondition, "prefix is not registered")
		}
		return nil, grpcutil.Internal
	}

	// If we're past prefix's expiration, reject this stream.
	//
	// If the prefix doesn't have an expiration, use its creation time and apply
	// the maximum expiration.
	expirationTime := pfx.Expiration
	if expirationTime.IsZero() {
		expiration := endpoints.MinDuration(cfg.Coordinator.PrefixExpiration, pcfg.PrefixExpiration)
		if expiration > 0 {
			expirationTime = pfx.Created.Add(expiration)
		}
	}
	if now := clock.Now(c); expirationTime.IsZero() || !now.Before(expirationTime) {
		log.Fields{
			"expiration": expirationTime,
		}.Errorf(c, "The log stream Prefix has expired.")
		return nil, grpcutil.Errf(codes.FailedPrecondition, "prefix has expired")
	}

	// The prefix secret must match the request secret. If it does, we know this
	// is a legitimate registration attempt.
	if subtle.ConstantTimeCompare(pfx.Secret, req.Secret) != 1 {
		log.Errorf(c, "Request secret does not match prefix secret.")
		return nil, grpcutil.Errf(codes.InvalidArgument, "invalid secret")
	}

	// Check for registration, and that the prefix did not expire
	// (non-transactional).
	ls := &coordinator.LogStream{ID: logStreamID}
	lst := ls.State(c)
	// If true, the stream was terminated and is scheduled for an optimistic archival.
	// Semantically, it's as if "TerminateStream" was called on this stream.
	preTerminated := false

	if err := ds.Get(c, ls, lst); err != nil {
		if !anyNoSuchEntity(err) {
			log.WithError(err).Errorf(c, "Failed to check for log stream.")
			return nil, err
		}

		// The stream does not exist. Proceed with transactional registration.
		err = ds.RunInTransaction(c, func(c context.Context) error {
			// Load our state and stream (transactional).
			switch err := ds.Get(c, ls, lst); {
			case err == nil:
				// The stream is already registered.
				return nil

			case !anyNoSuchEntity(err):
				log.WithError(err).Errorf(c, "Failed to check for stream registration (transactional).")
				return err
			}

			// The stream is not yet registered.
			log.Infof(c, "Registering new log stream.")

			// Construct our LogStreamState.
			now := clock.Now(c).UTC()
			lst.Created = now
			lst.Updated = now
			lst.Secret = pfx.Secret // Copy Prefix Secret to reduce datastore Gets.

			// Construct our LogStream.
			ls.Created = now
			ls.ProtoVersion = req.ProtoVersion

			if err := ls.LoadDescriptor(&desc); err != nil {
				log.Fields{
					log.ErrorKey: err,
				}.Errorf(c, "Failed to load descriptor into LogStream.")
				return grpcutil.Errf(codes.InvalidArgument, "Failed to load descriptor.")
			}

			// If our registration request included a terminal index, terminate the
			// log stream state as well.
			if req.TerminalIndex >= 0 {
				log.Fields{
					"terminalIndex": req.TerminalIndex,
				}.Debugf(c, "Registration request included terminal index.")

				lst.TerminalIndex = req.TerminalIndex
				lst.TerminatedTime = now
				preTerminated = true
			} else {
				lst.TerminalIndex = -1
			}

			if err := ds.Put(c, ls, lst); err != nil {
				log.Fields{
					log.ErrorKey: err,
				}.Errorf(c, "Failed to Put LogStream.")
				return grpcutil.Internal
			}

			// Send archival task.
			set := coordinator.GetSettings(c)
			delay := 48 * time.Hour
			if preTerminated {
				delay = set.OptimisticArchivalDelay
			}
			return s.taskArchival(c, lst, delay)
		}, nil)
		if err != nil {
			log.Fields{
				log.ErrorKey: err,
			}.Errorf(c, "Failed to register LogStream.")
			return nil, err
		}
	}

	registerStreamMetric.Add(c, 1, req.Project, preTerminated)
	return &logdog.RegisterStreamResponse{
		Id:    string(ls.ID),
		State: buildLogStreamState(ls, lst),
	}, nil
}
