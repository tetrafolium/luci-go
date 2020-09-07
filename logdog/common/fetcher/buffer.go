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

package fetcher

import (
	"container/list"

	"github.com/tetrafolium/luci-go/logdog/api/logpb"
)

type logBuffer struct {
	l      list.List
	cur    []*logpb.LogEntry
	curIdx int

	count int
}

func (b *logBuffer) current() *logpb.LogEntry {
	for b.curIdx >= len(b.cur) {
		if b.l.Len() == 0 {
			return nil
		}
		b.cur = b.l.Remove(b.l.Front()).([]*logpb.LogEntry)
		b.curIdx = 0
	}

	return b.cur[b.curIdx]
}

func (b *logBuffer) next() {
	b.curIdx++
	b.count--
}

func (b *logBuffer) append(le ...*logpb.LogEntry) {
	b.l.PushBack(le)
	b.count += len(le)
}

func (b *logBuffer) size() int {
	return b.count
}
