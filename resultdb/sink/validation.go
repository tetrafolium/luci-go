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
	"time"

	"github.com/tetrafolium/luci-go/common/errors"

	"github.com/tetrafolium/luci-go/resultdb/pbutil"
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
	sinkpb "github.com/tetrafolium/luci-go/resultdb/sink/proto/v1"
)

// validateTestResult returns a non-nil error if msg is invalid.
func validateTestResult(now time.Time, msg *sinkpb.TestResult) (err error) {
	ec := checker{&err}
	switch {
	case msg == nil:
		return unspecified()
	case ec.isErr(pbutil.ValidateTestID(msg.TestId), "test_id"):
	case ec.isErr(pbutil.ValidateResultID(msg.ResultId), "result_id"):
	// skip `Expected`
	case ec.isErr(pbutil.ValidateEnum(int32(msg.Status), pb.TestStatus_name), "status"):
	case ec.isErr(pbutil.ValidateSummaryHTML(msg.SummaryHtml), "summary_html"):
	case ec.isErr(pbutil.ValidateStartTimeWithDuration(now, msg.StartTime, msg.Duration), ""):
	case ec.isErr(pbutil.ValidateStringPairs(msg.Tags), "tags"):
	case ec.isErr(validateArtifacts(msg.Artifacts), "artifacts"):
	case msg.TestLocation != nil && ec.isErr(pbutil.ValidateTestLocation(msg.TestLocation), "test_location"):
	}
	return err
}

// validateArtifact returns a non-nil error if art is invalid.
func validateArtifact(art *sinkpb.Artifact) error {
	if art.GetFilePath() == "" && art.GetContents() == nil {
		return errors.Reason("body: either file_path or contents must be provided").Err()
	}
	// TODO(1017288) - validate content_type
	// skip `ContentType`
	return nil
}

// validateArtifacts returns a non-nil error if any element of arts is invalid.
func validateArtifacts(arts map[string]*sinkpb.Artifact) error {
	for id, art := range arts {
		if art == nil {
			return errors.Reason("%s: %s", id, unspecified()).Err()
		}
		if err := pbutil.ValidateArtifactID(id); err != nil {
			return errors.Annotate(err, "%s", id).Err()
		}
		if err := validateArtifact(art); err != nil {
			return errors.Annotate(err, "%s", id).Err()
		}
	}
	return nil
}

type checker struct {
	lastCheckedErr *error
}

// isErr returns true if err is nil. False, otherwise.
//
// It also stores err into lastCheckedErr. If err was not nil, it wraps err with
// errors.Annotate before storing it in lastErr.
func (c *checker) isErr(err error, format string, args ...interface{}) bool {
	if err == nil {
		return false
	}
	*c.lastCheckedErr = errors.Annotate(err, format, args...).Err()
	return true
}

func unspecified() error {
	return errors.Reason("unspecified").Err()
}
