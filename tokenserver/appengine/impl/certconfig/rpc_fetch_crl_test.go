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

package certconfig

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/tetrafolium/luci-go/appengine/gaetesting"
	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/server/auth"

	"github.com/tetrafolium/luci-go/tokenserver/api/admin/v1"
	"github.com/tetrafolium/luci-go/tokenserver/appengine/impl/utils"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestFetchCRLRPC(t *testing.T) {
	Convey("with mock context", t, func() {
		ctx := gaetesting.TestingContext()
		ctx = auth.ModifyConfig(ctx, func(cfg auth.Config) auth.Config {
			cfg.AnonymousTransport = func(context.Context) http.RoundTripper {
				return http.DefaultTransport // mock URLFetch service
			}
			return cfg
		})

		importConfig := func(cfg string) {
			impl := ImportCAConfigsRPC{}
			_, err := impl.ImportCAConfigs(prepareCfg(ctx, cfg), nil)
			if err != nil {
				panic(err)
			}
		}

		callFetchCRL := func(cn string, force bool) error {
			impl := FetchCRLRPC{}
			_, err := impl.FetchCRL(ctx, &admin.FetchCRLRequest{
				Cn:    cn,
				Force: force,
			})
			return err
		}

		Convey("FetchCRL not configured", func() {
			// Prepare config (with empty crl_url).
			importConfig(`
				certificate_authority {
					cn: "Puppet CA: fake.ca"
					cert_path: "certs/fake.ca.crt"
				}
			`)
			// Use it, must fail.
			err := callFetchCRL("Puppet CA: fake.ca", false)
			So(err, ShouldErrLike, "doesn't have CRL defined")
		})

		Convey("FetchCRL works (der, no etags)", func() {
			ts := serveCRL()
			defer ts.Close()

			// Prepare config.
			importConfig(fmt.Sprintf(`
				certificate_authority {
					cn: "Puppet CA: fake.ca"
					cert_path: "certs/fake.ca.crt"
					crl_url: %q
				}
			`, ts.URL))

			// Import works.
			ts.CRL = fakeCACrl
			err := callFetchCRL("Puppet CA: fake.ca", true)
			So(err, ShouldBeNil)

			// CRL is there.
			crl := CRL{
				Parent: ds.NewKey(ctx, "CA", "Puppet CA: fake.ca", 0, nil),
			}
			err = ds.Get(ctx, &crl)
			So(err, ShouldBeNil)
			So(crl.RevokedCertsCount, ShouldEqual, 1) // fakeCACrl has only 1 SN
		})

		Convey("FetchCRL works (pem, no etags)", func() {
			ts := serveCRL()
			defer ts.Close()

			// Prepare config.
			importConfig(fmt.Sprintf(`
				certificate_authority {
					cn: "Puppet CA: fake.ca"
					cert_path: "certs/fake.ca.crt"
					crl_url: %q
				}
			`, ts.URL))

			// Import works.
			ts.CRL = fakeCACrl
			ts.ServePEM = true
			err := callFetchCRL("Puppet CA: fake.ca", true)
			So(err, ShouldBeNil)

			// CRL is there.
			crl := CRL{
				Parent: ds.NewKey(ctx, "CA", "Puppet CA: fake.ca", 0, nil),
			}
			err = ds.Get(ctx, &crl)
			So(err, ShouldBeNil)
			So(crl.RevokedCertsCount, ShouldEqual, 1) // fakeCACrl has only 1 SN
		})

		Convey("FetchCRL works (der, with etags)", func() {
			ts := serveCRL()
			defer ts.Close()

			// Prepare config.
			importConfig(fmt.Sprintf(`
				certificate_authority {
					cn: "Puppet CA: fake.ca"
					cert_path: "certs/fake.ca.crt"
					crl_url: %q
				}
			`, ts.URL))

			// Initial import works.
			ts.CRL = fakeCACrl
			ts.Etag = `"etag1"`
			So(callFetchCRL("Puppet CA: fake.ca", false), ShouldBeNil)

			// CRL is there.
			crl := CRL{
				Parent: ds.NewKey(ctx, "CA", "Puppet CA: fake.ca", 0, nil),
			}
			err := ds.Get(ctx, &crl)
			So(err, ShouldBeNil)
			So(crl.LastFetchETag, ShouldEqual, `"etag1"`)
			So(crl.EntityVersion, ShouldEqual, 1)

			// Refetch. No etag change.
			So(callFetchCRL("Puppet CA: fake.ca", false), ShouldBeNil)

			// Entity isn't touched.
			err = ds.Get(ctx, &crl)
			So(err, ShouldBeNil)
			So(crl.LastFetchETag, ShouldEqual, `"etag1"`)
			So(crl.EntityVersion, ShouldEqual, 1)

			// Refetch. Etag changes.
			ts.Etag = `"etag2"`
			So(callFetchCRL("Puppet CA: fake.ca", false), ShouldBeNil)

			// Entity is updated.
			err = ds.Get(ctx, &crl)
			So(err, ShouldBeNil)
			So(crl.LastFetchETag, ShouldEqual, `"etag2"`)
			So(crl.EntityVersion, ShouldEqual, 2)
		})
	})
}

