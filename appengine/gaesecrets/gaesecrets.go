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

// Package gaesecrets implements storage of secret blobs on top of datastore.
//
// It is not super secure, but we have what we have: there's no other better
// mechanism to persistently store non-static secrets on GAE.
//
// All secrets are global (live in default GAE namespace).
package gaesecrets

import (
	"context"
	"crypto/rand"
	"io"
	"strings"
	"time"

	ds "github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/gae/service/info"

	"github.com/tetrafolium/luci-go/common/clock"
	"github.com/tetrafolium/luci-go/common/retry/transient"
	"github.com/tetrafolium/luci-go/server/caching"
	"github.com/tetrafolium/luci-go/server/secrets"
)

// TODO(vadimsh): Add secrets rotation.

// cacheExp is how long to cache secrets in the process memory.
const cacheExp = time.Minute * 5

// Config can be used to tweak parameters of the store. It is fine to use
// default values.
type Config struct {
	NoAutogenerate bool      // if true, GetSecret will NOT generate secrets
	SecretLen      int       // length of generated secrets, 32 bytes default
	Prefix         string    // optional prefix for entity keys to namespace them
	Entropy        io.Reader // source of random numbers, crypto rand by default
}

// Use injects the GAE implementation of secrets.Store into the context.
// The context must be configured with GAE datastore implementation already.
func Use(ctx context.Context, cfg *Config) context.Context {
	config := Config{}
	if cfg != nil {
		config = *cfg
	}
	if strings.Contains(config.Prefix, ":") {
		panic("forbidden character ':' in Prefix")
	}
	if config.SecretLen == 0 {
		config.SecretLen = 32
	}
	if config.Entropy == nil {
		config.Entropy = rand.Reader
	}
	return secrets.SetFactory(ctx, func(ctx context.Context) secrets.Store {
		return &storeImpl{config, ctx}
	})
}

// full secret key (including prefix) => secrets.Secret.
var secretsCache = caching.RegisterLRUCache(100)

// storeImpl is implementation of secrets.Store bound to a GAE context.
type storeImpl struct {
	cfg Config
	ctx context.Context
}

// GetSecret returns a secret by its key.
func (s *storeImpl) GetSecret(k string) (secrets.Secret, error) {
	secret, err := secretsCache.LRU(s.ctx).GetOrCreate(s.ctx, s.cfg.Prefix+":"+string(k), func() (interface{}, time.Duration, error) {
		secret, err := s.getSecretFromDatastore(k)
		if err != nil {
			return nil, 0, err
		}
		return secret, cacheExp, nil
	})
	if err != nil {
		return secrets.Secret{}, err
	}
	return secret.(secrets.Secret), nil
}

// getSecretImpl uses non-transactional datastore (txnBuf.GetNoTxn) to grab a
// secret.
func (s *storeImpl) getSecretFromDatastore(k string) (secrets.Secret, error) {
	// Switch to default namespace.
	ctx, err := info.Namespace(s.ctx, "")
	if err != nil {
		panic(err) // should not happen, Namespace errors only on bad namespace name
	}
	ctx = ds.WithoutTransaction(ctx)

	// Grab existing.
	ent := secretEntity{ID: s.cfg.Prefix + ":" + string(k)}
	err = ds.Get(ctx, &ent)
	if err != nil && err != ds.ErrNoSuchEntity {
		return secrets.Secret{}, transient.Tag.Apply(err)
	}

	// Autogenerate and put into the datastore.
	if err == ds.ErrNoSuchEntity {
		if s.cfg.NoAutogenerate {
			return secrets.Secret{}, secrets.ErrNoSuchSecret
		}
		ent.Created = clock.Now(s.ctx).UTC()
		if ent.Secret, err = s.generateSecret(); err != nil {
			return secrets.Secret{}, transient.Tag.Apply(err)
		}
		err = ds.RunInTransaction(ctx, func(ctx context.Context) error {
			newOne := secretEntity{ID: ent.ID}
			switch err := ds.Get(ctx, &newOne); err {
			case nil:
				ent = newOne
				return nil
			case ds.ErrNoSuchEntity:
				return ds.Put(ctx, &ent)
			default:
				return err
			}
		}, nil)
		if err != nil {
			return secrets.Secret{}, transient.Tag.Apply(err)
		}
	}

	return secrets.Secret{
		Current: ent.Secret,
	}, nil
}

func (s *storeImpl) generateSecret() ([]byte, error) {
	out := make([]byte, s.cfg.SecretLen)
	_, err := io.ReadFull(s.cfg.Entropy, out)
	return out, err
}

////

type secretEntity struct {
	_kind  string         `gae:"$kind,gaesecrets.Secret"`
	_extra ds.PropertyMap `gae:"-,extra"`

	ID string `gae:"$id"`

	Secret  []byte `gae:",noindex"` // blob with the secret
	Created time.Time
}
