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

package cache

import (
	"crypto"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/tetrafolium/luci-go/common/data/text/units"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/isolated"
	"github.com/tetrafolium/luci-go/common/system/filesystem"
)

// Cache is a cache of objects holding content in disk.
//
// All implementations must be thread-safe.
type Cache struct {
	// Immutable.
	policies Policies
	path     string
	h        crypto.Hash

	// Lock protected.
	mu  sync.Mutex // This protects modification of cached entries under |path| too.
	lru lruDict    // Implements LRU based eviction.

	statsMu sync.Mutex // Protects the stats below
	// TODO(tikuta): Add stats about: # removed.
	// TODO(tikuta): stateFile
	added []int64
	used  []int64
}

// Policies is the policies to use on a cache to limit it's footprint.
//
// It's a cache, not a leak.
type Policies struct {
	// MaxSize trims if the cache gets larger than this value. If 0, the cache is
	// effectively a leak.
	MaxSize units.Size
	// MaxItems is the maximum number of items to keep in the cache. If 0, do not
	// enforce a limit.
	MaxItems int
	// MinFreeSpace trims if disk free space becomes lower than this value.
	// Only makes sense when using disk based cache.
	MinFreeSpace units.Size
}

// AddFlags adds flags for cache policy parameters.
func (p *Policies) AddFlags(f *flag.FlagSet) {
	p.MaxSize = cacheMaxSizeDefault
	f.Var(&p.MaxSize, "cache-max-size", "Cache is trimmed if the cache gets larger than this value.")
	f.IntVar(&p.MaxItems, "cache-max-items", cacheMaxItemsDefault, "Maximum number of items to keep in the cache.")
	f.Var(&p.MinFreeSpace, "cache-min-free-space", "Cache is trimmed if disk free space becomes lower than this value.")
}

// IsDefault returns whether some flags are set or not.
func (p *Policies) IsDefault() bool {
	return p.MaxSize == cacheMaxSizeDefault && p.MaxItems == cacheMaxItemsDefault && p.MinFreeSpace == 0
}

var ErrInvalidHash = errors.New("invalid hash")

// New creates a disk based cache.
//
// It may return both a valid Cache and an error if it failed to load the
// previous cache metadata. It is safe to ignore this error. This creates
// cache directory if it doesn't exist.
func New(policies Policies, path string, h crypto.Hash) (*Cache, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, errors.Annotate(err, "failed to call Abs(%s)", path).Err()
	}
	err = os.MkdirAll(path, 0700)
	if err != nil {
		return nil, errors.Annotate(err, "failed to call MkdirAll(%s)", path).Err()
	}

	d := &Cache{
		policies: policies,
		path:     path,
		h:        h,
		lru:      makeLRUDict(h),
	}
	p := d.statePath()

	err = func() error {
		f, err := os.Open(p)
		if err != nil && os.IsNotExist(err) {
			// The fact that the cache is new is not an error.
			return nil
		}
		if err != nil {
			return err
		}
		defer f.Close()
		return json.NewDecoder(f).Decode(&d.lru)
	}()

	if err != nil {
		// Do not use os.RemoveAll, due to strange 'Access Denied' error on windows
		// in os.MkDir after os.RemoveAll.
		// crbug.com/932396#c123
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, errors.Annotate(err, "failed to call ioutil.ReadDir(%s)", path).Err()
		}

		for _, file := range files {
			p := filepath.Join(path, file.Name())
			if err := os.RemoveAll(p); err != nil {
				return nil, errors.Annotate(err, "failed to call os.RemoveAll(%s)", p).Err()
			}
		}

		d.lru = makeLRUDict(h)
	}
	return d, err
}

func (d *Cache) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.lru.IsDirty() {
		return nil
	}
	f, err := os.Create(d.statePath())
	if err == nil {
		defer f.Close()
		err = json.NewEncoder(f).Encode(&d.lru)
	}
	return err
}

// Keys returns the list of all cached digests in LRU order.
func (d *Cache) Keys() isolated.HexDigests {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lru.keys()
}

