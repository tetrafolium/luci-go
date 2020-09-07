// Copyright 2015 The LUCI Authors.
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

package service

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/server/auth/service/protocol"
	"github.com/tetrafolium/luci-go/server/auth/signing"
	"github.com/tetrafolium/luci-go/server/auth/signing/signingtest"
	"github.com/tetrafolium/luci-go/server/caching"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestPubSubWorkflow(t *testing.T) {
	Convey("PubSub pull workflow works", t, func(c C) {
		ctx := caching.WithEmptyProcessCache(context.Background())

		fakeSigner := signingtest.NewSigner(nil)
		certs, _ := fakeSigner.Certificates(ctx)
		certsJSON, _ := json.Marshal(certs)

		// Expected calls.
		calls := []struct {
			Method   string
			URL      string
			Code     int
			Response string
		}{
			// Probing for existing subscription -> tell it is not found.
			{
				"GET",
				"/pubsub/projects/p1/subscriptions/sub",
				404,
				`{"error": {"code": 404}}`,
			},
			// Authorizing access to PubSub topic.
			{
				"POST",
				"/auth_service/api/v1/authdb/subscription/authorization",
				200,
				`{"topic": "projects/p2/topics/topic"}`,
			},
			// Creating the subscription.
			{
				"PUT",
				"/pubsub/projects/p1/subscriptions/sub",
				200,
				"",
			},
			// Probing for existing subscription again.
			{
				"GET",
				"/pubsub/projects/p1/subscriptions/sub",
				200,
				`{"pushConfig": {"pushEndpoint": "http://blah"}}`,
			},
			// Changing push URL.
			{
				"POST",
				"/pubsub/projects/p1/subscriptions/sub:modifyPushConfig",
				200,
				"",
			},
			// Pulling messages from it, all bad.
			{
				"POST",
				"/pubsub/projects/p1/subscriptions/sub:pull",
				200,
				`{"receivedMessages": [
					{
						"ackId": "ack1",
						"message": {"data": "broken"}
					}
				]}`,
			},
			// Fetching certificates from auth service to authenticate messages.
			{
				"GET",
				"/auth/api/v1/server/certificates",
				200,
				string(certsJSON),
			},
			// Bad messages are removed from the queue by ack.
			{
				"POST",
				"/pubsub/projects/p1/subscriptions/sub:acknowledge",
				200,
				"",
			},
			// Pulling messages from the subscription, again.
			{
				"POST",
				"/pubsub/projects/p1/subscriptions/sub:pull",
				200,
				fmt.Sprintf(`{"receivedMessages": [
					{
						"ackId": "ack1",
						"message": {"data": "broken"}
					},
					%s,
					%s
				]}`, fakePubSubMessage(ctx, "ack2", 122, fakeSigner), fakePubSubMessage(ctx, "ack2", 123, fakeSigner)),
			},
			// Acknowledging messages.
			{
				"POST",
				"/pubsub/projects/p1/subscriptions/sub:acknowledge",
				200,
				"",
			},
			// Removing existing subscription.
			{
				"DELETE",
				"/pubsub/projects/p1/subscriptions/sub",
				200,
				"{}",
			},
			// Removing already deleted subscription.
			{
				"DELETE",
				"/pubsub/projects/p1/subscriptions/sub",
				404,
				`{"error": {"code": 404}}`,
			},
		}
		counter := 0

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if counter >= len(calls) {
				c.So(fmt.Sprintf("%s %s is unexpected", r.Method, r.URL.Path), ShouldBeNil)
			}
			call := &calls[counter]
			c.So(r.Method, ShouldEqual, call.Method)
			c.So(r.URL.Path, ShouldEqual, call.URL)
			w.WriteHeader(call.Code)
			if call.Response != "" {
				w.Write([]byte(call.Response))
			}
			counter++
		}))
		defer ts.Close()

		// Register.
		srv := AuthService{
			URL:           ts.URL,
			pubSubURLRoot: ts.URL + "/pubsub/",
		}
		So(srv.EnsureSubscription(ctx, "projects/p1/subscriptions/sub", "http://blah"), ShouldBeNil)

		// Reregister with no push url. For code coverage.
		So(srv.EnsureSubscription(ctx, "projects/p1/subscriptions/sub", ""), ShouldBeNil)

		// First pull. No valid messages.
		notify, err := srv.PullPubSub(ctx, "projects/p1/subscriptions/sub")
		So(err, ShouldBeNil)
		So(notify, ShouldBeNil)

		// Second pull. Have something.
		notify, err = srv.PullPubSub(ctx, "projects/p1/subscriptions/sub")
		So(err, ShouldBeNil)
		So(notify, ShouldNotBeNil)
		So(notify.Revision, ShouldEqual, 123)

		// Ack.
		So(notify.Acknowledge(ctx), ShouldBeNil)

		// Code coverage.
		notify, err = srv.ProcessPubSubPush(ctx, []byte(fakePubSubMessage(ctx, "", 456, fakeSigner)))
		So(err, ShouldBeNil)
		So(notify, ShouldNotBeNil)
		So(notify.Revision, ShouldEqual, 456)

		// Killing existing subscription.
		So(srv.DeleteSubscription(ctx, "projects/p1/subscriptions/sub"), ShouldBeNil)
		// Killing already removed subscription.
		So(srv.DeleteSubscription(ctx, "projects/p1/subscriptions/sub"), ShouldBeNil)
	})
}

