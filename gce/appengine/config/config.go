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

package config

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/appengine/gaeconfig"
	"go.chromium.org/luci/config/impl/remote"
	"go.chromium.org/luci/config/server/cfgclient"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"

	gce "go.chromium.org/luci/gce/api/config/v1"
	"go.chromium.org/luci/gce/appengine/rpc"
)

// kindsFile is the name of the kinds config file.
const kindsFile = "kinds.cfg"

// vmsFile is the name of the VMs config file.
const vmsFile = "vms.cfg"

// Config encapsulates the service config.
type Config struct {
	revision string
	Kinds    *gce.Kinds
	VMs      *gce.Configs
}

// cfgKey is the key to a config.Interface in the context.
var cfgKey = "cfg"

// withInterface returns a new context with the given config.Interface installed.
func withInterface(c context.Context, cfg config.Interface) context.Context {
	return context.WithValue(c, &cfgKey, cfg)
}

// getInterface returns the config.Interface installed in the current context.
func getInterface(c context.Context) config.Interface {
	return c.Value(&cfgKey).(config.Interface)
}

// srvKey is the key to a gce.ConfigurationServer in the context.
var srvKey = "srv"

// withServer returns a new context with the given gce.ConfigurationServer installed.
func withServer(c context.Context, srv gce.ConfigurationServer) context.Context {
	return context.WithValue(c, &srvKey, srv)
}

// getServer returns the gce.ConfigurationServer installed in the current context.
func getServer(c context.Context) gce.ConfigurationServer {
	return c.Value(&srvKey).(gce.ConfigurationServer)
}

// newInterface returns a new config.Interface. Panics on error.
func newInterface(c context.Context) config.Interface {
	s, err := gaeconfig.FetchCachedSettings(c)
	if err != nil {
		panic(err)
	}
	t, err := auth.GetRPCTransport(c, auth.AsSelf)
	if err != nil {
		panic(err)
	}
	return remote.New(s.ConfigServiceHost, false, func(c context.Context) (*http.Client, error) {
		return &http.Client{Transport: t}, nil
	})
}

// fetch fetches configs from the config service.
func fetch(c context.Context) (*Config, error) {
	cli := getInterface(c)
	set := cfgclient.CurrentServiceConfigSet(c)

	// If VMs and kinds are both non-empty, then their revisions must match
	// because VMs may depend on kinds. If VMs is empty but kinds is not,
	// then kinds are declared but unused (which is fine). If kinds is empty
	// but VMs is not, this may or may not be fine. Validation will tell us.
	rev := ""

	vms := &gce.Configs{}
	switch vmsCfg, err := cli.GetConfig(c, set, vmsFile, false); {
	case err == config.ErrNoConfig:
		logging.Debugf(c, "%q not found", vmsFile)
	case err != nil:
		return nil, errors.Annotate(err, "failed to fetch %q", vmsFile).Err()
	default:
		rev = vmsCfg.Revision
		logging.Debugf(c, "found %q revision %s", vmsFile, vmsCfg.Revision)
		if err := proto.UnmarshalText(vmsCfg.Content, vms); err != nil {
			return nil, errors.Annotate(err, "failed to load %q", vmsFile).Err()
		}
	}

	kinds := &gce.Kinds{}
	switch kindsCfg, err := cli.GetConfig(c, set, kindsFile, false); {
	case err == config.ErrNoConfig:
		logging.Debugf(c, "%q not found", kindsFile)
	case err != nil:
		return nil, errors.Annotate(err, "failed to fetch %s", kindsFile).Err()
	default:
		logging.Debugf(c, "found %q revision %s", kindsFile, kindsCfg.Revision)
		if rev != "" && kindsCfg.Revision != rev {
			return nil, errors.Reason("config revision mismatch").Err()
		}
		if err := proto.UnmarshalText(kindsCfg.Content, kinds); err != nil {
			return nil, errors.Annotate(err, "failed to load %q", kindsFile).Err()
		}
	}
	return &Config{
		revision: rev,
		Kinds:    kinds,
		VMs:      vms,
	}, nil
}

// validate validates configs.
func validate(c context.Context, cfg *Config) error {
	v := &validation.Context{Context: c}
	v.SetFile(kindsFile)
	cfg.Kinds.Validate(v)
	v.SetFile(vmsFile)
	cfg.VMs.Validate(v)
	return v.Finalize()
}

// merge merges validated configs.
// Each config's referenced Kind is used to fill out unset values in its attributes.
func merge(c context.Context, cfg *Config) error {
	kindsMap := cfg.Kinds.Map()
	for _, v := range cfg.VMs.GetVms() {
		if v.Kind != "" {
			k, ok := kindsMap[v.Kind]
			if !ok {
				return errors.Reason("unknown kind %q", v.Kind).Err()
			}
			// Merge the config's attributes into a copy of the kind's.
			// This ensures the config's attributes overwrite the kind's.
			attrs := proto.Clone(k.Attributes).(*gce.VM)
			// By default, proto.Merge concatenates repeated field values.
			// Instead, make repeated fields in the config override the kind.
			if len(v.Attributes.Disk) > 0 {
				attrs.Disk = nil
			}
			if len(v.Attributes.Metadata) > 0 {
				attrs.Metadata = nil
			}
			if len(v.Attributes.NetworkInterface) > 0 {
				attrs.NetworkInterface = nil
			}
			if len(v.Attributes.Tag) > 0 {
				attrs.Tag = nil
			}
			proto.Merge(attrs, v.Attributes)
			v.Attributes = attrs
		}
	}
	return nil
}

