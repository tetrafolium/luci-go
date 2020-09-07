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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tetrafolium/luci-go/server/auth/internal"
	"github.com/tetrafolium/luci-go/server/caching"
)

// "url:..." | "email:..." | "google_auth2_certs" => *PublicCertificates.
var certsCache = caching.RegisterLRUCache(1024)

const (
	robotCertsURL  = "https://www.googleapis.com/robot/v1/metadata/x509/"
	oauth2CertsURL = "https://www.googleapis.com/oauth2/v1/certs"
)

// CertsCacheExpiration defines how long to cache fetched certificates in local
// memory.
const CertsCacheExpiration = time.Hour

// Certificate is public certificate of some service. Must not be mutated once
// initialized.
type Certificate struct {
	// KeyName identifies the key used for signing.
	KeyName string `json:"key_name"`
	// X509CertificatePEM is PEM encoded certificate.
	X509CertificatePEM string `json:"x509_certificate_pem"`
}

// PublicCertificates is a bundle of recent certificates of some service. Must
// not be mutated once initialized.
type PublicCertificates struct {
	// AppID is GAE app ID of a service that owns the keys if it is on GAE.
	AppID string `json:"app_id,omitempty"`
	// ServiceAccountName is name of a service account that owns the key, if any.
	ServiceAccountName string `json:"service_account_name,omitempty"`
	// Certificates is the list of certificates.
	Certificates []Certificate `json:"certificates"`
	// Timestamp is Unix time (microseconds) of when this list was generated.
	Timestamp JSONTime `json:"timestamp"`

	lock  sync.RWMutex
	cache map[string]*x509.Certificate
}

// JSONTime is time.Time that serializes as unix timestamp (in microseconds).
type JSONTime time.Time

// Time casts value to time.Time.
func (t JSONTime) Time() time.Time {
	return time.Time(t)
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	ts, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*t = JSONTime(time.Unix(0, ts*1000))
	return nil
}

// MarshalJSON implements json.Marshaler.
func (t JSONTime) MarshalJSON() ([]byte, error) {
	ts := t.Time().UnixNano() / 1000
	return []byte(strconv.FormatInt(ts, 10)), nil
}

// FetchCertificates fetches certificates from the given URL.
//
// The server is expected to reply with JSON described by PublicCertificates
// struct (like LUCI services do). Uses the process cache to cache them for
// CertsCacheExpiration minutes.
//
// LUCI services serve certificates at /auth/api/v1/server/certificates.
func FetchCertificates(c context.Context, url string) (*PublicCertificates, error) {
	certs, err := certsCache.LRU(c).GetOrCreate(c, "url:"+url, func() (interface{}, time.Duration, error) {
		certs := &PublicCertificates{}
		req := internal.Request{
			Method: "GET",
			URL:    url,
			Out:    certs,
		}
		if err := req.Do(c); err != nil {
			return nil, 0, err
		}
		return certs, CertsCacheExpiration, nil
	})
	if err != nil {
		return nil, err
	}
	return certs.(*PublicCertificates), nil
}

// FetchCertificatesFromLUCIService is shortcut for FetchCertificates
// that uses LUCI-specific endpoint.
//
// 'serviceURL' is root URL of the service (e.g. 'https://example.com').
func FetchCertificatesFromLUCIService(c context.Context, serviceURL string) (*PublicCertificates, error) {
	return FetchCertificates(c, serviceURL+"/auth/api/v1/server/certificates")
}

