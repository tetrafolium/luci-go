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

package delegation

import (
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/tetrafolium/luci-go/server/auth/delegation/messages"
	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	bqpb "github.com/tetrafolium/luci-go/tokenserver/api/bq"
	"github.com/tetrafolium/luci-go/tokenserver/api/minter/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMintedTokenInfo(t *testing.T) {
	t.Parallel()

	Convey("produces correct row map", t, func() {
		info := MintedTokenInfo{
			Request: &minter.MintDelegationTokenRequest{
				ValidityDuration: 3600,
				Intent:           "intent string",
				Tags:             []string{"k:v"},
			},
			Response: &minter.MintDelegationTokenResponse{
				Token:          "blah",
				ServiceVersion: "unit-tests/mocked-ver",
				DelegationSubtoken: &messages.Subtoken{
					Kind:              messages.Subtoken_BEARER_DELEGATION_TOKEN,
					SubtokenId:        1234,
					DelegatedIdentity: "user:delegated@example.com",
					RequestorIdentity: "user:requestor@example.com",
					CreationTime:      1422936306,
					ValidityDuration:  3600,
					Audience:          []string{"user:audience@example.com"},
					Services:          []string{"*"},
					Tags:              []string{"k:v"},
				},
			},
			ConfigRev: "config-rev",
			Rule: &admin.DelegationRule{
				Name: "rule-name",
			},
			PeerIP:    net.ParseIP("127.10.10.10"),
			RequestID: "gae-request-id",
			AuthDBRev: 123,
		}

		So(info.toBigQueryMessage(), ShouldResemble, &bqpb.DelegationToken{
			AuthDbRev:         123,
			ConfigRev:         "config-rev",
			ConfigRule:        "rule-name",
			DelegatedIdentity: "user:delegated@example.com",
			Expiration:        &timestamp.Timestamp{Seconds: 1422939906},
			Fingerprint:       "8b7df143d91c716ecfa5fc1730022f6b",
			GaeRequestId:      "gae-request-id",
			IssuedAt:          &timestamp.Timestamp{Seconds: 1422936306},
			PeerIp:            "127.10.10.10",
			RequestedIntent:   "intent string",
			RequestedValidity: 3600,
			RequestorIdentity: "user:requestor@example.com",
			ServiceVersion:    "unit-tests/mocked-ver",
			Tags:              []string{"k:v"},
			TargetAudience:    []string{"user:audience@example.com"},
			TargetServices:    []string{"*"},
			TokenId:           "1234",
			TokenKind:         messages.Subtoken_BEARER_DELEGATION_TOKEN,
		})
	})
}
