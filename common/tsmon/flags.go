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

package tsmon

import (
	"flag"
	"time"

	"github.com/tetrafolium/luci-go/common/tsmon/target"
)

// Flags defines command line flags related to tsmon.  Use NewFlags()
// to get a Flags struct with sensible default values.
type Flags struct {
	ConfigFile    string
	Endpoint      string
	Credentials   string
	ActAs         string
	Flush         FlushType
	FlushInterval time.Duration

	Target target.Flags
}

// NewFlags returns a Flags struct with sensible default values.
func NewFlags() Flags {
	return Flags{
		ConfigFile:    defaultConfigFilePath(),
		Endpoint:      "",
		Credentials:   "",
		ActAs:         "",
		Flush:         FlushAuto,
		FlushInterval: time.Minute,

		Target: target.NewFlags(),
	}
}

// Register adds tsmon related flags to a FlagSet.
func (fl *Flags) Register(f *flag.FlagSet) {
	f.StringVar(&fl.ConfigFile, "ts-mon-config-file", fl.ConfigFile,
		"path to a JSON config file that contains suitable values for "+
			"\"endpoint\" and \"credentials\" for this machine. This config file is "+
			"intended to be shared by all processes on the machine, as the values "+
			"depend on the machine's position in the network, IP whitelisting and "+
			"deployment of credentials.")
	f.StringVar(&fl.Endpoint, "ts-mon-endpoint", fl.Endpoint,
		"url (including file://, https://, pubsub://project/topic) to post "+
			"monitoring metrics to. If set, overrides the value in "+
			"--ts-mon-config-file")
	f.StringVar(&fl.Credentials, "ts-mon-credentials", fl.Credentials,
		"path to a pkcs8 json credential file. If set, overrides the value in "+
			"--ts-mon-config-file")
	f.StringVar(&fl.ActAs, "ts-mon-act-as", fl.ActAs,
		"(advanced) a service account email to impersonate when authenticating to "+
			"tsmon backends. Uses 'iam' scope and serviceAccountTokenCreator role. "+
			"If set, overrides the value in --ts-mon-config-file")
	f.Var(&fl.Flush, "ts-mon-flush",
		"metric push behavior: manual (only send when Flush() is called), or auto "+
			"(send automatically every --ts-mon-flush-interval)")
	f.DurationVar(&fl.FlushInterval, "ts-mon-flush-interval", fl.FlushInterval,
		"automatically push metrics on this interval if --ts-mon-flush=auto")

	fl.Target.Register(f)
}