// Touch updates the LRU position of an item to ensure it is kept in the
// cache.
//
// Returns true if item is in cache.
func (d *Cache) Touch(digest isolated.HexDigest) bool {
	if !digest.Validate(d.h) {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lru.touch(digest)
}

// Evict removes item from cache if it's there.
func (d *Cache) Evict(digest isolated.HexDigest) {
	if !digest.Validate(d.h) {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lru.pop(digest)
	_ = os.Remove(d.itemPath(digest))
}

// Read returns contents of the cached item.
func (d *Cache) Read(digest isolated.HexDigest) (io.ReadCloser, error) {
	if !digest.Validate(d.h) {
		return nil, os.ErrInvalid
	}

	d.mu.Lock()
	f, err := os.Open(d.itemPath(digest))
	if err != nil {
		d.mu.Unlock()
		return nil, err
	}
	d.lru.touch(digest)
	d.mu.Unlock()

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, errors.Annotate(err, "failed to get stat for %s", digest).Err()
	}

	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	d.used = append(d.used, fi.Size())
	return f, nil
}

// Add reads data from src and stores it in cache.
func (d *Cache) Add(digest isolated.HexDigest, src io.Reader) error {
	return d.add(digest, src, nil)
}

// AddFileWithoutValidation adds src as cache entry with hardlink.
// But this doesn't do any content validation.
//
// TODO(tikuta): make one function and control the behavior by option?
func (d *Cache) AddFileWithoutValidation(digest isolated.HexDigest, src string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return errors.Annotate(err, "failed to get stat").Err()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := os.Link(src, d.itemPath(digest)); err != nil && !os.IsExist(err) {
		return errors.Annotate(err, "failed to link %s to %s", src, digest).Err()
	}

	d.lru.pushFront(digest, units.Size(fi.Size()))
	if err := d.respectPolicies(); err != nil {
		d.lru.pop(digest)
		return err
	}

	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	d.added = append(d.added, fi.Size())
	return nil
}

// AddWithHardlink reads data from src and stores it in cache and hardlink file.
// This is to avoid file removal by shrink in Add().
func (d *Cache) AddWithHardlink(digest isolated.HexDigest, src io.Reader, dest string, perm os.FileMode) error {
	return d.add(digest, src, func() error {
		if err := d.hardlinkUnlocked(digest, dest, perm); err != nil {
			_ = os.Remove(d.itemPath(digest))
			return errors.Annotate(err, "failed to call Hardlink(%s, %s)", digest, dest).Err()
		}
		return nil
	})
}

// Hardlink ensures file at |dest| has the same content as cached |digest|.
//
// Note that the behavior when dest already exists is undefined. It will work
// on all POSIX and may or may not fail on Windows depending on the
// implementation used. Do not rely on this behavior.
func (d *Cache) Hardlink(digest isolated.HexDigest, dest string, perm os.FileMode) error {
	if runtime.GOOS == "darwin" {
		// Accessing the path, which is being replaced, with os.Link
		// seems to cause flaky 'operation not permitted' failure on
		// macOS (https://crbug.com/1076468). So prevent that by holding
		// lock here.
		d.mu.Lock()
		defer d.mu.Unlock()
	}
	return d.hardlinkUnlocked(digest, dest, perm)
}

// GetAdded returns a list of file size added to cache.
func (d *Cache) GetAdded() []int64 {
	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	return append([]int64{}, d.added...)
}

// GetUsed returns a list of file size used from cache.
func (d *Cache) GetUsed() []int64 {
	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	return append([]int64{}, d.used...)
}

// Private details.

const maxUint = ^uint(0)
const maxInt = int(maxUint >> 1)

const cacheMaxSizeDefault = math.MaxInt64
const cacheMaxItemsDefault = maxInt

func (d *Cache) add(digest isolated.HexDigest, src io.Reader, cb func() error) error {
	if !digest.Validate(d.h) {
		return os.ErrInvalid
	}
	tmp, err := ioutil.TempFile(d.path, string(digest)+".*.tmp")
	if err != nil {
		return errors.Annotate(err, "failed to create tempfile for %s", digest).Err()
	}
	// TODO(maruel): Use a LimitedReader flavor that fails when reaching limit.
	h := d.h.New()
	size, err := io.Copy(tmp, io.TeeReader(src, h))
	if err2 := tmp.Close(); err == nil {
		err = err2
	}
	fname := tmp.Name()
	if err != nil {
		_ = os.Remove(fname)
		return err
	}
	if d := isolated.Sum(h); d != digest {
		_ = os.Remove(fname)
		return errors.Annotate(ErrInvalidHash, "invalid hash, got=%s, want=%s", d, digest).Err()
	}
	if units.Size(size) > d.policies.MaxSize {
		_ = os.Remove(fname)
		return errors.Reason("item too large, size=%d, limit=%d", size, d.policies.MaxSize).Err()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := os.Rename(fname, d.itemPath(digest)); err != nil {
		_ = os.Remove(fname)
		return errors.Annotate(err, "failed to rename %s -> %s", fname, d.itemPath(digest)).Err()
	}

	if cb != nil {
		if err := cb(); err != nil {
			return err
		}
	}

	d.lru.pushFront(digest, units.Size(size))
	if err := d.respectPolicies(); err != nil {
		d.lru.pop(digest)
		return err
	}
	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	d.added = append(d.added, size)
	return nil
}

func (d *Cache) hardlinkUnlocked(digest isolated.HexDigest, dest string, perm os.FileMode) error {
	if !digest.Validate(d.h) {
		return os.ErrInvalid
	}
	src := d.itemPath(digest)
	// - Windows, if dest exists, the call fails. In particular, trying to
	//   os.Remove() will fail if the file's ReadOnly bit is set. What's worse is
	//   that the ReadOnly bit is set on the file inode, shared on all hardlinks
	//   to this inode. This means that in the case of a file with the ReadOnly
	//   bit set, it would have to do:
	//   - If dest exists:
	//    - If dest has ReadOnly bit:
	//      - If file has any other inode:
	//        - Remove the ReadOnly bit.
	//        - Remove dest.
	//        - Set the ReadOnly bit on one of the inode found.
	//   - Call os.Link()
	//  In short, nobody ain't got time for that.
	//
	// - On any other (sane) OS, if dest exists, it is silently overwritten.
	if err := os.Link(src, dest); err != nil {
		if _, serr := os.Stat(src); errors.Contains(serr, os.ErrNotExist) {
			// In Windows, os.Link may fail with access denied error even if |src| isn't there.
			// And this is to normalize returned error in such case.
			// https://crbug.com/1098265
			err = errors.Annotate(serr, "%s doesn't exist and os.Link failed: %v", src, err).Err()
		}
		debugInfo := fmt.Sprintf("Stats:\n*  src: %s\n*  dest: %s\n*  destDir: %s\nUID=%d GID=%d", statsStr(src), statsStr(dest), statsStr(filepath.Dir(dest)), os.Getuid(), os.Getgid())
		return errors.Annotate(err, "failed to call os.Link(%s, %s)\n%s", src, dest, debugInfo).Err()
	}

	if err := os.Chmod(dest, perm); err != nil {
		return errors.Annotate(err, "failed to call os.Chmod(%s, %#o)", dest, perm).Err()
	}

	fi, err := os.Stat(dest)
	if err != nil {
		return errors.Annotate(err, "failed to call os.Stat(%s)", dest).Err()
	}
	size := fi.Size()
	d.statsMu.Lock()
	defer d.statsMu.Unlock()
	// If this succeeds directly, it means the file is already cached on the
	// disk, so we put it into LRU.
	d.used = append(d.used, size)

	return nil
}

func (d *Cache) itemPath(digest isolated.HexDigest) string {
	return filepath.Join(d.path, string(digest))
}

func (d *Cache) statePath() string {
	return filepath.Join(d.path, "state.json")
}

func (d *Cache) respectPolicies() error {
	minFreeSpaceWanted := uint64(d.policies.MinFreeSpace)
	for {
		freeSpace, err := filesystem.GetFreeSpace(d.path)
		if err != nil {
			return errors.Annotate(err, "couldn't estimate the free space at %s", d.path).Err()
		}
		if d.lru.length() <= d.policies.MaxItems && d.lru.sum <= d.policies.MaxSize && freeSpace >= minFreeSpaceWanted {
			break
		}
		if d.lru.length() == 0 {
			return errors.Reason("no more space to free in %s: current free space=%d policies.MinFreeSpace=%d", d.path, freeSpace, minFreeSpaceWanted).Err()
		}
		k, _ := d.lru.popOldest()
		_ = os.Remove(d.itemPath(k))
	}
	return nil
}

func statsStr(path string) string {
	fi, err := os.Stat(path)
	return fmt.Sprintf("path=%s FileInfo=%+v err=%v", path, fi, err)
}
