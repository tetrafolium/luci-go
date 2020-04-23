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

package deriver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"

	swarmingAPI "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/isolatedclient/isolatedfake"

	"go.chromium.org/luci/resultdb/internal"
	"go.chromium.org/luci/resultdb/internal/services/deriver/chromium"
	"go.chromium.org/luci/resultdb/internal/services/deriver/chromium/formats"
	"go.chromium.org/luci/resultdb/internal/span"
	"go.chromium.org/luci/resultdb/internal/tasks"
	"go.chromium.org/luci/resultdb/internal/testutil"
	"go.chromium.org/luci/resultdb/pbutil"
	pb "go.chromium.org/luci/resultdb/proto/rpc/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestValidateDeriveChromiumInvocationRequest(t *testing.T) {
	Convey(`TestValidateDeriveChromiumInvocationRequest`, t, func() {
		Convey(`Valid`, func() {
			req := &pb.DeriveChromiumInvocationRequest{
				SwarmingTask: &pb.DeriveChromiumInvocationRequest_SwarmingTask{
					Hostname: "swarming-host.appspot.com",
					Id:       "beeff00d",
				},
			}
			So(validateDeriveChromiumInvocationRequest(req), ShouldBeNil)
		})

		Convey(`Invalid swarming_task`, func() {
			Convey(`missing`, func() {
				req := &pb.DeriveChromiumInvocationRequest{}
				So(validateDeriveChromiumInvocationRequest(req), ShouldErrLike, "swarming_task missing")
			})

			Convey(`missing hostname`, func() {
				req := &pb.DeriveChromiumInvocationRequest{
					SwarmingTask: &pb.DeriveChromiumInvocationRequest_SwarmingTask{
						Id: "beeff00d",
					},
				}
				So(validateDeriveChromiumInvocationRequest(req), ShouldErrLike, "swarming_task.hostname missing")
			})

			Convey(`bad hostname`, func() {
				req := &pb.DeriveChromiumInvocationRequest{
					SwarmingTask: &pb.DeriveChromiumInvocationRequest_SwarmingTask{
						Hostname: "https://swarming-host.appspot.com",
						Id:       "beeff00d",
					},
				}
				So(validateDeriveChromiumInvocationRequest(req), ShouldErrLike,
					"swarming_task.hostname should not have prefix")
			})

			Convey(`missing id`, func() {
				req := &pb.DeriveChromiumInvocationRequest{
					SwarmingTask: &pb.DeriveChromiumInvocationRequest_SwarmingTask{
						Hostname: "swarming-host.appspot.com",
					},
				}
				So(validateDeriveChromiumInvocationRequest(req), ShouldErrLike, "swarming_task.id missing")
			})
		})
	})
}

