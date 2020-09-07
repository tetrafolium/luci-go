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
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/errors"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/endpoints"
)

// server is a service supporting privileged support services.
//
// This endpoint is restricted to LogDog support service accounts.
type server struct {
	// We pick a random starting queue number and then rotate round-robin through
	// the queues in LeaseArchiveTasks.
	lastQueueNumberMu          sync.Mutex
	lastQueueNumberInitialized bool
	lastQueueNumber            int

	settings ServerSettings
}

func (s *server) getNextArchiveQueueName(c context.Context) (string, int32) {
	s.lastQueueNumberMu.Lock()
	if s.lastQueueNumberInitialized {
		s.lastQueueNumber = (s.lastQueueNumber + 1) % s.settings.NumQueues
	} else {
		s.lastQueueNumberInitialized = true
		s.lastQueueNumber = mathrand.Intn(c, s.settings.NumQueues)
	}
	ret := int32(s.lastQueueNumber)
	s.lastQueueNumberMu.Unlock()
	return RawArchiveQueueName(ret), ret
}

// ServerSettings are settings for the LogDog RPC service.
type ServerSettings struct {
	// NumQueues is the number of queues to use for Archival tasks.
	//
	// Note that cloud task queues have a maximum throughput of 1000 qps on
	// average. Each log STREAM in LogDog will require processing AT LEAST two
	// tasks. It is recommended that you monitor the queue throughput of the
	// logdog deployment and increase this value when getting close to the qps
	// limit.
	//
	// NOTE:
	//   * Decreasing this value will cause some tasks to be un-issued.
	//     DO NOT DO THIS without coding some other workaround (for example, in
	//     LeaseArchiveTasks, inspect ALL available queues and issue randomly from
	//     all of them; Could be done by maintaining a `maxNumQueues` alongside
	//     `numQueues` where `maxNumQueues` is kept high while draining the higher
	//     queues).
	//   * Increasing this value is OK. Leased tasks embed their queue number into
	//     their TaskName field, which is round-tripped through the Archivist.
	//     When the DeleteArchiveTasks RPC is invoked, each task will be removed
	//     from the queue number embedded in TaskName.
	//
	// Reqired. Must be >0.
	NumQueues int
}

// New creates a new authenticating ServicesServer instance.
//
// Panics if `settings` is invalid.
func New(settings ServerSettings) logdog.ServicesServer {
	if settings.NumQueues <= 0 {
		panic(errors.Reason("settings.NumQueues <= 0: %d", settings.NumQueues))
	}

	return &logdog.DecoratedServices{
		Service: &server{settings: settings},
		Prelude: func(c context.Context, methodName string, req proto.Message) (context.Context, error) {

			// Only service users may access this endpoint.
			if err := coordinator.IsServiceUser(c); err != nil {
				log.WithError(err).Errorf(c, "Failed to authenticate user as a service.")

				if !coordinator.IsMembershipError(err) {
					// Not a membership error. Something went wrong on the server's end.
					return nil, grpcutil.Internal
				}
				return nil, grpcutil.PermissionDenied
			}

			return maybeEnterProjectNamespace(c, req)
		},
	}
}

// maybeEnterProjectNamespace enters a datastore namespace based on the request
// message type.
func maybeEnterProjectNamespace(c context.Context, req proto.Message) (context.Context, error) {
	if pbm, ok := req.(endpoints.ProjectBoundMessage); ok {
		project := pbm.GetMessageProject()
		log.Fields{
			"project": project,
		}.Debugf(c, "Request is entering project namespace.")
		if err := coordinator.WithProjectNamespace(&c, project, coordinator.NamespaceAccessNoAuth); err != nil {
			return c, err
		}
	}
	return c, nil
}
