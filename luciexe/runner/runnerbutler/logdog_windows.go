// Copyright 2019 The LUCI Authors.
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

package runnerbutler

import (
	"context"
	"fmt"
	"os"

	"go.chromium.org/luci/logdog/client/butler/streamserver"
)

// newLogDogStreamServerForPlatform creates a StreamServer instance usable on
// Windows.
func newLogDogStreamServerForPlatform(ctx context.Context, workDir string) (streamserver.StreamServer, error) {
	// Windows, use named pipe.
	return streamserver.NewNamedPipeServer(ctx, fmt.Sprintf("LUCILogDogRunBuild_%d", os.Getpid()))
}