// Copyright 2016 The LUCI Authors.
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

package grpcmon

import (
	"context"
	"fmt"
	"time"

	gcode "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/tsmon/distribution"
	"github.com/tetrafolium/luci-go/common/tsmon/field"
	"github.com/tetrafolium/luci-go/common/tsmon/metric"
	"github.com/tetrafolium/luci-go/common/tsmon/types"
)

var (
	grpcServerCount = metric.NewCounter(
		"grpc/server/count",
		"Total number of RPCs.",
		nil,
		field.String("method"),         // full name of the grpc method
		field.Int("code"),              // grpc.Code of the result
		field.String("canonical_code")) // String representation of the code above

	grpcServerDuration = metric.NewCumulativeDistribution(
		"grpc/server/duration",
		"Distribution of server-side RPC duration (in milliseconds).",
		&types.MetricMetadata{Units: types.Milliseconds},
		distribution.DefaultBucketer,
		field.String("method"),         // full name of the grpc method
		field.Int("code"),              // grpc.Code of the result
		field.String("canonical_code")) // String representation of the code above
)

// UnaryServerInterceptor is a grpc.UnaryServerInterceptor that gathers RPC
// handler metrics and sends them to tsmon.
//
// It assumes the RPC context has tsmon initialized already.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	started := clock.Now(ctx)
	panicking := true
	defer func() {
		// We don't want to recover anything, but we want to log Internal error
		// in case of a panic. We pray here reportServerRPCMetrics is very
		// lightweight and it doesn't panic itself.
		code := codes.OK
		switch {
		case err != nil:
			code = grpc.Code(err)
		case panicking:
			code = codes.Internal
		}
		reportServerRPCMetrics(ctx, info.FullMethod, code, clock.Now(ctx).Sub(started))
	}()
	resp, err = handler(ctx, req)
	panicking = false // normal exit, no panic happened, disarms defer
	return
}

// reportServerRPCMetrics sends metrics after RPC handler has finished.
func reportServerRPCMetrics(ctx context.Context, method string, code codes.Code, dur time.Duration) {
	canon, ok := gcode.Code_name[int32(code)]
	if !ok {
		canon = fmt.Sprintf("Code(%d)", int64(code))
	}

	grpcServerCount.Add(ctx, 1, method, int(code), canon)
	grpcServerDuration.Add(ctx, float64(dur.Nanoseconds()/1e6), method, int(code), canon)
}
