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

package backend

import (
	"context"
	"encoding/base64"
	"sync"

	buildbucketpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/buildbucket/protoutil"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/sync/parallel"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/grpc/appstatus"
	milopb "github.com/tetrafolium/luci-go/milo/api/service/v1"
	"github.com/tetrafolium/luci-go/milo/common"
	"github.com/tetrafolium/luci-go/milo/common/model"
	"github.com/tetrafolium/luci-go/milo/git"
	"google.golang.org/protobuf/proto"
)

// QueryBlamelist implements milopb.MiloInternal service
func (s *MiloInternalService) QueryBlamelist(ctx context.Context, req *milopb.QueryBlamelistRequest) (*milopb.QueryBlamelistResponse, error) {
	startCommitID, err := prepareQueryBlamelistRequest(req)
	if err != nil {
		return nil, appstatus.BadRequest(err)
	}

	pageSize := adjustPageSize(req.PageSize)

	// Fetch one more commit to check whether there are more commits in the
	// blamelist.
	opts := &git.LogOptions{Limit: pageSize + 1, WithFiles: true}
	commits, err := git.Get(ctx).Log(ctx, req.GitilesCommit.Host, req.GitilesCommit.Project, startCommitID, opts)
	if err != nil {
		return nil, err
	}

	q := datastore.NewQuery("BuildSummary").Eq("BuilderID", common.LegacyBuilderIDString(req.Builder))
	blameLength := len(commits)
	m := sync.Mutex{}

	// Find the first other commit that has an associated build and update
	// blameLength.
	err = parallel.WorkPool(8, func(c chan<- func() error) {
		// Skip the first commit, it should always be included in the blamelist.
		for i, commit := range commits[1:] {
			newBlameLength := i + 1 // +1 since we skipped the first one.

			m.Lock()
			foundBuild := newBlameLength >= blameLength
			m.Unlock()

			// We have already found a build before this commit, no point looking
			// further.
			if foundBuild {
				break
			}

			curGC := &buildbucketpb.GitilesCommit{Host: req.GitilesCommit.Host, Project: req.GitilesCommit.Project, Id: commit.Id}
			c <- func() error {
				// Check whether this commit has an associated build.
				hasAssociatedBuild := false
				err := datastore.Run(ctx, q.Eq("BuildSet", protoutil.GitilesBuildSet(curGC)), func(build *model.BuildSummary) error {
					switch build.Summary.Status {
					case model.InfraFailure, model.Expired, model.Canceled:
						return nil
					default:
						hasAssociatedBuild = true
						return datastore.Stop
					}
				})
				if err != nil {
					return err
				}

				if hasAssociatedBuild {
					m.Lock()
					if newBlameLength < blameLength {
						blameLength = newBlameLength
					}
					m.Unlock()
				}
				return nil
			}
		}
	})
	if err != nil {
		return nil, err
	}

	// If there's more commits than needed, reserve the last commit as the pivot
	// for the next page.
	nextPageToken := ""
	if blameLength >= pageSize+1 {
		blameLength = pageSize
		nextPageToken, err = serializeQueryBlamelistPageToken(&milopb.QueryBlamelistPageToken{
			NextCommitId: commits[blameLength].Id,
		})
		if err != nil {
			return nil, err
		}
	}

	return &milopb.QueryBlamelistResponse{
		Commits:       commits[:blameLength],
		NextPageToken: nextPageToken,
	}, nil
}

// prepareQueryBlamelistRequest
//  * validates the request params.
//  * extracts start commit ID from page token or gittles Commit ID.
func prepareQueryBlamelistRequest(req *milopb.QueryBlamelistRequest) (startCommitID string, err error) {
	switch {
	case req.PageSize < 0:
		return "", errors.Reason("page_size can not be negative").Err()
	case req.GitilesCommit == nil:
		return "", errors.Reason("gitiles_commit is required").Err()
	case req.GitilesCommit.Host == "":
		return "", errors.Reason("gitiles_commit.host is required").Err()
	case req.GitilesCommit.Project == "":
		return "", errors.Reason("gitiles_commit.project is required").Err()
	case req.GitilesCommit.Id == "":
		return "", errors.Reason("gitiles_commit.id is required").Err()
	}

	if req.PageToken != "" {
		token, err := parseQueryBlamelistPageToken(req.PageToken)
		if err != nil {
			return "", errors.Annotate(err, "unable to parse page_token").Err()
		}
		return token.NextCommitId, nil
	}

	return req.GitilesCommit.Id, nil
}

func parseQueryBlamelistPageToken(tokenStr string) (token *milopb.QueryBlamelistPageToken, err error) {
	bytes, err := base64.StdEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, err
	}
	token = &milopb.QueryBlamelistPageToken{}
	err = proto.Unmarshal(bytes, token)
	return
}

func serializeQueryBlamelistPageToken(token *milopb.QueryBlamelistPageToken) (string, error) {
	bytes, err := proto.Marshal(token)
	return base64.StdEncoding.EncodeToString(bytes), err
}

const (
	pageSizeMax     = 1000
	pageSizeDefault = 100
)

// adjustPageSize takes the given requested pageSize and adjusts as necessary.
func adjustPageSize(pageSize int32) int {
	switch {
	case pageSize >= pageSizeMax:
		return pageSizeMax
	case pageSize > 0:
		return int(pageSize)
	default:
		return pageSizeDefault
	}
}
