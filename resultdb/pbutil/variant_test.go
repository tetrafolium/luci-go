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
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateVariant(t *testing.T) {
	t.Parallel()
	Convey(`TestValidateVariant`, t, func() {
		Convey(`empty`, func() {
			err := ValidateVariant(Variant())
			So(err, ShouldBeNil)
		})

		Convey(`invalid`, func() {
			err := ValidateVariant(Variant("1", "b"))
			So(err, ShouldErrLike, `key: does not match`)
		})
	})
}

func TestVariantUtils(t *testing.T) {
	t.Parallel()

	Convey(`Conversion to pair strings works`, t, func() {
		v := Variant(
			"key/with/part/k3", "v3",
			"k1", "v1",
			"key/k2", "v2",
		)
		So(VariantToStrings(v), ShouldResemble, []string{
			"k1:v1", "key/k2:v2", "key/with/part/k3:v3",
		})
	})

	Convey(`Conversion from pair strings works`, t, func() {
		Convey(`for valid pairs`, func() {
			vr, err := VariantFromStrings([]string{"k1:v1", "key/k2:v2", "key/with/part/k3:v3"})
			So(err, ShouldBeNil)
			So(vr, ShouldResembleProto, Variant(
				"k1", "v1",
				"key/k2", "v2",
				"key/with/part/k3", "v3",
			))
		})

		Convey(`for empty list returns nil`, func() {
			vr, err := VariantFromStrings([]string{})
			So(vr, ShouldBeNil)
			So(err, ShouldBeNil)
		})
	})

	Convey(`Key sorting works`, t, func() {
		vr := Variant(
			"k2", "v2",
			"k3", "v3",
			"k1", "v1",
		)
		So(SortedVariantKeys(vr), ShouldResemble, []string{"k1", "k2", "k3"})
	})
}
