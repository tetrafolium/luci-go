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

package engine

// This file contains helpers used by the rest of tests.

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"

	"google.golang.org/api/pubsub/v1"

	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/tsmon"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/store"
	"github.com/tetrafolium/luci-go/common/tsmon/target"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/server/auth"
	"github.com/tetrafolium/luci-go/server/auth/authtest"
	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/auth/signing/signingtest"
	"github.com/tetrafolium/luci-go/server/secrets/testsecrets"

	"github.com/tetrafolium/luci-go/scheduler/appengine/catalog"
	"github.com/tetrafolium/luci-go/scheduler/appengine/internal"
	"github.com/tetrafolium/luci-go/scheduler/appengine/messages"
	"github.com/tetrafolium/luci-go/scheduler/appengine/task"
)

const fakeAppID = "scheduler-app-id"

var epoch = time.Unix(1442270520, 0).UTC()

// getSentMetric returns sent value or nil if value wasn't sent.
func getSentMetric(c context.Context, m types.Metric, fieldVals ...interface{}) interface{} {
	return tsmon.GetState(c).Store().Get(c, m, time.Time{}, fieldVals)
}

// getSentDistrValue returns the value that was added to distribution after
// ensuring there was exactly 1 value sent.
func getSentDistrValue(c context.Context, m types.Metric, fieldVals ...interface{}) float64 {
	switch d, ok := getSentMetric(c, m, fieldVals...).(*distribution.Distribution); {
	case !ok:
		panic(errors.New("not a distribution"))
	case d.Count() != 1:
		panic(fmt.Errorf("expected 1 value, but %d values were sent with sum of %f", d.Count(), d.Sum()))
	default:
		return d.Sum()
	}
}

func allJobs(c context.Context) []Job {
	datastore.GetTestable(c).CatchupIndexes()
	entities := []Job{}
	if err := datastore.GetAll(c, datastore.NewQuery("Job"), &entities); err != nil {
		panic(err)
	}
	// Strip UTC location pointers from zero time.Time{} so that ShouldResemble
	// can compare it to default time.Time{}. nil location is UTC too.
	for i := range entities {
		ent := &entities[i]
		if ent.Cron.LastRewind.IsZero() {
			ent.Cron.LastRewind = time.Time{}
		}
		if ent.Cron.LastTick.When.IsZero() {
			ent.Cron.LastTick.When = time.Time{}
		}
	}
	return entities
}

func sortedJobIds(jobs []*Job) []string {
	ids := stringset.New(len(jobs))
	for _, j := range jobs {
		ids.Add(j.JobID)
	}
	asSlice := ids.ToSlice()
	sort.Strings(asSlice)
	return asSlice
}

func newTestContext(now time.Time) context.Context {
	c := memory.UseWithAppID(context.Background(), fakeAppID)
	c = clock.Set(c, testclock.New(now))
	c = mathrand.Set(c, rand.New(rand.NewSource(1000)))
	c = testsecrets.Use(c)

	// Signer is used by ShouldEnforceRealmACL to discover app ID.
	c = auth.ModifyConfig(c, func(cfg auth.Config) auth.Config {
		cfg.Signer = signingtest.NewSigner(&signing.ServiceInfo{
			AppID: fakeAppID,
		})
		return cfg
	})

	c, _, _ = tsmon.WithFakes(c)
	fake := store.NewInMemory(&target.Task{})
	tsmon.GetState(c).SetStore(fake)

	datastore.GetTestable(c).AddIndexes(&datastore.IndexDefinition{
		Kind: "Job",
		SortBy: []datastore.IndexColumn{
			{Property: "Enabled"},
			{Property: "ProjectID"},
		},
	})
	datastore.GetTestable(c).CatchupIndexes()

	return c
}

func newTestEngine() (*engineImpl, *fakeTaskManager) {
	mgr := &fakeTaskManager{}
	cat := catalog.New()
	cat.RegisterTaskManager(mgr)
	return NewEngine(Config{
		Catalog:        cat,
		Dispatcher:     &tq.Dispatcher{},
		PubSubPushPath: "/push-url",
	}).(*engineImpl), mgr
}

func mockEnforceRealmACL(realm string) authtest.MockedDatum {
	return authtest.MockRealmData(realm, &protocol.RealmData{
		EnforceInService: []string{fakeAppID},
	})
}

////

// fakeTaskManager implement task.Manager interface.
type fakeTaskManager struct {
	launchTask         func(ctx context.Context, ctl task.Controller) error
	abortTask          func(ctx context.Context, ctl task.Controller) error
	handleNotification func(ctx context.Context, msg *pubsub.PubsubMessage) error
	handleTimer        func(ctx context.Context, ctl task.Controller, name string, payload []byte) error
}

func (m *fakeTaskManager) Name() string {
	return "fake"
}

func (m *fakeTaskManager) ProtoMessageType() proto.Message {
	return (*messages.NoopTask)(nil)
}

func (m *fakeTaskManager) Traits() task.Traits {
	return task.Traits{}
}

func (m *fakeTaskManager) ValidateProtoMessage(c *validation.Context, msg proto.Message) {}

func (m *fakeTaskManager) LaunchTask(c context.Context, ctl task.Controller) error {
	return m.launchTask(c, ctl)
}

func (m *fakeTaskManager) AbortTask(c context.Context, ctl task.Controller) error {
	if m.abortTask != nil {
		return m.abortTask(c, ctl)
	}
	return nil
}

func (m *fakeTaskManager) HandleNotification(c context.Context, ctl task.Controller, msg *pubsub.PubsubMessage) error {
	return m.handleNotification(c, msg)
}

func (m fakeTaskManager) HandleTimer(c context.Context, ctl task.Controller, name string, payload []byte) error {
	return m.handleTimer(c, ctl, name, payload)
}

func (m fakeTaskManager) GetDebugState(c context.Context, ctl task.ControllerReadOnly) (*internal.DebugManagerState, error) {
	return nil, errors.New("not implemented")
}
