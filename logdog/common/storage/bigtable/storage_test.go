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

package bigtable

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/tetrafolium/luci-go/common/data/recordio"
	"github.com/tetrafolium/luci-go/logdog/common/storage"
	"github.com/tetrafolium/luci-go/logdog/common/storage/memory"
	"github.com/tetrafolium/luci-go/logdog/common/types"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func mustGetIndex(e *storage.Entry) types.MessageIndex {
	idx, err := e.GetStreamIndex()
	if err != nil {
		panic(err)
	}
	return idx
}

func TestStorage(t *testing.T) {
	t.Parallel()

	Convey(`A BigTable storage instance bound to a testing BigTable instance`, t, func() {
		c := context.Background()

		var cache memory.Cache
		s := NewMemoryInstance(&cache)
		defer s.Close()

		project := "test-project"
		get := func(path string, index int, limit int, keysOnly bool) ([]string, error) {
			req := storage.GetRequest{
				Project:  project,
				Path:     types.StreamPath(path),
				Index:    types.MessageIndex(index),
				Limit:    limit,
				KeysOnly: keysOnly,
			}
			var got []string
			err := s.Get(c, req, func(e *storage.Entry) bool {
				if keysOnly {
					got = append(got, strconv.Itoa(int(mustGetIndex(e))))
				} else {
					got = append(got, string(e.D))
				}
				return true
			})
			return got, err
		}

		put := func(path string, index int, d ...string) error {
			data := make([][]byte, len(d))
			for i, v := range d {
				data[i] = []byte(v)
			}

			return s.Put(c, storage.PutRequest{
				Project: project,
				Path:    types.StreamPath(path),
				Index:   types.MessageIndex(index),
				Values:  data,
			})
		}

		ekey := func(path string, v, c int64) string {
			return newRowKey(project, path, v, c).encode()
		}
		records := func(s ...string) []byte {
			buf := bytes.Buffer{}
			w := recordio.NewWriter(&buf)

			for _, v := range s {
				if _, err := w.Write([]byte(v)); err != nil {
					panic(err)
				}
				if err := w.Flush(); err != nil {
					panic(err)
				}
			}

			return buf.Bytes()
		}

		Convey(`With an artificial maximum BigTable row size of two records`, func() {
			// Artificially constrain row size. 4 = 2*{size/1, data/1} RecordIO
			// entries.
			s.SetMaxRowSize(4)

			Convey(`Will split row data that overflows the table into multiple rows.`, func() {
				So(put("A", 0, "0", "1", "2", "3"), ShouldBeNil)

				So(s.DataMap(), ShouldResemble, map[string][]byte{
					ekey("A", 1, 2): records("0", "1"),
					ekey("A", 3, 2): records("2", "3"),
				})
			})

			Convey(`Loading a single row data beyond the maximum row size will fail.`, func() {
				So(put("A", 0, "0123"), ShouldErrLike, "single row entry exceeds maximum size")
			})
		})

		Convey(`With row data: A{0, 1, 2, 3, 4}, B{10, 12, 13}`, func() {
			So(put("A", 0, "0", "1", "2"), ShouldBeNil)
			So(put("A", 3, "3", "4"), ShouldBeNil)
			So(put("B", 10, "10"), ShouldBeNil)
			So(put("B", 12, "12", "13"), ShouldBeNil)
			So(put("C", 0, "0", "1", "2"), ShouldBeNil)
			So(put("C", 4, "4"), ShouldBeNil)

			Convey(`Testing "Put"...`, func() {
				Convey(`Loads the row data.`, func() {
					So(s.DataMap(), ShouldResemble, map[string][]byte{
						ekey("A", 2, 3):  records("0", "1", "2"),
						ekey("A", 4, 2):  records("3", "4"),
						ekey("B", 10, 1): records("10"),
						ekey("B", 13, 2): records("12", "13"),
						ekey("C", 2, 3):  records("0", "1", "2"),
						ekey("C", 4, 1):  records("4"),
					})
				})
			})

			Convey(`Testing "Get"...`, func() {
				Convey(`Can fetch the full row, "A".`, func() {
					got, err := get("A", 0, 0, false)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"0", "1", "2", "3", "4"})
				})

				Convey(`Will fetch A{1, 2, 3, 4} with index=1.`, func() {
					got, err := get("A", 1, 0, false)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"1", "2", "3", "4"})
				})

				Convey(`Will fetch A{1, 2} with index=1 and limit=2.`, func() {
					got, err := get("A", 1, 2, false)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"1", "2"})
				})

				Convey(`Will fetch B{10, 12, 13} for B.`, func() {
					got, err := get("B", 0, 0, false)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"10", "12", "13"})
				})

				Convey(`Will fetch B{12, 13} when index=11.`, func() {
					got, err := get("B", 11, 0, false)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"12", "13"})
				})

				Convey(`Will fetch {} for INVALID.`, func() {
					got, err := get("INVALID", 0, 0, false)
					So(err, ShouldBeNil)
					So(got, ShouldBeEmpty)
				})
			})

			Convey(`Testing "Get" (keys only)`, func() {
				Convey(`Can fetch the full row, "A".`, func() {
					got, err := get("A", 0, 0, true)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"0", "1", "2", "3", "4"})
				})

				Convey(`Will fetch A{1, 2, 3, 4} with index=1.`, func() {
					got, err := get("A", 1, 0, true)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"1", "2", "3", "4"})
				})

				Convey(`Will fetch A{1, 2} with index=1 and limit=2.`, func() {
					got, err := get("A", 1, 2, true)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"1", "2"})
				})

				Convey(`Will fetch B{10, 12, 13} for B.`, func() {
					got, err := get("B", 0, 0, true)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"10", "12", "13"})
				})

				Convey(`Will fetch B{12, 13} when index=11.`, func() {
					got, err := get("B", 11, 0, true)
					So(err, ShouldBeNil)
					So(got, ShouldResemble, []string{"12", "13"})
				})
			})

			Convey(`Testing "Tail"...`, func() {
				tail := func(path string) (string, error) {
					e, err := s.Tail(c, project, types.StreamPath(path))
					if err != nil {
						return "", err
					}
					return string(e.D), nil
				}

				Convey(`A tail request for "A" returns A{4}.`, func() {
					got, err := tail("A")
					So(err, ShouldBeNil)
					So(got, ShouldEqual, "4")

					Convey(`(Cache) A second request also returns A{4}.`, func() {
						got, err := tail("A")
						So(err, ShouldBeNil)
						So(got, ShouldEqual, "4")
					})
				})

				Convey(`A tail request for "B" returns nothing (no contiguous logs).`, func() {
					_, err := tail("B")
					So(err, ShouldEqual, storage.ErrDoesNotExist)
				})

				Convey(`A tail request for "C" returns 2.`, func() {
					got, err := tail("C")
					So(err, ShouldBeNil)
					So(got, ShouldEqual, "2")

					Convey(`(Cache) A second request also returns 2.`, func() {
						got, err := tail("C")
						So(err, ShouldBeNil)
						So(got, ShouldEqual, "2")
					})

					Convey(`(Cache) After "3" is added, a second request returns 4.`, func() {
						So(put("C", 3, "3"), ShouldBeNil)

						got, err := tail("C")
						So(err, ShouldBeNil)
						So(got, ShouldEqual, "4")
					})
				})

				Convey(`A tail request for "INVALID" errors NOT FOUND.`, func() {
					_, err := tail("INVALID")
					So(err, ShouldEqual, storage.ErrDoesNotExist)
				})
			})
		})
	})
}
