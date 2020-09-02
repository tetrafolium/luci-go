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

package buffer

import (
	"time"

	"github.com/tetrafolium/luci-go/common/retry"
)

// Batch represents a collection of individual work items and associated
// metadata.
//
// Batches are are cut by the Channel according to Options.Buffer, and can be
// manipulated by ErrorFn and SendFn.
//
// ErrorFn and SendFn may manipulate the contents of the Batch (Data and Meta)
// to do things such as:
//   * Associate a UID with the Batch (e.g. in the Meta field) to identify it to
//     remote services for deduplication.
//   * Remove already-processed items from Data in case the SendFn partially
//     succeeded.
//
// The dispatcher accounts for the number of work items in the Batch as it
// leases the Batch out; initially the Batch's length will be len(Data). If the
// SendFn reduces the length of Data before the NACK, the accounted number of
// work items will be accordingly reduced. The accounted length can never grow
// (e.g. extending Data doesn't do anything).
type Batch struct {
	// Data is the individual work items pushed into the Buffer.
	Data []interface{}

	// Meta is an object which dispatcher.Channel will treat as totally opaque;
	// You may manipulate it in SendFn or ErrorFn as you see fit. This can be used
	// for e.g. associating a nonce with the Batch for retries, or stashing
	// a constructed RPC proto, etc.
	Meta interface{}

	// id is a 1-based counter which is generated by Buffer when the Batch
	// is created. Within a Buffer it is monotonically increasing.
	id uint64

	// retry is the retry.Iterator associated with this Batch. Its Next method
	// will be called when it is NACK'd.
	retry retry.Iterator

	// nextSend is the next timestamp after which this Batch is eligible for
	// sending.
	//
	// While the batch is the `currentBatch` in the buffer, this timestamp
	// represents the deadline for cutting this batch.
	nextSend time.Time

	// countedSize is the length of this Batch as the Buffer counts it. It starts
	// as the original value of len(Batch.Data) and can decrease if
	// len(Batch.Data) is smaller on a NACK().
	countedSize int
}
