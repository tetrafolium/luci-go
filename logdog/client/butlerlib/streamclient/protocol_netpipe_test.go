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

// +build windows

package streamclient

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/microsoft/go-winio"

	"go.chromium.org/luci/logdog/client/butlerlib/streamproto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNamedPipe(t *testing.T) {
	t.Parallel()

	counter := 0

	Convey(`test windows NamedPipe`, t, func() {
		defer timebomb()()

		ctx, cancel := mkTestCtx()
		defer cancel()

		name := fmt.Sprintf(`streamclient.test.%d.%d`, os.Getpid(), counter)
		counter++

		dataChan, _, closer := acceptOn(ctx, func() (net.Listener, error) {
			return winio.ListenPipe(streamproto.LocalNamedPipePath(name), nil)
		})
		defer closer()
		client, err := New("net.pipe:"+name, "")
		So(err, ShouldBeNil)

		runWireProtocolTest(ctx, dataChan, client, true)
	})
}