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

package server

import (
	"context"
	"net/http"

	"github.com/tetrafolium/luci-go/auth/identity"
	"github.com/tetrafolium/luci-go/server/auth"
)

// InboundAppIDAuthMethod implements auth.Method by checking special HTTP
// header (X-Appengine-Inbound-Appid), that is set iff one GAE app talks to
// another.
type InboundAppIDAuthMethod struct{}

// Authenticate extracts peer's identity from the incoming request.
func (m InboundAppIDAuthMethod) Authenticate(ctx context.Context, r *http.Request) (*auth.User, error) {
	appID := r.Header.Get("X-Appengine-Inbound-Appid")
	if appID == "" {
		return nil, nil
	}
	id, err := identity.MakeIdentity("service:" + appID)
	if err != nil {
		return nil, err
	}
	return &auth.User{Identity: id}, nil
}
