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

package pbutil

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/tetrafolium/luci-go/common/errors"
)

const (
	artifactIDPattern = `[[:word:]]([[:print:]]{0,254}[[:word:]])?`
)

var (
	artifactIDRe                  = regexpf("^%s$", artifactIDPattern)
	invocationArtifactNamePattern = fmt.Sprintf("invocations/(%s)/artifacts/(.+)", invocationIDPattern)
	testResultArtifactNamePattern = fmt.Sprintf("invocations/(%s)/tests/([^/]+)/results/(%s)/artifacts/(.+)", invocationIDPattern, resultIDPattern)
	invocationArtifactNameRe      = regexpf("^%s$", invocationArtifactNamePattern)
	testResultArtifactNameRe      = regexpf("^%s$", testResultArtifactNamePattern)
	artifactNameRe                = regexpf("^%s|%s$", testResultArtifactNamePattern, invocationArtifactNamePattern)
)

// ValidateArtifactID returns a non-nil error if id is invalid.
func ValidateArtifactID(id string) error {
	return validateWithRe(artifactIDRe, id)
}

// ValidateArtifactName returns a non-nil error if name is invalid.
func ValidateArtifactName(name string) error {
	return validateWithRe(artifactNameRe, name)
}

// ParseArtifactName extracts the invocation ID, unescaped test id, result ID
// and artifact ID.
// The testID and resultID are empty if this is an invocation-level artifact.
func ParseArtifactName(name string) (invocationID, testID, resultID, artifactID string, err error) {
	if name == "" {
		err = unspecified()
		return
	}

	unescape := func(escaped string, re *regexp.Regexp) (string, error) {
		unescaped, err := url.PathUnescape(escaped)
		if err != nil {
			return "", errors.Annotate(err, "%q", escaped).Err()
		}

		if err := validateWithRe(re, unescaped); err != nil {
			return "", errors.Annotate(err, "%q", unescaped).Err()
		}

		return unescaped, nil
	}

	if m := invocationArtifactNameRe.FindStringSubmatch(name); m != nil {
		invocationID = m[1]
		artifactID, err = unescape(m[2], artifactIDRe)
		err = errors.Annotate(err, "artifact ID").Err()
		return
	}

	if m := testResultArtifactNameRe.FindStringSubmatch(name); m != nil {
		invocationID = m[1]
		if testID, err = unescape(m[2], testIDRe); err != nil {
			err = errors.Annotate(err, "test ID").Err()
			return
		}
		resultID = m[3]
		artifactID, err = unescape(m[4], artifactIDRe)
		err = errors.Annotate(err, "artifact ID").Err()
		return
	}

	err = doesNotMatch(artifactNameRe)
	return
}

// InvocationArtifactName synthesizes a name of an invocation-level artifact.
// Does not validate IDs, use ValidateInvocationID and ValidateArtifactID.
func InvocationArtifactName(invocationID, artifactID string) string {
	return fmt.Sprintf("invocations/%s/artifacts/%s", invocationID, url.PathEscape(artifactID))
}

// TestResultArtifactName synthesizes a name of an test-result-level artifact.
// Does not validate IDs, use ValidateInvocationID, ValidateTestID,
// ValidateResultID and ValidateArtifactID.
func TestResultArtifactName(invocationID, testID, resulID, artifactID string) string {
	return fmt.Sprintf("invocations/%s/tests/%s/results/%s/artifacts/%s", invocationID, url.PathEscape(testID), resulID, url.PathEscape(artifactID))
}
