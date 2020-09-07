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

package cas

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/server/auth"

	api "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"
)

// Public returns publicly exposed implementation of cipd.Storage service that
// wraps the given internal implementation with ACLs.
func Public(internal api.StorageServer) api.StorageServer {
	return &api.DecoratedStorage{
		Service: internal,
		Prelude: aclPrelude,
	}
}

// aclPrelude is called before each RPC to check ACLs.
func aclPrelude(ctx context.Context, methodName string, req proto.Message) (context.Context, error) {
	acl, ok := perMethodACL[methodName]
	if !ok {
		panic(fmt.Sprintf("method %q is not defined in perMethodACL", methodName))
	}
	if acl.group != "*" {
		switch yep, err := auth.IsMember(ctx, acl.group); {
		case err != nil:
			logging.WithError(err).Errorf(ctx, "IsMember(%q) failed", acl.group)
			return nil, status.Errorf(codes.Internal, "failed to check ACL")
		case !yep:
			return nil, status.Errorf(codes.PermissionDenied, "not allowed")
		}
	}
	if acl.check != nil {
		if err := acl.check(ctx, req); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

// perMethodACL defines a group to check when authorizing an RPC call plus a
// callback for more detailed check.
//
// Group "*" means "allow anyone to call the method".
var perMethodACL = map[string]struct {
	group string
	check func(ctx context.Context, req proto.Message) error
}{
	"GetObjectURL": {"administrators", nil},

	// Upload operations are initiated by the backend, but finalized by whoever
	// uploads the data, thus 'FinishUpload' and 'CancelUpload' is accessible to
	// anyone (the authorization happens through upload operation IDs which should
	// be treated as secrets). Except we don't trust external API users to assign
	// hashes, so usage of 'force_hash' field is forbidden.
	"BeginUpload":  {"administrators", nil},
	"FinishUpload": {"*", denyForceHash},
	"CancelUpload": {"*", nil},
}

func denyForceHash(ctx context.Context, req proto.Message) error {
	if req.(*api.FinishUploadRequest).ForceHash != nil {
		return status.Errorf(codes.PermissionDenied, "usage of 'force_hash' is forbidden")
	}
	return nil
}
