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

package serviceaccountsv2

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils/policy"

	. "github.com/smartystreets/goconvey/convey"
)

const fakeMappingConfig = `
mapping {
	project: "proj1"
	project: "proj2"
	service_account: "sa1@example.com"
	service_account: "sa2@example.com"
}

mapping {
	project: "proj3"
	service_account: "sa3@example.com"
}

mapping {
	project: "proj4"
}
`

func TestMapping(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		ctx := context.Background()

		mapping, err := loadMapping(ctx, fakeMappingConfig)
		So(err, ShouldBeNil)
		So(mapping, ShouldNotBeNil)

		So(mapping.CanProjectUseAccount("proj1", "sa1@example.com"), ShouldBeTrue)
		So(mapping.CanProjectUseAccount("proj2", "sa1@example.com"), ShouldBeTrue)
		So(mapping.CanProjectUseAccount("proj3", "sa1@example.com"), ShouldBeFalse)
		So(mapping.CanProjectUseAccount("proj4", "sa1@example.com"), ShouldBeFalse)

		So(mapping.CanProjectUseAccount("proj1", "sa2@example.com"), ShouldBeTrue)
		So(mapping.CanProjectUseAccount("proj2", "sa2@example.com"), ShouldBeTrue)
		So(mapping.CanProjectUseAccount("proj3", "sa2@example.com"), ShouldBeFalse)
		So(mapping.CanProjectUseAccount("proj4", "sa2@example.com"), ShouldBeFalse)

		So(mapping.CanProjectUseAccount("proj1", "sa3@example.com"), ShouldBeFalse)
		So(mapping.CanProjectUseAccount("proj2", "sa3@example.com"), ShouldBeFalse)
		So(mapping.CanProjectUseAccount("proj3", "sa3@example.com"), ShouldBeTrue)
		So(mapping.CanProjectUseAccount("proj4", "sa3@example.com"), ShouldBeFalse)
	})
}

func loadMapping(ctx context.Context, text string) (*Mapping, error) {
	cfg := &admin.ServiceAccountsProjectMapping{}
	err := proto.UnmarshalText(text, cfg)
	if err != nil {
		return nil, err
	}
	mapping, err := prepareMapping(ctx, policy.ConfigBundle{configFileName: cfg}, "fake-revision")
	if err != nil {
		return nil, err
	}
	return mapping.(*Mapping), nil
}
