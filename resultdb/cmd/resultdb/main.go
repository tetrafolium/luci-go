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

package main

import (
	"flag"

	"github.com/tetrafolium/luci-go/common/flag/stringmapflag"
	"github.com/tetrafolium/luci-go/server"

	"github.com/tetrafolium/luci-go/resultdb/internal"
	"github.com/tetrafolium/luci-go/resultdb/internal/artifactcontent"
	"github.com/tetrafolium/luci-go/resultdb/internal/services/resultdb"
)

func main() {
	opts := resultdb.Options{
		ContentHostnameMap: map[string]string{},
	}
	flag.BoolVar(
		&opts.InsecureSelfURLs,
		"insecure-self-urls",
		false,
		"Use http:// (not https://) for URLs pointing back to ResultDB",
	)

	hostFlag := stringmapflag.Value(opts.ContentHostnameMap)
	flag.Var(
		&hostFlag,
		"user-content-host-map",
		"Key=value map where key is a ResultDB API hostname and value is a "+
			"hostname to use for user-content URLs produced there. "+
			"Key '*' indicates a fallback.")

	artifactcontent.RegisterRBEInstanceFlag(flag.CommandLine, &opts.ArtifactRBEInstance)

	internal.Main(func(srv *server.Server) error {
		return resultdb.InitServer(srv, opts)
	})
}
