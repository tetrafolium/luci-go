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

package delegation

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/server/auth/delegation/messages"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/auth/signing/signingtest"

	admin "github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestInspectDelegationToken(t *testing.T) {
	ctx := context.Background()
	ctx, tc := testclock.UseTime(ctx, testclock.TestTimeUTC)

	signer := signingtest.NewSigner(&signing.ServiceInfo{
		ServiceAccountName: "service@example.com",
	})
	rpc := InspectDelegationTokenRPC{
		Signer: signer,
	}

	original := &messages.Subtoken{
		DelegatedIdentity: "user:delegated@example.com",
		RequestorIdentity: "user:requestor@example.com",
		CreationTime:      clock.Now(ctx).Unix(),
		ValidityDuration:  3600,
		Audience:          []string{"*"},
		Services:          []string{"*"},
	}

	tok, _ := SignToken(ctx, rpc.Signer, original)

	Convey("Happy path", t, func() {
		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: tok,
		})
		So(err, ShouldBeNil)

		resp.Envelope.Pkcs1Sha256Sig = nil
		resp.Envelope.SerializedSubtoken = nil
		So(resp, ShouldResembleProto, &admin.InspectDelegationTokenResponse{
			Valid:      true,
			Signed:     true,
			NonExpired: true,
			Envelope: &messages.DelegationToken{
				SignerId:     "user:service@example.com",
				SigningKeyId: signer.KeyNameForTest(),
			},
			Subtoken: original,
		})
	})

	Convey("Not base64", t, func() {
		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: "@@@@@@@@@@@@@",
		})
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, &admin.InspectDelegationTokenResponse{
			InvalidityReason: "not base64 - illegal base64 data at input byte 0",
		})
	})

	Convey("Not valid envelope proto", t, func() {
		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: "zzzz",
		})
		So(err, ShouldBeNil)
		So(resp.InvalidityReason, ShouldStartWith, "can't unmarshal the envelope - proto")
	})

	Convey("Bad signature", t, func() {
		env, _, _ := deserializeForTest(ctx, tok, rpc.Signer)
		env.Pkcs1Sha256Sig = []byte("lalala")
		blob, _ := proto.Marshal(env)
		tok := base64.RawURLEncoding.EncodeToString(blob)

		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: tok,
		})
		So(err, ShouldBeNil)

		resp.Envelope.Pkcs1Sha256Sig = nil
		resp.Envelope.SerializedSubtoken = nil
		So(resp, ShouldResembleProto, &admin.InspectDelegationTokenResponse{
			Valid:            false,
			InvalidityReason: "bad signature - crypto/rsa: verification error",
			Signed:           false,
			NonExpired:       true,
			Envelope: &messages.DelegationToken{
				SignerId:     "user:service@example.com",
				SigningKeyId: signer.KeyNameForTest(),
			},
			Subtoken: original,
		})
	})

	Convey("Expired", t, func() {
		tc.Add(2 * time.Hour)

		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: tok,
		})
		So(err, ShouldBeNil)

		resp.Envelope.Pkcs1Sha256Sig = nil
		resp.Envelope.SerializedSubtoken = nil
		So(resp, ShouldResembleProto, &admin.InspectDelegationTokenResponse{
			Valid:            false,
			InvalidityReason: "expired",
			Signed:           true,
			NonExpired:       false,
			Envelope: &messages.DelegationToken{
				SignerId:     "user:service@example.com",
				SigningKeyId: signer.KeyNameForTest(),
			},
			Subtoken: original,
		})
	})
}
