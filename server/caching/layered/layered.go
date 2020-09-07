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

// Package layered provides a two-layer cache for serializable objects.
package layered

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/data/caching/lru"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/server/caching"
)

// ErrCantSatisfyMinTTL is returned by GetOrCreate if the factory function
// produces an item that expires sooner than the requested MinTTL.
var ErrCantSatisfyMinTTL = errors.New("new item produced by the factory has insufficient TTL")

// Cache implements a cache of serializable objects on top of process and
// global caches.
//
// If the global cache is not available or fails, degrades to using only process
// cache.
//
// Since global cache errors are ignored, gives no guarantees of consistency or
// item uniqueness. Thus supposed to be used only when caching results of
// computations without side effects.
type Cache struct {
	// ProcessLRUCache is a handle to a process LRU cache that holds the data.
	ProcessLRUCache caching.LRUHandle
	// GlobalNamespace is a global cache namespace to use for the data.
	GlobalNamespace string
	// Marshal converts an item being cached to a byte blob.
	Marshal func(item interface{}) ([]byte, error)
	// Unmarshal takes output of Marshal and converts it to an item to return.
	Unmarshal func(blob []byte) (interface{}, error)
}

// Option is a base interface of options for GetOrCreate call.
type Option interface {
	apply(opts *options)
}

// WithMinTTL specifies minimal acceptable TTL (Time To Live) of the returned
// cached item.
//
// If the currently cached item expires sooner than the requested TTL, it will
// be forcefully refreshed. If the new (refreshed) item also expires sooner
// than the requested min TTL, GetOrCreate will return ErrCantSatisfyMinTTL.
func WithMinTTL(ttl time.Duration) Option {
	if ttl <= 0 {
		panic("ttl must be positive")
	}
	return minTTLOpt(ttl)
}

// WithRandomizedExpiration enables randomized early expiration.
//
// This is only useful if cached items are used highly concurrently from many
// goroutines.
//
// On each cache access if the remaining TTL of the cached item is less than
// 'threshold', it may randomly be considered already expired (with probability
// increasing when item nears its true expiration).
//
// This is useful to avoid a situation when many concurrent consumers discover
// at the same time that the item has expired, and then all proceed waiting
// for a refresh. With randomized early expiration only the most unlucky
// consumer will trigger the refresh and will be blocked on it.
func WithRandomizedExpiration(threshold time.Duration) Option {
	if threshold < 0 {
		panic("threshold must be positive")
	}
	return expRandThresholdOpt(threshold)
}

// GetOrCreate attempts to grab an item from process or global cache, or create
// it if it's not cached yet.
//
// Fetching an item from the global cache or instantiating a new item happens
// under a per-key lock.
//
// Expiration time is used with seconds precision. Zero expiration time means
// the item doesn't expire on its own.
func (c *Cache) GetOrCreate(ctx context.Context, key string, fn lru.Maker, opts ...Option) (interface{}, error) {
	if c.GlobalNamespace == "" {
		panic("empty namespace is forbidden, please specify GlobalNamespace")
	}

	o := options{}
	for _, opt := range opts {
		opt.apply(&o)
	}

	now := clock.Now(ctx)
	lru := c.ProcessLRUCache.LRU(ctx)

	// Check that the item is in the local cache, its TTL is acceptable and we
	// don't want to randomly prematurely expire it, see WithRandomizedExpiration.
	var ignored *itemWithExp
	if v, ok := lru.Get(ctx, key); ok {
		item := v.(*itemWithExp)
		if item.isAcceptableTTL(now, o.minTTL) && !item.randomlyExpired(ctx, now, o.expRandThreshold) {
			return item.val, nil
		}
		ignored = item
	}

	// Either the item is not in the local cache, or the cached copy expires too
	// soon or we randomly decided that we want to prematurely refresh it. Attempt
	// to fetch from the global cache or create a new one. Disable expiration
	// randomization at this point, it has served its purpose already, since only
	// unlucky callers will reach this code path.
	v, err := lru.Create(ctx, key, func() (interface{}, time.Duration, error) {
		// Now that we have the lock, recheck that the item still needs a refresh.
		// Purposely ignore an item we decided we want to prematurely expire.
		if v, ok := lru.Get(ctx, key); ok {
			if item := v.(*itemWithExp); item != ignored && item.isAcceptableTTL(now, o.minTTL) {
				return item, item.expiration(now), nil
			}
		}

		// Attempt to grab it from the global cache, verifying TTL is acceptable.
		if item := c.maybeFetchItem(ctx, key); item != nil && item.isAcceptableTTL(now, o.minTTL) {
			return item, item.expiration(now), nil
		}

		// Either a cache miss, problems with the cached item or its TTL is not
		// acceptable. Need a to make a new item.
		var item itemWithExp
		val, exp, err := fn()
		item.val = val
		switch {
		case err != nil:
			return nil, 0, err
		case exp < 0:
			panic("the expiration time must be non-negative")
		case exp > 0: // note: if exp == 0 we want item.exp to be zero
			item.exp = now.Add(exp)
			if !item.isAcceptableTTL(now, o.minTTL) {
				// If 'fn' is incapable of generating an item with sufficient TTL there's
				// nothing else we can do.
				return nil, 0, ErrCantSatisfyMinTTL
			}
		}

		// Store the new item in the global cache. We may accidentally override
		// an item here if someone else refreshed it already. But this is
		// unavoidable given GlobalCache semantics and generally rare and harmless
		// (given Cache guarantees or rather lack of there of).
		if err := c.maybeStoreItem(ctx, key, &item, now); err != nil {
			return nil, 0, err
		}
		return &item, item.expiration(now), nil
	})

	if err != nil {
		return nil, err
	}
	return v.(*itemWithExp).val, nil
}

