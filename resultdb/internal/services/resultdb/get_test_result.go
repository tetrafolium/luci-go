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

	"github.com/tetrafolium/luci-go/resultdb/internal/testresults"
	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

func validateGetTestResultRequest(req *pb.GetTestResultRequest) error {
	if err := pbutil.ValidateTestResultName(req.Name); err != nil {
		return errors.Annotate(err, "name").Err()
	}

	return nil
}

func (s *resultDBServer) GetTestResult(ctx context.Context, in *pb.GetTestResultRequest) (*pb.TestResult, error) {
	if err := validateGetTestResultRequest(in); err != nil {
		return nil, appstatus.BadRequest(err)
	}

	ctx, cancel := span.ReadOnlyTransaction(ctx)
	defer cancel()
	tr, err := testresults.Read(ctx, in.Name)
	if err != nil {
		return nil, err
	}

	return tr, nil
}
