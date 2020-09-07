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
	"regexp/syntax"
	"strings"

	"github.com/tetrafolium/luci-go/common/errors"

	pb "github.com/tetrafolium/luci-go/resultdb/proto/v1"
)

// testObjectPredicate is implemented by both *pb.TestResultPredicate
// and *pb.TestExonerationPredicate.
type testObjectPredicate interface {
	GetTestIdRegexp() string
	GetVariant() *pb.VariantPredicate
}

// validateTestObjectPredicate returns a non-nil error if p is determined to be
// invalid.
func validateTestObjectPredicate(p testObjectPredicate) error {
	if err := validateRegexp(p.GetTestIdRegexp()); err != nil {
		return errors.Annotate(err, "test_id_regexp").Err()
	}

	if p.GetVariant() != nil {
		if err := ValidateVariantPredicate(p.GetVariant()); err != nil {
			return errors.Annotate(err, "variant").Err()
		}
	}
	return nil
}

// ValidateTestResultPredicate returns a non-nil error if p is determined to be
// invalid.
func ValidateTestResultPredicate(p *pb.TestResultPredicate) error {
	if err := ValidateEnum(int32(p.GetExpectancy()), pb.TestResultPredicate_Expectancy_name); err != nil {
		return errors.Annotate(err, "expectancy").Err()
	}

	if p.GetExcludeExonerated() && p.GetExpectancy() == pb.TestResultPredicate_ALL {
		return errors.Reason("exclude_exonerated and expectancy=ALL are mutually exclusive").Err()
	}

	return validateTestObjectPredicate(p)
}

// ValidateTestExonerationPredicate returns a non-nil error if p is determined to be
// invalid.
func ValidateTestExonerationPredicate(p *pb.TestExonerationPredicate) error {
	return validateTestObjectPredicate(p)
}

// validateRegexp returns a non-nil error if re is an invalid regular
// expression.
func validateRegexp(re string) error {
	// Note: regexp.Compile uses syntax.Perl.
	if _, err := syntax.Parse(re, syntax.Perl); err != nil {
		return err
	}

	// Do not allow ^ and $ in the regexp, because we need to be able to prepend
	// a pattern to the user-supplied pattern.
	if strings.HasPrefix(re, "^") {
		return errors.Reason("must not start with ^; it is prepended automatically").Err()
	}
	if strings.HasSuffix(re, "$") {
		return errors.Reason("must not end with $; it is appended automatically").Err()
	}

	return nil
}

// ValidateVariantPredicate returns a non-nil error if p is determined to be
// invalid.
func ValidateVariantPredicate(p *pb.VariantPredicate) error {
	switch pr := p.Predicate.(type) {
	case *pb.VariantPredicate_Equals:
		return errors.Annotate(ValidateVariant(pr.Equals), "equals").Err()
	case *pb.VariantPredicate_Contains:
		return errors.Annotate(ValidateVariant(pr.Contains), "contains").Err()
	case nil:
		return unspecified()
	default:
		panic("impossible")
	}
}
