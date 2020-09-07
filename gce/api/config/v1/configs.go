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
	"strings"

	"github.com/tetrafolium/luci-go/config/validation"
)

// Validate validates these configs.
func (cfgs *Configs) Validate(c *validation.Context) {
	prefixes := make([]string, 0, len(cfgs.GetVms()))
	for i, cfg := range cfgs.GetVms() {
		c.Enter("vms config %d", i)
		if cfg.Prefix == "" {
			c.Errorf("prefix is required")
		}
		// Ensure no prefix is a prefix of any other prefix. Building a prefix tree
		// and waiting until the end to check this is faster, but config validation
		// isn't particularly time sensitive since configs are processed asynchronously.
		for _, p := range prefixes {
			switch {
			case strings.HasPrefix(p, cfg.Prefix):
				c.Errorf("prefix %q is a prefix of %q", cfg.Prefix, p)
			case strings.HasPrefix(cfg.Prefix, p):
				c.Errorf("prefix %q is a prefix of %q", p, cfg.Prefix)
			}
		}
		prefixes = append(prefixes, cfg.Prefix)
		cfg.Validate(c)
		c.Exit()
	}
}
