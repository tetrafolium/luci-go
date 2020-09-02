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

package experiments

import (
	"context"
	"fmt"

	"github.com/tetrafolium/luci-go/common/errors"

	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	swarmingpb "github.com/tetrafolium/luci-go/swarming/proto/api"
)

func init() {
	Register("luci.use_realms", func(ctx context.Context, b *bbpb.Build, task *swarmingpb.TaskRequest) error {
		project := b.GetBuilder().GetProject()
		bucket := b.GetBuilder().GetBucket()
		if project == "" || bucket == "" {
			return errors.Reason("incomplete Builder ID, need both `project` and `bucket` set").Err()
		}
		task.Realm = fmt.Sprintf("%s:%s", project, bucket)
		return nil
	})
}
