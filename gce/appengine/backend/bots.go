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

package backend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto"

	"google.golang.org/api/googleapi"

	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/gae/service/memcache"

	"github.com/tetrafolium/luci-go/gce/api/tasks/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/backend/internal/metrics"
	"github.com/tetrafolium/luci-go/gce/appengine/model"
)

// utcRFC3339 is the timestamp format used by Swarming.
// Similar to RFC3339 but with an implicit UTC time zone.
const utcRFC3339 = "2006-01-02T15:04:05"

// setConnected sets the Swarming bot as connected in the datastore if it isn't already.
func setConnected(c context.Context, id, hostname string, at time.Time) error {
	// Check dscache in case vm.Connected is set now, to avoid a costly transaction.
	vm := &model.VM{
		ID: id,
	}
	switch err := datastore.Get(c, vm); {
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM").Err()
	case vm.Hostname != hostname:
		return errors.Reason("bot %q does not exist", hostname).Err()
	case vm.Connected > 0:
		return nil
	}
	put := false
	err := datastore.RunInTransaction(c, func(c context.Context) error {
		put = false
		switch err := datastore.Get(c, vm); {
		case err != nil:
			return errors.Annotate(err, "failed to fetch VM").Err()
		case vm.Hostname != hostname:
			return errors.Reason("bot %q does not exist", hostname).Err()
		case vm.Connected > 0:
			return nil
		}
		vm.Connected = at.Unix()
		if err := datastore.Put(c, vm); err != nil {
			return errors.Annotate(err, "failed to store VM").Err()
		}
		put = true
		return nil
	}, nil)
	if put && err == nil {
		metrics.ReportConnectionTime(c, float64(vm.Connected-vm.Created), vm.Prefix, vm.Attributes.GetProject(), vm.Swarming, vm.Attributes.GetZone())
	}
	return err
}

// manageMissingBot manages a missing Swarming bot.
func manageMissingBot(c context.Context, vm *model.VM) error {
	// Set that the bot has not yet connected to Swarming.
	switch {
	case vm.Lifetime > 0 && vm.Created+vm.Lifetime < time.Now().Unix():
		logging.Debugf(c, "deadline %d exceeded", vm.Created+vm.Lifetime)
		return destroyInstanceAsync(c, vm.ID, vm.URL)
	case vm.Drained:
		logging.Debugf(c, "VM drained")
		return destroyInstanceAsync(c, vm.ID, vm.URL)
	case vm.Timeout > 0 && vm.Created+vm.Timeout < time.Now().Unix():
		logging.Debugf(c, "timeout %d exceeded", vm.Created+vm.Timeout)
		return destroyInstanceAsync(c, vm.ID, vm.URL)
	default:
		return nil
	}
}

