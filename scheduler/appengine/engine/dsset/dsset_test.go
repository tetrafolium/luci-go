// Copyright 2017 The LUCI Authors.
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

package dsset

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/data/stringset"

	. "github.com/smartystreets/goconvey/convey"
)

func testingContext() context.Context {
	c := memory.Use(context.Background())
	c = clock.Set(c, testclock.New(time.Unix(1442270520, 0).UTC()))
	c = mathrand.Set(c, rand.New(rand.NewSource(1000)))
	return c
}

// pop pops a bunch of items from the set and returns items that were popped.
func pop(c context.Context, s *Set, listing *Listing, ids []string) (popped []string, tombs Garbage, err error) {
	op, err := s.BeginPop(c, listing)
	if err != nil {
		return nil, nil, err
	}
	for _, id := range ids {
		if op.Pop(id) {
			popped = append(popped, id)
		}
	}
	if tombs, err = FinishPop(c, op); err != nil {
		return nil, nil, err
	}
	return popped, tombs, nil
}

func TestSet(t *testing.T) {
	t.Parallel()

	Convey("item one lifecycle", t, func() {
		c := testingContext()

		set := Set{
			ID:              "test",
			ShardCount:      3,
			TombstonesRoot:  datastore.NewKey(c, "Root", "root", 0, nil),
			TombstonesDelay: time.Minute,
		}

		// Add one item.
		So(set.Add(c, []Item{{ID: "abc"}}), ShouldBeNil)

		// The item is returned by the listing.
		listing, err := set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldResemble, []Item{{ID: "abc"}})
		So(listing.Garbage, ShouldBeNil)

		// Pop it!
		var cleanup Garbage
		err = datastore.RunInTransaction(c, func(c context.Context) error {
			popped, tombs, err := pop(c, &set, listing, []string{"abc"})
			So(err, ShouldBeNil)
			So(popped, ShouldResemble, []string{"abc"})
			So(len(tombs), ShouldEqual, 1)
			So(tombs[0].id, ShouldEqual, "abc")
			So(len(tombs[0].storage), ShouldEqual, 1)
			cleanup = tombs
			return nil
		}, nil)
		So(err, ShouldBeNil)

		// The listing no longer returns it, but we have a fresh tombstone that can
		// be cleaned up.
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldBeNil)
		So(len(listing.Garbage), ShouldEqual, 1)
		So(listing.Garbage[0].id, ShouldEqual, "abc")

		// Cleaning up the storage using tombstones from Pop works.
		So(CleanupGarbage(c, cleanup), ShouldBeNil)

		// The listing no longer returns the item, and there's no tombstones to
		// cleanup.
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldBeNil)
		So(listing.Garbage, ShouldBeNil)

		// Attempt to add it back (should be ignored). Add a bunch of times to make
		// sure to fill in many shards (this is pseudo-random).
		for i := 0; i < 5; i++ {
			So(set.Add(c, []Item{{ID: "abc"}}), ShouldBeNil)
		}

		// The listing still doesn't returns it, but we now have a tombstone to
		// cleanup (again).
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldBeNil)
		So(len(listing.Garbage), ShouldEqual, 1)
		So(listing.Garbage[0].old, ShouldBeFalse)
		So(len(listing.Garbage[0].storage), ShouldEqual, 3) // all shards

		// Popping it again doesn't work either.
		err = datastore.RunInTransaction(c, func(c context.Context) error {
			popped, tombs, err := pop(c, &set, listing, []string{"abc"})
			So(err, ShouldBeNil)
			So(popped, ShouldBeNil)
			So(tombs, ShouldBeNil)
			return nil
		}, nil)
		So(err, ShouldBeNil)

		// Cleaning up the storage, again. This should make List stop returning
		// the tombstone (since it has no storage items associated with it and it's
		// not ready to be evicted yet).
		So(CleanupGarbage(c, listing.Garbage), ShouldBeNil)
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldBeNil)
		So(listing.Garbage, ShouldBeNil)

		// Time passes, tombstone expires.
		clock.Get(c).(testclock.TestClock).Add(2 * time.Minute)

		// Listing now returns expired tombstone.
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldBeNil)
		So(len(listing.Garbage), ShouldEqual, 1)
		So(len(listing.Garbage[0].storage), ShouldEqual, 0) // cleaned already

		// Cleanup storage keys.
		So(CleanupGarbage(c, listing.Garbage), ShouldBeNil)

		// Cleanup the tombstones themselves.
		err = datastore.RunInTransaction(c, func(c context.Context) error {
			popped, tombs, err := pop(c, &set, listing, nil)
			So(err, ShouldBeNil)
			So(popped, ShouldBeNil)
			So(tombs, ShouldBeNil)
			return nil
		}, nil)
		So(err, ShouldBeNil)

		// No tombstones returned any longer.
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldBeNil)
		So(listing.Garbage, ShouldBeNil)

		// And the item can be added back now, since no trace of it is left.
		So(set.Add(c, []Item{{ID: "abc"}}), ShouldBeNil)

		// Yep, it is there.
		listing, err = set.List(c)
		So(err, ShouldBeNil)
		So(listing.Items, ShouldResemble, []Item{{ID: "abc"}})
		So(listing.Garbage, ShouldBeNil)
	})

	Convey("stress", t, func() {
		// Add 1000 items in parallel from N goroutines, and (also in parallel),
		// run N instances of "List and pop all", collecting the result in single
		// list. There should be no duplicates in the final list!
		c := testingContext()

		set := Set{
			ID:              "test",
			ShardCount:      3,
			TombstonesRoot:  datastore.NewKey(c, "Root", "root", 0, nil),
			TombstonesDelay: time.Minute,
		}

		producers := 3
		consumers := 5
		items := 100

		wakeups := make(chan string)

		lock := sync.Mutex{}
		var consumed []string

		for i := 0; i < producers; i++ {
			go func() {
				for j := 0; j < items; j++ {
					set.Add(c, []Item{{ID: fmt.Sprintf("%d", j)}})
					// Wake up 3 consumers, so they "fight".
					wakeups <- "wake"
					wakeups <- "wake"
					wakeups <- "wake"
				}
				for i := 0; i < consumers; i++ {
					wakeups <- "done"
				}
			}()
		}

		consume := func() {
			listing, err := set.List(c)
			if err != nil || len(listing.Items) == 0 {
				return
			}

			keys := make([]string, len(listing.Items))
			for i, itm := range listing.Items {
				keys[i] = itm.ID
			}

			// Try to pop all.
			var popped []string
			var tombs Garbage
			err = datastore.RunInTransaction(c, func(c context.Context) error {
				var err error
				popped, tombs, err = pop(c, &set, listing, keys)
				return err
			}, nil)
			// Best-effort storage cleanup on success.
			if err == nil {
				CleanupGarbage(c, tombs)
			}

			// Consider items consumed only if transaction has landed.
			if err == nil && len(popped) != 0 {
				lock.Lock()
				consumed = append(consumed, popped...)
				lock.Unlock()
			}
		}

		wg := sync.WaitGroup{}
		wg.Add(consumers)
		for i := 0; i < consumers; i++ {
			go func() {
				defer wg.Done()
				done := false
				for !done {
					done = (<-wakeups) == "done"
					consume()
				}
			}()
		}

		wg.Wait() // this waits for completion of the entire pipeline

		// Make sure 'consumed' is the initially produced set.
		dedup := stringset.New(len(consumed))
		for _, itm := range consumed {
			dedup.Add(itm)
		}
		So(dedup.Len(), ShouldEqual, len(consumed)) // no dups
		So(len(consumed), ShouldEqual, items)       // all are accounted for
	})
}