func TestGetSnapshot(t *testing.T) {
	Convey("GetSnapshot works", t, func(c C) {
		ctx := context.Background()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.So(r.URL.Path, ShouldEqual, "/auth_service/api/v1/authdb/revisions/123")
			body, digest := generateSnapshot(123)
			w.Write([]byte(fmt.Sprintf(`{"snapshot": {
				"auth_db_rev": 123,
				"sha256": "%s",
				"created_ts": 1446599918304238,
				"deflated_body": "%s"
			}}`, digest, body)))
		}))
		defer ts.Close()

		srv := AuthService{
			URL: ts.URL,
		}
		snap, err := srv.GetSnapshot(ctx, 123)
		So(err, ShouldBeNil)

		So(snap, ShouldResemble, &Snapshot{
			AuthDB:         &protocol.AuthDB{},
			AuthServiceURL: ts.URL,
			Rev:            123,
			Created:        time.Unix(0, 1446599918304238000),
		})
	})
}

func TestDeflateInflate(t *testing.T) {
	Convey("Deflate then Inflate works", t, func() {
		initial := &protocol.AuthDB{
			OauthClientId:     "abc",
			OauthClientSecret: "def",
		}
		blob, err := DeflateAuthDB(initial)
		So(err, ShouldBeNil)
		inflated, err := InflateAuthDB(blob)
		So(err, ShouldBeNil)
		So(inflated, ShouldResembleProto, initial)
	})
}

///

func generateSnapshot(rev int64) (body string, digest string) {
	blob, err := proto.Marshal(&protocol.ReplicationPushRequest{
		Revision: &protocol.AuthDBRevision{
			AuthDbRev:  rev,
			PrimaryId:  "primaryId",
			ModifiedTs: 1446599918304238,
		},
		AuthDb: &protocol.AuthDB{},
	})
	if err != nil {
		panic(err)
	}

	buf := bytes.Buffer{}
	w := zlib.NewWriter(&buf)
	_, err = w.Write(blob)
	if err != nil {
		panic(err)
	}
	w.Close()

	body = base64.StdEncoding.EncodeToString(buf.Bytes())
	hash := sha256.Sum256(blob)
	digest = hex.EncodeToString(hash[:])
	return
}

func fakePubSubMessage(c context.Context, ackID string, rev int64, signer signing.Signer) string {
	msg := protocol.ChangeNotification{
		Revision: &protocol.AuthDBRevision{
			AuthDbRev:  rev,
			PrimaryId:  "primaryId",
			ModifiedTs: 1000,
		},
	}
	blob, _ := proto.Marshal(&msg)
	key, sig, _ := signer.SignBytes(c, blob)
	ps := pubSubMessage{
		AckID: ackID,
	}
	ps.Message.Data = base64.StdEncoding.EncodeToString(blob)
	ps.Message.Attributes = map[string]string{
		"X-AuthDB-SigKey-v1": key,
		"X-AuthDB-SigVal-v1": base64.StdEncoding.EncodeToString(sig),
	}
	out, _ := json.Marshal(&ps)
	return string(out)
}
