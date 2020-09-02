// Copyright 2018 The LUCI Authors.
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

package lucicfg

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	luciproto "github.com/tetrafolium/luci-go/common/proto"
	"github.com/tetrafolium/luci-go/starlark/starlarkproto"

	_ "google.golang.org/protobuf/types/known/anypb"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/emptypb"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	_ "google.golang.org/protobuf/types/known/wrapperspb"

	_ "google.golang.org/genproto/googleapis/type/calendarperiod"
	_ "google.golang.org/genproto/googleapis/type/color"
	_ "google.golang.org/genproto/googleapis/type/date"
	_ "google.golang.org/genproto/googleapis/type/dayofweek"
	_ "google.golang.org/genproto/googleapis/type/expr"
	_ "google.golang.org/genproto/googleapis/type/fraction"
	_ "google.golang.org/genproto/googleapis/type/latlng"
	_ "google.golang.org/genproto/googleapis/type/money"
	_ "google.golang.org/genproto/googleapis/type/postaladdress"
	_ "google.golang.org/genproto/googleapis/type/quaternion"
	_ "google.golang.org/genproto/googleapis/type/timeofday"

	_ "github.com/tetrafolium/luci-go/buildbucket/proto"
	_ "github.com/tetrafolium/luci-go/common/proto/config"
	_ "github.com/tetrafolium/luci-go/common/proto/realms"
	_ "github.com/tetrafolium/luci-go/cv/api/config/v2"
	_ "github.com/tetrafolium/luci-go/logdog/api/config/svcconfig"
	_ "github.com/tetrafolium/luci-go/luci_notify/api/config"
	_ "github.com/tetrafolium/luci-go/milo/api/config"
	_ "github.com/tetrafolium/luci-go/resultdb/proto/v1"
	_ "github.com/tetrafolium/luci-go/scheduler/appengine/messages"
)

// Collection of built-in descriptor sets built from the protobuf registry
// embedded into the lucicfg binary.
var (
	wellKnownDescSet *starlarkproto.DescriptorSet
	googTypesDescSet *starlarkproto.DescriptorSet
	luciTypesDescSet *starlarkproto.DescriptorSet
)

// init initializes DescSet global vars.
//
// Uses the protobuf registry embedded into the binary. It visits imports in
// topological order, to make sure all cross-file references are correctly
// resolved. We assume there are no circular dependencies (if there are, they'll
// be caught by hanging unit tests).
func init() {
	visited := stringset.New(0)

	// Various well-known proto types (see also starlark/internal/descpb.star).
	wellKnownDescSet = builtinDescriptorSet("google/protobuf", []string{
		"google/protobuf/any.proto",
		"google/protobuf/descriptor.proto",
		"google/protobuf/duration.proto",
		"google/protobuf/empty.proto",
		"google/protobuf/field_mask.proto",
		"google/protobuf/struct.proto",
		"google/protobuf/timestamp.proto",
		"google/protobuf/wrappers.proto",
	}, visited)

	// Google API types (see also starlark/internal/descpb.star).
	googTypesDescSet = builtinDescriptorSet("google/type", []string{
		"google/type/calendar_period.proto",
		"google/type/color.proto",
		"google/type/date.proto",
		"google/type/dayofweek.proto",
		"google/type/expr.proto",
		"google/type/fraction.proto",
		"google/type/latlng.proto",
		"google/type/money.proto",
		"google/type/postal_address.proto",
		"google/type/quaternion.proto",
		"google/type/timeofday.proto",
	}, visited, wellKnownDescSet)

	// LUCI protos used by stdlib (see also starlark/internal/luci/descpb.star).
	luciTypesDescSet = builtinDescriptorSet("lucicfg/stdlib", []string{
		"github.com/tetrafolium/luci-go/buildbucket/proto/common.proto",
		"github.com/tetrafolium/luci-go/buildbucket/proto/project_config.proto",
		"github.com/tetrafolium/luci-go/common/proto/config/project_config.proto",
		"github.com/tetrafolium/luci-go/common/proto/realms/realms_config.proto",
		"github.com/tetrafolium/luci-go/cv/api/config/v2/cq.proto",
		"github.com/tetrafolium/luci-go/logdog/api/config/svcconfig/project.proto",
		"github.com/tetrafolium/luci-go/luci_notify/api/config/notify.proto",
		"github.com/tetrafolium/luci-go/milo/api/config/project.proto",
		"github.com/tetrafolium/luci-go/resultdb/proto/v1/invocation.proto",
		"github.com/tetrafolium/luci-go/resultdb/proto/v1/predicate.proto",
		"github.com/tetrafolium/luci-go/scheduler/appengine/messages/config.proto",
	}, visited, wellKnownDescSet, googTypesDescSet)
}

// builtinDescriptorSet assembles a *DescriptorSet from descriptors embedded
// into the binary in the protobuf registry.
//
// Visits 'files' and all their dependencies (not already visited per 'visited'
// set), adding them in topological order to the new DescriptorSet, updating
// 'visited' along the way.
//
// 'name' and 'deps' are passed verbatim to NewDescriptorSet(...).
//
// Panics on errors. Built-in descriptors can't be invalid.
func builtinDescriptorSet(name string, files []string, visited stringset.Set, deps ...*starlarkproto.DescriptorSet) *starlarkproto.DescriptorSet {
	var descs []*descriptorpb.FileDescriptorProto
	for _, f := range files {
		var err error
		if descs, err = visitRegistry(descs, f, visited); err != nil {
			panic(fmt.Errorf("%s: %s", f, err))
		}
	}
	ds, err := starlarkproto.NewDescriptorSet(name, descs, deps)
	if err != nil {
		panic(err)
	}
	return ds
}

// visitRegistry visits dependencies of 'path', and then 'path' itself.
//
// Appends discovered file descriptors to fds and returns it.
func visitRegistry(fds []*descriptorpb.FileDescriptorProto, path string, visited stringset.Set) ([]*descriptorpb.FileDescriptorProto, error) {
	if !visited.Add(path) {
		return fds, nil // visited it already
	}
	fd, err := protoregistry.GlobalFiles.FindFileByPath(path)
	if err != nil {
		return fds, err
	}
	fdp := protodesc.ToFileDescriptorProto(fd)
	for _, d := range fdp.GetDependency() {
		if fds, err = visitRegistry(fds, d, visited); err != nil {
			return fds, fmt.Errorf("%s: %s", d, err)
		}
	}
	return append(fds, fdp), nil
}

// protoMessageDoc returns the message name and a link to its schema doc.
//
// Extracts it from `option (lucicfg.file_metadata) = {...}` embedded
// into the file descriptor proto.
//
// If there's no documentation, returns two empty strings.
func protoMessageDoc(msg *starlarkproto.Message) (name, doc string) {
	fd := msg.MessageType().Descriptor().ParentFile()
	if fd == nil {
		return "", ""
	}
	opts := fd.Options().(*descriptorpb.FileOptions)
	if opts != nil && proto.HasExtension(opts, luciproto.E_FileMetadata) {
		meta := proto.GetExtension(opts, luciproto.E_FileMetadata).(*luciproto.Metadata)
		if meta.GetDocUrl() != "" {
			return string(msg.MessageType().Descriptor().Name()), meta.GetDocUrl()
		}
	}
	return "", "" // not a public proto
}
