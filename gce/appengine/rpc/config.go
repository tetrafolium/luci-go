// Copyright 2018 The LUCI Authors.
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

package rpc

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/proto/paged"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/auth"

	"github.com/tetrafolium/luci-go/gce/api/config/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/model"
)

// Config implements config.ConfigurationServer.
type Config struct {
}

// Ensure Config implements config.ConfigurationServer.
var _ config.ConfigurationServer = &Config{}

// Delete handles a request to delete a config.
// For app-internal use only.
func (*Config) Delete(c context.Context, req *config.DeleteRequest) (*empty.Empty, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID is required")
	}
	if err := datastore.Delete(c, &model.Config{ID: req.Id}); err != nil {
		return nil, errors.Annotate(err, "failed to delete config").Err()
	}
	return &empty.Empty{}, nil
}

// Ensure handles a request to create or update a config.
// For app-internal use only.
func (*Config) Ensure(c context.Context, req *config.EnsureRequest) (*config.Config, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID is required")
	}
	cfg := &model.Config{
		ID: req.Id,
	}
	if err := datastore.RunInTransaction(c, func(c context.Context) error {
		var priorAmount int32
		switch err := datastore.Get(c, cfg); {
		case err == nil:
			// Don't forget potentially custom amount set via Update RPC.
			priorAmount = cfg.Config.CurrentAmount
		case err == datastore.ErrNoSuchEntity:
			priorAmount = 0
		default:
			return errors.Annotate(err, "failed to fetch config").Err()
		}
		cfg.Config = *req.Config
		cfg.Config.CurrentAmount = priorAmount
		if err := datastore.Put(c, cfg); err != nil {
			return errors.Annotate(err, "failed to store config").Err()
		}
		return nil
	}, nil); err != nil {
		return nil, err
	}
	return &cfg.Config, nil
}

// Get handles a request to get a config.
func (*Config) Get(c context.Context, req *config.GetRequest) (*config.Config, error) {
	if req.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "ID is required")
	}

	cfg, err := getConfigByID(c, req.Id)
	if err != nil {
		return nil, err
	}

	return &cfg.Config, nil
}

// List handles a request to list all configs.
func (*Config) List(c context.Context, req *config.ListRequest) (*config.ListResponse, error) {
	rsp := &config.ListResponse{}
	if err := paged.Query(c, req.GetPageSize(), req.GetPageToken(), rsp, datastore.NewQuery(model.ConfigKind), func(cfg *model.Config) error {
		rsp.Configs = append(rsp.Configs, &cfg.Config)
		return nil
	}); err != nil {
		return nil, err
	}
	return rsp, nil
}

// Update handles a request to update a config.
func (*Config) Update(c context.Context, req *config.UpdateRequest) (*config.Config, error) {
	switch {
	case req.GetId() == "":
		return nil, status.Errorf(codes.InvalidArgument, "ID is required")
	case len(req.UpdateMask.GetPaths()) == 0:
		return nil, status.Errorf(codes.InvalidArgument, "update mask is required")
	}
	for _, p := range req.UpdateMask.Paths {
		if p != "config.current_amount" {
			return nil, status.Errorf(codes.InvalidArgument, "field %q is invalid or immutable", p)
		}
	}

	var ret *config.Config

	if err := datastore.RunInTransaction(c, func(c context.Context) error {
		cfg, err := getConfigByID(c, req.Id)
		if err != nil {
			return err
		}
		ret = &cfg.Config

		amt, err := cfg.Config.ComputeAmount(req.Config.GetCurrentAmount(), clock.Now(c))
		switch {
		case err != nil:
			return errors.Annotate(err, "failed to parse amount").Err()
		case amt == cfg.Config.CurrentAmount:
			return nil
		default:
			cfg.Config.CurrentAmount = amt
			if err := datastore.Put(c, cfg); err != nil {
				return errors.Annotate(err, "failed to store config").Err()
			}
			return nil
		}
	}, nil); err != nil {
		return nil, err
	}

	return ret, nil
}

// configPrelude ensures the user is authorized to use the config API.
func configPrelude(c context.Context, methodName string, req proto.Message) (context.Context, error) {
	if methodName == "Update" || methodName == "Get" {
		// Update performs its own authorization checks, so allow all callers through.
		logging.Debugf(c, "%s called %q:\n%s", auth.CurrentIdentity(c), methodName, req)
		return c, nil
	}
	if !isReadOnly(methodName) {
		return c, status.Errorf(codes.PermissionDenied, "unauthorized user")
	}
	switch is, err := auth.IsMember(c, admins, writers, readers); {
	case err != nil:
		return c, err
	case is:
		logging.Debugf(c, "%s called %q:\n%s", auth.CurrentIdentity(c), methodName, req)
		return c, nil
	}
	return c, status.Errorf(codes.PermissionDenied, "unauthorized user")
}

// NewConfigurationServer returns a new configuration server.
func NewConfigurationServer() config.ConfigurationServer {
	return &config.DecoratedConfiguration{
		Prelude:  configPrelude,
		Service:  &Config{},
		Postlude: gRPCifyAndLogErr,
	}
}

func getConfigByID(c context.Context, id string) (*model.Config, error) {
	cfg := &model.Config{
		ID: id,
	}
	switch err := datastore.Get(c, cfg); err {
	case nil:
	case datastore.ErrNoSuchEntity:
		return nil, notFoundErr(id)
	default:
		return nil, errors.Annotate(err, "failed to fetch config").Err()
	}

	switch is, err := auth.IsMember(c, cfg.Config.GetOwner()...); {
	case err != nil:
		return nil, err
	case !is:
		return nil, notFoundErr(id)
	}
	return cfg, nil
}

func notFoundErr(id string) error {
	// To avoid revealing information about config existence to unauthorized users,
	// not found and permission denied responses should be ambiguous.
	return status.Errorf(codes.NotFound, "no config found with ID %q or unauthorized user", id)
}