// manageExistingBot manages an existing Swarming bot.
func manageExistingBot(c context.Context, bot *swarming.SwarmingRpcsBotInfo, vm *model.VM) error {
	// A bot connected to Swarming may be executing workload.
	// To destroy the instance, terminate the bot first to avoid interruptions.
	// Termination can be skipped if the bot is deleted, dead, or already terminated.
	switch {
	case bot.Deleted:
		logging.Debugf(c, "bot deleted")
		return destroyInstanceAsync(c, vm.ID, vm.URL)
	case bot.IsDead:
		logging.Debugf(c, "bot dead")
		return destroyInstanceAsync(c, vm.ID, vm.URL)
	}
	// This value of vm.Connected may be several seconds old, because the VM was fetched
	// prior to sending an RPC to Swarming. Still, check it here to save a costly operation
	// in setConnected, since manageExistingBot may be called thousands of times per minute.
	if vm.Connected == 0 {
		t, err := time.Parse(utcRFC3339, bot.FirstSeenTs)
		if err != nil {
			return errors.Annotate(err, "failed to parse bot connection time").Err()
		}
		if err := setConnected(c, vm.ID, vm.Hostname, t); err != nil {
			return err
		}
	}
	srv := getSwarming(c, vm.Swarming).Bot
	// bot_terminate occurs when the bot starts the termination task and is normally followed
	// by task_completed and bot_shutdown. A terminated bot has no further events. Responses
	// also include the full set of dimensions when the event was recorded. Limit response size
	// by fetching only recent events, and only the type of each.
	events, err := srv.Events(vm.Hostname).Context(c).Fields("items/event_type").Limit(5).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			logErrors(c, gerr)
		}
		return errors.Annotate(err, "failed to fetch bot events").Err()
	}
	for _, e := range events.Items {
		if e.EventType == "bot_terminate" {
			logging.Debugf(c, "bot terminated")
			return destroyInstanceAsync(c, vm.ID, vm.URL)
		}
	}
	switch {
	case vm.Lifetime > 0 && vm.Created+vm.Lifetime < time.Now().Unix():
		logging.Debugf(c, "deadline %d exceeded", vm.Created+vm.Lifetime)
		return terminateBotAsync(c, vm.ID, vm.Hostname)
	case vm.Drained:
		logging.Debugf(c, "VM drained")
		return terminateBotAsync(c, vm.ID, vm.Hostname)
	}
	return nil
}

// manageBotQueue is the name of the manage bot task handler queue.
const manageBotQueue = "manage-bot"

// manageBot manages a Swarming bot.
func manageBot(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.ManageBot)
	switch {
	case !ok:
		return errors.Reason("unexpected payload %q", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	}
	vm := &model.VM{
		ID: task.Id,
	}
	switch err := datastore.Get(c, vm); {
	case err == datastore.ErrNoSuchEntity:
		return nil
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM").Err()
	case vm.URL == "":
		logging.Debugf(c, "instance %q does not exist", vm.Hostname)
		return nil
	}
	// Drain the VM if necessary. No-op if unnecessary or already drained.
	if err := drainVM(c, vm); err != nil {
		return errors.Annotate(err, "failed to drain VM").Err()
	}

	logging.Debugf(c, "fetching bot %q: %s", vm.Hostname, vm.Swarming)
	bot, err := getSwarming(c, vm.Swarming).Bot.Get(vm.Hostname).Context(c).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			if gerr.Code == http.StatusNotFound {
				logging.Debugf(c, "bot not found")
				return manageMissingBot(c, vm)
			}
			logErrors(c, gerr)
		}
		return errors.Annotate(err, "failed to fetch bot").Err()
	}
	logging.Debugf(c, "found bot")
	return manageExistingBot(c, bot, vm)
}

// terminateBotAsync schedules a task queue task to terminate a Swarming bot.
func terminateBotAsync(c context.Context, id, hostname string) error {
	t := &tq.Task{
		Payload: &tasks.TerminateBot{
			Id:       id,
			Hostname: hostname,
		},
	}
	if err := getDispatcher(c).AddTask(c, t); err != nil {
		return errors.Annotate(err, "failed to schedule terminate task").Err()
	}
	return nil
}

// terminateBotQueue is the name of the terminate bot task handler queue.
const terminateBotQueue = "terminate-bot"

