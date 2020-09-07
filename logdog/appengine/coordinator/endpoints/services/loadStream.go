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

	ds "github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	"google.golang.org/grpc/codes"
)

// LoadStream loads the log stream state.
func (s *server) LoadStream(c context.Context, req *logdog.LoadStreamRequest) (*logdog.LoadStreamResponse, error) {
	log.Fields{
		"project": req.Project,
		"id":      req.Id,
	}.Infof(c, "Loading log stream state.")

	id := coordinator.HashID(req.Id)
	if err := id.Normalize(); err != nil {
		log.WithError(err).Errorf(c, "Invalid stream ID.")
		return nil, grpcutil.Errf(codes.InvalidArgument, "Invalid ID (%s): %s", id, err)
	}

	ls := &coordinator.LogStream{ID: coordinator.HashID(req.Id)}
	lst := ls.State(c)

	if err := ds.Get(c, lst, ls); err != nil {
		if anyNoSuchEntity(err) {
			log.WithError(err).Errorf(c, "No such entity in datastore.")

			// The state isn't registered, so this stream does not exist.
			return nil, grpcutil.Errf(codes.NotFound, "Log stream was not found.")
		}

		log.WithError(err).Errorf(c, "Failed to load log stream.")
		return nil, grpcutil.Internal
	}

	// The log stream and state loaded successfully.
	resp := logdog.LoadStreamResponse{
		State: buildLogStreamState(ls, lst),
	}
	if req.Desc {
		resp.Desc = ls.Descriptor
	}
	resp.ArchivalKey = lst.ArchivalKey
	resp.Age = google.NewDuration(ds.RoundTime(clock.Now(c)).Sub(lst.Updated))

	log.Fields{
		"id":              lst.ID(),
		"terminalIndex":   resp.State.TerminalIndex,
		"archived":        resp.State.Archived,
		"purged":          resp.State.Purged,
		"age":             google.DurationFromProto(resp.Age),
		"archivalKeySize": len(resp.ArchivalKey),
	}.Infof(c, "Successfully loaded log stream state.")
	return &resp, nil
}

func anyNoSuchEntity(err error) bool {
	return errors.Any(err, func(err error) bool {
		return err == ds.ErrNoSuchEntity
	})
}
