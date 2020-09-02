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

package gaetesting

import (
	"context"
	"flag"

	"github.com/tetrafolium/luci-go/common/logging/gologger"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/server/caching"
	"github.com/tetrafolium/luci-go/server/secrets/testsecrets"
)

var goLogger = flag.Bool("test.gologger", false, "Enable console logging during test.")

// TestingContext returns context with base services installed:
//   * github.com/tetrafolium/luci-go/gae/impl/memory (in-memory appengine services)
//   * github.com/tetrafolium/luci-go/server/caching (access to process cache)
//   * github.com/tetrafolium/luci-go/server/secrets/testsecrets (access to fake secret keys)
func TestingContext() context.Context {
	ctx := context.Background()
	ctx = memory.Use(ctx)
	return commonTestingContext(ctx)
}

// TestingContextWithAppID returns context with the specified App ID and base
// services installed:
//   * github.com/tetrafolium/luci-go/gae/impl/memory (in-memory appengine services)
//   * github.com/tetrafolium/luci-go/server/caching (access to process cache)
//   * github.com/tetrafolium/luci-go/server/secrets/testsecrets (access to fake secret keys)
func TestingContextWithAppID(appID string) context.Context {
	ctx := context.Background()
	ctx = memory.UseWithAppID(ctx, appID)
	return commonTestingContext(ctx)
}

func commonTestingContext(ctx context.Context) context.Context {
	ctx = testsecrets.Use(ctx)
	ctx = caching.WithEmptyProcessCache(ctx)

	if *goLogger {
		ctx = gologger.StdConfig.Use(ctx)
	}

	return ctx
}
