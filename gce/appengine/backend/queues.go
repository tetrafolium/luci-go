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

package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"

	"google.golang.org/api/googleapi"

	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/gce/api/tasks/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/backend/internal/metrics"
	"github.com/tetrafolium/luci-go/gce/appengine/model"
)

// countVMsQueue is the name of the count VMs task handler queue.
const countVMsQueue = "count-vms"

// countVMs counts the VMs for a given config.
func countVMs(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.CountVMs)
	switch {
	case !ok:
		return errors.Reason("unexpected payload type %T", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	}
	// Count VMs per project, server and zone.
	// VMs created from the same config eventually have the same project, server,
	// and zone but may currently exist for a previous version of the config.
	vms := &metrics.InstanceCount{}

	// Get the configured count.
	cfg := &model.Config{
		ID: task.Id,
	}
	switch err := datastore.Get(c, cfg); {
	case err == datastore.ErrNoSuchEntity:
	case err != nil:
		return errors.Annotate(err, "failed to fetch config").Err()
	default:
		vms.AddConfigured(int(cfg.Config.CurrentAmount), cfg.Config.Attributes.Project)
	}

	// Get the actual (connected, created) counts.
	vm := &model.VM{}
	q := datastore.NewQuery(model.VMKind).Eq("config", task.Id)
	if err := datastore.Run(c, q, func(k *datastore.Key) error {
		id := k.StringID()
		vm.ID = id
		switch err := datastore.Get(c, vm); {
		case err == datastore.ErrNoSuchEntity:
			return nil
		case err != nil:
			return errors.Annotate(err, "failed to fetch VM").Err()
		default:
			if vm.Created > 0 {
				vms.AddCreated(1, vm.Attributes.Project, vm.Attributes.Zone)
			}
			if vm.Connected > 0 {
				vms.AddConnected(1, vm.Attributes.Project, vm.Swarming, vm.Attributes.Zone)
			}
			return nil
		}
	}); err != nil {
		return errors.Annotate(err, "failed to fetch VMs").Err()
	}
	if err := vms.Update(c, task.Id); err != nil {
		return errors.Annotate(err, "failed to update count").Err()
	}
	return nil
}

// drainVM drains a given VM if necessary.
func drainVM(c context.Context, vm *model.VM) error {
	if vm.Drained {
		return nil
	}
	cfg := &model.Config{
		ID: vm.Config,
	}
	switch err := datastore.Get(c, cfg); {
	case err == datastore.ErrNoSuchEntity:
		logging.Debugf(c, "config %q does not exist", cfg.ID)
	case err != nil:
		return errors.Annotate(err, "failed to fetch config").Err()
	}
	if cfg.Config.CurrentAmount > vm.Index {
		return nil
	}
	logging.Debugf(c, "config %q only specifies %d VMs", cfg.ID, cfg.Config.CurrentAmount)
	return datastore.RunInTransaction(c, func(c context.Context) error {
		switch err := datastore.Get(c, vm); {
		case err == datastore.ErrNoSuchEntity:
			vm.Drained = true
			return nil
		case err != nil:
			return errors.Annotate(err, "failed to fetch VM").Err()
		case vm.Drained:
			return nil
		}
		vm.Drained = true
		if err := datastore.Put(c, vm); err != nil {
			return errors.Annotate(err, "failed to store VM").Err()
		}
		return nil
	}, nil)
}

// getSuffix returns a random suffix to use when naming a GCE instance.
func getSuffix(c context.Context) string {
	const allowed = "abcdefghijklmnopqrstuvwxyz0123456789"
	suf := make([]byte, 4)
	for i := range suf {
		suf[i] = allowed[mathrand.Intn(c, len(allowed))]
	}
	return string(suf)
}

// createVMQueue is the name of the create VM task handler queue.
const createVMQueue = "create-vm"

// createVM creates a VM if it doesn't already exist.
func createVM(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.CreateVM)
	switch {
	case !ok:
		return errors.Reason("unexpected payload type %T", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	case task.GetConfig() == "":
		return errors.Reason("config is required").Err()
	}
	vm := &model.VM{
		ID:         task.Id,
		Config:     task.Config,
		Configured: clock.Now(c).Unix(),
		Hostname:   fmt.Sprintf("%s-%d-%s", task.Prefix, task.Index, getSuffix(c)),
		Index:      task.Index,
		Lifetime:   task.Lifetime,
		Prefix:     task.Prefix,
		Revision:   task.Revision,
		Swarming:   task.Swarming,
		Timeout:    task.Timeout,
	}
	if task.Attributes != nil {
		vm.Attributes = *task.Attributes
		// TODO(crbug/942301): Auto-select zone if zone is unspecified.
		vm.Attributes.SetZone(vm.Attributes.GetZone())
		vm.IndexAttributes()
	}
	// createVM is called repeatedly, so do a fast check outside the transaction.
	// In most cases, this will skip the more expensive transactional check.
	switch err := datastore.Get(c, vm); {
	case err == datastore.ErrNoSuchEntity:
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM").Err()
	default:
		return nil
	}
	return datastore.RunInTransaction(c, func(c context.Context) error {
		switch err := datastore.Get(c, vm); {
		case err == datastore.ErrNoSuchEntity:
		case err != nil:
			return errors.Annotate(err, "failed to fetch VM").Err()
		default:
			return nil
		}
		if err := datastore.Put(c, vm); err != nil {
			return errors.Annotate(err, "failed to store VM").Err()
		}
		return nil
	}, nil)
}

