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

package buildmerge

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	bbpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/common/types"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func mkDgram(build *bbpb.Build) *logpb.LogEntry {
	data, err := proto.Marshal(build)
	So(err, ShouldBeNil)
	return &logpb.LogEntry{
		Content: &logpb.LogEntry_Datagram{Datagram: &logpb.Datagram{
			Data: data,
		}},
	}
}

// cleanupProtoError inspects the SummaryMarkdown field of build proto
// and if it contains proto err, it will strip out the proto error message.
// The reason we are doing this is that the proto lib discourage us to perform
// error string comparison. See:
// https://github.com/protocolbuffers/protobuf-go/blob/cd108d00a8df3bba55927ef35ca07438b895d7aa/internal/errors/errors.go#L26-L34
func cleanupProtoError(builds ...*bbpb.Build) {
	re := regexp.MustCompile(`Error in build protocol:.*\sproto:`)
	for _, build := range builds {
		if loc := re.FindStringIndex(build.SummaryMarkdown); loc != nil {
			build.SummaryMarkdown = build.SummaryMarkdown[:loc[1]]
		}
	}
}

func assertStateEqual(actual, expected *buildState) {
	cleanupProtoError(actual.build, expected.build)
	So(actual.build, ShouldResembleProto, expected.build)
	So(actual.closed, ShouldEqual, expected.closed)
	So(actual.final, ShouldEqual, expected.final)
	So(actual.invalid, ShouldEqual, expected.invalid)
}

