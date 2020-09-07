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

// Package gaeemulation provides a server module that adds implementation of
// some https://godoc.org/github.com/tetrafolium/luci-go/gae APIs to the global server context.
//
// The implementation is based on regular Cloud APIs and works from anywhere
// (not necessarily from Appengine).
//
// Usage:
//
//   func main() {
//     modules := []module.Module{
//       gaeemulation.NewModuleFromFlags(),
//     }
//     server.Main(nil, modules, func(srv *server.Server) error {
//       srv.Routes.GET("/", ..., func(c *router.Context) {
//         ent := Entity{ID: "..."}
//         err := datastore.Get(c.Context, &ent)
//         ...
//       })
//       return nil
//     })
//   }
//
// TODO(vadimsh): Currently provides datastore API only.
package gaeemulation

import (
	"context"
	"flag"
	"os"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"

	"github.com/tetrafolium/luci-go/gae/filter/txndefer"
	"github.com/tetrafolium/luci-go/gae/impl/cloud"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/module"
)

// ModuleOptions are empty for now but exist to make the gaeemulation interface
// similar to interfaces of other modules.
type ModuleOptions struct{}

// Register registers the command line flags.
func (o *ModuleOptions) Register(f *flag.FlagSet) {}

// NewModule returns a server module that adds implementation of
// some https://godoc.org/github.com/tetrafolium/luci-go/gae APIs to the global server context.
func NewModule(opts *ModuleOptions) module.Module {
	if opts == nil {
		opts = &ModuleOptions{}
	}
	return &gaeModule{opts: opts}
}

// NewModuleFromFlags is a variant of NewModule that initializes options through
// command line flags.
//
// Calling this function registers flags in flag.CommandLine. They are usually
// parsed in server.Main(...).
func NewModuleFromFlags() module.Module {
	opts := &ModuleOptions{}
	opts.Register(flag.CommandLine)
	return NewModule(opts)
}

// gaeModule implements module.Module.
type gaeModule struct {
	opts *ModuleOptions
}

// Name is part of module.Module interface.
func (*gaeModule) Name() string {
	return "github.com/tetrafolium/luci-go/server/gaeemulation"
}

// Initialize is part of module.Module interface.
func (m *gaeModule) Initialize(ctx context.Context, host module.Host, opts module.HostOptions) (context.Context, error) {
	var client *datastore.Client
	if opts.CloudProject != "" {
		var err error
		if client, err = m.initDSClient(ctx, host, opts.CloudProject); err != nil {
			return nil, err
		}
	}
	cfg := &cloud.ConfigLite{
		IsDev:     !opts.Prod,
		ProjectID: opts.CloudProject,
		DS:        client, // if nil, datastore calls will fail gracefully(-ish)
	}
	return txndefer.FilterRDS(cfg.Use(ctx)), nil
}

// initDSClient sets up Cloud Datastore client that uses AsSelf server token
// source.
func (m *gaeModule) initDSClient(ctx context.Context, host module.Host, cloudProject string) (*datastore.Client, error) {
	logging.Infof(ctx, "Setting up datastore client for project %q", cloudProject)

	// Enable auth only when using the real datastore.
	var clientOpts []option.ClientOption
	if addr := os.Getenv("DATASTORE_EMULATOR_HOST"); addr == "" {
		ts, err := auth.GetTokenSource(ctx, auth.AsSelf, auth.WithScopes(auth.CloudOAuthScopes...))
		if err != nil {
			return nil, errors.Annotate(err, "failed to initialize the token source").Err()
		}
		clientOpts = []option.ClientOption{option.WithTokenSource(ts)}
	}

	client, err := datastore.NewClient(ctx, cloudProject, clientOpts...)
	if err != nil {
		return nil, errors.Annotate(err, "failed to instantiate the datastore client").Err()
	}

	host.RegisterCleanup(func(ctx context.Context) {
		if err := client.Close(); err != nil {
			logging.Warningf(ctx, "Failed to close the datastore client - %s", err)
		}
	})

	// TODO(vadimsh): "Ping" the datastore to verify the credentials are correct?

	return client, nil
}
