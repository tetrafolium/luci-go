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

package tokensigning

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/server/auth/delegation/messages"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/auth/signing/signingtest"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestSignToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	original := &messages.Subtoken{
		DelegatedIdentity: "user:delegated@example.com",
		RequestorIdentity: "user:requestor@example.com",
		CreationTime:      1477624966,
		ValidityDuration:  3600,
		Audience:          []string{"*"},
		Services:          []string{"*"},
	}
	signer := signingtest.NewSigner(&signing.ServiceInfo{
		ServiceAccountName: "service@example.com",
	})

	Convey("Works", t, func() {
		tok, err := signerForTest(signer, "").SignToken(ctx, original)
		So(err, ShouldBeNil)
		So(tok, ShouldHaveLength, 276)

		envelope, back, err := deserializeForTest(ctx, tok, signer)
		So(err, ShouldBeNil)
		So(back, ShouldResembleProto, original)

		envelope.Pkcs1Sha256Sig = nil
		envelope.SerializedSubtoken = nil
		So(envelope, ShouldResemble, &messages.DelegationToken{
			SignerId:     "user:service@example.com",
			SigningKeyId: signer.KeyNameForTest(),
		})
	})

	Convey("SigningContext works", t, func() {
		const contextString = "Some context string"

		signer := &capturingSigner{signer, nil}
		tok, err := signerForTest(signer, contextString).SignToken(ctx, original)
		So(err, ShouldBeNil)
		So(tok, ShouldHaveLength, 276)

		ctxPart := signer.blobs[0][:len(contextString)+1]
		So(string(ctxPart), ShouldEqual, contextString+"\x00")

		msgPart := signer.blobs[0][len(contextString)+1:]
		msg := &messages.Subtoken{}
		So(proto.Unmarshal(msgPart, msg), ShouldBeNil)
		So(msg, ShouldResembleProto, original)
	})
}

type capturingSigner struct {
	signing.Signer

	blobs [][]byte
}

func (s *capturingSigner) SignBytes(c context.Context, blob []byte) (string, []byte, error) {
	s.blobs = append(s.blobs, blob)
	return s.Signer.SignBytes(c, blob)
}

func signerForTest(signer signing.Signer, signingCtx string) *Signer {
	return &Signer{
		Signer:         signer,
		SigningContext: signingCtx,
		Wrap: func(t *Unwrapped) proto.Message {
			return &messages.DelegationToken{
				SignerId:           "user:" + t.SignerID,
				SigningKeyId:       t.KeyID,
				SerializedSubtoken: t.Body,
				Pkcs1Sha256Sig:     t.RsaSHA256Sig,
			}
		},
	}
}

func deserializeForTest(c context.Context, tok string, signer signing.Signer) (*messages.DelegationToken, *messages.Subtoken, error) {
	blob, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return nil, nil, err
	}
	env := &messages.DelegationToken{}
	if err = proto.Unmarshal(blob, env); err != nil {
		return nil, nil, err
	}
	certs, err := signer.Certificates(c)
	if err != nil {
		return nil, nil, err
	}
	if err = certs.CheckSignature(env.SigningKeyId, env.SerializedSubtoken, env.Pkcs1Sha256Sig); err != nil {
		return nil, nil, err
	}
	subtoken := &messages.Subtoken{}
	if err = proto.Unmarshal(env.SerializedSubtoken, subtoken); err != nil {
		return nil, nil, err
	}
	return env, subtoken, nil
}
