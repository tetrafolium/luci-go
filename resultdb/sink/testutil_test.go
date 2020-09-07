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

package sink

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc/metadata"

	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/grpc/prpc"

	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
	sinkpb "github.com/tetrafolium/luci-go/resultdb/sink/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func installTestListener(cfg *ServerConfig) (string, func() error) {
	l, err := net.Listen("tcp", "localhost:0")
	So(err, ShouldBeNil)
	cfg.testListener = l
	cfg.Address = fmt.Sprint("localhost:", l.Addr().(*net.TCPAddr).Port)

	// return the serving address
	return fmt.Sprint("localhost:", l.Addr().(*net.TCPAddr).Port), l.Close
}

func reportTestResults(ctx context.Context, host, authToken string, in *sinkpb.ReportTestResultsRequest) (*sinkpb.ReportTestResultsResponse, error) {
	sinkClient := sinkpb.NewSinkPRPCClient(&prpc.Client{
		Host:    host,
		Options: &prpc.Options{Insecure: true},
	})
	// install the auth token into the context, if present
	if authToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, AuthTokenKey, authTokenValue(authToken))
	}
	return sinkClient.ReportTestResults(ctx, in)
}

func testServerConfig(ctl *gomock.Controller, addr, tk string) ServerConfig {
	return ServerConfig{
		Address:          addr,
		AuthToken:        tk,
		ArtifactUploader: &ArtifactUploader{Client: &http.Client{}, Host: "example.org"},
		Recorder:         pb.NewMockRecorderClient(ctl),
		Invocation:       "invocations/u-foo-1587421194_893166206",
		invocationID:     "u-foo-1587421194_893166206",
		UpdateToken:      "UpdateToken-ABC",
	}
}

func testArtifactWithFile(writer func(f *os.File)) *sinkpb.Artifact {
	f, err := ioutil.TempFile("", "test-artifact")
	So(err, ShouldBeNil)
	defer f.Close()
	writer(f)

	return &sinkpb.Artifact{
		Body:        &sinkpb.Artifact_FilePath{FilePath: f.Name()},
		ContentType: "text/plain",
	}
}

func testArtifactWithContents(contents []byte) *sinkpb.Artifact {
	return &sinkpb.Artifact{
		Body:        &sinkpb.Artifact_Contents{contents},
		ContentType: "text/plain",
	}
}

// validTestResult returns a valid sinkpb.TestResult sample message.
func validTestResult() (*sinkpb.TestResult, func()) {
	now := testclock.TestRecentTimeUTC
	st, _ := ptypes.TimestampProto(now.Add(-2 * time.Minute))
	artf := testArtifactWithFile(func(f *os.File) {
		_, err := f.WriteString("a sample artifact")
		So(err, ShouldBeNil)
	})
	cleanup := func() { os.Remove(artf.GetFilePath()) }

	return &sinkpb.TestResult{
		TestId:      "this is testID",
		ResultId:    "result_id1",
		Expected:    true,
		Status:      pb.TestStatus_PASS,
		SummaryHtml: "HTML summary",
		StartTime:   st,
		Duration:    ptypes.DurationProto(time.Minute),
		Tags:        pbutil.StringPairs("k1", "v1"),
		Artifacts: map[string]*sinkpb.Artifact{
			"art1": artf,
		},
		TestLocation: &pb.TestLocation{
			FileName: "//a_test.cc",
		},
	}, cleanup
}

type BatchCreateTestResultsRequestMatcher struct {
	invocation string
	trs        []*pb.TestResult
}

func matchBatchCreateTestResultsRequest(inv string, trs ...*pb.TestResult) gomock.Matcher {
	return BatchCreateTestResultsRequestMatcher{inv, trs}
}

func (m BatchCreateTestResultsRequestMatcher) Matches(x interface{}) bool {
	req, ok := x.(*pb.BatchCreateTestResultsRequest)
	if !ok {
		return false
	}
	if req.Invocation != m.invocation {
		return false
	}

	for i, r := range req.Requests {
		if gomock.Eq(m.trs[i]).Matches(r.TestResult) == false {
			return false
		}
	}

	return true
}

func (m BatchCreateTestResultsRequestMatcher) String() string {
	ret := &strings.Builder{}
	fmt.Fprintf(ret, "has invocation:%q ", m.invocation)
	fmt.Fprintf(ret, "requests:<")

	for i, tr := range m.trs {
		if i > 0 {
			fmt.Fprintf(ret, ", ")
		}
		fmt.Fprintf(ret, "[%d]: %s", i, tr.String())
	}
	fmt.Fprintf(ret, ">")
	return ret.String()
}
