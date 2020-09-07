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

package cas

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/tetrafolium/luci-go/cipd/appengine/impl/testutil"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/grpc/grpcutil"
	"github.com/tetrafolium/luci-go/server/caching"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestGetSignedURL(t *testing.T) {
	t.Parallel()

	Convey("with context", t, func() {
		ctx := caching.WithEmptyProcessCache(context.Background())
		// Use TestRecentTimeUTC, not TestRecentTimeLocal, so the
		// timestamps in the following tests do not depend on the
		// local timezone.
		ctx, cl := testclock.UseTime(ctx, testclock.TestRecentTimeUTC.Local())

		var signed []byte
		var signature string
		var signErr error
		signer := func(context.Context) (*signer, error) {
			return &signer{
				Email: "test@example.com",
				SignBytes: func(_ context.Context, data []byte) (key string, sig []byte, err error) {
					signed = data
					return "", []byte(signature), signErr
				},
			}, nil
		}

		Convey("Works", func() {
			signature = "sig1"
			url1, size, err := getSignedURL(ctx, "/bucket/path", "", signer, &mockedSignerGS{exists: true})
			So(err, ShouldBeNil)
			So(url1, ShouldEqual, "https://storage.googleapis.com"+
				"/bucket/path?Expires=1454479506&"+
				"GoogleAccessId=test%40example.com&"+
				"Signature=c2lnMQ%3D%3D")
			So(size, ShouldEqual, 123)
			So(string(signed), ShouldEqual, "GET\n\n\n1454479506\n/bucket/path")

			// 1h later returns same cached URL.
			cl.Add(time.Hour)

			signature = "sig2" // must not be picked up
			url2, _, err := getSignedURL(ctx, "/bucket/path", "", signer, &mockedSignerGS{exists: true})
			So(err, ShouldBeNil)
			So(url2, ShouldEqual, url1)

			// 31min later the cache expires and new link is generated.
			cl.Add(31 * time.Minute)

			signature = "sig3"
			url3, _, err := getSignedURL(ctx, "/bucket/path", "", signer, &mockedSignerGS{exists: true})
			So(err, ShouldBeNil)
			So(url3, ShouldEqual, "https://storage.googleapis.com"+
				"/bucket/path?Expires=1454484966&"+
				"GoogleAccessId=test%40example.com&"+
				"Signature=c2lnMw%3D%3D")
		})

		Convey("Absence is cached", func() {
			gs := &mockedSignerGS{exists: false}
			_, _, err := getSignedURL(ctx, "/bucket/path", "", signer, gs)
			So(err, ShouldErrLike, "doesn't exist")
			So(grpcutil.Code(err), ShouldEqual, codes.NotFound)
			So(gs.calls, ShouldEqual, 1)

			// 30 sec later same check is reused.
			cl.Add(30 * time.Second)
			_, _, err = getSignedURL(ctx, "/bucket/path", "", signer, gs)
			So(err, ShouldErrLike, "doesn't exist")
			So(gs.calls, ShouldEqual, 1)

			// 31 sec later the cache expires and new check is made.
			cl.Add(31 * time.Second)
			_, _, err = getSignedURL(ctx, "/bucket/path", "", signer, gs)
			So(err, ShouldErrLike, "doesn't exist")
			So(gs.calls, ShouldEqual, 2)
		})

		Convey("Signing error", func() {
			signErr = fmt.Errorf("boo, error")
			_, _, err := getSignedURL(ctx, "/bucket/path", "", signer, &mockedSignerGS{exists: true})
			So(err, ShouldErrLike, "boo, error")
			So(grpcutil.Code(err), ShouldEqual, codes.Internal)
		})

		Convey("Content-Disposition", func() {
			signature = "sig1"
			url, _, _ := getSignedURL(ctx, "/bucket/path", "name.txt", signer, &mockedSignerGS{exists: true})
			So(url, ShouldEqual, "https://storage.googleapis.com"+
				"/bucket/path?Expires=1454479506&"+
				"GoogleAccessId=test%40example.com&"+
				"Signature=c2lnMQ%3D%3D&"+
				"response-content-disposition=attachment%3B+filename%3D%22name.txt%22")
		})
	})
}

type mockedSignerGS struct {
	testutil.NoopGoogleStorage

	exists bool
	calls  int
}

func (m *mockedSignerGS) Size(ctx context.Context, path string) (size uint64, exists bool, err error) {
	m.calls++
	return 123, m.exists, nil
}
