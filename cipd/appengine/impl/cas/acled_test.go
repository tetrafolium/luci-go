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
	"testing"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"

	api "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestACLDecorator(t *testing.T) {
	t.Parallel()

	anon := identity.AnonymousIdentity
	someone := identity.Identity("user:someone@example.com")
	admin := identity.Identity("user:admin@example.com")

	state := &authtest.FakeState{
		FakeDB: authtest.NewFakeDB(
			authtest.MockMembership(admin, "administrators"),
		),
	}
	ctx := auth.WithState(context.Background(), state)

	noForceHash := &api.FinishUploadRequest{}
	withForceHash := &api.FinishUploadRequest{ForceHash: &api.ObjectRef{}}
	cancelReq := &api.CancelUploadRequest{}

	var cases = []struct {
		method  string
		caller  identity.Identity
		request proto.Message
		allowed bool
	}{
		{"GetObjectURL", anon, nil, false},
		{"GetObjectURL", someone, nil, false},
		{"GetObjectURL", admin, nil, true},

		{"BeginUpload", anon, nil, false},
		{"BeginUpload", someone, nil, false},
		{"BeginUpload", admin, nil, true},

		{"FinishUpload", anon, noForceHash, true},
		{"FinishUpload", someone, noForceHash, true},
		{"FinishUpload", admin, noForceHash, true},

		{"FinishUpload", anon, withForceHash, false},
		{"FinishUpload", someone, withForceHash, false},
		{"FinishUpload", admin, withForceHash, false},

		{"CancelUpload", anon, cancelReq, true},
		{"CancelUpload", someone, cancelReq, true},
		{"CancelUpload", admin, cancelReq, true},
	}

	for idx, cs := range cases {
		Convey(fmt.Sprintf("%d - %s by %s", idx, cs.method, cs.caller), t, func() {
			state.Identity = cs.caller
			_, err := aclPrelude(ctx, cs.method, cs.request)
			if cs.allowed {
				So(err, ShouldBeNil)
			} else {
				So(grpc.Code(err), ShouldEqual, codes.PermissionDenied)
			}
		})
	}
}