// terminateBot terminates an existing Swarming bot.
func terminateBot(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.TerminateBot)
	switch {
	case !ok:
		return errors.Reason("unexpected payload %q", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	case task.GetHostname() == "":
		return errors.Reason("hostname is required").Err()
	}
	vm := &model.VM{
		ID: task.Id,
	}
	switch err := datastore.Get(c, vm); {
	case err == datastore.ErrNoSuchEntity:
		return nil
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM").Err()
	case vm.Hostname != task.Hostname:
		// Instance is already destroyed and replaced. Don't terminate the new bot.
		logging.Debugf(c, "bot %q does not exist", task.Hostname)
		return nil
	}
	// 1 terminate task should normally suffice to terminate a bot, but in case
	// users or bugs interfere, resend terminate task after 1 hour. Even if
	// memcache content is cleared, more frequent terminate requests can now be
	// handled by the swarming server and after crbug/982840 will be auto-deduped.
	mi := memcache.NewItem(c, fmt.Sprintf("terminate-%s/%s", vm.Swarming, vm.Hostname))
	if err := memcache.Get(c, mi); err == nil {
		logging.Debugf(c, "bot %q already has terminate task from us", task.Hostname)
		return nil
	}
	logging.Debugf(c, "terminating bot %q: %s", vm.Hostname, vm.Swarming)
	srv := getSwarming(c, vm.Swarming)
	if _, err := srv.Bot.Terminate(vm.Hostname).Context(c).Do(); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			if gerr.Code == http.StatusNotFound {
				// Bot is already deleted.
				logging.Debugf(c, "bot not found")
				return nil
			}
			logErrors(c, gerr)
		}
		return errors.Annotate(err, "failed to terminate bot").Err()
	}
	if err := memcache.Set(c, mi.SetExpiration(time.Hour)); err != nil {
		logging.Warningf(c, "failed to record terminate task in memcache: %s", err)
	}
	return nil
}

// deleteBotAsync schedules a task queue task to delete a Swarming bot.
func deleteBotAsync(c context.Context, id, hostname string) error {
	t := &tq.Task{
		Payload: &tasks.DeleteBot{
			Id:       id,
			Hostname: hostname,
		},
	}
	if err := getDispatcher(c).AddTask(c, t); err != nil {
		return errors.Annotate(err, "failed to schedule delete task").Err()
	}
	return nil
}

// deleteBotQueue is the name of the delete bot task handler queue.
const deleteBotQueue = "delete-bot"

// deleteBot deletes an existing Swarming bot.
func deleteBot(c context.Context, payload proto.Message) error {
	task, ok := payload.(*tasks.DeleteBot)
	switch {
	case !ok:
		return errors.Reason("unexpected payload %q", payload).Err()
	case task.GetId() == "":
		return errors.Reason("ID is required").Err()
	case task.GetHostname() == "":
		return errors.Reason("hostname is required").Err()
	}
	vm := &model.VM{
		ID: task.Id,
	}
	switch err := datastore.Get(c, vm); {
	case err == datastore.ErrNoSuchEntity:
		return nil
	case err != nil:
		return errors.Annotate(err, "failed to fetch VM").Err()
	case vm.Hostname != task.Hostname:
		// Instance is already destroyed and replaced. Don't delete the new bot.
		return errors.Reason("bot %q does not exist", task.Hostname).Err()
	}
	logging.Debugf(c, "deleting bot %q: %s", vm.Hostname, vm.Swarming)
	srv := getSwarming(c, vm.Swarming).Bot
	_, err := srv.Delete(vm.Hostname).Context(c).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			if gerr.Code == http.StatusNotFound {
				// Bot is already deleted.
				logging.Debugf(c, "bot not found")
				return deleteVM(c, task.Id, vm.Hostname)
			}
			logErrors(c, gerr)
		}
		return errors.Annotate(err, "failed to delete bot").Err()
	}
	return deleteVM(c, task.Id, vm.Hostname)
}

// deleteVM deletes the given VM from the datastore if it exists.
func deleteVM(c context.Context, id, hostname string) error {
	vm := &model.VM{
		ID: id,
	}
	return datastore.RunInTransaction(c, func(c context.Context) error {
		switch err := datastore.Get(c, vm); {
		case err == datastore.ErrNoSuchEntity:
			return nil
		case err != nil:
			return errors.Annotate(err, "failed to fetch VM").Err()
		case vm.Hostname != hostname:
			logging.Debugf(c, "VM %q does not exist", hostname)
			return nil
		}
		if err := datastore.Delete(c, vm); err != nil {
			return errors.Annotate(err, "failed to delete VM").Err()
		}
		return nil
	}, nil)
}
