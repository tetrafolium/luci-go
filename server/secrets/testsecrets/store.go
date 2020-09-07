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

// Package testsecrets provides a dumb in-memory secret store to use in unit
// tests. Use secrets.Set(c, &testsecrets.Store{...}) to inject it into
// the context.
package testsecrets

import (
	"context"
	"math/rand"
	"sync"

	"github.com/tetrafolium/luci-go/server/secrets"
)

// Store implements secrets.Store in the simplest way possible using memory as
// a backend and very dumb deterministic "randomness" source for secret key
// autogeneration. Useful in unit tests. Can be modified directly (use lock if
// doing it concurrently). NEVER use it outside of tests.
type Store struct {
	sync.Mutex

	Secrets        map[string]secrets.Secret // current map of all secrets
	NoAutogenerate bool                      // if true, GetSecret will NOT generate secrets
	SecretLen      int                       // length of generated secret, 8 bytes default
	Rand           *rand.Rand                // used to generate missing secrets
}

// GetSecret is a part of Store interface.
func (t *Store) GetSecret(k string) (secrets.Secret, error) {
	t.Lock()
	defer t.Unlock()

	if s, ok := t.Secrets[k]; ok {
		return s, nil
	}

	if t.NoAutogenerate {
		return secrets.Secret{}, secrets.ErrNoSuchSecret
	}

	// Initialize defaults.
	if t.Secrets == nil {
		t.Secrets = map[string]secrets.Secret{}
	}
	if t.SecretLen == 0 {
		t.SecretLen = 8
	}
	if t.Rand == nil {
		t.Rand = rand.New(rand.NewSource(0))
	}

	// Generate deterministic secret.
	secret := make([]byte, t.SecretLen)
	for i := range secret {
		secret[i] = byte(t.Rand.Int31n(256))
	}
	t.Secrets[k] = secrets.Secret{Current: secret}
	return t.Secrets[k], nil
}

// Use installs default testing store into the context.
func Use(c context.Context) context.Context {
	return secrets.Set(c, &Store{})
}
