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

package recorder

import (
	"context"
	"time"

	"google.golang.org/genproto/googleapis/bytestream"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/server"
	"github.com/tetrafolium/luci-go/server/router"

	"github.com/tetrafolium/luci-go/resultdb/internal"
	"github.com/tetrafolium/luci-go/resultdb/internal/artifactcontent"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// recorderServer implements pb.RecorderServer.
//
// It does not return gRPC-native errors; use DecoratedRecorder with
// internal.CommonPostlude.
type recorderServer struct {
	*Options
}

// Options is recorder server configuration.
type Options struct {
	// Duration since invocation creation after which to delete expected test
	// results.
	ExpectedResultsExpiration time.Duration

	// ArtifactRBEInstance is the name of the RBE instance to use for artifact
	// storage. Example: "projects/luci-resultdb/instances/artifacts".
	ArtifactRBEInstance string
}

// InitServer initializes a recorder server.
func InitServer(srv *server.Server, opt Options) error {
	pb.RegisterRecorderServer(srv.PRPC, &pb.DecoratedRecorder{
		Service:  &recorderServer{Options: &opt},
		Postlude: internal.CommonPostlude,
	})

	return installArtifactCreationHandler(srv, &opt)
}

// installArtifactCreationHandler installs artifact creation handler.
func installArtifactCreationHandler(srv *server.Server, opt *Options) error {
	if opt.ArtifactRBEInstance == "" {
		return errors.Reason("opt.ArtifactRBEInstance is required").Err()
	}

	conn, err := artifactcontent.RBEConn(srv.Context)
	if err != nil {
		return err
	}

	bs := bytestream.NewByteStreamClient(conn)
	ach := &artifactCreationHandler{
		RBEInstance: opt.ArtifactRBEInstance,
		NewCASWriter: func(ctx context.Context) (bytestream.ByteStream_WriteClient, error) {
			return bs.Write(ctx)
		},
	}

	// Ideally we define more specific routes, but
	// "github.com/julienschmidt/httprouter" does not support routes over
	// unescaped paths: https://github.com/julienschmidt/httprouter/issues/208
	srv.Routes.PUT("invocations/*rest", router.MiddlewareChain{}, ach.Handle)
	return nil
}
