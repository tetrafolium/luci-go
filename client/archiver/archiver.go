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

package archiver

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/tetrafolium/luci-go/client/internal/common"
	"github.com/tetrafolium/luci-go/client/internal/progress"
	"github.com/tetrafolium/luci-go/common/api/isolate/isolateservice/v1"
	"github.com/tetrafolium/luci-go/common/data/text/units"
	"github.com/tetrafolium/luci-go/common/isolated"
	"github.com/tetrafolium/luci-go/common/isolatedclient"
	"github.com/tetrafolium/luci-go/common/logging"
)

const (
	groupFound      progress.Group   = 0
	groupFoundFound progress.Section = 0

	groupHash         progress.Group   = 1
	groupHashDone     progress.Section = 0
	groupHashDoneSize progress.Section = 1
	groupHashTodo     progress.Section = 2

	groupLookup     progress.Group   = 2
	groupLookupDone progress.Section = 0
	groupLookupTodo progress.Section = 1

	groupUpload         progress.Group   = 3
	groupUploadDone     progress.Section = 0
	groupUploadDoneSize progress.Section = 1
	groupUploadTodo     progress.Section = 2
	groupUploadTodoSize progress.Section = 3
)

var headers = [][]progress.Column{
	{{Name: "found"}},
	{
		{Name: "hashed"},
		{Name: "size", Formatter: units.SizeToString},
		{Name: "to hash"},
	},
	{{Name: "looked up"}, {Name: "to lookup"}},
	{
		{Name: "uploaded"},
		{Name: "size", Formatter: units.SizeToString},
		{Name: "to upload"},
		{Name: "size", Formatter: units.SizeToString},
	},
}

// UploadStat is the statistic for a single upload.
type UploadStat struct {
	Duration time.Duration
	Size     units.Size
	Name     string
}

// Stats is statistics from the Archiver.
type Stats struct {
	Hits   []units.Size  // Bytes; each item is immutable.
	Pushed []*UploadStat // Misses; each item is immutable.
}

// TotalHits is the number of cache hits on the server.
func (s *Stats) TotalHits() int {
	return len(s.Hits)
}

// TotalBytesHits is the number of bytes not uploaded due to cache hits on the
// server.
func (s *Stats) TotalBytesHits() units.Size {
	out := units.Size(0)
	for _, i := range s.Hits {
		out += i
	}
	return out
}

// PackedHits returns the size of hit items in packed format.
func (s *Stats) PackedHits() ([]byte, error) {
	hits := make([]int64, len(s.Hits))
	for i, v := range s.Hits {
		hits[i] = int64(v)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i] < hits[j] })
	return isolated.Pack(hits)
}

// TotalMisses returns the number of cache misses on the server.
func (s *Stats) TotalMisses() int {
	return len(s.Pushed)
}

// TotalBytesPushed returns the sum of bytes uploaded.
func (s *Stats) TotalBytesPushed() units.Size {
	out := units.Size(0)
	for _, i := range s.Pushed {
		out += i.Size
	}
	return out
}

// PackedMisses returns size of missed items in packed format.
func (s *Stats) PackedMisses() ([]byte, error) {
	misses := make([]int64, len(s.Pushed))
	for i, v := range s.Pushed {
		misses[i] = int64(v.Size)
	}
	sort.Slice(misses, func(i, j int) bool { return misses[i] < misses[j] })
	return isolated.Pack(misses)
}

func (s *Stats) deepCopy() *Stats {
	// Only need to copy the slice, not the items themselves.
	return &Stats{s.Hits, s.Pushed}
}