func TestBuildState(t *testing.T) {
	t.Parallel()

	Convey(`buildState`, t, func() {
		now, err := ptypes.TimestampProto(testclock.TestRecentTimeLocal)
		So(err, ShouldBeNil)
		ctx, _ := testclock.UseTime(context.Background(), testclock.TestRecentTimeLocal)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		merger := New(ctx, "u/", &bbpb.Build{}, func(ns, stream types.StreamName) (url, viewURL string) {
			return fmt.Sprintf("url://%s%s", ns, stream), fmt.Sprintf("view://%s%s", ns, stream)
		})
		defer merger.Close()

		informChan := make(chan struct{}, 1)
		merger.informNewData = func() {
			informChan <- struct{}{}
		}
		wait := func() {
			<-informChan
		}

		Convey(`opened in error state`, func() {
			bs := newBuildStateTracker(ctx, merger, "ns/", false, errors.New("nope"))
			wait() // for final build
			assertStateEqual(bs.getLatest(), &buildState{
				build: &bbpb.Build{
					SummaryMarkdown: "\n\nError in build protocol: nope",
					Status:          bbpb.Status_INFRA_FAILURE,
					UpdateTime:      now,
					EndTime:         now,
				},
				closed: true,
				final:  true,
			})
		})

		Convey(`basic`, func() {
			bs := newBuildStateTracker(ctx, merger, "ns/", false, nil)

			Convey(`ignores updates when merger cancels context`, func() {
				cancel()
				wait() // cancel generated an 'informNewData'

				bs.handleNewData(mkDgram(&bbpb.Build{
					SummaryMarkdown: "some stuff",
					Steps: []*bbpb.Step{
						{Name: "Parent"},
						{Name: "Parent|Child"},
						{
							Name: "Parent|Merge",
							Logs: []*bbpb.Log{{
								Name: "$build.proto",
								Url:  "Parent/Merge/build.proto",
							}},
						},
					},
				}))
				// No wait, because this handleNewData was ignored.

				assertStateEqual(bs.GetFinal(), &buildState{
					build: &bbpb.Build{
						EndTime:         now,
						UpdateTime:      now,
						Status:          bbpb.Status_INFRA_FAILURE,
						SummaryMarkdown: "Never received any build data.",
					},
					closed: true,
					final:  true,
				})

				Convey(`can still close, though`, func() {
					bs.handleNewData(nil)
					assertStateEqual(bs.GetFinal(), &buildState{
						closed: true,
						final:  true,
						build: &bbpb.Build{
							EndTime:         now,
							UpdateTime:      now,
							Status:          bbpb.Status_INFRA_FAILURE,
							SummaryMarkdown: "Never received any build data.",
						},
					})
				})
			})

			Convey(`no updates`, func() {
				Convey(`getLatest`, func() {
					assertStateEqual(bs.getLatest(), &buildState{
						build: &bbpb.Build{
							Status:          bbpb.Status_SCHEDULED,
							SummaryMarkdown: "build.proto not found",
						}})
				})

				Convey(`handleNewData(nil)`, func() {
					bs.handleNewData(nil)
					wait()
					assertStateEqual(bs.GetFinal(), &buildState{
						closed: true,
						final:  true,
						build: &bbpb.Build{
							EndTime:         now,
							UpdateTime:      now,
							Status:          bbpb.Status_INFRA_FAILURE,
							SummaryMarkdown: "Never received any build data.",
						}},
					)

					Convey(`convey close noop`, func() {
						// not a valid state, but bs is closed, so handleNewData should do nothing
						// when invoked.
						bs.latestState = &buildState{
							closed: true,
							final:  true,
							build:  &bbpb.Build{SummaryMarkdown: "wat"},
						}
						bs.handleNewData(nil)
						assertStateEqual(bs.GetFinal(), &buildState{
							closed: true,
							final:  true,
							build:  &bbpb.Build{SummaryMarkdown: "wat"},
						})
					})
				})
			})

			Convey(`valid update`, func() {
				bs.handleNewData(mkDgram(&bbpb.Build{
					SummaryMarkdown: "some stuff",
					Steps: []*bbpb.Step{
						{Name: "Parent"},
						{Name: "Parent|Child"},
						{
							Name: "Parent|Merge",
							Logs: []*bbpb.Log{{
								Name: "$build.proto",
								Url:  "Parent/Merge/build.proto",
							}},
						},
					},
				}))
				wait()

				assertStateEqual(bs.getLatest(), &buildState{
					build: &bbpb.Build{
						SummaryMarkdown: "some stuff",
						Steps: []*bbpb.Step{
							{Name: "Parent"},
							{Name: "Parent|Child"},
							{
								Name: "Parent|Merge",
								Logs: []*bbpb.Log{{
									Name:    "$build.proto",
									Url:     "url://ns/Parent/Merge/build.proto",
									ViewUrl: "view://ns/Parent/Merge/build.proto",
								}},
							},
						},
						UpdateTime: now,
					},
				})

				Convey(`followed by garbage`, func() {
					bs.handleNewData(&logpb.LogEntry{
						Content: &logpb.LogEntry_Datagram{Datagram: &logpb.Datagram{
							Data: []byte("narpnarp"),
						}},
					})
					wait()
					wait() // for final build
					assertStateEqual(bs.GetFinal(), &buildState{
						closed:  true,
						final:   true,
						invalid: true,
						build: &bbpb.Build{
							SummaryMarkdown: ("some stuff\n\n" +
								"Error in build protocol: parsing Build: proto:"),
							Status: bbpb.Status_INFRA_FAILURE,
							Steps: []*bbpb.Step{
								{Name: "Parent", EndTime: now, Status: bbpb.Status_CANCELED,
									SummaryMarkdown: "step was never finalized; did the build crash?"},
								{Name: "Parent|Child", EndTime: now, Status: bbpb.Status_CANCELED,
									SummaryMarkdown: "step was never finalized; did the build crash?"},
								{
									Name: "Parent|Merge",
									Logs: []*bbpb.Log{{
										Name:    "$build.proto",
										Url:     "url://ns/Parent/Merge/build.proto",
										ViewUrl: "view://ns/Parent/Merge/build.proto",
									}},
								},
							},
							UpdateTime: now,
							EndTime:    now,
						},
					})
				})

				Convey(`handleNewData(nil)`, func() {
					bs.handleNewData(nil)
					wait()
					assertStateEqual(bs.GetFinal(), &buildState{
						closed: true,
						final:  true,
						build: &bbpb.Build{
							SummaryMarkdown: ("some stuff\n\nError in build protocol: " +
								"Expected a terminal build status, " +
								"got STATUS_UNSPECIFIED."),
							Status: bbpb.Status_INFRA_FAILURE,
							Steps: []*bbpb.Step{
								{Name: "Parent", EndTime: now, Status: bbpb.Status_CANCELED,
									SummaryMarkdown: "step was never finalized; did the build crash?"},
								{Name: "Parent|Child", EndTime: now, Status: bbpb.Status_CANCELED,
									SummaryMarkdown: "step was never finalized; did the build crash?"},
								{
									Name: "Parent|Merge",
									Logs: []*bbpb.Log{{
										Name:    "$build.proto",
										Url:     "url://ns/Parent/Merge/build.proto",
										ViewUrl: "view://ns/Parent/Merge/build.proto",
									}},
								},
							},
							EndTime:    now,
							UpdateTime: now,
						},
					})
				})
			})

			FocusConvey(`invalid build data`, func() {
				bs.handleNewData(&logpb.LogEntry{
					Content: &logpb.LogEntry_Datagram{Datagram: &logpb.Datagram{
						Data: []byte("narpnarp"),
					}},
				})
				wait()
				wait() // for final build
				assertStateEqual(bs.GetFinal(), &buildState{
					closed:  true,
					final:   true,
					invalid: true,
					build: &bbpb.Build{
						SummaryMarkdown: ("\n\nError in build protocol: parsing Build: proto:"),
						Status:          bbpb.Status_INFRA_FAILURE,
						UpdateTime:      now,
						EndTime:         now,
					},
				})

				Convey(`ignores further updates`, func() {
					bs.handleNewData(mkDgram(&bbpb.Build{SummaryMarkdown: "hi"}))
					assertStateEqual(bs.GetFinal(), &buildState{
						invalid: true,
						closed:  true,
						build: &bbpb.Build{
							SummaryMarkdown: ("\n\nError in build protocol: parsing Build: proto:"),
							Status:          bbpb.Status_INFRA_FAILURE,
							UpdateTime:      now,
							EndTime:         now,
						},
					})
				})
			})

			Convey(`bad log url`, func() {
				bs.handleNewData(mkDgram(&bbpb.Build{
					Steps: []*bbpb.Step{
						{
							Name: "hi",
							Logs: []*bbpb.Log{{
								Name: "log",
								Url:  "!!badnews!!",
							}},
						},
					},
				}))
				wait()
				wait() // for final build
				assertStateEqual(bs.GetFinal(), &buildState{
					closed:  true,
					final:   true,
					invalid: true,
					build: &bbpb.Build{
						SummaryMarkdown: ("\n\nError in build protocol: " +
							"step[\"hi\"].logs[\"log\"].Url = \"!!badnews!!\": " +
							"segment (at 0) must begin with alphanumeric character"),
						Steps: []*bbpb.Step{
							{
								Name:            "hi",
								Status:          bbpb.Status_INFRA_FAILURE,
								EndTime:         now,
								SummaryMarkdown: "bad log url: \"!!badnews!!\"",
								Logs: []*bbpb.Log{{
									Name: "log",
									Url:  "!!badnews!!",
								}},
							},
						},
						Status:     bbpb.Status_INFRA_FAILURE,
						UpdateTime: now,
						EndTime:    now,
					},
				})
			})
		})
	})
}
