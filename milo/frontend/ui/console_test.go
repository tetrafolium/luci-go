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

package ui

import (
	"testing"

	"github.com/tetrafolium/luci-go/milo/common/model"

	. "github.com/smartystreets/goconvey/convey"
)

type testBuilder struct {
	Builder  *BuilderRef
	Category []string
}

// Test helpers
func buildVerifyRoot(name string, builders []testBuilder, expectChildren int) *Category {
	root := NewCategory(name)
	for _, builder := range builders {
		root.AddBuilder(builder.Category, builder.Builder)
	}
	So(len(root.Children()), ShouldEqual, expectChildren)
	So(root.Name, ShouldEqual, name)
	return root
}

func verifyCategory(e ConsoleElement, expectChildren int, expectName string) *Category {
	cat := e.(*Category)
	So(len(cat.Children()), ShouldEqual, expectChildren)
	So(cat.Name, ShouldEqual, expectName)
	return cat
}

func TestCategory(t *testing.T) {
	Convey("Category structure", t, func() {
		// Test structures
		var emptycat []string
		cat1 := []string{"66__bbl"}
		cat2 := []string{"test.data"}
		deepcat := []string{"Hi", "Goodbye"}
		br1 := &BuilderRef{
			ID:        "test 1",
			ShortName: "t1",
			Build:     []*model.BuildSummary{},
		}
		br2 := &BuilderRef{
			ID:        "test 2",
			ShortName: "t2",
			Build:     []*model.BuildSummary{},
		}

		// Tests
		Convey("Root category", func() {
			buildVerifyRoot("root", []testBuilder{}, 0)
		})

		Convey("With builder", func() {
			root := buildVerifyRoot("_root_", []testBuilder{{br1, emptycat}}, 1)
			So(root.Children()[0].(*BuilderRef).ID, ShouldEqual, br1.ID)
		})

		Convey("With nested categories", func() {
			root := buildVerifyRoot("o_o", []testBuilder{{br1, deepcat}}, 1)
			child1 := verifyCategory(root.Children()[0], 1, deepcat[0])
			child2 := verifyCategory(child1.Children()[0], 1, deepcat[1])
			So(child2.Children()[0].(*BuilderRef).ID, ShouldEqual, br1.ID)
		})

		Convey("Multiple categories", func() {
			root := buildVerifyRoot("@_@", []testBuilder{
				{br1, cat1},
				{br2, cat2},
			}, 2)
			child1 := verifyCategory(root.Children()[0], 1, cat1[0])
			So(child1.Children()[0].(*BuilderRef).ID, ShouldEqual, br1.ID)
			child2 := verifyCategory(root.Children()[1], 1, cat2[0])
			So(child2.Children()[0].(*BuilderRef).ID, ShouldEqual, br2.ID)
		})

		Convey("Reusing existing categories", func() {
			root := buildVerifyRoot("rut", []testBuilder{
				{br1, cat1},
				{br2, cat1},
			}, 1)
			child := verifyCategory(root.Children()[0], 2, cat1[0])
			So(child.Children()[0].(*BuilderRef).ID, ShouldEqual, br1.ID)
			So(child.Children()[1].(*BuilderRef).ID, ShouldEqual, br2.ID)
		})

		Convey("Caches number of leaf nodes in a category", func() {
			root := buildVerifyRoot("rut", []testBuilder{{br1, cat1}}, 1)
			So(root.cachedNumLeafNodes, ShouldEqual, -1)
			So(root.Children()[0].(*Category).cachedNumLeafNodes, ShouldEqual, -1)
			So(root.NumLeafNodes(), ShouldEqual, 1)
			So(root.cachedNumLeafNodes, ShouldEqual, 1)
			So(root.Children()[0].(*Category).cachedNumLeafNodes, ShouldEqual, 1)

			root.AddBuilder(cat1, br2) // this must invalidate cached values
			So(root.cachedNumLeafNodes, ShouldEqual, -1)
			So(root.Children()[0].(*Category).cachedNumLeafNodes, ShouldEqual, -1)
			So(root.NumLeafNodes(), ShouldEqual, 2)
		})
	})
}