////////////////////////////////////////////////////////////////////////////////

// formatVersionByte indicates what serialization format is used, it is stored
// as a first byte of the serialized data.
//
// Serialized items with different value of the first byte are rejected.
const formatVersionByte = 1

// options is collection of options for GetOrCreate.
type options struct {
	minTTL           time.Duration
	expRandThreshold time.Duration
}

type minTTLOpt time.Duration
type expRandThresholdOpt time.Duration

func (o minTTLOpt) apply(opts *options)           { opts.minTTL = time.Duration(o) }
func (o expRandThresholdOpt) apply(opts *options) { opts.expRandThreshold = time.Duration(o) }

// itemWithExp is what is actually stored (pointer to it) in the process cache.
//
// It is a user-generated value plus its expiration time (or zero time if it
// doesn't expire).
type itemWithExp struct {
	val interface{}
	exp time.Time
}

// isAcceptableTTL returns true if item's TTL is large enough.
func (i *itemWithExp) isAcceptableTTL(now time.Time, minTTL time.Duration) bool {
	if i.exp.IsZero() {
		return true // never expires
	}
	// Note: '>=' must not be used here, since minTTL may be 0, and we don't want
	// to return true on zero expiration.
	return i.exp.Sub(now) > minTTL
}

// randomlyExpired returns true if the item must be considered already expired.
//
// See WithRandomizedExpiration for the rationale. The context is used only to
// grab RNG.
func (i *itemWithExp) randomlyExpired(ctx context.Context, now time.Time, threshold time.Duration) bool {
	if i.exp.IsZero() {
		return false // never expires
	}

	ttl := i.exp.Sub(now)
	if ttl > threshold {
		return false // far from expiration, no need to enable randomization
	}

	// TODO(vadimsh): The choice of distribution here was made arbitrary. Some
	// literature suggests to use exponential distribution instead, but it's not
	// clear how to pick parameters for it. In practice what we do here seems good
	// enough. On each check we randomly expire the item with probability
	// p = (threshold - ttl) / threshold. Closer the item to its true expiration
	// (ttl is smaller), higher the probability.
	rnd := time.Duration(mathrand.Int63n(ctx, int64(threshold)))
	return rnd > ttl
}

// expiration returns expiration time to use when storing this item.
//
// Zero return value means "does not expire" (as understood by both LRU and
// Global caches). Panics if the calculated expiration is negative. Use
// isAcceptableTTL to detect this case beforehand.
func (i *itemWithExp) expiration(now time.Time) time.Duration {
	if i.exp.IsZero() {
		return 0 // never expires
	}
	d := i.exp.Sub(now)
	if d <= 0 {
		panic("item is already expired, isAcceptableTTL should have detected this")
	}
	return d
}

// maybeFetchItem attempts to fetch the item from the global cache.
//
// If the global cache is not available or the cached item there is broken
// returns nil. Logs errors inside.
func (c *Cache) maybeFetchItem(ctx context.Context, key string) *itemWithExp {
	g := caching.GlobalCache(ctx, c.GlobalNamespace)
	if g == nil {
		return nil
	}

	blob, err := g.Get(ctx, key)
	if err != nil {
		if err != caching.ErrCacheMiss {
			logging.WithError(err).Errorf(ctx, "Failed to read item %q from the global cache", key)
		}
		return nil
	}

	item, err := c.deserializeItem(blob)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to deserialize item %q", key)
		return nil
	}
	return item
}

// maybeStoreItem puts the item in the global cache, if possible.
//
// It returns an error only if the serialization fails. It generally means the
// serialization code is buggy and should be adjusted.
//
// Global cache errors are logged and ignored.
func (c *Cache) maybeStoreItem(ctx context.Context, key string, item *itemWithExp, now time.Time) error {
	g := caching.GlobalCache(ctx, c.GlobalNamespace)
	if g == nil {
		return nil
	}

	blob, err := c.serializeItem(item)
	if err != nil {
		return err
	}

	if err = g.Set(ctx, key, blob, item.expiration(now)); err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to store item %q in the global cache", key)
	}
	return nil
}

// serializeItem packs item and its expiration time into a byte blob.
func (c *Cache) serializeItem(item *itemWithExp) ([]byte, error) {
	blob, err := c.Marshal(item.val)
	if err != nil {
		return nil, err
	}

	var deadline uint64
	if !item.exp.IsZero() {
		deadline = uint64(item.exp.Unix())
	}

	// <version_byte> + <uint64 deadline timestamp> + <blob>
	output := make([]byte, 9+len(blob))
	output[0] = formatVersionByte
	binary.LittleEndian.PutUint64(output[1:], deadline)
	copy(output[9:], blob)
	return output, nil
}

// deserializeItem is reverse of serializeItem.
func (c *Cache) deserializeItem(blob []byte) (item *itemWithExp, err error) {
	if len(blob) < 9 {
		err = fmt.Errorf("the received buffer is too small")
		return
	}
	if blob[0] != formatVersionByte {
		err = fmt.Errorf("bad format version, expecting %d, got %d", formatVersionByte, blob[0])
		return
	}
	item = &itemWithExp{}
	deadline := binary.LittleEndian.Uint64(blob[1:])
	if deadline != 0 {
		item.exp = time.Unix(int64(deadline), 0)
	}
	item.val, err = c.Unmarshal(blob[9:])
	return
}
