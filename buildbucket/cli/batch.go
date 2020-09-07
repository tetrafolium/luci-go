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

package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/tetrafolium/luci-go/common/proto"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/common/cli"
	"github.com/tetrafolium/luci-go/common/errors"

	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
)

func cmdBatch(p Params) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `batch [flags]`,
		ShortDesc: "calls buildbucket.v2.Builds.Batch",
		LongDesc: doc(`
			Calls buildbucket.v2.Builds.Batch.

			Stdin must be buildbucket.v2.BatchRequest in JSON format.
			Stdout will be buildbucket.v2.BatchResponse in JSON format.
			Exits with code 1 if at least one sub-request fails.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &batchRun{}
			r.RegisterDefaultFlags(p)
			return r
		},
	}
}

type batchRun struct {
	baseCommandRun
	pb.BatchRequest
}

func (r *batchRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	if err := r.initClients(ctx); err != nil {
		return 0
	}

	if len(args) != 0 {
		return r.done(ctx, fmt.Errorf("unexpected argument"))
	}

	requestBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return r.done(ctx, errors.Annotate(err, "failed to read stdin").Err())
	}
	requestBytes, err = proto.FixFieldMasksBeforeUnmarshal(requestBytes, reflect.TypeOf(pb.BatchRequest{}))
	if err != nil {
		return r.done(ctx, errors.Annotate(err, "failed to parse BatchRequest from stdin").Err())
	}
	req := &pb.BatchRequest{}
	if err := jsonpb.Unmarshal(bytes.NewReader(requestBytes), req); err != nil {
		return r.done(ctx, errors.Annotate(err, "failed to parse BatchRequest from stdin").Err())
	}

	res, err := r.client.Batch(ctx, req)
	if err != nil {
		return r.done(ctx, err)
	}

	m := &jsonpb.Marshaler{}
	if err := m.Marshal(os.Stdout, res); err != nil {
		return r.done(ctx, err)
	}

	for _, r := range res.Responses {
		if _, ok := r.Response.(*pb.BatchResponse_Response_Error); ok {
			return 1
		}
	}
	return 0
}
