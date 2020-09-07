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
	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

const invocationIDPattern = `[a-z][a-z0-9_\-:.]{0,99}`

var invocationIDRe = regexpf("^%s$", invocationIDPattern)
var invocationNameRe = regexpf("^invocations/(%s)$", invocationIDPattern)

// ValidateInvocationID returns a non-nil error if id is invalid.
func ValidateInvocationID(id string) error {
	return validateWithRe(invocationIDRe, id)
}

// ValidateInvocationName returns a non-nil error if name is invalid.
func ValidateInvocationName(name string) error {
	_, err := ParseInvocationName(name)
	return err
}

// ParseInvocationName extracts the invocation id.
func ParseInvocationName(name string) (id string, err error) {
	if name == "" {
		return "", unspecified()
	}

	m := invocationNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", doesNotMatch(invocationNameRe)
	}
	return m[1], nil
}

// InvocationName synthesizes an invocation name from an id.
// Does not validate id, use ValidateInvocationID.
func InvocationName(id string) string {
	return "invocations/" + id
}

// NormalizeInvocation converts inv to the canonical form.
func NormalizeInvocation(inv *pb.Invocation) {
	sortStringPairs(inv.Tags)
}
