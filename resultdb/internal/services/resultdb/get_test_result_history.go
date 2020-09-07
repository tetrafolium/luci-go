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

	"github.com/tetrafolium/luci-go/common/errors"

	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// GetTestResultHistory implements pb.ResultDBServer.
func (s *resultDBServer) GetTestResultHistory(ctx context.Context, in *pb.GetTestResultHistoryRequest) (*pb.GetTestResultHistoryResponse, error) {
	return nil, errors.Reason("Not yet implemented").Err()
}
