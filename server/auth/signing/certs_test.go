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

package signing

import (
	"context"
	"encoding/pem"
	"fmt"
	"net/http"
	"testing"

	"github.com/tetrafolium/luci-go/server/auth/internal"
	"github.com/tetrafolium/luci-go/server/caching"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

var certBlob = `-----BEGIN CERTIFICATE-----
MIIBDjCBu6ADAgECAgEBMAsGCSqGSIb3DQEBCzAAMCAXDTAxMDkwOTAxNDY0MFoY
DzIyODYxMTIwMTc0NjQwWjAAMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAMGYtc/k
vp1Sr2zZFWPu534tqX9chKxhADlLbPR4A+ojKl/EchYCV6DE7Ikogx02PFpYZe3A
3a4hccSufwr3wtMCAwEAAaMgMB4wDgYDVR0PAQH/BAQDAgCAMAwGA1UdEwEB/wQC
MAAwCwYJKoZIhvcNAQELA0EAI/3v5eWNzA2oudenR8Vo5EY0j3zCUVhlHRErlcUR
I69yAHZUpJ9lzcwmHcaCJ76m/jDINZrYoL/4aSlDEGgHmw==
-----END CERTIFICATE-----
`

func TestFetchCertificates(t *testing.T) {
	t.Parallel()

	const testURL = "https://test.example.com"

	Convey("With empty cache", t, func() {
		ctx := caching.WithEmptyProcessCache(context.Background())

		Convey("Works", func() {
			ctx := internal.WithTestTransport(ctx, func(r *http.Request, body string) (int, string) {
				So(r.URL.String(), ShouldEqual, testURL)
				return 200, fmt.Sprintf(`{
				"service_account_name": "blah@blah.com",
				"certificates": [{
					"key_name": "abc",
					"x509_certificate_pem": %q
				}],
				"timestamp": 1446166229439210
			}`, certBlob)
			})

			certs, err := FetchCertificates(ctx, testURL)
			So(err, ShouldBeNil)
			So(certs.ServiceAccountName, ShouldEqual, "blah@blah.com")
			So(len(certs.Certificates), ShouldEqual, 1)
		})

		Convey("Errors", func() {
			ctx := internal.WithTestTransport(ctx, func(r *http.Request, body string) (int, string) {
				return 401, "fail"
			})

			_, err := FetchCertificates(ctx, testURL)
			So(err, ShouldNotBeNil)
		})

		Convey("Bad JSON", func() {
			ctx := internal.WithTestTransport(ctx, func(r *http.Request, body string) (int, string) {
				return 200, fmt.Sprintf(`{
				"certificates": [{
					"key_name": "abc",
					"x509_certificate_pem": %q
				}],
				"timestamp": "not an int"
			}`, certBlob)
			})

			_, err := FetchCertificates(ctx, testURL)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestFetchCertificatesForServiceAccount(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		ctx := caching.WithEmptyProcessCache(context.Background())
		ctx = internal.WithTestTransport(ctx, func(r *http.Request, body string) (int, string) {
			So(r.URL.String(), ShouldEqual, "https://www.googleapis.com/robot/v1/metadata/x509/robot%40robots.gserviceaccount.com")
			return 200, `{
				"0392f9886770640357cbb29e57d3698291b1e805": "-----BEGIN CERTIFICATE-----\nblah 1\n-----END CERTIFICATE-----\n",
				"f5db308971078d1496c262cc06b6e7f87652af55": "-----BEGIN CERTIFICATE-----\nblah 2\n-----END CERTIFICATE-----\n"
			}`
		})

		certs, err := FetchCertificatesForServiceAccount(ctx, "robot@robots.gserviceaccount.com")
		So(err, ShouldBeNil)
		So(certs.ServiceAccountName, ShouldEqual, "robot@robots.gserviceaccount.com")
		So(certs.Certificates, ShouldResemble, []Certificate{
			{
				KeyName:            "0392f9886770640357cbb29e57d3698291b1e805",
				X509CertificatePEM: "-----BEGIN CERTIFICATE-----\nblah 1\n-----END CERTIFICATE-----\n",
			},
			{
				KeyName:            "f5db308971078d1496c262cc06b6e7f87652af55",
				X509CertificatePEM: "-----BEGIN CERTIFICATE-----\nblah 2\n-----END CERTIFICATE-----\n",
			},
		})
	})
}

func TestFetchGoogleOAuth2Certificates(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		ctx := caching.WithEmptyProcessCache(context.Background())
		ctx = internal.WithTestTransport(ctx, func(r *http.Request, body string) (int, string) {
			So(r.URL.String(), ShouldEqual, "https://www.googleapis.com/oauth2/v1/certs")
			return 200, `{
				"0392f9886770640357cbb29e57d3698291b1e805": "-----BEGIN CERTIFICATE-----\nblah 1\n-----END CERTIFICATE-----\n",
				"f5db308971078d1496c262cc06b6e7f87652af55": "-----BEGIN CERTIFICATE-----\nblah 2\n-----END CERTIFICATE-----\n"
			}`
		})

		certs, err := FetchGoogleOAuth2Certificates(ctx)
		So(err, ShouldBeNil)
		So(certs.Certificates, ShouldResemble, []Certificate{
			{
				KeyName:            "0392f9886770640357cbb29e57d3698291b1e805",
				X509CertificatePEM: "-----BEGIN CERTIFICATE-----\nblah 1\n-----END CERTIFICATE-----\n",
			},
			{
				KeyName:            "f5db308971078d1496c262cc06b6e7f87652af55",
				X509CertificatePEM: "-----BEGIN CERTIFICATE-----\nblah 2\n-----END CERTIFICATE-----\n",
			},
		})
	})
}

func TestCertificateForKey(t *testing.T) {
	t.Parallel()

	Convey("Works", t, func() {
		certs := PublicCertificates{
			Certificates: []Certificate{
				{
					KeyName:            "abc",
					X509CertificatePEM: certBlob,
				},
			},
		}
		cert, err := certs.CertificateForKey("abc")
		So(err, ShouldBeNil)
		So(cert, ShouldNotBeNil)

		// Code coverage for cache hit.
		cert, err = certs.CertificateForKey("abc")
		So(err, ShouldBeNil)
		So(cert, ShouldNotBeNil)
	})

	Convey("Bad PEM", t, func() {
		certs := PublicCertificates{
			Certificates: []Certificate{
				{
					KeyName:            "abc",
					X509CertificatePEM: "not a pem",
				},
			},
		}
		cert, err := certs.CertificateForKey("abc")
		So(err, ShouldErrLike, "not PEM")
		So(cert, ShouldBeNil)
	})

	Convey("Bad cert", t, func() {
		certs := PublicCertificates{
			Certificates: []Certificate{
				{
					KeyName: "abc",
					X509CertificatePEM: string(pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: []byte("not a certificate"),
					})),
				},
			},
		}
		cert, err := certs.CertificateForKey("abc")
		So(err, ShouldErrLike, "structure error")
		So(cert, ShouldBeNil)
	})

	Convey("Missing key", t, func() {
		certs := PublicCertificates{}
		cert, err := certs.CertificateForKey("abc")
		So(err, ShouldErrLike, "no such certificate")
		So(cert, ShouldBeNil)
	})
}

