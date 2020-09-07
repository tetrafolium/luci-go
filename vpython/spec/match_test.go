// Copyright 2017 The LUCI Authors.
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

package spec

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tetrafolium/luci-go/vpython/api/vpython"

	. "github.com/smartystreets/goconvey/convey"
)

func mkTag(python, abi, platform string) *vpython.PEP425Tag {
	return &vpython.PEP425Tag{
		Python:   python,
		Abi:      abi,
		Platform: platform,
	}
}

func tagString(tags []*vpython.PEP425Tag) string {
	parts := make([]string, len(tags))
	for i, tag := range tags {
		parts[i] = tag.TagString()
	}
	return strings.Join(parts, ", ")
}

func TestPEP425Matches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		tags       []*vpython.PEP425Tag
		matches    []*vpython.PEP425Tag
		notMatches []*vpython.PEP425Tag
	}{
		{
			tags: nil,
			notMatches: []*vpython.PEP425Tag{
				mkTag("", "", ""),
				mkTag("cp27", "cp27mu", "manylinux1_x86_64"),
			},
		},
		{
			tags: []*vpython.PEP425Tag{
				mkTag("cp27", "cp27mu", "manylinux1_x86_64"),
				mkTag("py2", "cp27m", "macosx_10_9_universal"),
			},
			matches: []*vpython.PEP425Tag{
				mkTag("cp27", "", ""),
				mkTag("", "cp27mu", ""),
				mkTag("", "", "manylinux1_x86_64"),
				mkTag("py2", "", ""),
				mkTag("", "cp27m", ""),
				mkTag("", "", "macosx_10_9_universal"),
				mkTag("", "cp27mu", "manylinux1_x86_64"),
			},
			notMatches: []*vpython.PEP425Tag{
				mkTag("", "", ""),
				mkTag("cp27", "cp27mu", "win_amd64"),
				mkTag("cp27", "cp27mu", "macosx_10_9_universal"),
			},
		},
		{
			tags: []*vpython.PEP425Tag{
				mkTag("cp27", "cp27mu", ""),
			},
			matches: []*vpython.PEP425Tag{
				mkTag("cp27", "cp27mu", ""),
			},
			notMatches: []*vpython.PEP425Tag{
				mkTag("", "", ""),
				mkTag("cp27", "cp27mu", "otherArch"),
			},
		},
	}

	Convey(`Test cases for PEP425 tag matching`, t, func() {
		for _, tc := range testCases {
			Convey(fmt.Sprintf(`With system tags: %s`, tagString(tc.tags)), func() {
				for _, m := range tc.matches {
					Convey(fmt.Sprintf(`Tag matches: %s`, m.TagString()), func() {
						So(PEP425Matches(m, tc.tags), ShouldBeTrue)
					})
				}

				for _, m := range tc.notMatches {
					Convey(fmt.Sprintf(`Tag doesn't match: %s`, m.TagString()), func() {
						So(PEP425Matches(m, tc.tags), ShouldBeFalse)
					})
				}
			})
		}
	})
}

func TestPackageMatches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		tags         []*vpython.PEP425Tag
		matchPkgs    []*vpython.Spec_Package
		notMatchPkgs []*vpython.Spec_Package
	}{
		{
			tags: nil,
			matchPkgs: []*vpython.Spec_Package{
				{Name: "NoTags"},
			},
			notMatchPkgs: []*vpython.Spec_Package{
				{
					Name:     "EmptyMatch",
					MatchTag: []*vpython.PEP425Tag{mkTag("", "", "")},
				},
				{
					Name:     "MissingMatch",
					MatchTag: []*vpython.PEP425Tag{mkTag("cp27", "cp27mu", "manylinux1_x86_64")},
				},
			},
		},
		{
			tags: []*vpython.PEP425Tag{
				mkTag("cp27", "cp27mu", "manylinux1_x86_64"),
				mkTag("py2", "cp27m", "macosx_10_9_universal"),
			},
			matchPkgs: []*vpython.Spec_Package{
				{Name: "NoTags"},
				{
					Name:     "OneMatchingTag",
					MatchTag: []*vpython.PEP425Tag{mkTag("cp27", "", "")},
				},
				{
					Name: "MultipleMatchingTag",
					MatchTag: []*vpython.PEP425Tag{
						mkTag("cp27", "", ""),
						mkTag("", "cp27m", ""),
					},
				},
			},
			notMatchPkgs: []*vpython.Spec_Package{
				{
					Name:     "EmptyMatch",
					MatchTag: []*vpython.PEP425Tag{mkTag("", "", "")},
				},
				{
					Name:     "MissingMatch",
					MatchTag: []*vpython.PEP425Tag{mkTag("none", "none", "none")},
				},
				{
					Name:        "NotMatchTag",
					NotMatchTag: []*vpython.PEP425Tag{mkTag("", "cp27mu", "")},
				},
				{
					Name:        "NotMatchTagWithMatchTag",
					MatchTag:    []*vpython.PEP425Tag{mkTag("py2", "", "")},
					NotMatchTag: []*vpython.PEP425Tag{mkTag("", "cp27mu", "")},
				},
			},
		},
	}

	Convey(`Test cases for package tag matching`, t, func() {
		for _, tc := range testCases {
			Convey(fmt.Sprintf(`With system tags: %s`, tagString(tc.tags)), func() {
				for _, m := range tc.matchPkgs {
					Convey(fmt.Sprintf(`Package %q matches: %s`, m.Name, tagString(m.MatchTag)), func() {
						So(PackageMatches(m, tc.tags), ShouldBeTrue)
					})
				}

				for _, m := range tc.notMatchPkgs {
					Convey(fmt.Sprintf(`Package %q doesn't match: %s`, m.Name, tagString(m.MatchTag)), func() {
						So(PackageMatches(m, tc.tags), ShouldBeFalse)
					})
				}
			})
		}
	})
}
