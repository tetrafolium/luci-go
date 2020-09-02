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

	ds "github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/clock"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
)

var (
	terminateStreamMetric = metric.NewCounter(
		"logdog/endpoints/terminate_stream",
		"Requests to terminate stream",
		nil,
		field.String("project"))
)

// TerminateStream is an idempotent stream state terminate operation.
func (s *server) TerminateStream(c context.Context, req *logdog.TerminateStreamRequest) (*empty.Empty, error) {
	log.Fields{
		"project":       req.Project,
		"id":            req.Id,
		"terminalIndex": req.TerminalIndex,
	}.Infof(c, "Request to terminate log stream.")

	if req.TerminalIndex < 0 {
		return nil, grpcutil.Errf(codes.InvalidArgument, "Negative terminal index.")
	}

	id := coordinator.HashID(req.Id)
	if err := id.Normalize(); err != nil {
		return nil, grpcutil.Errf(codes.InvalidArgument, "Invalid ID (%s): %s", id, err)
	}

	// Initialize our log stream state.
	lst := coordinator.NewLogStreamState(c, id)

	// Transactionally validate and update the terminal index.
	err := ds.RunInTransaction(c, func(c context.Context) error {
		if err := ds.Get(c, lst); err != nil {
			if err == ds.ErrNoSuchEntity {
				log.Debugf(c, "Log stream state not found.")
				return grpcutil.Errf(codes.NotFound, "Log stream %q is not registered", id)
			}

			log.WithError(err).Errorf(c, "Failed to load LogEntry.")
			return grpcutil.Internal
		}

		switch {
		case subtle.ConstantTimeCompare(lst.Secret, req.Secret) != 1:
			log.Errorf(c, "Secrets do not match.")
			return grpcutil.Errf(codes.InvalidArgument, "Request secret doesn't match the stream secret.")

		case lst.Terminated():
			// Succeed if this is non-conflicting (idempotent).
			if lst.TerminalIndex == req.TerminalIndex {
				log.Fields{
					"terminalIndex": lst.TerminalIndex,
				}.Infof(c, "Log stream is already terminated.")
				return nil
			}

			log.Fields{
				"terminalIndex": lst.TerminalIndex,
			}.Warningf(c, "Log stream is already incompatibly terminated.")
			return grpcutil.Errf(codes.FailedPrecondition, "Log stream is incompatibly terminated.")

		default:
			// Everything looks good, let's proceed...
			now := clock.Now(c).UTC()
			lst.Updated = now
			lst.TerminalIndex = req.TerminalIndex
			lst.TerminatedTime = now

			if err := ds.Put(c, lst); err != nil {
				log.Fields{
					log.ErrorKey: err,
				}.Errorf(c, "Failed to Put() LogStream.")
				return grpcutil.Internal
			}
		}

		return s.taskArchival(c, lst, coordinator.GetSettings(c).OptimisticArchivalDelay)
	}, nil)
	if err != nil {
		log.Fields{
			log.ErrorKey: err,
		}.Errorf(c, "Failed to update LogStream.")
		return nil, err
	}

	terminateStreamMetric.Add(c, 1, req.Project)
	return &empty.Empty{}, nil
}