func TestCheckSignature(t *testing.T) {
	// See signingtest/signer_test.go for where this cert and signature were
	// generated. 'signingtest' module itself can't be imported due to import
	// cycle.

	t.Parallel()

	Convey("Works", t, func() {
		certs := PublicCertificates{
			Certificates: []Certificate{
				{
					KeyName:            "abc",
					X509CertificatePEM: certBlob,
				},
			},
		}

		blob := []byte("some blob")

		signature := []byte{
			0x66, 0x2d, 0xa6, 0xa0, 0x65, 0x63, 0x8b, 0x83, 0xc5, 0x45, 0xeb, 0xfd,
			0x88, 0xec, 0x9, 0x41, 0x59, 0x92, 0xd0, 0x48, 0x78, 0x37, 0xc2, 0x45,
			0x74, 0xfc, 0x8b, 0x13, 0xa, 0xca, 0x47, 0x7d, 0xd1, 0x24, 0x2c, 0x6c,
			0xbe, 0x3a, 0xea, 0xc5, 0x12, 0x76, 0xb4, 0xe1, 0xa9, 0x4a, 0x40, 0x40,
			0x24, 0xf7, 0x1e, 0x7c, 0x91, 0x91, 0xe3, 0x71, 0x4f, 0x21, 0xf4, 0xe4,
			0xec, 0x65, 0x87, 0x1c,
		}

		err := certs.CheckSignature("abc", blob, signature)
		So(err, ShouldBeNil)

		err = certs.CheckSignature("abc", blob, []byte{1, 2, 3})
		So(err, ShouldNotBeNil)

		err = certs.CheckSignature("no key", blob, signature)
		So(err, ShouldNotBeNil)
	})
}
