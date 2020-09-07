// Copyright 2020 The LUCI Authors.
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

package resultdb

import (
	"context"

	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/trace"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/realms"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/invocations"
)

var (
	permGetInvocation      = realms.RegisterPermission("resultdb.invocations.get")
	permGetTestExoneration = realms.RegisterPermission("resultdb.testExonerations.get")
	permGetTestResult      = realms.RegisterPermission("resultdb.testResults.get")
	permGetArtifact        = realms.RegisterPermission("resultdb.artifacts.get")

	permListTestExonerations = realms.RegisterPermission("resultdb.testExonerations.list")
	permListTestResults      = realms.RegisterPermission("resultdb.testResults.list")
	permListArtifacts        = realms.RegisterPermission("resultdb.artifacts.list")
)

// verifyPermission checks if the caller has the specified permission on the
// realm that the invocation with the specified id belongs to.
func verifyPermission(ctx context.Context, permission realms.Permission, id invocations.ID) error {
	return verifyPermissionBatch(ctx, permission, invocations.NewIDSet(id))
}

// verifyPermissionBatch is like verifyPermission, but checks multiple
// invocations.
func verifyPermissionBatch(ctx context.Context, permission realms.Permission, ids invocations.IDSet) (err error) {
	ctx, ts := trace.StartSpan(ctx, "resultdb.resultdb.verifyPermissionBatch")
	defer func() { ts.End(err) }()

	realms, err := invocations.ReadRealms(span.Single(ctx), ids)
	if err != nil {
		return err
	}

	checked := stringset.New(1)
	for id, realm := range realms {
		if !checked.Add(realm) {
			continue
		}
		// Note: HasPermission does not make RPCs.
		switch allowed, err := auth.HasPermission(ctx, permission, realm); {
		case err != nil:
			return err
		case !allowed:
			return appstatus.Errorf(codes.PermissionDenied, `caller does not have permission %s in realm of invocation %s`, permission, id)
		}
	}
	return nil
}

// verifyPermissionInvNames does the same as verifyPermission but accepts
// invocation names (variadic)  instead of a single  invocations.ID.
func verifyPermissionInvNames(ctx context.Context, permission realms.Permission, invNames ...string) error {
	ids, err := invocations.ParseNames(invNames)
	if err != nil {
		return appstatus.BadRequest(err)
	}
	return verifyPermissionBatch(ctx, permission, ids)
}
