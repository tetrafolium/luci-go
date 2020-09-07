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

package git

import (
	"context"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	gitpb "github.com/tetrafolium/luci-go/common/proto/git"
	gitilespb "github.com/tetrafolium/luci-go/common/proto/gitiles"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
	"github.com/tetrafolium/luci-go/gae/service/memcache"
	"github.com/tetrafolium/luci-go/milo/common"
	"github.com/tetrafolium/luci-go/server/auth"
)

// LogOptions are options for Log function.
type LogOptions struct {
	Limit     int
	WithFiles bool
}

// Log implements Client interface.
func (p *implementation) Log(c context.Context, host, project, commitish string, inputOptions *LogOptions) (commits []*gitpb.Commit, err error) {
	defer func() { err = common.TagGRPC(c, err) }()

	allowed, err := p.acls.IsAllowed(c, host, project)
	switch {
	case err != nil:
		return
	case !allowed:
		logging.Warningf(c, "%q not allowed for host %q project %q", auth.CurrentIdentity(c), host, project)
		err = status.Errorf(codes.NotFound, "not found")
		return
	}

	return p.log(c, host, project, commitish, "", inputOptions)
}

func (p *implementation) log(c context.Context, host, project, commitish, ancestor string, inputOptions *LogOptions) (commits []*gitpb.Commit, err error) {
	var opts LogOptions
	if inputOptions != nil {
		opts = *inputOptions
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	remaining := opts.Limit // defined as (limit - len(commits))
	// add appends page to commits and updates remaining.
	// If page starts with commits' last commit, page's first commit
	// is skipped.
	add := func(page []*gitpb.Commit) {
		switch {
		case len(commits) == 0:
			commits = page
		case len(page) == 0:
			// Do nothing.
		default:
			if page[0].Id != commits[len(commits)-1].Id {
				panic(fmt.Sprintf(
					"pages are not contiguous; want page start %q, got %q",
					commits[len(commits)-1].Id, page[0].Id))
			}
			commits = append(commits, page[1:]...)
		}
		remaining = opts.Limit - len(commits)
	}

	req := &logReq{
		factory:   p,
		host:      host,
		project:   project,
		commitish: commitish,
		ancestor:  ancestor,
		withFiles: opts.WithFiles,
		min:       100,
	}

	var page []*gitpb.Commit
	for remaining > 100 {
		// We need to fetch >100 commits, but one logReq can handle only 100.
		// Call it in a loop.
		page, err = req.call(c)
		switch {
		case err != nil:
			return
		case len(page) < 100:
			// This can happen iff there are no more commits.
			add(page)
			return
		case len(page) > 100:
			panic("impossible: logReq.call() returned >100 commits")
		default:
			// There may be more commits.
			// Continue from the last fetched commit.
			req.commitish = page[len(page)-1].Id
			add(page)
			if remaining == 0 {
				panic("impossible: remaining reached 0")
			}
		}
	}

	// last page. One logReq.call() can handle the rest.
	req.min = remaining
	page, err = req.call(c)
	if err != nil {
		return
	}
	add(page)

	// we may end up with more than we were asked for because
	// gitReq's parameter is minimum, not limit.
	if len(commits) > opts.Limit {
		commits = commits[:opts.Limit]
	}
	return
}

// possible values for "result" field of logCounter metric below.
const (
	cacheHit        = "hit"
	cacheMiss       = "miss"
	cacheFailure    = "failure-cache"
	decodingFailure = "failure-decoding"
)

var logCounter = metric.NewCounter(
	"luci/milo/git/log/cache",
	"The number of hits we get in git.Log",
	nil,
	field.String("result"), // for possible value see consts above.
	field.String("host"),
	field.String("project"),
	field.String("ref"))

var latencyMetric = metric.NewCumulativeDistribution(
	"luci/milo/git/log/latency",
	"Gitiles response latency",
	&types.MetricMetadata{Units: types.Milliseconds},
	distribution.DefaultBucketer,
	field.Bool("treeDiff"),
	field.String("host"),
	field.String("project"))

var gitHash = regexp.MustCompile("^[0-9a-fA-F]{40}$")

// logReq is the implementation of Log().
//
// Cached commits are keyed not only off of the requested commitish,
// but also ancestors, for example if received 100 ancestor
// commits C100, C99, .., C2, C1,
// cache with C99, C98, ... C1, too.
//
// The last byte of a cache value specifies how to interpret the rest:
//   0:       the rest is protobuf-marshalled gitilespb.LogResponse message
//   1-99:    the rest is byte id of a descendant commit that
//            the commit-in-cache-key is reachable from.
//            The last byte indicates the distance between the two commits.
//   100-255: reserved for future.
type logReq struct {
	factory   *implementation
	host      string
	project   string
	commitish string
	ancestor  string
	withFiles bool
	min       int // must be in [1..100]

	// fields below are set in call()

	commitishIsHash bool
	commitishEntry  memcache.Item
}

func (l *logReq) call(c context.Context) ([]*gitpb.Commit, error) {
	if l.min < 1 || l.min > 100 {
		panic(fmt.Sprintf("invalid min %d", l.min))
	}

	c = logging.SetFields(c, logging.Fields{
		"host":      l.host,
		"project":   l.project,
		"commitish": l.commitish,
	})

	l.commitishIsHash = gitHash.MatchString(l.commitish)
	l.commitishEntry = l.mkCache(c, l.commitish)
	if !l.commitishIsHash {
		// committish is not pinned, may move, so set a short expiration.
		l.commitishEntry.SetExpiration(30 * time.Second)
	}

	cacheResult := ""
	defer func() {
		ref := l.commitish
		if l.commitishIsHash {
			ref = "PINNED"
		}
		logCounter.Add(c, 1, cacheResult, l.host, l.project, ref)
	}()

	cacheResult, commits, ok := l.readCache(c)
	if ok {
		return commits, nil
	}

	// cache miss, cache failure or corrupted cache.
	// Call Gitiles.
	req := &gitilespb.LogRequest{
		Project:    l.project,
		Committish: l.commitish,
		PageSize:   100,
		TreeDiff:   l.withFiles,
	}
	if l.ancestor != "" {
		req.ExcludeAncestorsOf = l.ancestor
	}
	logging.Infof(c, "gitiles(%q).Log(%#v)", l.host, req)
	client, err := l.factory.gitilesClient(c, l.host)
	if err != nil {
		return nil, err
	}
	start := clock.Now(c)
	res, err := client.Log(c, req)
	latency := float64(clock.Now(c).Sub(start).Nanoseconds()) / 1000000.0
	logging.Debugf(c, "gitiles took %fms", latency)
	latencyMetric.Add(c, latency, l.withFiles, l.host, l.project)
	if err != nil {
		return nil, errors.Annotate(err, "gitiles.Log").Err()
	}

	l.writeCache(c, res)
	return res.Log, nil
}

func (l *logReq) readCache(c context.Context) (cacheResult string, commits []*gitpb.Commit, ok bool) {
	e := l.commitishEntry
	dist := byte(0)
	maxDist := byte(100 - l.min)
	for {
		switch err := memcache.Get(c, e); {
		case err == memcache.ErrCacheMiss:
			logging.Infof(c, "cache miss")
			cacheResult = cacheMiss

		case err != nil:
			logging.WithError(err).Errorf(c, "cache failure")
			cacheResult = cacheFailure

		case len(e.Value()) == 0:
			logging.WithError(err).Errorf(c, "empty cache value at key %q", e.Key())
			cacheResult = decodingFailure
		default:
			n := len(e.Value())
			// see logReq for cache value format.
			data := e.Value()[:n-1]
			meta := e.Value()[n-1]
			switch {
			case meta == 0:
				var decoded gitilespb.LogResponse
				if err := proto.Unmarshal(data, &decoded); err != nil {
					logging.WithError(err).Errorf(c, "could not decode cached commits at key %q", e.Key())
					cacheResult = decodingFailure
				} else {
					cacheResult = cacheHit
					commits = decoded.Log[dist:]
					ok = true
				}

			case meta >= 100:
				logging.WithError(err).Errorf(c, "unexpected last byte %d in cache value at key %q", meta, e.Key())
				cacheResult = decodingFailure

			case dist+meta <= maxDist:
				// meta is distance, and the referenced cache is sufficient for reuse
				dist += meta
				descendant := hex.EncodeToString(data)
				logging.Debugf(c, "recursing into cache %s with distance %d", descendant, meta)
				e = l.mkCache(c, descendant)
				// cacheResult is not set => continue the loop.
			default:
				// TODO(nodir): if we need commits C200..C100, but we have
				// C200->C250 here (C250 entry has C250..C150), instead of
				// discarding cache, reuse C200..C150 from C250 and fetch
				// C150..C50.
				logging.Debugf(c, "distance at key %q is too large", e.Key())
				cacheResult = cacheMiss
			}
		}

		if cacheResult != "" {
			return
		}
	}
}

func (l *logReq) writeCache(c context.Context, res *gitilespb.LogResponse) {
	marshalled, err := proto.Marshal(res)
	if err != nil {
		logging.WithError(err).Errorf(c, "failed to marshal gitiles log %s", res)
		return
	}

	// see logReq comment for cache value format.
	l.commitishEntry.SetValue(append(marshalled, 0))

	// Cache entries to set.
	caches := make([]memcache.Item, 1, len(res.Log)+1)
	caches[0] = l.commitishEntry
	if !l.commitishIsHash && len(res.Log) > 0 {
		// cache with commit hash cache key too.
		e := l.mkCache(c, res.Log[0].Id)
		e.SetValue(l.commitishEntry.Value())
		caches = append(caches, e)
	}

	if len(res.Log) > 1 {
		// Also potentially cache with ancestors as cache keys.
		ancestorCaches := make([]memcache.Item, 0, len(res.Log)-1)
		for i := 1; i < len(res.Log) && len(res.Log[i-1].Parents) == 1; i++ {
			ancestorCaches = append(ancestorCaches, l.mkCache(c, res.Log[i].Id))
		}
		if err := memcache.Get(c, ancestorCaches...); err != nil {
			merr, ok := err.(errors.MultiError)
			if !ok {
				merr = errors.MultiError{err}
			}
			for i, ierr := range merr {
				e := ancestorCaches[i]
				if ierr != nil && ierr != memcache.ErrCacheMiss {
					logging.WithError(err).Errorf(c, "Failed to retrieve cache entry at %q", e.Key())
				}
				if ierr != nil {
					e.SetValue(nil)
				}
			}
		}

		topCommitID, err := hex.DecodeString(res.Log[0].Id)
		if err != nil {
			logging.WithError(err).Errorf(c, "commit id %q is not a valid hex", res.Log[0].Id)
		} else {
			for i, e := range ancestorCaches {
				dist := byte(i + 1)
				if v := e.Value(); len(v) > 0 && v[len(v)-1] <= dist {
					// This cache entry is not worse than what we can offer.
				} else {
					// We have data with a shorter distance.
					// see logReq comment for format of this cache value.
					v := make([]byte, len(topCommitID)+1)
					copy(v, topCommitID)
					v[len(v)-1] = dist
					e.SetValue(v)
					caches = append(caches, e)
				}
			}
		}
	}

	// This could be potentially improved by using CAS,
	// but it would significantly complicate this code.
	if err := memcache.Set(c, caches...); err != nil {
		logging.WithError(err).Errorf(c, "Failed to cache gitiles log")
	} else {
		logging.Debugf(c, "wrote %d entries", len(caches))
	}
}

func (l *logReq) mkCache(c context.Context, commitish string) memcache.Item {
	// note: better not to include limit in the cache key.
	key := fmt.Sprintf("git-log-%s|%s|%s|%s|%t", l.host, l.project, commitish,
		l.ancestor, l.withFiles)
	item := memcache.NewItem(c, key)
	// do not pollute memcache with items we probably won't need soon.
	item.SetExpiration(12 * time.Hour)
	return item
}
