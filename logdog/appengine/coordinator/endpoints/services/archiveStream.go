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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tetrafolium/luci-go/common/clock"
	log "github.com/tetrafolium/luci-go/common/logging"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"

	"google.golang.org/grpc/codes"
)

func (b *server) ArchiveStream(c context.Context, req *logdog.ArchiveStreamRequest) (*empty.Empty, error) {
	log.Fields{
		"project":       req.Project,
		"id":            req.Id,
		"complete":      req.Complete(),
		"terminalIndex": req.TerminalIndex,
		"logEntryCount": req.LogEntryCount,
		"error":         req.Error,
	}.Infof(c, "Received archival request.")

	id := coordinator.HashID(req.Id)
	if err := id.Normalize(); err != nil {
		return nil, grpcutil.Errf(codes.InvalidArgument, "Invalid ID (%s): %s", id, err)
	}

	// Verify that the request is minimially valid.
	switch {
	case req.IndexUrl == "":
		return nil, grpcutil.Errf(codes.InvalidArgument, "missing required index archive URL")
	case req.StreamUrl == "":
		return nil, grpcutil.Errf(codes.InvalidArgument, "missing required stream archive URL")
	}

	lst := coordinator.NewLogStreamState(c, id)

	// Post the archival results to the Coordinator.
	now := clock.Now(c).UTC()
	var ierr error
	err := ds.RunInTransaction(c, func(c context.Context) error {
		ierr = nil

		// Note that within this transaction, we have two return values:
		// - Non-nil to abort the transaction.
		// - Specific error via "ierr".
		if err := ds.Get(c, lst); err != nil {
			return err
		}

		switch as := lst.ArchivalState(); {
		case as.Archived():
			// Return nil if the log stream is already archived (idempotent).
			log.Warningf(c, "Log stream is already archived.")
			return nil

			// If our log stream is not in in a tasked archival state, we will reject
			// this archive request with FailedPrecondition.
		case as != coordinator.ArchiveTasked:
			log.Fields{
				"state": as,
			}.Errorf(c, "Log stream archival is not tasked.")
			ierr = grpcutil.Errf(codes.FailedPrecondition, "Log stream has not tasked an archival.")
			return ierr
		}

		// If this request contained an error, we will record an empty archival and
		// log a warning.
		if req.Error != "" {
			log.Fields{
				"archiveError": req.Error,
			}.Warningf(c, "Log stream archival indicated error. Archiving empty stream.")

			req.TerminalIndex = -1
			req.LogEntryCount = 0
		}

		// Update archival information. Make sure this actually marks the stream as
		// archived.
		lst.Updated = now
		lst.ArchivedTime = now
		lst.ArchivalKey = nil // No point in wasting datastore space on this.

		if lst.TerminalIndex < 0 {
			// Also set the terminated time.
			lst.TerminatedTime = now
		}
		lst.TerminalIndex = req.TerminalIndex

		lst.ArchiveLogEntryCount = req.LogEntryCount
		lst.ArchiveStreamURL = req.StreamUrl
		lst.ArchiveStreamSize = req.StreamSize
		lst.ArchiveIndexURL = req.IndexUrl
		lst.ArchiveIndexSize = req.IndexSize

		// Update the log stream.
		if err := ds.Put(c, lst); err != nil {
			log.WithError(err).Errorf(c, "Failed to update log stream.")
			return err
		}

		log.Infof(c, "Successfully marked stream as archived.")
		return nil
	}, nil)
	if ierr != nil {
		log.WithError(ierr).Errorf(c, "Failed to mark stream as archived.")
		return nil, ierr
	}
	if err != nil {
		log.WithError(err).Errorf(c, "Internal error.")
		return nil, grpcutil.Internal
	}

	return &empty.Empty{}, nil
}
