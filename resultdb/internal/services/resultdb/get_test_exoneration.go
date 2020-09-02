// Copyright 2019 The LUCI Authors.
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

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/span"

	"github.com/tetrafolium/luci-go/resultdb/internal/exonerations"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

func validateGetTestExonerationRequest(req *pb.GetTestExonerationRequest) error {
	if err := pbutil.ValidateTestExonerationName(req.Name); err != nil {
		return errors.Annotate(err, "name").Err()
	}

	return nil
}

func (s *resultDBServer) GetTestExoneration(ctx context.Context, in *pb.GetTestExonerationRequest) (*pb.TestExoneration, error) {
	if err := validateGetTestExonerationRequest(in); err != nil {
		return nil, appstatus.BadRequest(err)
	}

	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()
	return exonerations.Read(ctx, in.Name)
}
