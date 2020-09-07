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

package delegation

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
)

// ImportDelegationConfigsRPC implements Admin.ImportDelegationConfigs method.
type ImportDelegationConfigsRPC struct {
	RulesCache *RulesCache // usually GlobalRulesCache, but replaced in tests
}

// ImportDelegationConfigs fetches configs from from luci-config right now.
func (r *ImportDelegationConfigsRPC) ImportDelegationConfigs(c context.Context, _ *empty.Empty) (*admin.ImportedConfigs, error) {
	rev, err := r.RulesCache.ImportConfigs(c)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to fetch delegation configs")
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &admin.ImportedConfigs{Revision: rev}, nil
}

// SetupConfigValidation registers the config validation rules.
func (r *ImportDelegationConfigsRPC) SetupConfigValidation(rules *validation.RuleSet) {
	r.RulesCache.SetupConfigValidation(rules)
}
