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

package jobcreate

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"

	swarming "github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
	"github.com/tetrafolium/luci-go/led/job"
)

var train = flag.Bool("train", false, "If set, write testdata/*out.json")

func readTestFixture(fixtureBaseName string) *job.Definition {
	data, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.json", fixtureBaseName))
	So(err, ShouldBeNil)

	req := &swarming.SwarmingRpcsNewTaskRequest{}
	So(json.NewDecoder(bytes.NewReader(data)).Decode(req), ShouldBeNil)

	jd, err := FromNewTaskRequest(
		context.Background(), req,
		"test_name", "swarming.example.com",
		job.NoKitchenSupport())
	So(err, ShouldBeNil)
	So(jd, ShouldNotBeNil)

	outFile := fmt.Sprintf("testdata/%s.job.json", fixtureBaseName)
	marshaler := &jsonpb.Marshaler{
		OrigName: true,
		Indent:   "  ",
	}
	if *train {
		oFile, err := os.Create(outFile)
		So(err, ShouldBeNil)
		defer oFile.Close()

		So(marshaler.Marshal(oFile, jd), ShouldBeNil)
	} else {
		current, err := ioutil.ReadFile(outFile)
		So(err, ShouldBeNil)

		actual, err := marshaler.MarshalToString(jd)
		So(err, ShouldBeNil)

		So(string(current), ShouldEqual, actual)
	}

	return jd
}

func TestCreateSwarmRaw(t *testing.T) {
	t.Parallel()

	Convey(`consume non-buildbucket swarming task`, t, func() {
		jd := readTestFixture("raw")

		So(jd.GetSwarming(), ShouldNotBeNil)
		So(jd.Info().SwarmingHostname(), ShouldEqual, "swarming.example.com")
		So(jd.Info().TaskName(), ShouldEqual, "led: test_name")
	})
}

func TestCreateBBagent(t *testing.T) {
	t.Parallel()

	Convey(`consume bbagent buildbucket swarming task`, t, func() {
		jd := readTestFixture("bbagent")

		So(jd.GetBuildbucket(), ShouldNotBeNil)
		So(jd.Info().SwarmingHostname(), ShouldEqual, "chromium-swarm-dev.appspot.com")
		So(jd.Info().TaskName(), ShouldEqual, "led: test_name")
	})
}
