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

package serviceaccountsv2

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
)

// ImportProjectOwnedAccountsConfigsRPC implements the corresponding method.
type ImportProjectOwnedAccountsConfigsRPC struct {
	MappingCache *MappingCache // usually GlobalMappingCache, but replaced in tests
}

// ImportProjectOwnedAccountsConfigs fetches configs from luci-config right now.
func (r *ImportProjectOwnedAccountsConfigsRPC) ImportProjectOwnedAccountsConfigs(ctx context.Context, _ *empty.Empty) (*admin.ImportedConfigs, error) {
	rev, err := r.MappingCache.ImportConfigs(ctx)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to fetch service accounts configs")
		return nil, status.Errorf(codes.Internal, "%s", err.Error())
	}
	return &admin.ImportedConfigs{Revision: rev}, nil
}

// SetupConfigValidation registers the config validation rules.
func (r *ImportProjectOwnedAccountsConfigsRPC) SetupConfigValidation(rules *validation.RuleSet) {
	r.MappingCache.SetupConfigValidation(rules)
}