// FetchCertificatesForServiceAccount fetches certificates of some Google
// service account.
//
// Works only with Google service accounts (@*.gserviceaccount.com). Uses the
// process cache to cache them for CertsCacheExpiration minutes.
//
// Usage (roughly):
//
//   certs, err := signing.FetchCertificatesForServiceAccount(ctx, <email>)
//   if certs.CheckSignature(<key id>, <blob>, <signature>) == nil {
//     <signature is valid!>
//   }
func FetchCertificatesForServiceAccount(c context.Context, email string) (*PublicCertificates, error) {
	// Do only basic validation and offload full validation to the google backend.
	if !strings.HasSuffix(email, ".gserviceaccount.com") {
		return nil, fmt.Errorf("signature: not a google service account %q", email)
	}
	certs, err := certsCache.LRU(c).GetOrCreate(c, "email:"+email, func() (interface{}, time.Duration, error) {
		certs, err := fetchCertsJSON(c, robotCertsURL+url.QueryEscape(email))
		if err != nil {
			return nil, 0, err
		}
		certs.ServiceAccountName = email
		return certs, CertsCacheExpiration, nil
	})
	if err != nil {
		return nil, err
	}
	return certs.(*PublicCertificates), nil
}

// FetchGoogleOAuth2Certificates fetches root certificates of Google OAuth2
// service.
//
// They can be used to verify signatures on various JWTs issued by Google
// OAuth2 backends (like OpenID identity tokens and GCE signed metadata JWTs).
//
// Uses the process cache to cache them for CertsCacheExpiration minutes.
func FetchGoogleOAuth2Certificates(c context.Context) (*PublicCertificates, error) {
	certs, err := certsCache.LRU(c).GetOrCreate(c, "google_auth2_certs", func() (interface{}, time.Duration, error) {
		certs, err := fetchCertsJSON(c, oauth2CertsURL)
		if err != nil {
			return nil, 0, err
		}
		return certs, CertsCacheExpiration, nil
	})
	if err != nil {
		return nil, err
	}
	return certs.(*PublicCertificates), nil
}

// fetchCertsJSON loads certificates from a JSON dict "key id => x509 PEM cert".
//
// This is the format served by Google certificate endpoints.
func fetchCertsJSON(c context.Context, url string) (*PublicCertificates, error) {
	keysAndCerts := map[string]string{}
	req := internal.Request{
		Method: "GET",
		URL:    url,
		Out:    &keysAndCerts,
	}
	if err := req.Do(c); err != nil {
		return nil, err
	}

	// Sort by key for reproducibility of return values.
	keys := make([]string, 0, len(keysAndCerts))
	for key := range keysAndCerts {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Convert to PublicCertificates struct.
	certs := &PublicCertificates{}
	for _, key := range keys {
		certs.Certificates = append(certs.Certificates, Certificate{
			KeyName:            key,
			X509CertificatePEM: keysAndCerts[key],
		})
	}
	return certs, nil
}

// CertificateForKey finds the certificate for given key and deserializes it.
func (pc *PublicCertificates) CertificateForKey(key string) (*x509.Certificate, error) {
	// Use fast reader lock first.
	pc.lock.RLock()
	cert, ok := pc.cache[key]
	pc.lock.RUnlock()
	if ok {
		return cert, nil
	}

	// Grab the write lock and recheck the cache.
	pc.lock.Lock()
	defer pc.lock.Unlock()
	if cert, ok := pc.cache[key]; ok {
		return cert, nil
	}

	for _, cert := range pc.Certificates {
		if cert.KeyName == key {
			block, _ := pem.Decode([]byte(cert.X509CertificatePEM))
			if block == nil {
				return nil, fmt.Errorf("signature: the certificate %q is not PEM encoded", key)
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			if pc.cache == nil {
				pc.cache = make(map[string]*x509.Certificate)
			}
			pc.cache[key] = cert
			return cert, nil
		}
	}

	return nil, fmt.Errorf("signature: no such certificate %q", key)
}

// CheckSignature returns nil if `signed` was indeed signed by given key.
func (pc *PublicCertificates) CheckSignature(key string, signed, signature []byte) error {
	cert, err := pc.CertificateForKey(key)
	if err != nil {
		return err
	}
	return cert.CheckSignature(x509.SHA256WithRSA, signed, signature)
}