// updateCurrentAmount updates CurrentAmount if necessary.
// Returns up-to-date config entity and the reference timestamp.
func updateCurrentAmount(c context.Context, id string) (cfg *model.Config, now time.Time, err error) {
	cfg = &model.Config{
		ID: id,
	}
	// Avoid transaction if possible.
	if err = datastore.Get(c, cfg); err != nil {
		err = errors.Annotate(err, "failed to fetch config").Err()
		return
	}

	now = clock.Now(c)
	var amt int32
	switch amt, err = cfg.Config.ComputeAmount(cfg.Config.CurrentAmount, now); {
	case err != nil:
		err = errors.Annotate(err, "failed to parse amount").Err()
		return
	case cfg.Config.CurrentAmount == amt:
		return
	}

	err = datastore.RunInTransaction(c, func(c context.Context) error {
		var err error
		if err = datastore.Get(c, cfg); err != nil {
			return errors.Annotate(err, "failed to fetch config").Err()
		}

		now = clock.Now(c)
		switch amt, err = cfg.Config.ComputeAmount(cfg.Config.CurrentAmount, now); {
		case err != nil:
			return errors.Annotate(err, "failed to parse amount").Err()
		case cfg.Config.CurrentAmount == amt:
			return nil
		}
		cfg.Config.CurrentAmount = amt
		if err = datastore.Put(c, cfg); err != nil {
			return errors.Annotate(err, "failed to store config").Err()
		}
		return nil
	}, nil)
	return
}

// expandConfigQueue is the name of the expand config task handler queue.
const expandConfigQueue = "expand-config"

// expandConfig creates task queue tasks to create each VM in the given config.
func expandConfig(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.ExpandConfig)
	switch {
	case !ok:
		return errors.Reason("unexpected payload type %T", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	}
	cfg, now, err := updateCurrentAmount(c, task.Id)
	if err != nil {
		return err
	}
	t := make([]*tq.Task, cfg.Config.CurrentAmount)
	for i := int32(0); i < cfg.Config.CurrentAmount; i++ {
		t[i] = &tq.Task{
			Payload: &tasks.CreateVM{
				Id:         fmt.Sprintf("%s-%d", cfg.Config.Prefix, i),
				Attributes: cfg.Config.Attributes,
				Config:     task.Id,
				Created: &timestamp.Timestamp{
					Seconds: now.Unix(),
				},
				Index:    i,
				Lifetime: cfg.Config.Lifetime.GetSeconds(),
				Prefix:   cfg.Config.Prefix,
				Revision: cfg.Config.Revision,
				Swarming: cfg.Config.Swarming,
				Timeout:  cfg.Config.Timeout.GetSeconds(),
			},
		}
	}
	logging.Debugf(c, "creating %d VMs", len(t))
	if err := getDispatcher(c).AddTask(c, t...); err != nil {
		return errors.Annotate(err, "failed to schedule tasks").Err()
	}
	return nil
}

// reportQuotaQueue is the name of the report quota task handler queue.
const reportQuotaQueue = "report-quota"

// reportQuota reports GCE quota utilization.
func reportQuota(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.ReportQuota)
	switch {
	case !ok:
		return errors.Reason("unexpected payload type %T", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	}
	p := &model.Project{
		ID: task.Id,
	}
	if err := datastore.Get(c, p); err != nil {
		return errors.Annotate(err, "failed to fetch project").Err()
	}
	mets := stringset.NewFromSlice(p.Config.Metric...)
	regs := stringset.NewFromSlice(p.Config.Region...)
	rsp, err := getCompute(c).Regions.List(p.Config.Project).Context(c).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			logErrors(c, gerr)
		}
		return errors.Annotate(err, "failed to fetch quota").Err()
	}
	for _, r := range rsp.Items {
		if regs.Has(r.Name) {
			for _, q := range r.Quotas {
				if mets.Has(q.Metric) {
					metrics.UpdateQuota(c, q.Limit, q.Usage, q.Metric, p.Config.Project, r.Name)
				}
			}
		}
	}
	return nil
}
