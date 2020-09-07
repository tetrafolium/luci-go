// Copyright 2015 The LUCI Authors.
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

package tsmon

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/tsmon/monitor"
	"github.com/tetrafolium/luci-go/common/tsmon/store"
	"github.com/tetrafolium/luci-go/common/tsmon/target"

	"github.com/tetrafolium/luci-go/hardcoded/chromeinfra"
)

// Store returns the global metric store that contains all the metric values for
// this process.  Applications shouldn't need to access this directly - instead
// use the metric objects which provide type-safe accessors.
func Store(c context.Context) store.Store {
	return GetState(c).Store()
}

// Monitor returns the global monitor that sends metrics to monitoring
// endpoints.  Defaults to a nil monitor, but changed by InitializeFromFlags.
func Monitor(c context.Context) monitor.Monitor {
	return GetState(c).Monitor()
}

// SetStore changes the global metric store.  All metrics that were registered
// with the old store will be re-registered on the new store.
func SetStore(c context.Context, s store.Store) {
	GetState(c).SetStore(s)
}

// InitializeFromFlags configures the tsmon library from flag values.
//
// This will set a Target (information about what's reporting metrics) and a
// Monitor (where to send the metrics to).
func InitializeFromFlags(c context.Context, fl *Flags) error {
	// Load the config file, and override its values with flags.
	cfg, err := loadConfig(fl.ConfigFile)
	if err != nil {
		return errors.Annotate(err, "failed to load config file at [%s]", fl.ConfigFile).Err()
	}

	if fl.Endpoint != "" {
		cfg.Endpoint = fl.Endpoint
	}
	if fl.Credentials != "" {
		cfg.Credentials = fl.Credentials
	}
	if fl.ActAs != "" {
		cfg.ActAs = fl.ActAs
	}

	mon, err := initMonitor(c, cfg)
	switch {
	case err != nil:
		return errors.Annotate(err, "failed to initialize monitor").Err()
	case mon == nil:
		return nil // tsmon is disabled
	}

	// Monitoring is enabled, so get the expensive default values for hostname,
	// etc.
	if cfg.AutoGenHostname {
		fl.Target.AutoGenHostname = true
	}
	if cfg.Hostname != "" {
		if fl.Target.DeviceHostname == "" {
			fl.Target.DeviceHostname = cfg.Hostname
		}
		if fl.Target.TaskHostname == "" {
			fl.Target.TaskHostname = cfg.Hostname
		}
	}
	if cfg.Region != "" {
		if fl.Target.DeviceRegion == "" {
			fl.Target.DeviceRegion = cfg.Region
		}
		if fl.Target.TaskRegion == "" {
			fl.Target.TaskRegion = cfg.Region
		}
	}
	fl.Target.SetDefaultsFromHostname()
	t, err := target.NewFromFlags(&fl.Target)
	if err != nil {
		return errors.Annotate(err, "failed to configure target from flags").Err()
	}

	Initialize(c, mon, store.NewInMemory(t))

	state := GetState(c)
	if state.flusher != nil {
		logging.Infof(c, "Canceling previous tsmon auto flush")
		state.flusher.stop()
		state.flusher = nil
	}

	if fl.Flush == FlushAuto {
		state.flusher = &autoFlusher{}
		state.flusher.start(c, fl.FlushInterval)
	}

	return nil
}

// Initialize configures the tsmon library with the given monitor and store.
func Initialize(c context.Context, m monitor.Monitor, s store.Store) {
	state := GetState(c)
	state.SetMonitor(m)
	state.SetStore(s)
}

// Shutdown gracefully terminates the tsmon by doing the final flush and
// disabling auto flush (if it was enabled).
//
// It resets Monitor and Store.
//
// Logs error to standard logger. Does nothing if tsmon wasn't initialized.
func Shutdown(c context.Context) {
	state := GetState(c)
	if store.IsNilStore(state.Store()) {
		return
	}

	if state.flusher != nil {
		logging.Debugf(c, "Stopping tsmon auto flush")
		state.flusher.stop()
		state.flusher = nil
	}

	// Flush logs errors inside.
	Flush(c)

	// Reset the state as if 'InitializeFromFlags' was never called.
	Initialize(c, monitor.NewNilMonitor(), store.NewNilStore())
}

// ResetCumulativeMetrics resets only cumulative metrics.
func ResetCumulativeMetrics(c context.Context) {
	GetState(c).ResetCumulativeMetrics(c)
}

// initMonitor examines flags and config and initializes a monitor.
//
// It returns (nil, nil) if tsmon should be disabled.
func initMonitor(c context.Context, cfg config) (monitor.Monitor, error) {
	if cfg.Endpoint == "" {
		logging.Infof(c, "tsmon is disabled because no endpoint is configured")
		return nil, nil
	}
	if strings.ToLower(cfg.Endpoint) == "none" {
		logging.Infof(c, "tsmon is explicitly disabled")
		return nil, nil
	}

	endpointURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	switch endpointURL.Scheme {
	case "file":
		return monitor.NewDebugMonitor(endpointURL.Path), nil
	case "http", "https":
		client, err := newAuthenticator(c, cfg.Credentials, cfg.ActAs, monitor.ProdxmonScopes).Client()
		if err != nil {
			return nil, err
		}

		return monitor.NewHTTPMonitor(c, client, endpointURL)
	default:
		return nil, fmt.Errorf("unknown tsmon endpoint url: %s", cfg.Endpoint)
	}
}

// newAuthenticator returns a new authenticator for HTTP requests.
func newAuthenticator(ctx context.Context, credentials, actAs string, scopes []string) *auth.Authenticator {
	// TODO(vadimsh): Don't hardcode auth options here, pass them from outside
	// somehow.
	authOpts := chromeinfra.DefaultAuthOptions()
	authOpts.ServiceAccountJSONPath = credentials
	authOpts.Scopes = scopes
	authOpts.ActAsServiceAccount = actAs
	return auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts)
}
