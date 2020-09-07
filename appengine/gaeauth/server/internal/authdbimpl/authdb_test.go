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

package authdbimpl

import (
	"context"
	"fmt"
	"testing"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/auth/service"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestConfigureAuthService(t *testing.T) {
	Convey("Initial config", t, func() {
		srv := &fakeAuthService{LatestRev: 123}
		ctx := setAuthService(gaetesting.TestingContext(), srv)

		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service"), ShouldBeNil)
		So(srv.Calls, ShouldResemble, []string{
			`EnsureSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service" ""`,
		})

		info, err := GetLatestSnapshotInfo(ctx)
		So(err, ShouldBeNil)
		So(info, ShouldResemble, &SnapshotInfo{
			AuthServiceURL: "http://auth-service",
			Rev:            123,
		})

		// Coverage for GetAuthDBSnapshot.
		_, err = GetAuthDBSnapshot(ctx, "missing")
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)
		snap, err := GetAuthDBSnapshot(ctx, info.GetSnapshotID())
		So(err, ShouldBeNil)
		So(snap, ShouldResembleProto, &protocol.AuthDB{
			OauthClientId:     "client-id-for-rev-123",
			OauthClientSecret: "secret",
		})

		// Same config call again triggers resubsciption.
		srv.Calls = nil
		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service"), ShouldBeNil)
		So(srv.Calls, ShouldResemble, []string{
			`EnsureSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service" ""`,
		})
	})

	Convey("Switching cfg", t, func() {
		srv := &fakeAuthService{LatestRev: 123}
		ctx := setAuthService(gaetesting.TestingContext(), srv)

		// Initial config.
		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service-1"), ShouldBeNil)
		// Change URL of the service.
		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service-2"), ShouldBeNil)

		info, err := GetLatestSnapshotInfo(ctx)
		So(err, ShouldBeNil)
		So(info, ShouldResemble, &SnapshotInfo{
			AuthServiceURL: "http://auth-service-2",
			Rev:            123,
		})

		So(srv.Calls, ShouldResemble, []string{
			`EnsureSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service-1" ""`,
			`EnsureSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service-2" ""`,
			`DeleteSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service-1"`,
		})
	})

	Convey("Removing cfg", t, func() {
		srv := &fakeAuthService{LatestRev: 123}
		ctx := setAuthService(gaetesting.TestingContext(), srv)

		// Initial config.
		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service-1"), ShouldBeNil)
		// Remove.
		So(ConfigureAuthService(ctx, "http://base_url", ""), ShouldBeNil)

		info, err := GetLatestSnapshotInfo(ctx)
		So(err, ShouldBeNil)
		So(info, ShouldBeNil)

		So(srv.Calls, ShouldResemble, []string{
			`EnsureSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service-1" ""`,
			`DeleteSubscription "projects/app/subscriptions/dev-app-server-v1+auth-service-1"`,
		})
	})
}

func TestSyncAuthDB(t *testing.T) {
	Convey("No new changes", t, func() {
		srv := &fakeAuthService{LatestRev: 123}
		ctx := setAuthService(gaetesting.TestingContext(), srv)
		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service"), ShouldBeNil)

		info, err := syncAuthDB(ctx)
		So(err, ShouldBeNil)
		So(info, ShouldResemble, &SnapshotInfo{
			AuthServiceURL: "http://auth-service",
			Rev:            123,
		})
	})

	Convey("Have update", t, func() {
		srv := &fakeAuthService{LatestRev: 123}
		ctx := setAuthService(gaetesting.TestingContext(), srv)
		So(ConfigureAuthService(ctx, "http://base_url", "http://auth-service"), ShouldBeNil)

		srv.LatestRev = 456

		info, err := syncAuthDB(ctx)
		So(err, ShouldBeNil)
		So(info, ShouldResemble, &SnapshotInfo{
			AuthServiceURL: "http://auth-service",
			Rev:            456,
		})
	})
}

///

type fakeAuthService struct {
	LatestRev    int64
	Calls        []string
	Notification *service.Notification
}

func (f *fakeAuthService) EnsureSubscription(ctx context.Context, subscription, pushURL string) error {
	f.Calls = append(f.Calls, fmt.Sprintf("EnsureSubscription %q %q", subscription, pushURL))
	return nil
}

func (f *fakeAuthService) DeleteSubscription(ctx context.Context, subscription string) error {
	f.Calls = append(f.Calls, fmt.Sprintf("DeleteSubscription %q", subscription))
	return nil
}

func (f *fakeAuthService) PullPubSub(ctx context.Context, subscription string) (*service.Notification, error) {
	f.Calls = append(f.Calls, fmt.Sprintf("PullPubSub %q", subscription))
	return f.Notification, nil
}

func (f *fakeAuthService) ProcessPubSubPush(ctx context.Context, body []byte) (*service.Notification, error) {
	f.Calls = append(f.Calls, "ProcessPubSubPush")
	return f.Notification, nil
}

func (f *fakeAuthService) GetLatestSnapshotRevision(ctx context.Context) (int64, error) {
	return f.LatestRev, nil
}

func (f *fakeAuthService) GetSnapshot(ctx context.Context, rev int64) (*service.Snapshot, error) {
	if rev != f.LatestRev {
		return nil, fmt.Errorf("fakeAuthService: no snapshot for rev %d", rev)
	}
	return &service.Snapshot{
		AuthDB: &protocol.AuthDB{
			OauthClientId:     fmt.Sprintf("client-id-for-rev-%d", f.LatestRev),
			OauthClientSecret: "secret",
		},
		Rev: f.LatestRev,
	}, nil
}
