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

package common

import (
	"context"
	"fmt"
	"html/template"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/auth/identity"
	buildbucketpb "github.com/tetrafolium/luci-go/buildbucket/proto"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/server/auth"
)

// MergeStrings merges multiple string slices together into a single slice,
// removing duplicates.
func MergeStrings(sss ...[]string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, ss := range sss {
		for _, s := range ss {
			if seen[s] {
				continue
			}
			seen[s] = true
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

// ObfuscateEmail converts a string containing email address email@address.com
// into email<junk>@address.com.
func ObfuscateEmail(email string) template.HTML {
	email = template.HTMLEscapeString(email)
	return template.HTML(strings.Replace(
		email, "@", "<span style=\"display:none\">ohnoyoudont</span>@", -1))
}

// ShortenEmail shortens Google emails.
func ShortenEmail(email string) string {
	return strings.Replace(email, "@google.com", "", -1)
}

// TagGRPC annotates some gRPC with Milo specific semantics, specifically:
// * Marks the error as Unauthorized if the user is not logged in,
// and the underlying error was a 403 or 404.
// * Otherwise, tag the error with the original error code.
func TagGRPC(c context.Context, err error) error {
	loggedIn := auth.CurrentIdentity(c) != identity.AnonymousIdentity
	code := grpcutil.Code(err)
	if code == codes.NotFound || code == codes.PermissionDenied {
		// Mask the errors, so they look the same.
		if loggedIn {
			return errors.Reason("not found").Tag(grpcutil.NotFoundTag).Err()
		}
		return errors.Reason("not logged in").Tag(grpcutil.UnauthenticatedTag).Err()
	}
	return grpcutil.ToGRPCErr(err)
}

// ParseIntFromForm parses an integer from a form.
func ParseIntFromForm(form url.Values, key string, base int, bitSize int) (int64, error) {
	input, err := ReadExactOneFromForm(form, key)
	if err != nil {
		return 0, err
	}
	ret, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return 0, errors.Annotate(err, "invalid %v; expected an integer; actual value: %v", key, input).Err()
	}
	return ret, nil
}

// ReadExactOneFromForm read a string from a form.
// There must be exactly one and non-empty entry of the given key in the form.
func ReadExactOneFromForm(form url.Values, key string) (string, error) {
	input := form[key]
	if len(input) != 1 || input[0] == "" {
		return "", fmt.Errorf("multiple or missing %v; actual value: %v", key, input)
	}
	return input[0], nil
}

// LegacyBuilderIDString returns a legacy string identifying the builder.
// It is used in the Milo datastore.
func LegacyBuilderIDString(bid *buildbucketpb.BuilderID) string {
	return fmt.Sprintf("buildbucket/luci.%s.%s/%s", bid.Project, bid.Bucket, bid.Builder)
}
