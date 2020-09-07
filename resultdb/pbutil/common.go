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
	"regexp"
	"time"

	"github.com/golang/protobuf/ptypes"
	durationpb "github.com/golang/protobuf/ptypes/duration"
	tspb "github.com/golang/protobuf/ptypes/timestamp"

	"github.com/tetrafolium/luci-go/common/errors"
)

var requestIDRe = regexp.MustCompile(`^[[:ascii:]]{0,36}$`)

func regexpf(patternFormat string, subpatterns ...interface{}) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(patternFormat, subpatterns...))
}

func doesNotMatch(r *regexp.Regexp) error {
	return errors.Reason("does not match %s", r).Err()
}

func unspecified() error {
	return errors.Reason("unspecified").Err()
}

func validateWithRe(re *regexp.Regexp, value string) error {
	if value == "" {
		return unspecified()
	}
	if !re.MatchString(value) {
		return doesNotMatch(re)
	}
	return nil
}

// MustTimestampProto converts a time.Time to a *tspb.Timestamp and panics
// on failure.
func MustTimestampProto(t time.Time) *tspb.Timestamp {
	ts, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return ts
}

// MustTimestamp converts a *tspb.Timestamp to a time.Time and panics
// on failure.
func MustTimestamp(ts *tspb.Timestamp) time.Time {
	t, err := ptypes.Timestamp(ts)
	if err != nil {
		panic(err)
	}
	return t
}

// ValidateRequestID returns a non-nil error if requestID is invalid.
// Returns nil if requestID is empty.
func ValidateRequestID(requestID string) error {
	if !requestIDRe.MatchString(requestID) {
		return doesNotMatch(requestIDRe)
	}
	return nil
}

// ValidateBatchRequestCount validates the number of requests in a batch
// request.
func ValidateBatchRequestCount(count int) error {
	const limit = 500
	if count > limit {
		return errors.Reason("the number of requests in the batch exceeds %d", limit).Err()
	}
	return nil
}

// ValidateMaxStaleness returns a non-nil error if maxStaleness is invalid.
func ValidateMaxStaleness(maxStaleness *durationpb.Duration) error {
	if maxStaleness == nil {
		return unspecified()
	}

	switch d, err := ptypes.Duration(maxStaleness); {
	case err != nil:
		return err
	case d < 0, d > 30*time.Minute:
		return errors.Reason("must between 0 and 30m, inclusive").Err()
	default:
		return nil
	}
}

// ValidateEnum returns a non-nil error if the value is not among valid values.
func ValidateEnum(value int32, validValues map[int32]string) error {
	if _, ok := validValues[value]; !ok {
		return errors.Reason("invalid value %d", value).Err()
	}
	return nil
}

// MustDuration converts a *durationpb.Duration to a time.Duration and panics
// on failure.
func MustDuration(du *durationpb.Duration) time.Duration {
	d, err := ptypes.Duration(du)
	if err != nil {
		panic(err)
	}
	return d
}