// Valid CRL signed by key that corresponds to fakeCACrt.
//
// Contains only one revoked SN: "2".
const fakeCACrl = `-----BEGIN X509 CRL-----
MIICuzCBpAIBATANBgkqhkiG9w0BAQUFADAdMRswGQYDVQQDDBJQdXBwZXQgQ0E6
IGZha2UuY2EXDTE2MDMxNTAzNDk0NloXDTIxMDMxNDAzNDk0N1owIjAgAgECFw0x
NjAzMTUwMzQ5NDdaMAwwCgYDVR0VBAMKAQGgLzAtMB8GA1UdIwQYMBaAFOeGP1Os
e9spvhIIrGMEZEpoeiDqMAoGA1UdFAQDAgEBMA0GCSqGSIb3DQEBBQUAA4ICAQA8
LeRLqrgl1ed5UbFQyWnmpOW58PzIDEdCtRutVc12VlMKu+FyJ6DELXDpmZjkam32
gMrH9zHbLywO3O6qGl8WaKMVPhKyhdemQa9/TrqFr/lqEsfM9g6ZY4b3dO9VFy42
9SMTQF6iu7ZRfhjui50DZlbD+VtfgTAJpeVTKR3E6ntuYQ+noJ568xcwcswAR6hT
iAvv49kExuflo2ntg9uSHZYvo/PMmUZZ/ThMK+EfalWsz//N1JOSahLl1qakEBKz
OD6QsZB0K3160hsPO5O8iC2FdYa1xiamTiYOKAIqIRgX8+WH2cfc4Wg8mGz4DtJE
BlPZCIhxjbzymi55B2N1Mo/KuYD73j24NN6IG7s6JSohjn/In7h7T9gkOGwkxM5P
jZrNiLYELrfMMVl9z3uiA31qVPoVa2MPsfwY3pWtTVZ3lJ/mWAFesrgCl2FSgBcr
t2WZsEUA7W8l45nbNg8m8l+nOEBCM7Pjycy8ZV7XFdT9iATn44huQi1CGw2xUpEX
8FOcDDS2tb78R3ZoyqFS5l/P5Kd0DitivPhRNQXQboFqT5XL9EBKcyExnR+y72+B
7fIzS92HZavZYpO/YKHweFWonSuNcGOwqLyI/ZZealwOQROD4AC6ZMUeY9oQkbEE
3QbCiGRlaGEOA9SCEoSTNPN9LQ1nHKoaFDy1B5ralA==
-----END X509 CRL-----
`

type crlServer struct {
	*httptest.Server

	Lock     sync.Mutex
	CRL      string
	Etag     string
	ServePEM bool
}

// serveCRL starts a test server that serves CRL file.
func serveCRL() *crlServer {
	s := &crlServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Lock.Lock()
		defer s.Lock.Unlock()

		var blob []byte
		if s.ServePEM {
			blob = []byte(s.CRL)
		} else {
			der, err := utils.ParsePEM(s.CRL, "X509 CRL")
			if err != nil {
				w.WriteHeader(500)
				return
			}
			blob = der
		}

		if s.Etag != "" {
			w.Header().Set("ETag", s.Etag)
		}
		w.WriteHeader(200)
		w.Write(blob)
	}))
	return s
}
