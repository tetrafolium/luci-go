// Copyright 2018 The LUCI Authors.
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

package apiservers

import (
	"context"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	schedulerpb "github.com/tetrafolium/luci-go/scheduler/api/scheduler/v1"
	"github.com/tetrafolium/luci-go/scheduler/appengine/catalog"
	"github.com/tetrafolium/luci-go/scheduler/appengine/engine"
	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
	"github.com/tetrafolium/luci-go/server/auth"
)

// AdminServerWithACL returns AdminServer implementation that checks all callers
// are in the given administrator group.
func AdminServerWithACL(e engine.EngineInternal, c catalog.Catalog, adminGroup string) internal.AdminServer {
	return &internal.DecoratedAdmin{
		Service: &adminServer{
			Engine:  e,
			Catalog: c,
		},

		Prelude: func(c context.Context, methodName string, req proto.Message) (context.Context, error) {
			caller := auth.CurrentIdentity(c)
			logging.Warningf(c, "Admin call %q by %q", methodName, caller)
			switch yes, err := auth.IsMember(c, adminGroup); {
			case err != nil:
				return nil, status.Errorf(codes.Internal, "failed to check ACL")
			case !yes:
				return nil, status.Errorf(codes.PermissionDenied, "not an administrator")
			default:
				return c, nil
			}
		},

		Postlude: func(c context.Context, methodName string, rsp proto.Message, err error) error {
			return grpcutil.GRPCifyAndLogErr(c, err)
		},
	}
}

// adminServer implements internal.admin.Admin API without ACL check.
//
// It also returns regular errors, NOT gRPC errors. AdminServerWithACL takes
// care of authorization and conversion of errors to grpc ones.
type adminServer struct {
	Engine  engine.EngineInternal
	Catalog catalog.Catalog
}

// GetDebugJobState implements the corresponding RPC method.
func (s *adminServer) GetDebugJobState(c context.Context, r *schedulerpb.JobRef) (resp *internal.DebugJobState, err error) {
	switch state, err := s.Engine.GetDebugJobState(c, r.Project+"/"+r.Job); {
	case err == engine.ErrNoSuchJob:
		return nil, status.Errorf(codes.NotFound, "no such job")
	case err != nil:
		return nil, err
	default:
		return &internal.DebugJobState{
			Enabled:    state.Job.Enabled,
			Paused:     state.Job.Paused,
			LastTriage: google.NewTimestamp(state.Job.LastTriage),
			CronState: &internal.DebugJobState_CronState{
				Enabled:       state.Job.Cron.Enabled,
				Generation:    state.Job.Cron.Generation,
				LastRewind:    google.NewTimestamp(state.Job.Cron.LastRewind),
				LastTickWhen:  google.NewTimestamp(state.Job.Cron.LastTick.When),
				LastTickNonce: state.Job.Cron.LastTick.TickNonce,
			},
			ManagerState:        state.ManagerState,
			ActiveInvocations:   state.Job.ActiveInvocations,
			FinishedInvocations: state.FinishedInvocations,
			RecentlyFinishedSet: state.RecentlyFinishedSet,
			PendingTriggersSet:  state.PendingTriggersSet,
		}, nil
	}
}
