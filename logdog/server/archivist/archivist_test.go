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

package archivist

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/gcloud/gs"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	logdog "github.com/tetrafolium/luci-go/logdog/api/endpoints/coordinator/services/v1"
	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/common/storage"
	"github.com/tetrafolium/luci-go/logdog/common/storage/memory"
	"github.com/tetrafolium/luci-go/logdog/common/types"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

// testServicesClient implements logdog.ServicesClient sufficient for testing
// and instrumentation.
type testServicesClient struct {
	logdog.ServicesClient

	lsCallback func(*logdog.LoadStreamRequest) (*logdog.LoadStreamResponse, error)
	asCallback func(*logdog.ArchiveStreamRequest) error
}

func (sc *testServicesClient) LoadStream(c context.Context, req *logdog.LoadStreamRequest, o ...grpc.CallOption) (
	*logdog.LoadStreamResponse, error) {
	if cb := sc.lsCallback; cb != nil {
		return cb(req)
	}
	return nil, errors.New("no callback implemented")
}

func (sc *testServicesClient) ArchiveStream(c context.Context, req *logdog.ArchiveStreamRequest, o ...grpc.CallOption) (
	*empty.Empty, error) {
	if cb := sc.asCallback; cb != nil {
		if err := cb(req); err != nil {
			return nil, err
		}
	}
	return &empty.Empty{}, nil
}

// testGSClient is a testing implementation of the gsClient interface.
//
// It does not actually retain any of the written data, since that level of
// testing is done in the archive package.
type testGSClient struct {
	sync.Mutex
	gs.Client

	// objs is a map of filename to "write amount". The write amount is the
	// cumulative amount of data written to the Writer for a given GS path.
	objs   map[gs.Path]int64
	closed bool

	closeErr     error
	newWriterErr func(w *testGSWriter) error
	deleteErr    func(gs.Path) error
	renameErr    func(gs.Path, gs.Path) error
}

func (c *testGSClient) NewWriter(p gs.Path) (gs.Writer, error) {
	w := testGSWriter{
		client: c,
		path:   p,
	}
	if c.newWriterErr != nil {
		if err := c.newWriterErr(&w); err != nil {
			return nil, err
		}
	}
	return &w, nil
}

func (c *testGSClient) Close() error {
	if c.closed {
		panic("double close")
	}
	if err := c.closeErr; err != nil {
		return err
	}
	c.closed = true
	return nil
}

func (c *testGSClient) Delete(p gs.Path) error {
	if c.deleteErr != nil {
		if err := c.deleteErr(p); err != nil {
			return err
		}
	}

	c.Lock()
	defer c.Unlock()

	delete(c.objs, p)
	return nil
}

func (c *testGSClient) Rename(src, dst gs.Path) error {
	if c.renameErr != nil {
		if err := c.renameErr(src, dst); err != nil {
			return err
		}
	}

	c.Lock()
	defer c.Unlock()

	c.objs[dst] = c.objs[src]
	delete(c.objs, src)
	return nil
}

type testGSWriter struct {
	client *testGSClient

	path       gs.Path
	closed     bool
	writeCount int64

	writeErr error
	closeErr error
}

func (w *testGSWriter) Write(d []byte) (int, error) {
	if err := w.writeErr; err != nil {
		return 0, err
	}

	w.client.Lock()
	defer w.client.Unlock()

	if w.client.objs == nil {
		w.client.objs = make(map[gs.Path]int64)
	}
	w.client.objs[w.path] += int64(len(d))
	w.writeCount += int64(len(d))
	return len(d), nil
}

func (w *testGSWriter) Close() error {
	if w.closed {
		panic("double close")
	}
	if err := w.closeErr; err != nil {
		return err
	}
	w.closed = true
	return nil
}

func (w *testGSWriter) Count() int64 {
	return w.writeCount
}

