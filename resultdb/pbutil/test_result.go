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

package pbutil

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes"
	dpb "github.com/golang/protobuf/ptypes/duration"
	tspb "github.com/golang/protobuf/ptypes/timestamp"

	"github.com/tetrafolium/luci-go/common/errors"

	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

const (
	resultIDPattern   = `[a-z0-9\-_.]{1,32}`
	maxLenSummaryHTML = 4 * 1024
)

var (
	resultIDRe       = regexpf("^%s$", resultIDPattern)
	testIDRe         = regexp.MustCompile(`^[[:print:]]{1,512}$`)
	testResultNameRe = regexpf(
		"^invocations/(%s)/tests/([^/]+)/results/(%s)$", invocationIDPattern,
		resultIDPattern,
	)
)

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

// ValidateTestID returns a non-nil error if testID is invalid.
func ValidateTestID(testID string) error {
	return validateWithRe(testIDRe, testID)
}

// ValidateTestResultID returns a non-nil error if resultID is invalid.
func ValidateResultID(resultID string) error {
	return validateWithRe(resultIDRe, resultID)
}

// ValidateTestResultName returns a non-nil error if name is invalid.
func ValidateTestResultName(name string) error {
	_, _, _, err := ParseTestResultName(name)
	return err
}

// ValidateSummaryHTML returns a non-nil error if summary is invalid.
func ValidateSummaryHTML(summary string) error {
	if len(summary) > maxLenSummaryHTML {
		return errors.Reason("exceeds the maximum size of %d bytes", maxLenSummaryHTML).Err()
	}
	return nil
}

// ValidateStartTimeWithDuration returns a non-nil error if startTime and duration are invalid.
func ValidateStartTimeWithDuration(now time.Time, startTime *tspb.Timestamp, duration *dpb.Duration) error {
	t, err := ptypes.Timestamp(startTime)
	if startTime != nil && err != nil {
		return err
	}

	d, err := ptypes.Duration(duration)
	if duration != nil && err != nil {
		return err
	}

	switch {
	case startTime != nil && now.Before(t):
		return errors.Reason("start_time: cannot be in the future").Err()
	case duration != nil && d < 0:
		return errors.Reason("duration: is < 0").Err()
	case startTime != nil && duration != nil && now.Before(t.Add(d)):
		return errors.Reason("start_time + duration: cannot be in the future").Err()
	}
	return nil
}

// ValidateTestResult returns a non-nil error if msg is invalid.
func ValidateTestResult(now time.Time, msg *pb.TestResult) (err error) {
	ec := checker{&err}
	switch {
	case msg == nil:
		return unspecified()
	// skip `Name`
	case ec.isErr(ValidateTestID(msg.TestId), "test_id"):
	case ec.isErr(ValidateResultID(msg.ResultId), "result_id"):
	case ec.isErr(ValidateVariant(msg.Variant), "variant"):
	// skip `Expected`
	case ec.isErr(ValidateEnum(int32(msg.Status), pb.TestStatus_name), "status"):
	case ec.isErr(ValidateSummaryHTML(msg.SummaryHtml), "summary_html"):
	case ec.isErr(ValidateStartTimeWithDuration(now, msg.StartTime, msg.Duration), ""):
	case ec.isErr(ValidateStringPairs(msg.Tags), "tags"):
	case msg.TestLocation != nil && ec.isErr(ValidateTestLocation(msg.TestLocation), "test_location"):
	}
	return err
}

// ValidateTestLocation returns a non-nil error if loc is invalid.
func ValidateTestLocation(loc *pb.TestLocation) error {
	switch {
	case loc.FileName == "":
		return errors.Reason("file_name: unspecified").Err()
	case len(loc.FileName) > 512:
		return errors.Reason("file_name: length exceeds 512").Err()
	case loc.Line < 0:
		return errors.Reason("line: must not be negative").Err()
	}

	return nil
}

// ParseTestResultName extracts the invocation ID, unescaped test id, and
// result ID.
func ParseTestResultName(name string) (invID, testID, resultID string, err error) {
	if name == "" {
		err = unspecified()
		return
	}

	m := testResultNameRe.FindStringSubmatch(name)
	if m == nil {
		err = doesNotMatch(testResultNameRe)
		return
	}
	unescapedTestID, err := url.PathUnescape(m[2])
	if err != nil {
		err = errors.Annotate(err, "test id %q", m[2]).Err()
		return
	}

	if ve := validateWithRe(testIDRe, unescapedTestID); ve != nil {
		err = errors.Annotate(ve, "test id %q", unescapedTestID).Err()
		return
	}
	return m[1], unescapedTestID, m[3], nil
}

// TestResultName synthesizes a test result name from its parts.
// Does not validate parts; use ValidateTestResultName.
func TestResultName(invID, testID, resultID string) string {
	return fmt.Sprintf(
		"invocations/%s/tests/%s/results/%s", invID, url.PathEscape(testID), resultID)
}

// NormalizeTestResult converts inv to the canonical form.
func NormalizeTestResult(tr *pb.TestResult) {
	sortStringPairs(tr.Tags)
}

// NormalizeTestResultSlice converts trs to the canonical form.
func NormalizeTestResultSlice(trs []*pb.TestResult) {
	for _, tr := range trs {
		NormalizeTestResult(tr)
	}
	sort.Slice(trs, func(i, j int) bool {
		a := trs[i]
		b := trs[j]
		if a.TestId != b.TestId {
			return a.TestId < b.TestId
		}
		return a.Name < b.Name
	})
}