// New returns a thread-safe Archiver instance.
//
// If not nil, out will contain tty-oriented progress information.
//
// ctx will be used for logging.
func New(ctx context.Context, c *isolatedclient.Client, out io.Writer) *Archiver {
	// TODO(maruel): Cache hashes and server cache presence.
	ctx2, cancel := context.WithCancel(ctx)
	a := &Archiver{
		ctx:      ctx2,
		cancel:   cancel,
		progress: progress.New(headers, out),
		c:        c,
		stage1:   make(chan *PendingItem),
	}
	// Tuning parameters.

	// Stage 1 is single threaded.

	// Stage 2; Disk I/O bound.
	// TODO(maruel): Reduce to 1 or 2.
	const maxConcurrentHash = 5

	// Stage 3; Control server overload due to parallelism (DDoS).
	const maxConcurrentContains = 64
	const containsBatchingDelay = 100 * time.Millisecond
	const containsBatchSize = 50

	// Stage 4; Network I/O bound.
	const maxConcurrentUpload = 8

	stage2 := make(chan *PendingItem)
	stage3 := make(chan *PendingItem)
	stage4 := make(chan *PendingItem)
	a.wg.Add(1)
	go func() {
		defer func() {
			a.wg.Done()
			close(stage2)
		}()
		a.stage1DedupeLoop(a.stage1, stage2)
	}()

	// TODO(todd): Create on-disk cache in a new stage inserted between stages 1
	// and 2. This should be a separate interface with its own implementation
	// that 'archiver' keeps a reference to as member.

	a.wg.Add(1)
	go func() {
		defer func() {
			a.wg.Done()
			close(stage3)
		}()
		a.stage2HashLoop(stage2, stage3, maxConcurrentHash)
	}()

	a.wg.Add(1)
	go func() {
		defer func() {
			a.wg.Done()
			close(stage4)
		}()
		a.stage3LookupLoop(stage3, stage4, maxConcurrentContains, containsBatchingDelay, containsBatchSize)
	}()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.stage4UploadLoop(stage4, maxConcurrentUpload)
	}()

	// Push an nil item to enforce stage1DedupeLoop() woke up. Otherwise this
	// could lead to a race condition if Close() is called too quickly.
	a.stage1 <- nil
	return a
}

// PendingItem is an item being processed.
//
// It is caried over from pipeline stage to stage to do processing on it.
type PendingItem struct {
	// Immutable.
	DisplayName string         // Name to use to qualify this item
	wgHashed    sync.WaitGroup // Released once .digestItem.Digest is set
	priority    int64          // Lower values - earlier hashing and uploading.
	path        string         // Set when the source is a file on disk.
	a           *Archiver

	// Mutable.
	lock       sync.Mutex
	err        error                                    // PendingItem specific error
	digestItem isolateservice.HandlersEndpointsV1Digest // Mutated by hashLoop(), used by doContains()
	linked     []*PendingItem                           // Deduplicated item.

	// Mutable but not accessible externally.
	source isolatedclient.Source     // Source of data
	state  *isolatedclient.PushState // Server-side push state for cache miss
}

func newItem(a *Archiver, displayName, path string, source isolatedclient.Source, priority int64) *PendingItem {
	i := &PendingItem{a: a, DisplayName: displayName, path: path, source: source, priority: priority}
	i.wgHashed.Add(1)
	return i
}

// done is called when the PendingItem processing is done.
//
// It can be because there was an error, it was linked to another item, it was
// present on the server or finally it was uploaded successfully.
func (i *PendingItem) done() error {
	i.a = nil
	return nil
}

// WaitForHashed hangs until the item hash is known.
func (i *PendingItem) WaitForHashed() {
	i.wgHashed.Wait()
}

// Error returns any error that occurred for this item if any.
func (i *PendingItem) Error() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	return i.err
}

// Digest returns the calculated digest once calculated, empty otherwise.
func (i *PendingItem) Digest() isolated.HexDigest {
	i.lock.Lock()
	defer i.lock.Unlock()
	return isolated.HexDigest(i.digestItem.Digest)
}

func (i *PendingItem) isFile() bool {
	return len(i.path) != 0
}