func TestDeriveChromiumInvocation(t *testing.T) {
	Convey(`TestDeriveChromiumInvocation`, t, func(c C) {
		ctx := testutil.SpannerTestContext(t)

		testutil.MustApply(ctx, testutil.InsertInvocation("inserted", pb.Invocation_FINALIZED, nil))

		Convey(`calling to shouldWriteInvocation works`, func() {
			Convey(`if we already have the invocation written`, func() {
				doWrite, err := shouldWriteInvocation(ctx, span.Client(ctx).Single(), "inserted")
				So(err, ShouldBeNil)
				So(doWrite, ShouldBeFalse)
			})

			Convey(`if we don't yet have the invocation written`, func() {
				doWrite, err := shouldWriteInvocation(ctx, span.Client(ctx).Single(), "another")
				So(err, ShouldBeNil)
				So(doWrite, ShouldBeTrue)
			})
		})

		ctx = internal.WithHTTPClient(ctx, &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		})

		// Set up fake isolatedserver.
		isoFake := isolatedfake.New()
		isoServer := httptest.NewServer(isoFake)
		defer isoServer.Close()

		// Inject isolated objects.
		testJSON := []byte(`
		{
			"version": 3,
			"interrupted": false,
			"path_delimiter": "/",
			"tests": {
				"c1": {
					"t1.html": {
						"actual": "FAIL PASS",
						"expected": "PASS"
					}
				},
				"c2": {
					"t3.html": {
						"actual": "FAIL",
						"expected": "PASS",
						"artifacts": {
							"log": ["relative/path/to/log"]
						}
					}
				}
			}
		}`)
		size := int64(len(testJSON))
		fileDigest := isoFake.Inject("ns", testJSON)
		isoOut := isolated.Isolated{
			Files: map[string]isolated.File{"output.json": {Digest: fileDigest, Size: &size}},
		}
		isoOutBytes, err := json.Marshal(isoOut)
		So(err, ShouldBeNil)
		outputsDigest := isoFake.Inject("ns", isoOutBytes)

		// Set up fake swarming service.
		swarmingFake := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := &swarmingAPI.SwarmingRpcsTaskResult{
				CreatedTs: "2019-10-14T13:49:16.01",
				Tags: []string{
					"bucket:bkt",
					"buildername:blder",
					"test_suite:foo_unittests",
					"ninja_target://tests:tests",
				},
				State:       "COMPLETED",
				CompletedTs: "2019-10-14T14:49:16.01",
			}

			swarmingAPIEndpoint := "_ah/api/swarming/v1/"
			switch r.URL.Path {
			case fmt.Sprintf("/%stask/completed-task/result", swarmingAPIEndpoint):
				resp.TaskId = "completed-task"
				resp.RunId = "completed-task"
				resp.OutputsRef = &swarmingAPI.SwarmingRpcsFilesRef{
					Isolatedserver: isoServer.URL,
					Namespace:      "ns",
					Isolated:       string(outputsDigest),
				}
			case fmt.Sprintf("/%stask/origin/result", swarmingAPIEndpoint):
				resp.TaskId = "origin"
				resp.RunId = "origin"
				resp.OutputsRef = &swarmingAPI.SwarmingRpcsFilesRef{
					Isolatedserver: isoServer.URL,
					Namespace:      "ns",
					Isolated:       string(outputsDigest),
				}
			case fmt.Sprintf("/%stask/deduped/result", swarmingAPIEndpoint):
				resp.TaskId = "deduped"
				resp.RunId = "deduped"
				resp.DedupedFrom = "origin"
			}
			err := json.NewEncoder(w).Encode(resp)
			c.So(err, ShouldBeNil)
		}))
		defer swarmingFake.Close()

		// Define base request we'll be using.
		swarmingHostname := strings.TrimPrefix(swarmingFake.URL, "https://")
		req := &pb.DeriveChromiumInvocationRequest{
			SwarmingTask: &pb.DeriveChromiumInvocationRequest_SwarmingTask{
				Hostname: swarmingHostname,
			},
		}

		deriver := newTestDeriverServer()
		deriver.InvBQTable = &pb.BigQueryExport{
			Project:     "project",
			Dataset:     "dataset",
			Table:       "table",
			TestResults: &pb.BigQueryExport_TestResults{},
		}

		Convey(`inserts a new invocation`, func() {
			req.SwarmingTask.Id = "completed-task"
			inv, err := deriver.DeriveChromiumInvocation(ctx, req)
			So(err, ShouldBeNil)

			So(inv, ShouldResembleProto, &pb.Invocation{
				Name:                inv.Name, // inv.Name is non-determinisic in this test
				State:               pb.Invocation_FINALIZED,
				CreateTime:          &tspb.Timestamp{Seconds: 1571060956, Nanos: 1e7},
				Tags:                pbutil.StringPairs(formats.OriginalFormatTagKey, formats.FormatJTR),
				FinalizeTime:        &tspb.Timestamp{Seconds: 1571064556, Nanos: 1e7},
				Deadline:            &tspb.Timestamp{Seconds: 1571064556, Nanos: 1e7},
				IncludedInvocations: []string{inv.Name + "::batch::0"},
				ProducerResource:    fmt.Sprintf("//%s/tasks/completed-task", swarmingHostname),
			})

			// Assert we wrote correct test results.
			txn := span.Client(ctx).ReadOnlyTransaction()
			defer txn.Close()

			invIDs, err := span.ReadReachableInvocations(ctx, txn, 100, span.NewInvocationIDSet(span.MustParseInvocationName(inv.Name)))
			So(err, ShouldBeNil)
			trs, _, err := span.QueryTestResults(ctx, txn, span.TestResultQuery{
				InvocationIDs: invIDs,
				PageSize:      100,
			})
			So(err, ShouldBeNil)
			So(trs, ShouldHaveLength, 3)
			So(trs[0].TestId, ShouldEqual, "ninja://tests:tests/c1/t1.html")
			So(trs[0].Status, ShouldEqual, pb.TestStatus_FAIL)
			So(trs[0].Variant, ShouldResembleProto, pbutil.Variant(
				"bucket", "bkt",
				"builder", "blder",
				"test_suite", "foo_unittests",
			))

			So(trs[1].TestId, ShouldEqual, "ninja://tests:tests/c1/t1.html")
			So(trs[1].Status, ShouldEqual, pb.TestStatus_PASS)

			So(trs[2].TestId, ShouldEqual, "ninja://tests:tests/c2/t3.html")
			So(trs[2].Status, ShouldEqual, pb.TestStatus_FAIL)

			// Read InvocationTask to confirm it's added.
			taskKey := tasks.BQExport.Key(fmt.Sprintf("%s:0", span.MustParseInvocationName(inv.Name).RowID()))
			var payload []byte
			testutil.MustReadRow(ctx, "InvocationTasks", taskKey, map[string]interface{}{
				"Payload": &payload,
			})
			bqExports := &pb.BigQueryExport{}
			err = proto.Unmarshal(payload, bqExports)
			So(err, ShouldBeNil)
			So(bqExports, ShouldResembleProto, deriver.InvBQTable)
		})

		Convey(`inserts a deduped invocation`, func() {
			req.SwarmingTask.Id = "deduped"
			inv, err := deriver.DeriveChromiumInvocation(ctx, req)
			So(err, ShouldBeNil)
			So(len(inv.IncludedInvocations), ShouldEqual, 1)
			So(inv.IncludedInvocations[0], ShouldContainSubstring, "origin")

			// Assert we wrote correct test results.
			txn := span.Client(ctx).ReadOnlyTransaction()
			defer txn.Close()

			invIDs, err := span.ReadReachableInvocations(ctx, txn, 100, span.NewInvocationIDSet(span.MustParseInvocationName(inv.Name)))
			So(err, ShouldBeNil)
			So(len(invIDs), ShouldEqual, 3)

			trs, _, err := span.QueryTestResults(ctx, txn, span.TestResultQuery{
				InvocationIDs: invIDs,
				PageSize:      100,
			})
			So(err, ShouldBeNil)
			So(trs, ShouldHaveLength, 3)
			So(trs[0].TestId, ShouldEqual, "ninja://tests:tests/c1/t1.html")
			So(trs[0].Status, ShouldEqual, pb.TestStatus_FAIL)
			So(trs[0].Variant, ShouldResembleProto, pbutil.Variant(
				"bucket", "bkt",
				"builder", "blder",
				"test_suite", "foo_unittests",
			))

			So(trs[1].TestId, ShouldEqual, "ninja://tests:tests/c1/t1.html")
			So(trs[1].Status, ShouldEqual, pb.TestStatus_PASS)

			So(trs[2].TestId, ShouldEqual, "ninja://tests:tests/c2/t3.html")
			So(trs[2].Status, ShouldEqual, pb.TestStatus_FAIL)

			// Read InvocationTask to confirm it's added for origin task.
			taskKey := tasks.BQExport.Key(fmt.Sprintf("%s:0", span.MustParseInvocationName(inv.IncludedInvocations[0]).RowID()))
			var payload []byte
			testutil.MustReadRow(ctx, "InvocationTasks", taskKey, map[string]interface{}{
				"Payload": &payload,
			})
			bqExports := &pb.BigQueryExport{}
			err = proto.Unmarshal(payload, bqExports)
			So(err, ShouldBeNil)
			So(bqExports, ShouldResembleProto, deriver.InvBQTable)
		})
	})
}

