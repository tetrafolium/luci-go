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

package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/tetrafolium/luci-go/appengine/gaeauth/server/internal/authdbimpl"
	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/auth/authdb"
	"github.com/tetrafolium/luci-go/server/auth/service"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetAuthDB(t *testing.T) {
	Convey("Unconfigured", t, func() {
		ctx := gaetesting.TestingContext()
		authDB, err := GetAuthDB(ctx, nil)
		So(err, ShouldBeNil)
		So(authDB, ShouldHaveSameTypeAs, authdb.DevServerDB{})
	})

	Convey("Reuses instance if no changes", t, func() {
		ctx := gaetesting.TestingContext()

		bumpAuthDB(ctx, 123)
		authDB, err := GetAuthDB(ctx, nil)
		So(err, ShouldBeNil)
		So(authDB, ShouldHaveSameTypeAs, &authdb.SnapshotDB{})
		So(authDB.(*authdb.SnapshotDB).Rev, ShouldEqual, 123)

		newOne, err := GetAuthDB(ctx, authDB)
		So(err, ShouldBeNil)
		So(newOne, ShouldEqual, authDB) // exact same pointer

		bumpAuthDB(ctx, 124)
		anotherOne, err := GetAuthDB(ctx, authDB)
		So(err, ShouldBeNil)
		So(anotherOne, ShouldHaveSameTypeAs, &authdb.SnapshotDB{})
		So(anotherOne.(*authdb.SnapshotDB).Rev, ShouldEqual, 124)
	})
}

///

func bumpAuthDB(ctx context.Context, rev int64) {
	blob, err := service.DeflateAuthDB(&protocol.AuthDB{
		OauthClientId:     fmt.Sprintf("client-id-for-rev-%d", rev),
		OauthClientSecret: "secret",
	})
	if err != nil {
		panic(err)
	}
	info := authdbimpl.SnapshotInfo{
		AuthServiceURL: "https://fake-auth-service",
		Rev:            rev,
	}
	if err = ds.Put(ctx, &info); err != nil {
		panic(err)
	}
	err = ds.Put(ctx, &authdbimpl.Snapshot{
		ID:             info.GetSnapshotID(),
		AuthDBDeflated: blob,
	})
	if err != nil {
		panic(err)
	}
}
