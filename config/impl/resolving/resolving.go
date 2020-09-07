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

// Package resolving implements an interface that resolves ${var} placeholders
// in config set names and file paths before forwarding calls to some other
// interface.
package resolving

import (
	"context"
	"net/url"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/vars"
)

// New creates a config.Interface that resolves ${var} placeholders in config
// set names and file paths before forwarding calls to `next`.
func New(vars *vars.VarSet, next config.Interface) config.Interface {
	return &resolvingInterface{vars, next}
}

type resolvingInterface struct {
	vars *vars.VarSet
	next config.Interface
}

func (r *resolvingInterface) configSet(ctx context.Context, cs config.Set) (config.Set, error) {
	out, err := r.vars.RenderTemplate(ctx, string(cs))
	if err != nil {
		return "", errors.Annotate(err, "bad configSet %q", cs).Err()
	}
	return config.Set(out), nil
}

func (r *resolvingInterface) path(ctx context.Context, p string) (string, error) {
	out, err := r.vars.RenderTemplate(ctx, p)
	if err != nil {
		return "", errors.Annotate(err, "bad path %q", p).Err()
	}
	return out, nil
}

func (r *resolvingInterface) GetConfig(ctx context.Context, configSet config.Set, path string, metaOnly bool) (*config.Config, error) {
	configSet, err := r.configSet(ctx, configSet)
	if err != nil {
		return nil, err
	}
	path, err = r.path(ctx, path)
	if err != nil {
		return nil, err
	}
	return r.next.GetConfig(ctx, configSet, path, metaOnly)
}

func (r *resolvingInterface) GetConfigByHash(ctx context.Context, contentHash string) (string, error) {
	return r.next.GetConfigByHash(ctx, contentHash)
}

func (r *resolvingInterface) GetConfigSetLocation(ctx context.Context, configSet config.Set) (*url.URL, error) {
	configSet, err := r.configSet(ctx, configSet)
	if err != nil {
		return nil, err
	}
	return r.next.GetConfigSetLocation(ctx, configSet)
}

func (r *resolvingInterface) GetProjectConfigs(ctx context.Context, path string, metaOnly bool) ([]config.Config, error) {
	path, err := r.path(ctx, path)
	if err != nil {
		return nil, err
	}
	return r.next.GetProjectConfigs(ctx, path, metaOnly)
}

func (r *resolvingInterface) GetProjects(ctx context.Context) ([]config.Project, error) {
	return r.next.GetProjects(ctx)
}

func (r *resolvingInterface) ListFiles(ctx context.Context, configSet config.Set) ([]string, error) {
	configSet, err := r.configSet(ctx, configSet)
	if err != nil {
		return nil, err
	}
	return r.next.ListFiles(ctx, configSet)
}

func (r *resolvingInterface) GetRefConfigs(ctx context.Context, path string, metaOnly bool) ([]config.Config, error) {
	path, err := r.path(ctx, path)
	if err != nil {
		return nil, err
	}
	return r.next.GetRefConfigs(ctx, path, metaOnly)
}

func (r *resolvingInterface) GetRefs(ctx context.Context, projectID string) ([]string, error) {
	return r.next.GetRefs(ctx, projectID)
}