// SetErr forcibly set an item as failed. Normally not used by callers.
func (i *PendingItem) SetErr(err error) {
	if err == nil {
		panic("internal error")
	}
	if err != context.Canceled && err != context.DeadlineExceeded && i.a != nil {
		i.a.setErr(err)
	}
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.err == nil {
		i.err = err
		for _, child := range i.linked {
			child.lock.Lock()
			child.err = err
			child.lock.Unlock()
		}
	}
}

func (i *PendingItem) calcDigest() error {
	defer i.wgHashed.Done()
	var d isolateservice.HandlersEndpointsV1Digest

	src, err := i.source()
	if err != nil {
		return fmt.Errorf("source(%s) failed: %s", i.DisplayName, err)
	}
	defer src.Close()

	h := i.a.c.Hash().New()
	size, err := io.Copy(h, src)
	if err != nil {
		i.SetErr(err)
		return fmt.Errorf("read(%s) failed: %s", i.DisplayName, err)
	}
	d = isolateservice.HandlersEndpointsV1Digest{Digest: string(isolated.Sum(h)), IsIsolated: true, Size: size}

	i.lock.Lock()
	defer i.lock.Unlock()
	i.digestItem = d

	for _, child := range i.linked {
		child.lock.Lock()
		child.digestItem = d
		child.lock.Unlock()
		child.wgHashed.Done()
	}
	return nil
}

func (i *PendingItem) link(child *PendingItem) {
	child.lock.Lock()
	defer child.lock.Unlock()
	if !child.isFile() || child.state != nil || child.linked != nil || child.err != nil {
		panic("internal error")
	}
	i.lock.Lock()
	defer i.lock.Unlock()
	i.linked = append(i.linked, child)
	if i.digestItem.Digest != "" {
		child.digestItem = i.digestItem
		child.err = i.err
		child.wgHashed.Done()
	} else if i.err != nil {
		child.err = i.err
		child.wgHashed.Done()
	}
}

// Archiver is an high level interface to an isolatedclient.Client.
//
// Uses a 4 stages pipeline, each doing work concurrently:
//   - Deduplicating similar requests or known server hot cache hits.
//   - Hashing files.
//   - Batched cache hit lookups on the server.
//   - Uploading cache misses.
type Archiver struct {
	// Immutable.
	ctx      context.Context
	cancel   func()
	c        *isolatedclient.Client
	stage1   chan *PendingItem
	wg       sync.WaitGroup
	progress progress.Progress

	// Mutable.
	mu  sync.Mutex
	err error

	// stats uses a separate mutex for performance.
	statsMu sync.Mutex
	stats   Stats
}

// Close waits for all pending files to be done. If an error occurred during
// processing, it is returned.
func (a *Archiver) Close() error {
	// This is done so asynchronously calling push() won't crash.
	a.mu.Lock()
	ok := false
	if a.stage1 != nil {
		close(a.stage1)
		a.stage1 = nil
		ok = true
	}
	a.mu.Unlock()

	if !ok {
		return errors.New("was already closed")
	}
	a.wg.Wait()
	_ = a.progress.Close()
	err := a.ctx.Err()
	// Documentation says that not calling cancel() could cause a leak, so call
	// it, but not before taking the context's error, if any.
	a.cancel()
	a.mu.Lock()
	if a.err != nil {
		err = a.err
	}
	a.mu.Unlock()
	return err
}

// Hash returns the hashing algorithm used by this archiver.
func (a *Archiver) Hash() crypto.Hash {
	return a.c.Hash()
}

// Push schedules item upload to the isolate server. Smaller priority value
// means earlier processing.
func (a *Archiver) Push(displayName string, source isolatedclient.Source, priority int64) *PendingItem {
	return a.push(newItem(a, displayName, "", source, priority))
}

// PushFile schedules file upload to the isolate server. Smaller priority
// value means earlier processing.
func (a *Archiver) PushFile(displayName, path string, priority int64) *PendingItem {
	source := func() (io.ReadCloser, error) {
		return os.Open(path)
	}
	return a.push(newItem(a, displayName, path, source, priority))
}

