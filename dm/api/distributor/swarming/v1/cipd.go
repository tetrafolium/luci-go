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

package swarmingV1

import (
	"sort"

	swarm "github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
)

// ToCipdPackage converts this to a swarming api SwarmingRpcsCipdPackage.
func (c *CipdPackage) ToCipdPackage() *swarm.SwarmingRpcsCipdPackage {
	if c == nil {
		return nil
	}
	return &swarm.SwarmingRpcsCipdPackage{PackageName: c.Name, Version: c.Version}
}

// ToCipdInput converts this to a swarming api SwarmingRpcsCipdInput.
func (c *CipdSpec) ToCipdInput() *swarm.SwarmingRpcsCipdInput {
	if c == nil || c.Client == nil && len(c.ByPath) == 0 {
		return nil
	}
	ret := &swarm.SwarmingRpcsCipdInput{
		ClientPackage: c.Client.ToCipdPackage(),
	}
	if len(c.ByPath) > 0 {
		count := 0
		paths := make(sort.StringSlice, 0, len(c.ByPath))
		for path, pkgs := range c.ByPath {
			paths = append(paths, path)
			count += len(pkgs.Pkg)
		}
		ret.Packages = make([]*swarm.SwarmingRpcsCipdPackage, 0, count)
		for _, path := range paths {
			for _, pkg := range c.ByPath[path].Pkg {
				retPkg := pkg.ToCipdPackage()
				retPkg.Path = path
				ret.Packages = append(ret.Packages, retPkg)
			}
		}
	}
	return ret
}
