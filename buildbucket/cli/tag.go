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

package cli

import (
	"flag"

	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
	"github.com/tetrafolium/luci-go/common/data/strpair"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
	luciflag "github.com/tetrafolium/luci-go/common/flag"
)

type tagsFlag struct {
	tags strpair.Map
}

func (f *tagsFlag) Register(fs *flag.FlagSet, help string) {
	f.tags = strpair.Map{}
	fs.Var(luciflag.StringPairs(f.tags), "t", help)
}

func (f *tagsFlag) Tags() []*pb.StringPair {
	return protoutil.StringPairs(f.tags)
}
