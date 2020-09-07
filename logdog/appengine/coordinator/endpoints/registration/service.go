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

package registration

import (
	"context"

	"github.com/golang/protobuf/proto"
	log "github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/registration/v1"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator"
	"github.com/tetrafolium/luci-go/logdog/appengine/coordinator/endpoints"

	"google.golang.org/grpc/codes"
)

// server is a service supporting log stream registration.
type server struct{}

// New creates a new authenticating ServicesServer instance.
func New() logdog.RegistrationServer {
	return &logdog.DecoratedRegistration{
		Service: &server{},
		Prelude: func(c context.Context, methodName string, req proto.Message) (context.Context, error) {
			// Enter a datastore namespace based on the message type.
			//
			// We use a type switch here because this is a shared decorator. All user
			// messages must implement ProjectBoundMessage.
			pbm, ok := req.(endpoints.ProjectBoundMessage)
			if ok {
				// Enter the requested project namespace. This validates that the
				// current user has READ access.
				project := pbm.GetMessageProject()
				if project == "" {
					return nil, grpcutil.Errf(codes.InvalidArgument, "project is required")
				}

				log.Fields{
					"project": project,
				}.Debugf(c, "User is accessing project.")
				if err := coordinator.WithProjectNamespace(&c, project, coordinator.NamespaceAccessWRITE); err != nil {
					return nil, getGRPCError(err)
				}
			}

			return c, nil
		},
	}
}

func getGRPCError(err error) error {
	switch {
	case err == nil:
		return nil

	case grpcutil.Code(err) != codes.Unknown:
		// If this is already a gRPC error, return it directly.
		return err

	default:
		// Generic empty internal error.
		return grpcutil.Internal
	}
}
