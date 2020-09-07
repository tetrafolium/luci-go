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

package lucictx

import (
	"context"

	"github.com/tetrafolium/luci-go/common/errors"
)

// ErrNoLocalAuthAccount is returned by SwitchLocalAccount if requested account
// is not available in the LUCI_CONTEXT.
var ErrNoLocalAuthAccount = errors.New("the requested logical account is not present in LUCI_CONTEXT")

// GetLocalAuth calls Lookup and returns a copy of the current LocalAuth from
// LUCI_CONTEXT if it was present. If no LocalAuth is in the context, this
// returns nil.
func GetLocalAuth(ctx context.Context) *LocalAuth {
	ret := LocalAuth{}
	ok, err := Lookup(ctx, "local_auth", &ret)
	if err != nil {
		panic(err)
	}
	if !ok {
		return nil
	}
	return &ret
}

// SetLocalAuth sets the LocalAuth in the LUCI_CONTEXT.
func SetLocalAuth(ctx context.Context, la *LocalAuth) context.Context {
	return Set(ctx, "local_auth", la)
}

// SwitchLocalAccount changes default logical account selected in the context.
//
// For example, it can be used to switch the context into using "system" account
// by default. The default account is transparently used by LUCI-aware tools.
//
// If the requested account is available, modifies LUCI_CONTEXT["local_auth"]
// in the context and returns the new modified context.
//
// If the given account is already default, returns the context unchanged.
//
// If the given account is not available, returns (nil, ErrNoLocalAuthAccount).
func SwitchLocalAccount(ctx context.Context, accountID string) (context.Context, error) {
	if la := GetLocalAuth(ctx); la != nil {
		if la.DefaultAccountId == accountID {
			return ctx, nil
		}
		for _, acc := range la.Accounts {
			if acc.Id == accountID {
				la.DefaultAccountId = accountID
				return SetLocalAuth(ctx, la), nil
			}
		}
	}
	return nil, ErrNoLocalAuthAccount
}
