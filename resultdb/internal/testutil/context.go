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

package testutil

import (
	"context"
	"math/rand"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/common/logging/gologger"
	"github.com/tetrafolium/luci-go/server/secrets"
	"github.com/tetrafolium/luci-go/server/secrets/testsecrets"
)

func testingContext(mockClock bool) context.Context {
	ctx := context.Background()

	// Enable logging to stdout/stderr.
	ctx = gologger.StdConfig.Use(ctx)

	if mockClock {
		ctx, _ = testclock.UseTime(ctx, testclock.TestRecentTimeUTC)
	}

	// Set test secrets store for token generation/validation.
	ctx = secrets.Set(ctx, &testsecrets.Store{})

	ctx = mathrand.Set(ctx, rand.New(rand.NewSource(0)))

	return ctx
}

// TestingContext returns a context to be used in tests.
func TestingContext() context.Context {
	return testingContext(true)
}
