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

package projectscope

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/config/validation"

	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
)

// ImportProjectIdentityConfigsRPC implements Admin.ImportProjectIdentityConfigs method.
type ImportProjectIdentityConfigsRPC struct {
}

// ImportProjectIdentityConfigs fetches configs from from luci-config right now.
func (r *ImportProjectIdentityConfigsRPC) ImportProjectIdentityConfigs(c context.Context, _ *empty.Empty) (*admin.ImportedConfigs, error) {
	rev, err := ImportConfigs(c)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to fetch projects configs")
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &admin.ImportedConfigs{Revision: rev}, nil
}

// SetupConfigValidation registers the config validation rules.
func (r *ImportProjectIdentityConfigsRPC) SetupConfigValidation(rules *validation.RuleSet) {
	SetupConfigValidation(&validation.Rules)
}