// Stats returns a copy of the statistics.
func (a *Archiver) Stats() *Stats {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	return a.stats.deepCopy()
}

func (a *Archiver) setErr(err error) {
	a.mu.Lock()
	if a.err == nil {
		a.err = err
	}
	a.mu.Unlock()
}

func (a *Archiver) push(item *PendingItem) *PendingItem {
	if a.pushLocked(item) {
		a.progress.Update(groupFound, groupFoundFound, 1)
		return item
	}
	item.done()
	return nil
}

func (a *Archiver) pushLocked(item *PendingItem) bool {
	// The close(a.stage1) call is always occurring with the lock held.
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stage1 == nil {
		// Archiver was closed.
		return false
	}
	// stage1DedupeLoop must never block and must be as fast as it can because it
	// is done while holding a.mu.
	a.stage1 <- item
	return true
}

// stage1DedupeLoop merges duplicate entries.
//
// This can happen when a lot of isolated files are archives, referencing to
// common entries.
func (a *Archiver) stage1DedupeLoop(in <-chan *PendingItem, out chan<- *PendingItem) {
	seen := map[string]*PendingItem{}
	// Create our own goroutine-local channel buffer, which doesn't need to be
	// synchronized (unlike channels).
	buildUp := []*PendingItem{}
	for {
		// Pull or push an item, dependending if there is build up.
		var item *PendingItem
		ok := true
		if len(buildUp) == 0 {
			item, ok = <-in
		} else {
			select {
			case item, ok = <-in:
			case out <- buildUp[0]:
				// Pop first item from buildUp.
				buildUp = buildUp[1:]
				a.progress.Update(groupHash, groupHashTodo, 1)
			}
		}
		if !ok {
			break
		}
		if item == nil {
			continue
		}

		// This loop must never block and must be as fast as it can as it is
		// functionally equivalent to running with a.mu held.
		if err := a.ctx.Err(); err != nil {
			item.SetErr(err)
			item.done()
			item.wgHashed.Done()
			continue
		}
		// TODO(maruel): Resolve symlinks for further deduplication? Depends on the
		// use case, not sure the trade off is worth.
		if item.isFile() {
			if previous, ok := seen[item.path]; ok {
				previous.link(item)
				// TODO(maruel): Semantically weird.
				item.done()
				continue
			}
		}

		buildUp = append(buildUp, item)
		seen[item.path] = item
	}

	// Take care of the build up after the channel closed.
	for _, item := range buildUp {
		if err := a.ctx.Err(); err != nil {
			item.SetErr(err)
			item.done()
			item.wgHashed.Done()
		} else {
			// The Archiver is being closed, this has to happen synchronously.
			out <- item
			a.progress.Update(groupHash, groupHashTodo, 1)
		}
	}
}

// stage2HashLoop hashes individual files.
func (a *Archiver) stage2HashLoop(in <-chan *PendingItem, out chan<- *PendingItem, parallelism int) {
	pool := common.NewGoroutinePriorityPool(a.ctx, parallelism)
	defer func() {
		_ = pool.Wait()
	}()
	for file := range in {
		// This loop will implicitly buffer when stage1 is too fast by creating a
		// lot of hung goroutines in pool. This permits reducing the contention on
		// a.mu.
		// TODO(tandrii): Implement backpressure in GoroutinePool, e.g. when it
		// exceeds 20k or something similar.
		item := file
		pool.Schedule(item.priority, func() {
			// calcDigest calls SetErr() and update wgHashed even on failure.
			if err := item.calcDigest(); err != nil {
				// When an error occurs, early exit the process.
				a.cancel()
				item.SetErr(err)
				item.done()
				return
			}
			a.progress.Update(groupHash, groupHashDone, 1)
			a.progress.Update(groupHash, groupHashDoneSize, item.digestItem.Size)
			a.progress.Update(groupLookup, groupLookupTodo, 1)
			out <- item
		}, func() {
			item.SetErr(a.ctx.Err())
			item.done()
			item.wgHashed.Done()
		})
	}
}

