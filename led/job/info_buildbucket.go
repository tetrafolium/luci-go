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

package job

import (
	bbpb "go.chromium.org/luci/buildbucket/proto"
	swarmingpb "go.chromium.org/luci/swarming/proto/api"
)

type bbInfo struct {
	*Buildbucket

	userPayload *swarmingpb.CASTree
}

var _ Info = bbInfo{}

func (b bbInfo) SwarmingHostname() string {
	return b.GetBbagentArgs().GetBuild().GetInfra().GetSwarming().GetHostname()
}

func (b bbInfo) TaskName() string {
	return b.GetName()
}

func (b bbInfo) CurrentIsolated() (*swarmingpb.CASTree, error) {
	return b.userPayload, nil
}

func (b bbInfo) Env() (ret map[string]string, err error) {
	ret = make(map[string]string, len(b.EnvVars))
	for _, pair := range b.EnvVars {
		ret[pair.Key] = pair.Value
	}
	return
}

func (b bbInfo) Priority() int32 {
	return b.GetBbagentArgs().GetBuild().GetInfra().GetSwarming().GetPriority()
}

func (b bbInfo) PrefixPathEnv() (ret []string, err error) {
	for _, keyVals := range b.EnvPrefixes {
		if keyVals.Key == "PATH" {
			ret = make([]string, len(keyVals.Values))
			copy(ret, keyVals.Values)
			break
		}
	}
	return
}

func (b bbInfo) Tags() (ret []string) {
	panic("implement me")
}

func (b bbInfo) Experimental() bool {
	return b.GetBbagentArgs().GetBuild().GetInput().GetExperimental()
}

func (b bbInfo) Properties() (ret map[string]string, err error) {
	panic("implement me")
}

func (b bbInfo) GerritChanges() (ret []*bbpb.GerritChange) {
	panic("implement me")
}

func (b bbInfo) GitilesCommit() (ret *bbpb.GitilesCommit) {
	panic("implement me")
}

func (b bbInfo) TaskPayload() (cipdPkg, cipdVers string, pathInTask string) {
	panic("implement me")
}
