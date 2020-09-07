// Copyright 2017 The LUCI Authors.
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

package buildsource

import (
	"strings"

	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"

	"github.com/tetrafolium/luci-go/milo/common/model"
)

// BuilderID is the universal ID of a builder, and has the form:
//   buildbucket/bucket/builder
type BuilderID string

// Split breaks the BuilderID into pieces.
//   - backend is always 'buildbucket'
//   - backendGroup is either the bucket or master name
//   - builderName is the builder name.
//
// Returns an error if the BuilderID is malformed (wrong # slashes) or if any of
// the pieces are empty.
func (b BuilderID) Split() (backend, backendGroup, builderName string, err error) {
	toks := strings.SplitN(string(b), "/", 3)
	if len(toks) != 3 {
		err = errors.Reason("bad BuilderID: not enough tokens: %q", b).
			Tag(grpcutil.InvalidArgumentTag).Err()
		return
	}
	backend, backendGroup, builderName = toks[0], toks[1], toks[2]
	switch {
	case backend != "buildbucket":
		err = errors.Reason("bad BuilderID: unknown backend %q", backend).
			Tag(grpcutil.InvalidArgumentTag).Err()
	case backendGroup == "":
		err = errors.New("bad BuilderID: empty backendGroup", grpcutil.InvalidArgumentTag)
	case builderName == "":
		err = errors.New("bad BuilderID: empty builderName", grpcutil.InvalidArgumentTag)
	}
	return
}

// SelfLink returns LUCI URL of the builder.
func (b BuilderID) SelfLink(project string) string {
	return model.BuilderIDLink(string(b), project)
}