// stage3LookupLoop queries the server to determine which files must be
// uploaded.
//
// It is the first stage where the client contacts the server.
func (a *Archiver) stage3LookupLoop(in <-chan *PendingItem, out chan<- *PendingItem, parallelism int, batchingDelay time.Duration, batchSize int) {
	pool := common.NewGoroutinePool(a.ctx, parallelism)
	defer func() {
		_ = pool.Wait()
	}()
	items := []*PendingItem{}
	never := make(<-chan time.Time)
	timer := never
	loop := true
	for loop {
		select {
		case <-timer:
			batch := items
			pool.Schedule(func() { a.doContains(out, batch) }, nil)
			items = []*PendingItem{}
			timer = never

		case item, ok := <-in:
			if !ok {
				loop = false
				break
			}
			items = append(items, item)
			if len(items) == batchSize {
				batch := items
				pool.Schedule(func() { a.doContains(out, batch) }, nil)
				items = []*PendingItem{}
				timer = never
			} else if timer == never {
				timer = time.After(batchingDelay)
			}
		}
	}

	if len(items) != 0 {
		batch := items
		pool.Schedule(func() { a.doContains(out, batch) }, nil)
	}
}

// stage4UploadLoop uploads the cache misses.
func (a *Archiver) stage4UploadLoop(in <-chan *PendingItem, parallelism int) {
	pool := common.NewGoroutinePriorityPool(a.ctx, parallelism)
	defer func() {
		_ = pool.Wait()
	}()
	for state := range in {
		item := state
		pool.Schedule(item.priority, func() { a.doUpload(item) }, nil)
	}
}

// doContains is called by stage 3.
func (a *Archiver) doContains(out chan<- *PendingItem, items []*PendingItem) {
	tmp := make([]*isolateservice.HandlersEndpointsV1Digest, len(items))
	// No need to lock each item at that point, no mutation occurs on
	// PendingItem.digestItem after stage 2.
	for i, item := range items {
		tmp[i] = &item.digestItem
	}
	states, err := a.c.Contains(a.ctx, tmp)
	if err != nil {
		err = fmt.Errorf("contains(%d) failed: %s", len(items), err)
		// When an error occurs, early exit the process.
		a.cancel()
		for _, item := range items {
			item.SetErr(err)
			item.done()
		}
		return
	}
	a.progress.Update(groupLookup, groupLookupDone, int64(len(items)))
	for index, state := range states {
		size := items[index].digestItem.Size
		if state == nil {
			a.statsMu.Lock()
			a.stats.Hits = append(a.stats.Hits, units.Size(size))
			a.statsMu.Unlock()
			items[index].done()
		} else {
			items[index].state = state
			a.progress.Update(groupUpload, groupUploadTodo, 1)
			a.progress.Update(groupUpload, groupUploadTodoSize, items[index].digestItem.Size)
			out <- items[index]
		}
	}
	logging.Infof(a.ctx, "Looked up %d items", len(items))
}

// doUpload is called by stage 4.
func (a *Archiver) doUpload(item *PendingItem) {
	start := time.Now()
	if err := a.c.Push(a.ctx, item.state, item.source); err != nil {
		err = fmt.Errorf("push(%s) failed: %s", item.path, err)
		// When an error occurs, early exit the process.
		a.cancel()
		item.SetErr(err)
		item.done()
		return
	}
	a.progress.Update(groupUpload, groupUploadDone, 1)
	a.progress.Update(groupUpload, groupUploadDoneSize, item.digestItem.Size)
	item.done()
	size := units.Size(item.digestItem.Size)
	u := &UploadStat{time.Since(start), size, item.DisplayName}
	a.statsMu.Lock()
	a.stats.Pushed = append(a.stats.Pushed, u)
	a.statsMu.Unlock()
	logging.Infof(a.ctx, "Uploaded %7s: %s", size, item.DisplayName)
}
