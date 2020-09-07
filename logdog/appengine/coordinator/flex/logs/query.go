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

package logs

import (
	"context"

	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/logs/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"
	"github.com/tetrafolium/luci-go/logdog/common/types"

	"github.com/tetrafolium/luci-go/common/clock"
	log "github.com/tetrafolium/luci-go/common/logging"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	"google.golang.org/grpc/codes"
)

const (
	// queryResultLimit is the maximum number of log streams that will be
	// returned in a single query. If the user requests more, it will be
	// automatically called at this value.
	queryResultLimit = 500
)

// Query returns log stream paths that match the requested query.
func (s *server) Query(c context.Context, req *logdog.QueryRequest) (*logdog.QueryResponse, error) {
	// Non-admin users may not request purged results.
	canSeePurged := true
	if err := coordinator.IsAdminUser(c); err != nil {
		canSeePurged = false

		// Non-admin user.
		if req.Purged == logdog.QueryRequest_YES {
			log.Fields{
				log.ErrorKey: err,
			}.Errorf(c, "Non-superuser requested to see purged logs. Denying.")
			return nil, grpcutil.Errf(codes.InvalidArgument, "non-admin user cannot request purged log streams")
		}
	}

	// Scale the maximum number of results based on the number of queries in this
	// request. If the user specified a maximum result count of zero, use the
	// default maximum.
	//
	// If this scaling results in a limit that is <1 per request, we will return
	// back a BadRequest error.
	limit := s.resultLimit
	if limit == 0 {
		limit = queryResultLimit
	}

	// Execute our queries in parallel.
	resp := logdog.QueryResponse{}
	e := &queryRunner{
		Context:      log.SetField(c, "path", req.Path),
		QueryRequest: req,
		canSeePurged: canSeePurged,
		limit:        limit,
	}

	startTime := clock.Now(c)
	if err := e.runQuery(&resp); err != nil {
		// Transient errors would be handled at the "execute" level, so these are
		// specific failure errors. We must escalate individual errors to the user.
		// We will choose the most severe of the resulting errors.
		log.WithError(err).Errorf(c, "Failed to execute query.")
		return nil, err
	}
	log.Infof(c, "Query took: %s", clock.Now(c).Sub(startTime))
	return &resp, nil
}

type queryRunner struct {
	context.Context
	*logdog.QueryRequest

	canSeePurged bool
	limit        int
}

func (r *queryRunner) runQuery(resp *logdog.QueryResponse) error {
	if r.limit == 0 {
		return grpcutil.Errf(codes.InvalidArgument, "query limit is zero")
	}

	if int(r.MaxResults) > 0 && r.limit > int(r.MaxResults) {
		r.limit = int(r.MaxResults)
	}

	q, err := coordinator.NewLogStreamQuery(r.Path)
	if err != nil {
		log.Fields{
			log.ErrorKey: err,
			"path":       r.Path,
		}.Errorf(r, "Invalid query path.")
		return grpcutil.Errf(codes.InvalidArgument, "invalid query `path`")
	}

	if err := q.SetCursor(r, r.Next); err != nil {
		log.Fields{
			log.ErrorKey: err,
			"cursor":     r.Next,
		}.Errorf(r, "Failed to SetCursor.")
		return grpcutil.Errf(codes.InvalidArgument, "invalid `next` value")
	}

	q.OnlyContentType(r.ContentType)
	if st := r.StreamType; st != nil {
		if err := q.OnlyStreamType(st.Value); err != nil {
			return grpcutil.Errf(codes.InvalidArgument, "invalid query `streamType`: %s", st.Value)
		}
	}

	// By default q wll exclude purged data.
	//
	// If the user is allowed to, and `r.Purged in (YES, BOTH)`, include purged
	// results in the result.
	if r.canSeePurged && r.Purged != logdog.QueryRequest_NO {
		q.IncludePurged()
		// If the user requested to ONLY see purged results, further restrict the
		// query.
		if r.Purged == logdog.QueryRequest_YES {
			q.OnlyPurged()
		}
	}

	q.TimeBound(r.Newer, r.Older)

	// Add tag constraints.
	for k, v := range r.Tags {
		if err := types.ValidateTag(k, v); err != nil {
			log.Fields{
				log.ErrorKey: err,
				"key":        k,
				"value":      v,
			}.Errorf(r, "Invalid tag constraint.")
			return grpcutil.Errf(codes.InvalidArgument, "invalid tag constraint: %q", k)
		}
	}
	q.MustHaveTags(r.Tags)

	// The "State" boolean in the query request populates two pieces of data:
	//   1) The Desc field in logdog.QueryResponse_Stream
	//   2) The State field (of type logdog.LogStreamState) in
	//       logdog.QueryResponse_Stream.
	//
	// Note that the logdog.LogStreamState is actually composed of data from both
	//   * a coordinator.LogStream
	//   * a coordinator.LogStreamState
	//
	// As far as I can tell, there's no reason for this complexity, it's just
	// confusing.
	//
	// To handle this, we have two arrays populated during the Run query:
	//   * logStreamStates holds the coordinator LogStreamState objects we need
	//     to pull from datastore.
	//   * respLogStreamStates holds the similarly-named response objects.
	//
	// We partially populate the content of the respLogStreamStates objects during
	// the execution of Run (the portion populated from the coordinator.LogStream
	// object).
	//
	// After the query, if these arrays are non-empty, we load the LogStreamState
	// objects from datastore, and populate the rest of the response objects.
	var logStreamStates []*coordinator.LogStreamState
	var respLogStreamStates []*logdog.LogStreamState

	err = q.Run(r, func(ls *coordinator.LogStream, cb ds.CursorCB) error {
		toAdd := &logdog.QueryResponse_Stream{
			Path: string(ls.Path()),
		}
		resp.Streams = append(resp.Streams, toAdd)
		if r.State {
			// ls.State returns a coordinator.LogStreamState object with just its
			// Parent key field populated.
			logStreamStates = append(logStreamStates, ls.State(r))

			// generate and fill the response LogStreamState object, then track it for
			// later.
			toAdd.State = &logdog.LogStreamState{}
			fillStateFromLogStream(toAdd.State, ls)
			respLogStreamStates = append(respLogStreamStates, toAdd.State)

			if desc, err := ls.DescriptorProto(); err != nil {
				log.Errorf(r, "processing %q: loading descriptor: %s", toAdd.Path, err)
			} else {
				toAdd.Desc = desc
			}
		}
		if len(resp.Streams) == r.limit {
			cursor, err := cb()
			if err != nil {
				return err
			}
			resp.Next = cursor.String()
			return ds.Stop
		}
		return nil
	})
	if err != nil {
		log.Fields{
			log.ErrorKey: err,
		}.Errorf(r, "Failed to execute query.")
		return grpcutil.Errf(codes.Internal, "failed to execute query: %s", err)
	}

	if len(logStreamStates) > 0 {
		if err := ds.Get(r, logStreamStates); err != nil {
			log.WithError(err).Errorf(r, "Failed to load log stream states.")
			return grpcutil.Errf(codes.Internal, "failed to load log stream states: %s", err)
		}
		for i, state := range logStreamStates {
			fillStateFromLogStreamState(respLogStreamStates[i], state)
		}
	}

	return nil
}
