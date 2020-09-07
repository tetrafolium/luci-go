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

package buildbucket

import (
	"context"
	"time"

	"google.golang.org/api/googleapi"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/common/retry"
	"github.com/tetrafolium/luci-go/common/retry/transient"
)

// Fetch fetches builds matching the search criteria.
// It stops when all builds are found or when context is cancelled.
// The order of returned builds is from most-recently-created to least-recently-created.
//
// c.MaxBuilds value is used as a result page size, defaults to 100.
// limit, if >0, specifies the maximum number of builds to return.
//
// If ret is nil, retries transient errors with exponential back-off.
// Logs errors on retries.
//
// Returns nil only if the search results are exhausted.
// May return context.Canceled.
func (c *SearchCall) Fetch(limit int, ret retry.Factory) ([]*LegacyApiCommonBuildMessage, string, error) {
	// Default page size to 100 because we are fetching everything.
	maxBuildsKey := "max_builds"
	origMaxBuilds := c.urlParams_.Get(maxBuildsKey)
	if origMaxBuilds == "" {
		c.MaxBuilds(100)
		defer c.urlParams_.Set(maxBuildsKey, origMaxBuilds)
	}

	ch := make(chan *LegacyApiCommonBuildMessage)
	var err error
	var cursor string
	go func() {
		defer close(ch)
		cursor, err = c.Run(ch, limit, ret)
	}()

	var builds []*LegacyApiCommonBuildMessage
	for b := range ch {
		builds = append(builds, b)
	}
	return builds, cursor, err
}

// Run is like Fetch, but sends results to a channel and the default page size
// is defined by the server (10 as of Sep 2017).
//
// Run blocks on sending.
func (c *SearchCall) Run(builds chan<- *LegacyApiCommonBuildMessage, limit int, ret retry.Factory) (cursor string, err error) {
	if ret == nil {
		ret = transient.Only(retry.Default)
	}

	// We will be mutating c.
	// Guarantee that it will remain the same by the time function exits.
	origCtx := c.ctx_
	const (
		cursorKey = "start_cursor"
		fieldsKey = "fields"
	)
	origCursor := c.urlParams_.Get(cursorKey)
	origFields := c.urlParams_.Get(fieldsKey)
	defer func() {
		// Use the low-level API to be consistent with reads.
		c.ctx_ = origCtx
		c.urlParams_.Set(cursorKey, origCursor)
		c.urlParams_.Set(fieldsKey, origFields)
	}()

	// ensure "next_cursor" is included
	c.urlParams_.Set(fieldsKey, googleapi.CombineFields([]googleapi.Field{
		googleapi.Field(origFields),
		"next_cursor",
	}))

	// Make a non-nil context used by default in this function.
	ctx := origCtx
	if ctx == nil {
		// won't happen on AppEngine in practice.
		ctx = context.Background()
	}

	sent := 0
outer:
	for {
		var res *LegacyApiSearchResponseMessage
		err = retry.Retry(ctx, ret,
			func() error {
				var err error
				// Set a timeout for this particular RPC.
				var cancel context.CancelFunc
				c.ctx_, cancel = context.WithTimeout(ctx, time.Minute)
				defer cancel()
				res, err = c.Do()
				c.ctx_ = origCtx // for code clarity only

				switch apiErr, _ := err.(*googleapi.Error); {
				case apiErr != nil && apiErr.Code >= 500:
					return transient.Tag.Apply(err)
				case err == context.DeadlineExceeded && ctx.Err() == nil:
					return transient.Tag.Apply(err) // request-level timeout
				case err != nil:
					return err
				case res.Error != nil:
					return errors.New(res.Error.Message)
				default:
					return nil
				}
			},
			func(err error, wait time.Duration) {
				logging.WithError(err).Warningf(ctx, "RPC error while searching builds; will retry in %s", wait)
			})
		if err != nil {
			return
		}
		cursor = res.NextCursor

		for _, b := range res.Builds {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			case builds <- b:
				sent++
				if sent == limit {
					break outer
				}
			}
		}

		if len(res.Builds) == 0 || res.NextCursor == "" {
			break
		}
		c.urlParams_.Set(cursorKey, res.NextCursor)
	}

	return
}
