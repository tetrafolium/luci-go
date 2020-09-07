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

package mutate

import (
	"context"

	"github.com/tetrafolium/luci-go/dm/api/service/v1"
	"github.com/tetrafolium/luci-go/dm/appengine/model"
	"github.com/tetrafolium/luci-go/gae/service/datastore"
	"github.com/tetrafolium/luci-go/tumble"
)

// ActivateExecution executes an execution, moving it from the
// SCHEDULING->RUNNING state, and resetting the execution timeout (if any).
type ActivateExecution struct {
	Auth   *dm.Execution_Auth
	NewTok []byte
}

// Root implements tumble.Mutation.
func (a *ActivateExecution) Root(c context.Context) *datastore.Key {
	return model.AttemptKeyFromID(c, a.Auth.Id.AttemptID())
}

// RollForward implements tumble.Mutation
func (a *ActivateExecution) RollForward(c context.Context) (muts []tumble.Mutation, err error) {
	_, e, err := model.ActivateExecution(c, a.Auth, a.NewTok)
	if err == nil {
		err = ResetExecutionTimeout(c, e)
	}
	return
}

func init() {
	tumble.Register((*ActivateExecution)(nil))
}