func TestHandleArchive(t *testing.T) {
	t.Parallel()

	Convey(`A testing archive setup`, t, func() {
		c, tc := testclock.UseTime(context.Background(), testclock.TestTimeUTC)

		st := memory.Storage{}
		gsc := testGSClient{}
		gscFactory := func(context.Context, string) (gs.Client, error) {
			return &gsc, nil
		}

		// Set up our test log stream.
		project := "test-project"
		desc := logpb.LogStreamDescriptor{
			Prefix: "testing",
			Name:   "foo",
		}

		// Utility function to add a log entry for "ls".
		addTestEntry := func(p string, idxs ...int) {
			for _, v := range idxs {
				le := logpb.LogEntry{
					PrefixIndex: uint64(v),
					StreamIndex: uint64(v),
					Content: &logpb.LogEntry_Text{&logpb.Text{
						Lines: []*logpb.Text_Line{
							{
								Value:     []byte(fmt.Sprintf("line #%d", v)),
								Delimiter: "\n",
							},
						},
					}},
				}

				d, err := proto.Marshal(&le)
				if err != nil {
					panic(err)
				}

				err = st.Put(c, storage.PutRequest{
					Project: p,
					Path:    desc.Path(),
					Index:   types.MessageIndex(v),
					Values:  [][]byte{d},
				})
				if err != nil {
					panic(err)
				}

				// Advance the time for each log entry.
				tc.Add(time.Second)
			}
		}

		// Set up our testing archival task.
		expired := 10 * time.Minute
		task := &logdog.ArchiveTask{
			Project: project,
			Id:      "coordinator-stream-id",
		}
		expired++ // This represents a time PAST CompletePeriod.

		// Set up our test Coordinator client stubs.
		stream := logdog.LoadStreamResponse{
			State: &logdog.LogStreamState{
				ProtoVersion:  logpb.Version,
				TerminalIndex: -1,
				Archived:      false,
				Purged:        false,
			},
		}

		// Allow tests to modify the log stream descriptor.
		reloadDesc := func() {
			descBytes, err := proto.Marshal(&desc)
			if err != nil {
				panic(err)
			}
			stream.Desc = descBytes
		}
		reloadDesc()

		var archiveRequest *logdog.ArchiveStreamRequest
		var archiveStreamErr error
		sc := testServicesClient{
			lsCallback: func(req *logdog.LoadStreamRequest) (*logdog.LoadStreamResponse, error) {
				return &stream, nil
			},
			asCallback: func(req *logdog.ArchiveStreamRequest) error {
				archiveRequest = req
				return archiveStreamErr
			},
		}

		stBase := Settings{}

		ar := Archivist{
			Service: &sc,
			SettingsLoader: func(c context.Context, project string) (*Settings, error) {
				// Extra slashes to test concatenation,.
				st := stBase
				st.GSBase = gs.Path(fmt.Sprintf("gs://archival/%s/path/to/archive/", project))
				st.GSStagingBase = gs.Path(fmt.Sprintf("gs://archival-staging/%s/path/to/archive/", project))
				return &st, nil
			},
			Storage:         &st,
			GSClientFactory: gscFactory,
		}

		gsURL := func(project, name string) string {
			return fmt.Sprintf("gs://archival/%s/path/to/archive/%s/%s/%s", project, project, desc.Path(), name)
		}

		// hasStreams can be called to check that the retained archiveRequest had
		// data sizes for the named archive stream types.
		//
		// After checking, the values are set to zero. This allows us to use
		// ShouldEqual without hardcoding specific archival sizes into the results.
		hasStreams := func(log, index, data bool) bool {
			So(archiveRequest, ShouldNotBeNil)
			if (log && archiveRequest.StreamSize <= 0) ||
				(index && archiveRequest.IndexSize <= 0) {
				return false
			}

			archiveRequest.StreamSize = 0
			archiveRequest.IndexSize = 0
			return true
		}

		Convey(`Will return task and fail to archive if the specified stream state could not be loaded.`, func() {
			sc.lsCallback = func(*logdog.LoadStreamRequest) (*logdog.LoadStreamResponse, error) {
				return nil, errors.New("does not exist")
			}

			So(ar.archiveTaskImpl(c, task), ShouldErrLike, "does not exist")
		})

		Convey(`Will consume task and refrain from archiving if the stream is already archived.`, func() {
			stream.State.Archived = true

			So(ar.archiveTaskImpl(c, task), ShouldBeNil)
			So(archiveRequest, ShouldBeNil)
		})

		Convey(`Will consume task and refrain from archiving if the stream is purged.`, func() {
			stream.State.Purged = true

			So(ar.archiveTaskImpl(c, task), ShouldBeNil)
			So(archiveRequest, ShouldBeNil)
		})

		Convey(`With terminal index "3"`, func() {
			stream.State.TerminalIndex = 3

			Convey(`Will consume the task if the log stream has no entries.`, func() {
				So(st.Count(project, desc.Path()), ShouldEqual, 0)
				So(ar.archiveTaskImpl(c, task), ShouldBeNil)
				So(st.Count(project, desc.Path()), ShouldEqual, 0)
			})

			Convey(`Will archive {0, 1, 2, 4} (incomplete).`, func() {
				addTestEntry(project, 0, 1, 2, 4)
				So(st.Count(project, desc.Path()), ShouldEqual, 4)
				So(ar.archiveTaskImpl(c, task), ShouldBeNil)
				So(st.Count(project, desc.Path()), ShouldEqual, 0)
			})

			Convey(`Will successfully archive {0, 1, 2, 3, 4}, stopping at the terminal index.`, func() {
				addTestEntry(project, 0, 1, 2, 3, 4)

				So(st.Count(project, desc.Path()), ShouldEqual, 5)
				So(ar.archiveTaskImpl(c, task), ShouldBeNil)
				So(st.Count(project, desc.Path()), ShouldEqual, 0)

				So(hasStreams(true, true, true), ShouldBeTrue)

				So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
					Project:       project,
					Id:            task.Id,
					LogEntryCount: 4,
					TerminalIndex: 3,

					StreamUrl: gsURL(project, "logstream.entries"),
					IndexUrl:  gsURL(project, "logstream.index"),
				})
			})

			Convey(`When a transient archival error occurs, will not consume the task.`, func() {
				addTestEntry(project, 0, 1, 2, 3, 4)
				gsc.newWriterErr = func(*testGSWriter) error { return errors.New("test error", transient.Tag) }

				So(st.Count(project, desc.Path()), ShouldEqual, 5)
				So(ar.archiveTaskImpl(c, task), ShouldErrLike, "test error")
				So(st.Count(project, desc.Path()), ShouldEqual, 5)
			})

			Convey(`When a non-transient archival error occurs`, func() {
				addTestEntry(project, 0, 1, 2, 3, 4)
				archiveErr := errors.New("archive failure error")
				gsc.newWriterErr = func(*testGSWriter) error { return archiveErr }

				Convey(`If remote report returns an error, do not consume the task.`, func() {
					archiveStreamErr = errors.New("test error", transient.Tag)

					So(st.Count(project, desc.Path()), ShouldEqual, 5)
					So(ar.archiveTaskImpl(c, task), ShouldErrLike, "test error")
					So(st.Count(project, desc.Path()), ShouldEqual, 5)

					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project: project,
						Id:      task.Id,
						Error:   "archive failure error",
					})
				})

				Convey(`If remote report returns success, the task is consumed.`, func() {
					So(st.Count(project, desc.Path()), ShouldEqual, 5)
					So(ar.archiveTaskImpl(c, task), ShouldBeNil)
					So(st.Count(project, desc.Path()), ShouldEqual, 0)
					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project: project,
						Id:      task.Id,
						Error:   "archive failure error",
					})
				})

				Convey(`If an empty error string is supplied, the generic error will be filled in.`, func() {
					archiveErr = errors.New("")

					So(ar.archiveTaskImpl(c, task), ShouldBeNil)
					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project: project,
						Id:      task.Id,
						Error:   "archival error",
					})
				})
			})
		})

		Convey(`When not enforcing stream completeness`, func() {
			stream.Age = google.NewDuration(expired)

			Convey(`With no terminal index`, func() {
				Convey(`Will successfully archive if there are no entries.`, func() {
					So(ar.archiveTaskImpl(c, task), ShouldBeNil)

					So(hasStreams(true, true, false), ShouldBeTrue)
					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project:       project,
						Id:            task.Id,
						LogEntryCount: 0,
						TerminalIndex: -1,

						StreamUrl: gsURL(project, "logstream.entries"),
						IndexUrl:  gsURL(project, "logstream.index"),
					})
				})

				Convey(`With {0, 1, 2, 4} (incomplete) will archive the stream and update its terminal index.`, func() {
					addTestEntry(project, 0, 1, 2, 4)

					So(st.Count(project, desc.Path()), ShouldEqual, 4)
					So(ar.archiveTaskImpl(c, task), ShouldBeNil)
					So(st.Count(project, desc.Path()), ShouldEqual, 0)

					So(hasStreams(true, true, true), ShouldBeTrue)
					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project:       project,
						Id:            task.Id,
						LogEntryCount: 4,
						TerminalIndex: 4,

						StreamUrl: gsURL(project, "logstream.entries"),
						IndexUrl:  gsURL(project, "logstream.index"),
					})
				})
			})

			Convey(`With terminal index 3`, func() {
				stream.State.TerminalIndex = 3

				Convey(`Will successfully archive if there are no entries.`, func() {
					So(ar.archiveTaskImpl(c, task), ShouldBeNil)

					So(hasStreams(true, true, false), ShouldBeTrue)
					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project:       project,
						Id:            task.Id,
						LogEntryCount: 0,
						TerminalIndex: -1,

						StreamUrl: gsURL(project, "logstream.entries"),
						IndexUrl:  gsURL(project, "logstream.index"),
					})
				})

				Convey(`With {0, 1, 2, 4} (incomplete) will archive the stream and update its terminal index to 2.`, func() {
					addTestEntry(project, 0, 1, 2, 4)

					So(st.Count(project, desc.Path()), ShouldEqual, 4)
					So(ar.archiveTaskImpl(c, task), ShouldBeNil)
					So(st.Count(project, desc.Path()), ShouldEqual, 0)

					So(hasStreams(true, true, true), ShouldBeTrue)
					So(archiveRequest, ShouldResembleProto, &logdog.ArchiveStreamRequest{
						Project:       project,
						Id:            task.Id,
						LogEntryCount: 3,
						TerminalIndex: 2,

						StreamUrl: gsURL(project, "logstream.entries"),
						IndexUrl:  gsURL(project, "logstream.index"),
					})
				})
			})
		})

		Convey(`With an empty project name, will fail and consume the task.`, func() {
			task.Project = ""

			So(ar.archiveTaskImpl(c, task), ShouldBeNil)
		})

		Convey(`With an invalid project name, will fail and consume the task.`, func() {
			task.Project = "!!! invalid project name !!!"

			So(ar.archiveTaskImpl(c, task), ShouldBeNil)
		})

		// Simulate failures during the various stream generation operations.
		Convey(`Stream generation failures`, func() {
			stream.State.TerminalIndex = 3
			addTestEntry(project, 0, 1, 2, 3)

			for _, failName := range []string{"/logstream.entries", "/logstream.index"} {
				for _, testCase := range []struct {
					name  string
					setup func()
				}{
					{"writer create failure", func() {
						gsc.newWriterErr = func(w *testGSWriter) error {
							if strings.HasSuffix(string(w.path), failName) {
								return errors.New("test error", transient.Tag)
							}
							return nil
						}
					}},

					{"write failure", func() {
						gsc.newWriterErr = func(w *testGSWriter) error {
							if strings.HasSuffix(string(w.path), failName) {
								w.writeErr = errors.New("test error", transient.Tag)
							}
							return nil
						}
					}},

					{"rename failure", func() {
						gsc.renameErr = func(src, dst gs.Path) error {
							if strings.HasSuffix(string(src), failName) {
								return errors.New("test error", transient.Tag)
							}
							return nil
						}
					}},

					{"close failure", func() {
						gsc.newWriterErr = func(w *testGSWriter) error {
							if strings.HasSuffix(string(w.path), failName) {
								w.closeErr = errors.New("test error", transient.Tag)
							}
							return nil
						}
					}},

					{"delete failure after other failure", func() {
						// Simulate a write failure. This is the error that will actually
						// be returned.
						gsc.newWriterErr = func(w *testGSWriter) error {
							if strings.HasSuffix(string(w.path), failName) {
								w.writeErr = errors.New("test error", transient.Tag)
							}
							return nil
						}

						// This will trigger when NewWriter fails from the above
						// instrumentation.
						gsc.deleteErr = func(p gs.Path) error {
							if strings.HasSuffix(string(p), failName) {
								return errors.New("other error")
							}
							return nil
						}
					}},
				} {
					Convey(fmt.Sprintf(`Can handle %s for %s, and will not archive.`, testCase.name, failName), func() {
						testCase.setup()

						So(ar.archiveTaskImpl(c, task), ShouldErrLike, "test error")
						So(archiveRequest, ShouldBeNil)
					})
				}
			}
		})
	})
}
