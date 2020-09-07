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

package e2etest

import (
	"context"
	"testing"

	proto "github.com/golang/protobuf/proto"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/testing/prpctest"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

type service struct {
	R   *HelloReply
	err error
}

func (s *service) Greet(c context.Context, req *HelloRequest) (*HelloReply, error) {
	return s.R, s.err
}

func TestEndToEnd(t *testing.T) {
	Convey(`A client/server for the Greet service`, t, func() {
		c := context.Background()
		svc := service{}

		// Create a client/server for Greet service.
		ts := prpctest.Server{}
		RegisterHelloServer(&ts, &svc)
		ts.Start(c)
		defer ts.Close()

		prpcClient, err := ts.NewClient()
		if err != nil {
			panic(err)
		}
		client := NewHelloPRPCClient(prpcClient)

		Convey(`Can round-trip a hello message`, func() {
			svc.R = &HelloReply{Message: "sup"}

			resp, err := client.Greet(c, &HelloRequest{Name: "round-trip"})
			So(err, ShouldBeRPCOK)
			So(resp, ShouldResembleProto, svc.R)
		})

		Convey(`Can round-trip status details`, func() {
			detail := &errdetails.DebugInfo{Detail: "x"}

			s := status.New(codes.Internal, "internal")
			s, err := s.WithDetails(detail)
			So(err, ShouldBeNil)
			svc.err = s.Err()

			_, err = client.Greet(c, &HelloRequest{Name: "round-trip"})
			details := status.Convert(err).Details()
			So(details, ShouldResembleProto, []proto.Message{detail})
		})
	})
}
