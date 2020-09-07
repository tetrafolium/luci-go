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

// Package memlock allows multiple appengine handlers to coordinate best-effort
// mutual execution via memcache. "best-effort" here means "best-effort"...
// memcache is not reliable. However, colliding on memcache is a lot cheaper
// than, for example, colliding with datastore transactions.
package memlock

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	mc "github.com/tetrafolium/luci-go/gae/service/memcache"
)

// ErrFailedToLock is returned from TryWithLock when it fails to obtain a lock
// prior to invoking the user-supplied function.
var ErrFailedToLock = errors.New("memlock: failed to obtain lock")

// ErrEmptyClientID is returned from TryWithLock when you specify an empty
// clientID.
var ErrEmptyClientID = errors.New("memlock: empty clientID")

// memlockKeyPrefix is the memcache Key prefix for all user-supplied keys.
const memlockKeyPrefix = "memlock:"

type checkOp string

// var so we can override it in the tests
var delay = time.Second

type testStopCBKeyType int

var testStopCBKey testStopCBKeyType

const (
	release checkOp = "release"
	refresh         = "refresh"
)

// memcacheLockTime is the expiration time of the memcache entry. If the lock
// is correctly released, then it will be released before this time. It's a
// var so we can override it in the tests.
var memcacheLockTime = 16 * time.Second

// TryWithLock attempts to obtains the lock once, and then invokes f if
// successful. The context provided to f will be canceled (e.g. ctx.Done() will
// be closed) if memlock detects that we've lost the lock.
//
// TryWithLock function returns ErrFailedToLock if it fails to obtain the lock,
// otherwise returns the error that f returns.
//
// `key` is the memcache key to use (i.e. the name of the lock). Clients locking
// the same data must use the same key. clientID is the unique identifier for
// this client (lock-holder). If it's empty then TryWithLock() will return
// ErrEmptyClientID.
//
// Note that the lock provided by TryWithLock is a best-effort lock... some
// other form of locking or synchronization should be used inside of f (such as
// Datastore transactions) to ensure that f is, in fact, operating exclusively.
// The purpose of TryWithLock is to have a cheap filter to prevent unnecessary
// contention on heavier synchronization primitives like transactions.
func TryWithLock(ctx context.Context, key, clientID string, f func(context.Context) error) error {
	if len(clientID) == 0 {
		return ErrEmptyClientID
	}

	log := logging.Get(
		logging.SetFields(ctx, logging.Fields{
			"key":      key,
			"clientID": clientID,
		}))

	key = memlockKeyPrefix + key
	cid := []byte(clientID)

	// checkAnd gets the current value from memcache, and then attempts to do the
	// checkOp (which can either be `refresh` or `release`). These pieces of
	// functionality are necessarially intertwined, because CAS only works with
	// the exact-same *Item which was returned from a Get.
	//
	// refresh will attempt to CAS the item with the same content to reset it's
	// timeout.
	//
	// release will attempt to CAS the item to remove it's contents (clientID).
	// another lock observing an empty clientID will know that the lock is
	// obtainable.
	checkAnd := func(op checkOp) bool {
		limitedRetry := func() retry.Iterator {
			return &retry.Limited{
				Delay:   time.Second,
				Retries: 5,
			}
		}
		var itm mc.Item
		if err := retry.Retry(ctx, limitedRetry, func() (err error) {
			itm, err = mc.GetKey(ctx, key)
			return
		}, retry.LogCallback(ctx, "getting lock from memcache")); err != nil {
			log.Warningf("permanent error getting: %s", err)
			return false
		}

		if len(itm.Value()) > 0 && !bytes.Equal(itm.Value(), cid) {
			log.Infof("lock owned by %q", string(itm.Value()))
			return false
		}

		if op == refresh {
			itm.SetValue(cid).SetExpiration(memcacheLockTime)
		} else {
			if len(itm.Value()) == 0 {
				// it's already unlocked, no need to CAS
				log.Infof("lock already released")
				return true
			}
			itm.SetValue([]byte{}).SetExpiration(delay)
		}

		if err := mc.CompareAndSwap(ctx, itm); err != nil {
			log.Warningf("failed to %s lock: %q", op, err)
			return false
		}

		return true
	}

	// Now the actual logic begins. First we 'Add' the item, which will set it if
	// it's not present in the memcache, otherwise leaves it alone.

	err := mc.Add(ctx, mc.NewItem(ctx, key).SetValue(cid).SetExpiration(memcacheLockTime))
	if err != nil {
		if err != mc.ErrNotStored {
			log.Warningf("error adding: %s", err)
		}
		if !checkAnd(refresh) {
			return ErrFailedToLock
		}
	}

	// At this point we nominally have the lock (at least for memcacheLockTime).
	finished := make(chan struct{})
	subCtx, cancelFunc := context.WithCancel(ctx)
	defer func() {
		cancelFunc()
		<-finished
	}()

	testStopCB, _ := ctx.Value(testStopCBKey).(func())

	// This goroutine checks to see if we still possess the lock, and refreshes it
	// if we do.
	go func() {
		defer func() {
			cancelFunc()
			close(finished)
		}()

		tmr := clock.NewTimer(subCtx)
		defer tmr.Stop()
		for {
			tmr.Reset(delay)

			if tr := <-tmr.GetC(); tr.Incomplete() {
				if tr.Err != context.Canceled {
					log.Debugf("context done: %s", tr.Err)
				}
				break
			}
			if !checkAnd(refresh) {
				log.Warningf("lost lock: %s", err)
				return
			}
		}

		if testStopCB != nil {
			testStopCB()
		}
		checkAnd(release)
	}()

	return f(subCtx)
}