// deref dereferences VMs metadata by fetching referenced files.
func deref(c context.Context, cfg *Config) error {
	// Cache fetched files.
	fileMap := make(map[string]string)
	cli := getInterface(c)
	set := cfgclient.CurrentServiceConfigSet(c)
	for _, v := range cfg.VMs.GetVms() {
		for i, m := range v.GetAttributes().Metadata {
			if m.GetFromFile() != "" {
				parts := strings.SplitN(m.GetFromFile(), ":", 2)
				if len(parts) < 2 {
					return errors.Reason("metadata from file must be in key:value form").Err()
				}
				file := parts[1]
				if _, ok := fileMap[file]; !ok {
					fileCfg, err := cli.GetConfig(c, set, file, false)
					if err != nil {
						return errors.Annotate(err, "failed to fetch %q", file).Err()
					}
					logging.Debugf(c, "found %q revision %s", file, fileCfg.Revision)
					if fileCfg.Revision != cfg.revision {
						return errors.Reason("config revision mismatch %q", fileCfg.Revision).Err()
					}
					fileMap[file] = fileCfg.Content
				}
				// fileMap[file] definitely exists.
				key := parts[0]
				val := fileMap[file]
				v.Attributes.Metadata[i].Metadata = &gce.Metadata_FromText{
					FromText: fmt.Sprintf("%s:%s", key, val),
				}
			}
		}
	}
	return nil
}

// normalize normalizes VMs durations by converting them to seconds.
func normalize(c context.Context, cfg *Config) error {
	for _, v := range cfg.VMs.GetVms() {
		if err := v.Lifetime.Normalize(); err != nil {
			return errors.Annotate(err, "failed to normalize %q", v.Prefix).Err()
		}
		for _, ch := range v.Amount.GetChange() {
			if err := ch.Length.Normalize(); err != nil {
				return errors.Annotate(err, "failed to normalize %q", v.Prefix).Err()
			}
		}
	}
	return nil
}

// sync synchronizes the given validated configs.
func sync(c context.Context, cfg *Config) error {
	srv := getServer(c)
	ids := stringset.New(0)
	rsp, err := srv.List(c, &gce.ListRequest{})
	if err != nil {
		return errors.Annotate(err, "failed to fetch configs").Err()
	}
	for _, v := range rsp.Configs {
		logging.Debugf(c, "fetched config %q", v.Prefix)
		ids.Add(v.Prefix)
	}
	ens := &gce.EnsureRequest{}
	for _, v := range cfg.VMs.GetVms() {
		// Validation enforces prefix uniqueness, so use it as the ID.
		ens.Id = v.Prefix
		ens.Config = v
		if _, err := srv.Ensure(c, ens); err != nil {
			return errors.Annotate(err, "failed to ensure config %q", ens.Id).Err()
		}
		logging.Debugf(c, "stored config %q", ens.Id)
		ids.Del(v.Prefix)
	}
	del := &gce.DeleteRequest{}
	err = nil
	ids.Iter(func(id string) bool {
		del.Id = id
		if _, err := srv.Delete(c, del); err != nil {
			err = errors.Annotate(err, "failed to delete config %q", del.Id).Err()
			return false
		}
		logging.Debugf(c, "deleted config %q", del.Id)
		return true
	})
	return err
}

// Import fetches and validates configs from the config service.
func Import(c context.Context) error {
	cfg, err := fetch(c)
	if err != nil {
		return errors.Annotate(err, "failed to fetch configs").Err()
	}

	// Merge before validating. VMs may be invalid until referenced kinds are applied.
	if err := merge(c, cfg); err != nil {
		return errors.Annotate(err, "failed to merge kinds into configs").Err()
	}

	// Deref before validating. VMs may be invalid until metadata from file is imported.
	if err := deref(c, cfg); err != nil {
		return errors.Annotate(err, "failed to dereference files").Err()
	}

	if err := validate(c, cfg); err != nil {
		return errors.Annotate(err, "invalid configs").Err()
	}

	if err := normalize(c, cfg); err != nil {
		return errors.Annotate(err, "failed to normalize configs").Err()
	}

	if err := sync(c, cfg); err != nil {
		return errors.Annotate(err, "failed to synchronize configs").Err()
	}
	return nil
}

// importHandler imports the config from the config service.
func importHandler(c *router.Context) {
	c.Writer.Header().Set("Content-Type", "text/plain")

	if err := Import(c.Context); err != nil {
		errors.Log(c.Context, err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

// InstallHandlers installs HTTP request handlers into the given router.
func InstallHandlers(r *router.Router, mw router.MiddlewareChain) {
	mw = mw.Extend(func(c *router.Context, next router.Handler) {
		// Install the config interface and the configuration service.
		c.Context = withInterface(c.Context, newInterface(c.Context))
		c.Context = withServer(c.Context, &rpc.Config{})
		next(c)
	})
	r.GET("/internal/cron/import-config", mw, importHandler)
}
