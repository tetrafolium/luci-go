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

package projectscope

import (
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"

	bqpb "github.com/tetrafolium/luci-go/tokenserver/api/bq"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMintedTokenInfo(t *testing.T) {
	t.Parallel()

	Convey("produces correct row map", t, func() {
		info := MintedTokenInfo{
			Request: &minter.MintProjectTokenRequest{
				MinValidityDuration: 3600,
				AuditTags:           []string{"k:v"},
				LuciProject:         "someproject",
			},
			Response: &minter.MintProjectTokenResponse{
				ServiceAccountEmail: "foo@bar.com",
				AccessToken:         "blah",
				ServiceVersion:      "unit-tests/mocked-ver",
			},
			RequestedAt: &timestamp.Timestamp{Seconds: 1},
			Expiration:  &timestamp.Timestamp{Seconds: 1},
			PeerIP:      net.ParseIP("127.10.10.10"),
			RequestID:   "gae-request-id",
			AuthDBRev:   123,
		}

		So(info.toBigQueryMessage(), ShouldResemble, &bqpb.ProjectToken{
			Fingerprint:    "8b7df143d91c716ecfa5fc1730022f6b",
			ServiceAccount: "foo@bar.com",
			LuciProject:    "someproject",
			Expiration:     &timestamp.Timestamp{Seconds: 1},
			RequestedAt:    &timestamp.Timestamp{Seconds: 1},
			AuditTags:      []string{"k:v"},
			PeerIp:         "127.10.10.10",
			ServiceVersion: "unit-tests/mocked-ver",
			GaeRequestId:   "gae-request-id",
			AuthDbRev:      123,
		})
	})
}
