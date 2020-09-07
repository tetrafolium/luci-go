// Copyright 2017 The LUCI Authors.
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

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/config"
	"github.com/tetrafolium/luci-go/config/cfgclient"
	"github.com/tetrafolium/luci-go/config/validation"

	configPB "github.com/tetrafolium/luci-go/machine-db/api/config/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/model"
	"github.com/tetrafolium/luci-go/machine-db/common"
)

// datacentersFilename is the name of the config file enumerating datacenter files.
const datacentersFilename = "datacenters.cfg"

// switchMaxPorts is the maximum number of ports a switch may have.
const switchMaxPorts = 65535

// importDatacenters fetches, validates, and applies datacenter configs.
func importDatacenters(c context.Context, configSet config.Set, platformIds map[string]int64) error {
	cfg := &configPB.Datacenters{}
	metadata := &config.Meta{}
	if err := cfgclient.Get(c, configSet, datacentersFilename, cfgclient.ProtoText(cfg), metadata); err != nil {
		return errors.Annotate(err, "failed to load %s", datacentersFilename).Err()
	}
	logging.Infof(c, "Found %s revision %q", datacentersFilename, metadata.Revision)

	ctx := &validation.Context{Context: c}
	ctx.SetFile(datacentersFilename)
	validateDatacentersCfg(ctx, cfg)
	if err := ctx.Finalize(); err != nil {
		return errors.Annotate(err, "invalid config").Err()
	}

	// cfgs will be a map of datacenter config filename to Datacenter.
	cfgs := make(map[string]*configPB.Datacenter, len(cfg.Datacenter))
	// datacenters will be a slice of Datacenters.
	datacenters := make([]*configPB.Datacenter, 0, len(cfg.Datacenter))
	for _, datacenterFile := range cfg.Datacenter {
		datacenter := &configPB.Datacenter{}
		if err := cfgclient.Get(c, configSet, datacenterFile, cfgclient.ProtoText(datacenter), nil); err != nil {
			return errors.Annotate(err, "failed to load datacenter config %q", datacenterFile).Err()
		}
		cfgs[datacenterFile] = datacenter
		datacenters = append(datacenters, datacenter)
		logging.Infof(c, "Found configured datacenter %q", datacenter.Name)
	}

	validateDatacenters(ctx, cfgs)
	if err := ctx.Finalize(); err != nil {
		return errors.Annotate(err, "invalid config").Err()
	}

	datacenterIds, err := model.EnsureDatacenters(c, datacenters)
	if err != nil {
		return errors.Annotate(err, "failed to ensure datacenters").Err()
	}
	rackIds, err := model.EnsureRacks(c, datacenters, datacenterIds)
	if err != nil {
		return errors.Annotate(err, "failed to ensure racks").Err()
	}
	err = model.EnsureSwitches(c, datacenters, rackIds)
	if err != nil {
		return errors.Annotate(err, "failed to ensure switches").Err()
	}
	kvmIds, err := model.EnsureKVMs(c, datacenters, platformIds, rackIds)
	if err != nil {
		return errors.Annotate(err, "failed to ensure KVMs").Err()
	}
	// KVMs must reference the rack they exist in, while racks may optionally reference
	// a KVM that serves them. Because of this bidirectional foreign key constraint in
	// the database, we must first ensure that everything about racks except their KVM
	// matches the config, then ensure that KVMs match the config, then ensure the KVM
	// referred to by each rack matches the config.
	err = model.EnsureRackKVMs(c, datacenters, kvmIds)
	if err != nil {
		return errors.Annotate(err, "failed to ensure rack KVMs").Err()
	}
	return nil
}

// validateDatacentersCfg validates datacenters.cfg.
func validateDatacentersCfg(c *validation.Context, cfg *configPB.Datacenters) {
	// Datacenter filenames must be unique.
	// Keep records of which ones we've already seen.
	files := stringset.New(len(cfg.Datacenter))
	for _, file := range cfg.Datacenter {
		switch {
		case file == "":
			c.Errorf("datacenter filenames are required and must be non-empty")
		case !files.Add(file):
			c.Errorf("duplicate filename %q", file)
		}
	}
}

// validateDatacenters validates the individual datacenter.cfg files referenced in datacenters.cfg.
func validateDatacenters(c *validation.Context, datacenters map[string]*configPB.Datacenter) {
	// Datacenter, rack, and switch names must be unique.
	// Keep records of ones we've already seen.
	names := stringset.New(len(datacenters))
	racks := stringset.New(0)
	switches := stringset.New(0)
	kvms := stringset.New(0)

	for filename, dc := range datacenters {
		c.SetFile(filename)
		switch {
		case dc.Name == "":
			c.Errorf("datacenter names are required and must be non-empty")
		case !names.Add(dc.Name):
			c.Errorf("duplicate datacenter %q", dc.Name)
		}

		c.Enter("datacenter %q", dc.Name)
		for _, rack := range dc.Rack {
			switch {
			case rack.Name == "":
				c.Errorf("rack names are required and must be non-empty")
			case !racks.Add(rack.Name):
				c.Errorf("duplicate rack %q", rack.Name)
			}

			c.Enter("rack %q", rack.Name)
			for _, s := range rack.Switch {
				switch {
				case s.Name == "":
					c.Errorf("switch names are required and must be non-empty")
				case !switches.Add(s.Name):
					c.Errorf("duplicate switch %q", s.Name)
				}

				c.Enter("switch %q", s.Name)
				switch {
				case s.Ports < 1:
					c.Errorf("switches must have at least one port")
				case s.Ports > switchMaxPorts:
					c.Errorf("switches must have at most %d ports", switchMaxPorts)
				}
				c.Exit()
			}
			c.Exit()
		}
		for _, kvm := range dc.Kvm {
			switch {
			case kvm.Name == "":
				c.Errorf("KVM names are required and must be non-empty")
			case !kvms.Add(kvm.Name):
				c.Errorf("duplicate KVM %q", kvm.Name)
			}
			c.Enter("kvm %q", kvm.Name)
			if !racks.Has(kvm.Rack) {
				c.Errorf("rack %q does not exist", kvm.Rack)
			}
			_, err := common.ParseMAC48(kvm.MacAddress)
			if err != nil {
				c.Errorf("invalid MAC-48 address %q", kvm.MacAddress)
			}
			_, err = common.ParseIPv4(kvm.Ipv4)
			if err != nil {
				c.Errorf("invalid IPv4 address %q", kvm.Ipv4)
			}
			c.Exit()
		}
		for _, rack := range dc.Rack {
			if rack.Kvm != "" {
				c.Enter("rack %q", rack.Name)
				if !kvms.Has(rack.Kvm) {
					c.Errorf("KVM %q does not exist", rack.Kvm)
				}
				c.Exit()
			}
		}
		c.Exit()
	}
}
