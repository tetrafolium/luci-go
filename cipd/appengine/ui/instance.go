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

package ui

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/proto/google"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/server/router"
	"github.com/tetrafolium/luci-go/server/templates"

	api "github.com/tetrafolium/luci-go/cipd/api/cipd/v1"
	"github.com/tetrafolium/luci-go/cipd/appengine/impl"
	"github.com/tetrafolium/luci-go/cipd/common"
)

func instancePage(c *router.Context, pkg, ver string) error {
	pkg = strings.Trim(pkg, "/")
	if err := common.ValidatePackageName(pkg); err != nil {
		return status.Errorf(codes.InvalidArgument, "%s", err)
	}
	if err := common.ValidateInstanceVersion(ver); err != nil {
		return status.Errorf(codes.InvalidArgument, "%s", err)
	}

	// Resolve the version first (even if is already IID). This also checks ACLs
	// and verifies the instance exists.
	inst, err := impl.PublicRepo.ResolveVersion(c.Context, &api.ResolveVersionRequest{
		Package: pkg,
		Version: ver,
	})
	if err != nil {
		return err
	}

	// Do the rest in parallel. There can be only transient errors returned here,
	// so collect them all into single Internal error.
	var desc *api.DescribeInstanceResponse
	var url *api.ObjectURL
	err = parallel.FanOutIn(func(tasks chan<- func() error) {
		tasks <- func() (err error) {
			desc, err = impl.PublicRepo.DescribeInstance(c.Context, &api.DescribeInstanceRequest{
				Package:            inst.Package,
				Instance:           inst.Instance,
				DescribeRefs:       true,
				DescribeTags:       true,
				DescribeProcessors: true,
			})
			return
		}
		tasks <- func() (err error) {
			name := ""
			chunks := strings.Split(pkg, "/")
			if len(chunks) > 1 {
				name = fmt.Sprintf("%s-%s", chunks[len(chunks)-2], chunks[len(chunks)-1])
			} else {
				name = chunks[0]
			}
			url, err = impl.InternalCAS.GetObjectURL(c.Context, &api.GetObjectURLRequest{
				Object:           inst.Instance,
				DownloadFilename: name + ".zip",
			})
			return
		}
	})
	if err != nil {
		return status.Errorf(codes.Internal, "%s", err)
	}

	now := clock.Now(c.Context)
	templates.MustRender(c.Context, c.Writer, "pages/instance.html", map[string]interface{}{
		"Package":     pkg,
		"Version":     ver,
		"InstanceID":  common.ObjectRefToInstanceID(inst.Instance),
		"Breadcrumbs": breadcrumbs(pkg, ver),
		"HashAlgo":    inst.Instance.HashAlgo.String(),
		"HexDigest":   inst.Instance.HexDigest,
		"DownloadURL": url.SignedUrl,
		"Uploader":    strings.TrimPrefix(inst.RegisteredBy, "user:"),
		"Age":         humanize.RelTime(google.TimeFromProto(inst.RegisteredTs), now, "", ""),
		"Refs":        refsListing(desc.Refs, pkg, now),
		"Tags":        tagsListing(desc.Tags, pkg, now),
	})
	return nil
}