func TestBatchInsertTestResults(t *testing.T) {
	Convey(`TestBatchInsertTestResults`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		startTS := ptypes.TimestampNow()

		inv := &pb.Invocation{
			State:        pb.Invocation_FINALIZED,
			CreateTime:   startTS,
			FinalizeTime: startTS,
			Deadline:     startTS,
		}
		results := []*chromium.TestResult{
			{TestResult: &pb.TestResult{TestId: "Foo.DoBar", ResultId: "0", Status: pb.TestStatus_PASS}},
			{TestResult: &pb.TestResult{TestId: "Foo.DoBar", ResultId: "1", Status: pb.TestStatus_FAIL}},
			{TestResult: &pb.TestResult{TestId: "Foo.DoBar", ResultId: "2", Status: pb.TestStatus_CRASH}},
		}
		deriver := newTestDeriverServer()

		checkBatches := func(baseID span.InvocationID, actualInclusions span.InvocationIDSet, expectedBatches [][]*chromium.TestResult) {
			// Check included Invocations.
			expectedInclusions := make(span.InvocationIDSet, len(expectedBatches))
			for i := range expectedBatches {
				expectedInclusions.Add(batchInvocationID(baseID, i))
			}
			So(actualInclusions, ShouldResemble, expectedInclusions)

			txn := span.Client(ctx).ReadOnlyTransaction()
			defer txn.Close()

			// Check that the TestResults are batched as expected.
			for i, expectedBatch := range expectedBatches {
				actualResults, _, err := span.QueryTestResults(ctx, txn, span.TestResultQuery{
					InvocationIDs: span.NewInvocationIDSet(batchInvocationID(baseID, i)),
					PageSize:      100,
				})
				So(err, ShouldBeNil)
				So(actualResults, ShouldHaveLength, len(expectedBatch))

				for k, expectedResult := range expectedBatch {
					So(actualResults[k].TestId, ShouldEqual, expectedResult.TestId)
					So(actualResults[k].ResultId, ShouldEqual, strconv.Itoa(k))
					So(actualResults[k].Status, ShouldEqual, expectedResult.Status)
				}
			}
		}

		Convey(`for one batch`, func() {
			baseID := span.InvocationID("one_batch")
			inv.Name = baseID.Name()
			actualInclusions, err := deriver.batchInsertTestResults(ctx, inv, results, 5)
			So(err, ShouldBeNil)
			checkBatches(baseID, actualInclusions, [][]*chromium.TestResult{results})
		})

		Convey(`for multiple batches`, func() {
			baseID := span.InvocationID("multiple_batches")
			inv.Name = baseID.Name()
			actualInclusions, err := deriver.batchInsertTestResults(ctx, inv, results, 2)
			So(err, ShouldBeNil)
			checkBatches(baseID, actualInclusions, [][]*chromium.TestResult{results[:2], results[2:]})

			Convey(`with batch size a factor of number of TestResults`, func() {
				baseID := span.InvocationID("multiple_batches_factor")
				inv.Name = baseID.Name()
				actualInclusions, err := deriver.batchInsertTestResults(ctx, inv, results, 1)
				So(err, ShouldBeNil)
				checkBatches(baseID, actualInclusions,
					[][]*chromium.TestResult{results[:1], results[1:2], results[2:]})
			})
		})
	})
}
