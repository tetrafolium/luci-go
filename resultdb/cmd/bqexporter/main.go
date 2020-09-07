// Copyright 2020 The LUCI Authors.
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

package main

import (
	"flag"

	"golang.org/x/time/rate"

	"github.com/tetrafolium/luci-go/server"

	"github.com/tetrafolium/luci-go/resultdb/internal"
	"github.com/tetrafolium/luci-go/resultdb/internal/services/bqexporter"
)

func main() {
	opts := bqexporter.DefaultOptions()
	flag.IntVar(&opts.TaskWorkers, "task-workers", opts.TaskWorkers,
		"Number of invocations to export concurrently")
	flag.BoolVar(&opts.UseInsertIDs, "insert-ids", opts.UseInsertIDs,
		"Use InsertIDs when inserting data to BigQuery")
	flag.IntVar(&opts.MaxBatchSizeApprox, "max-batch-size-approx", opts.MaxBatchSizeApprox,
		"Maximum size of a batch in bytes, approximate")
	flag.IntVar(&opts.MaxBatchTotalSizeApprox, "batch-total-size-approx", opts.MaxBatchTotalSizeApprox,
		"Maximum total size of batches in bytes, approximate")
	rateLimit := int(opts.RateLimit)
	flag.IntVar(&rateLimit, "rate-limit", rateLimit,
		"Maximum BigQuery request rate")

	internal.Main(func(srv *server.Server) error {
		opts.RateLimit = rate.Limit(rateLimit)
		bqexporter.InitServer(srv, opts)
		return nil
	})
}
