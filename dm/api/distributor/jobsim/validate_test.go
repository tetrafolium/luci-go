// Copyright 2016 The LUCI Authors.
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

package jobsim

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestNormalization(t *testing.T) {
	positive := []struct {
		Name string

		Phrase string
		Parsed *Phrase
	}{
		{
			Name:   "FailureStage",
			Phrase: `{"stages": [{"failure": {"chance": 0.75}}]}`,
			Parsed: &Phrase{
				Stages: []*Stage{{StageType: &Stage_Failure{&FailureStage{Chance: .75}}}},
			},
		},
	}

	bad := []struct {
		Name string

		Phrase   string
		ParseErr interface{}
		NormErr  interface{}
	}{
		{
			Name:    "FailureStage",
			Phrase:  `{"stages": [{"failure": {"chance": -2}}]}`,
			NormErr: "too small FailureStage chance",
		},
	}

	Convey("TestNormalization", t, func() {
		for _, t := range positive {
			Convey(fmt.Sprintf("good: %s", t.Name), func() {
				p := Phrase{Name: "basic"}
				So(jsonpb.UnmarshalString(t.Phrase, &p), ShouldBeNil)
				So(p.Normalize(), ShouldBeNil)
				if t.Parsed != nil {
					So(t.Parsed, ShouldResemble, t.Parsed)
				}
			})
		}

		for _, t := range bad {
			Convey(fmt.Sprintf("bad: %s", t.Name), func() {
				p := Phrase{Name: "basic"}
				So(jsonpb.UnmarshalString(t.Phrase, &p), ShouldErrLike, t.ParseErr)
				So(p.Normalize(), ShouldErrLike, t.NormErr)
			})
		}
	})
}
