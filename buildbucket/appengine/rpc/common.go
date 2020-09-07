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

package rpc

import (
	"context"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/common/data/stringset"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/logging"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	"github.com/tetrafolium/luci-go/server/auth"

	"github.com/tetrafolium/luci-go/buildbucket/appengine/internal/perm"
	"github.com/tetrafolium/luci-go/buildbucket/appengine/model"
	pb "github.com/tetrafolium/luci-go/buildbucket/proto"
)

type tagValidationMode int

const (
	TagNew tagValidationMode = iota
	TagAppend
)

const (
	buildSetMaxLength = 1024
	// summaryMardkdownMaxLength is the maximum size of Build.summary_markdown field in bytes.
	// Find more details at https://godoc.org/github.com/tetrafolium/luci-go/buildbucket/proto#Build
	summaryMarkdownMaxLength = 4 * 1000
)

var (
	blocklistKeyForAppend = stringset.NewFromSlice("build_address", "buildset", "builder")
	reservedKeys          = stringset.NewFromSlice("build_address")
	gitilesCommitRegex    = regexp.MustCompile(`^commit/gitiles/([^/]+)/(.+?)/\+/([a-f0-9]{40})$`)
	gerritCLRegex         = regexp.MustCompile(`^patch/gerrit/([^/]+)/(\d+)/(\d+)$`)
)

// commonPostlude converts an appstatus error to a gRPC error and logs it.
func commonPostlude(ctx context.Context, methodName string, rsp proto.Message, err error) error {
	return appstatus.GRPCifyAndLog(ctx, err)
}

// teeErr saves `err` in `keep` and then returns `err`
func teeErr(err error, keep *error) error {
	*keep = err
	return err
}

// logDetails logs debug information about the request.
func logDetails(ctx context.Context, methodName string, req proto.Message) (context.Context, error) {
	logging.Debugf(ctx, "%q called %q with request %s", auth.CurrentIdentity(ctx), methodName, proto.MarshalTextString(req))
	return ctx, nil
}

func validatePageSize(pageSize int32) error {
	if pageSize < 0 {
		return errors.Reason("page_size cannot be negative").Err()
	}
	return nil
}

// decodeCursor decodes a datastore cursor from a page token.
// The returned error may be appstatus-annotated.
func decodeCursor(ctx context.Context, pageToken string) (datastore.Cursor, error) {
	if pageToken == "" {
		return nil, nil
	}

	cursor, err := datastore.DecodeCursor(ctx, pageToken)
	if err != nil {
		return nil, appstatus.Attachf(err, codes.InvalidArgument, "bad cursor")
	}

	return cursor, nil
}

// getBuild returns the build with the given ID or NotFound appstatus if it is
// not found.
func getBuild(ctx context.Context, id int64) (*model.Build, error) {
	bld := &model.Build{ID: id}
	switch err := datastore.Get(ctx, bld); {
	case err == datastore.ErrNoSuchEntity:
		return nil, perm.NotFoundErr(ctx)
	case err != nil:
		return nil, errors.Annotate(err, "error fetching build with ID %d", id).Err()
	default:
		return bld, nil
	}
}

// validateTags validates build tags.
// tagValidationMode should be one of the enum - TagNew, TagAppend
// Note: Duplicate tags can pass the validation, which will be eventually deduplicated when storing into DB.
func validateTags(tags []*pb.StringPair, m tagValidationMode) error {
	if tags == nil {
		return nil
	}
	var k, v string
	var seenBuilderTagValue string
	var seenGitilesCommit string
	for _, tag := range tags {
		k = tag.Key
		v = tag.Value
		if strings.Contains(k, ":") {
			return errors.Reason(`tag key "%s" cannot have a colon`, k).Err()
		}
		if m == TagAppend && blocklistKeyForAppend.Has(k) {
			return errors.Reason(`tag key "%s" cannot be added to an existing build`, k).Err()
		}
		if k == "buildset" {
			if err := validateBuildSet(v); err != nil {
				return err
			}
			if gitilesCommitRegex.MatchString(v) {
				if seenGitilesCommit != "" && v != seenGitilesCommit {
					return errors.Reason(`tag "buildset:%s" conflicts with tag "buildset:%s"`, v, seenGitilesCommit).Err()
				}
				seenGitilesCommit = v
			}
		}
		if k == "builder" {
			if seenBuilderTagValue == "" {
				seenBuilderTagValue = v
			} else if v != seenBuilderTagValue {
				return errors.Reason(`tag "builder:%s" conflicts with tag "builder:%s"`, v, seenBuilderTagValue).Err()
			}
		}
		if reservedKeys.Has(k) {
			return errors.Reason(`tag "%s" is reserved`, k).Err()
		}
	}
	return nil
}

func validateBuildSet(bs string) error {
	if len("buildset:")+len(bs) > buildSetMaxLength {
		return errors.Reason("buildset tag is too long").Err()
	}

	// Verify that a buildset with a known prefix is well formed.
	if strings.HasPrefix(bs, "commit/gitiles/") {
		matches := gitilesCommitRegex.FindStringSubmatch(bs)
		if len(matches) == 0 {
			return errors.Reason(`does not match regex "%s"`, gitilesCommitRegex).Err()
		}
		project := matches[2]
		if strings.HasPrefix(project, "a/") {
			return errors.Reason(`gitiles project must not start with "a/"`).Err()
		}
		if strings.HasSuffix(project, ".git") {
			return errors.Reason(`gitiles project must not end with ".git"`).Err()
		}
	} else if strings.HasPrefix(bs, "patch/gerrit/") {
		if !gerritCLRegex.MatchString(bs) {
			return errors.Reason(`does not match regex "%s"`, gerritCLRegex).Err()
		}
	}
	return nil
}

func validateSummaryMarkdown(md string) error {
	if len(md) > summaryMarkdownMaxLength {
		return errors.Reason("too big to accept (%d > %d bytes)", len(md), summaryMarkdownMaxLength).Err()
	}
	return nil
}

// TODO(ddoman): move proto validator functions to protoutil.

// validateCommitWithRef checks if `cm` is a valid commit with a ref.
func validateCommitWithRef(cm *pb.GitilesCommit) error {
	if cm.GetRef() == "" {
		return errors.Reason(`ref is required`).Err()
	}
	return validateCommit(cm)
}

// validateCommit validates the given Gitiles commit.
func validateCommit(cm *pb.GitilesCommit) error {
	switch {
	case cm.GetHost() == "":
		return errors.Reason("host is required").Err()
	case cm.GetProject() == "":
		return errors.Reason("project is required").Err()
	case cm.GetId() != "":
		switch {
		case cm.Ref != "" || cm.Position != 0:
			return errors.Reason("id is mutually exclusive with (ref and position)").Err()
		case !sha1Regex.MatchString(cm.Id):
			return errors.Reason("id must match %q", sha1Regex).Err()
		}
	case cm.GetRef() != "":
		if !strings.HasPrefix(cm.Ref, "refs/") {
			return errors.Reason("ref must match refs/.*").Err()
		}
	default:
		return errors.Reason("one of id or ref is required").Err()
	}
	return nil
}
