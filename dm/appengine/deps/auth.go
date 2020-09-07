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

package deps

import (
	"context"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/dm/api/acls"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/server/auth"

	"google.golang.org/grpc/codes"
)

func loadAcls(c context.Context) (ret *acls.Acls, err error) {
	ret = &acls.Acls{}
	if err := cfgclient.Get(c, "services/${appid}", "acls.cfg", cfgclient.ProtoText(ret), nil); err != nil {
		return nil, errors.Annotate(err, "").Tag(transient.Tag).
			InternalReason("loading config :: acls.cfg").Err()
	}
	return
}

func inGroups(c context.Context, groups []string) error {
	for _, grp := range groups {
		ok, err := auth.IsMember(c, grp)
		if err != nil {
			return grpcAnnotate(err, codes.Internal, "failed group check").Err()
		}
		if ok {
			return nil
		}
	}
	logging.Fields{
		"ident":  auth.CurrentIdentity(c),
		"groups": groups,
	}.Infof(c, "not authorized")
	return grpcutil.Errf(codes.PermissionDenied, "not authorized")
}

func canRead(c context.Context) (err error) {
	acl, err := loadAcls(c)
	if err != nil {
		return
	}
	if err = inGroups(c, acl.Readers); grpcutil.Code(err) == codes.PermissionDenied {
		err = inGroups(c, acl.Writers)
	}
	return
}

func canWrite(c context.Context) (err error) {
	acl, err := loadAcls(c)
	if err != nil {
		return
	}
	return inGroups(c, acl.Writers)
}
