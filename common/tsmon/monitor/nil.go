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

package monitor

import (
	"context"

	"github.com/tetrafolium/luci-go/common/tsmon/types"
)

type nilMonitor struct{}

// NewNilMonitor returns a Monitor that does nothing.
func NewNilMonitor() Monitor {
	return &nilMonitor{}
}

func (m *nilMonitor) ChunkSize() int {
	return 0
}

func (m *nilMonitor) Send(ctx context.Context, cells []types.Cell) error {
	return nil
}

func (m *nilMonitor) Close() error {
	return nil
}
