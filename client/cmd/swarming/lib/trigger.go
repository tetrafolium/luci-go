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

package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maruel/subcommands"

	"github.com/tetrafolium/luci-go/auth"
	"github.com/tetrafolium/luci-go/client/internal/common"
	"github.com/tetrafolium/luci-go/common/api/swarming/swarming/v1"
	"github.com/tetrafolium/luci-go/common/errors"
	"github.com/tetrafolium/luci-go/common/flag/flagenum"
	"github.com/tetrafolium/luci-go/common/flag/stringlistflag"
	"github.com/tetrafolium/luci-go/common/flag/stringmapflag"
	"github.com/tetrafolium/luci-go/common/system/signals"
)

// CmdTrigger returns an object for the `trigger` subcommand.
func CmdTrigger(defaultAuthOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "trigger <options>",
		ShortDesc: "Triggers a Swarming task",
		LongDesc:  "Triggers a Swarming task.",
		CommandRun: func() subcommands.CommandRun {
			r := &triggerRun{}
			r.Init(defaultAuthOpts)
			return r
		},
	}
}

type array []*swarming.SwarmingRpcsStringPair

func (a array) Len() int { return len(a) }
func (a array) Less(i, j int) bool {
	return (a[i].Key < a[j].Key) ||
		(a[i].Key == a[j].Key && a[i].Value < a[j].Value)
}
func (a array) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// mapToArray converts a stringmapflag.Value into an array of
// swarming.SwarmingRpcsStringPair, sorted by key and then value.
func mapToArray(m stringmapflag.Value) []*swarming.SwarmingRpcsStringPair {
	a := make([]*swarming.SwarmingRpcsStringPair, 0, len(m))
	for k, v := range m {
		a = append(a, &swarming.SwarmingRpcsStringPair{Key: k, Value: v})
	}

	sort.Sort(array(a))
	return a
}

// listToStringListPairArray converts a stringlistflag.Flag into an array of
// swarming.SwarmingRpcsStringListPair, sorted by key.
func listToStringListPairArray(m stringlistflag.Flag) []*swarming.SwarmingRpcsStringListPair {
	prefixes := make(map[string][]string)
	for _, f := range m {
		kv := strings.SplitN(f, "=", 2)
		prefixes[kv[0]] = append(prefixes[kv[0]], kv[1])
	}

	a := make([]*swarming.SwarmingRpcsStringListPair, 0, len(prefixes))

	for key, value := range prefixes {
		a = append(a, &swarming.SwarmingRpcsStringListPair{
			Key:   key,
			Value: value,
		})
	}

	sort.Slice(a, func(i, j int) bool {
		return a[i].Key < a[j].Key
	})
	return a
}

// namePartFromDimensions creates a string from a map of dimensions that can
// be used as part of the task name.  The dimensions are first sorted as
// described in mapToArray().
func namePartFromDimensions(m stringmapflag.Value) string {
	a := mapToArray(m)
	pairs := make([]string, 0, len(a))
	for i := 0; i < len(a); i++ {
		pairs = append(pairs, fmt.Sprintf("%s=%s", a[i].Key, a[i].Value))
	}
	return strings.Join(pairs, "_")
}

type containmentType string

func (c *containmentType) String() string {
	return string(*c)
}

func (c *containmentType) Set(v string) error {
	return containmentChoices.FlagSet(c, v)
}

var containmentChoices = flagenum.Enum{
	"none":       containmentType("NONE"),
	"auto":       containmentType("AUTO"),
	"job_object": containmentType("JOB_OBJECT"),
}

type triggerRun struct {
	commonFlags

	// Task properties.
	isolateServer             string
	namespace                 string
	isolated                  string
	dimensions                stringmapflag.Value
	env                       stringmapflag.Value
	envPrefix                 stringlistflag.Flag
	idempotent                bool
	lowerPriority             bool
	containmentType           containmentType
	limitProcesses            int64
	limitTotalCommittedMemory int64
	namedCache                stringmapflag.Value
	hardTimeout               int64
	ioTimeout                 int64
	cipdPackage               stringmapflag.Value
	outputs                   common.Strings
	serviceAccount            string
	relativeCwd               string

	// Task request.
	taskName       string
	priority       int64
	tags           common.Strings
	user           string
	expiration     int64
	enableResultDB bool
	realm          string

	// Other.
	rawCmd   bool
	dumpJSON string
}

func (c *triggerRun) Init(defaultAuthOpts auth.Options) {
	c.commonFlags.Init(defaultAuthOpts)

	// Task properties.
	c.Flags.StringVar(&c.isolateServer, "isolate-server", "", "URL of the Isolate Server to use.")
	c.Flags.StringVar(&c.isolateServer, "I", "", "Alias for -isolate-server.")
	c.Flags.StringVar(&c.namespace, "namespace", "default-gzip", "The namespace to use on the Isolate Server.")
	c.Flags.StringVar(&c.isolated, "isolated", "", "Hash of the .isolated to grab from the isolate server.")
	c.Flags.StringVar(&c.isolated, "s", "", "Alias for -isolated.")
	c.Flags.Var(&c.dimensions, "dimension", "Dimension to select the right kind of bot. In the form of `key=value`")
	c.Flags.Var(&c.dimensions, "d", "Alias for -dimension.")
	c.Flags.Var(&c.env, "env", "Environment variables to set.")
	c.Flags.Var(&c.envPrefix, "env-prefix", "Environment prefixes to set.")
	c.Flags.BoolVar(&c.idempotent, "idempotent", false, "When set, the server will actively try to find a previous task with the same parameter and return this result instead if possible.")
	c.Flags.BoolVar(&c.lowerPriority, "lower-priority", false, "When set, the task will run with a lower process priority.")
	c.containmentType = "NONE"
	c.Flags.Var(&c.containmentType, "containment-type", "Specify which type of process containment to use. Choices are: "+containmentChoices.Choices())
	c.Flags.Int64Var(&c.limitProcesses, "limit-processes", 0, "When set, limit the maximum number of concurrent processes the task can create.")
	c.Flags.Int64Var(&c.limitTotalCommittedMemory, "limit-total-committed-memory", 0, "When set, limit the maximum total amount of memory committed by the processes in the task.")
	c.Flags.Int64Var(&c.hardTimeout, "hard-timeout", 60*60, "Seconds to allow the task to complete.")
	c.Flags.Int64Var(&c.ioTimeout, "io-timeout", 20*60, "Seconds to allow the task to be silent.")
	c.Flags.Var(&c.cipdPackage, "cipd-package",
		"(repeatable) CIPD packages to install on the swarming bot. This takes a parameter of `[subdir:]pkgname=version`. "+
			"Using an empty version will remove the package. The subdir is optional and defaults to '.'.")
	c.Flags.Var(&c.namedCache, "named-cache", "This takes a parameter of `name=cachedir`.")
	c.Flags.Var(&c.outputs, "output", "(repeatable) Specify an output file or directory that can be retrieved via collect.")
	c.Flags.StringVar(&c.relativeCwd, "relative-cwd", "", "Use this flag instead of the isolated 'relative_cwd'; requires -raw-cmd.")
	c.Flags.StringVar(&c.serviceAccount, "service-account", "",
		`Email of a service account to run the task as, or literal "bot" string to indicate that the task should use the same account the bot itself is using to authenticate to Swarming. Don't use task service accounts if not given (default).`)

	// Task request.
	c.Flags.StringVar(&c.taskName, "task-name", "", "Display name of the task. Defaults to <base_name>/<dimensions>/<isolated hash>/<timestamp> if an  isolated file is provided, if a hash is provided, it defaults to <user>/<dimensions>/<isolated hash>/<timestamp>")
	c.Flags.Int64Var(&c.priority, "priority", 200, "The lower value, the more important the task.")
	c.Flags.Var(&c.tags, "tag", "Tags to assign to the task.")
	c.Flags.StringVar(&c.user, "user", "", "User associated with the task. Defaults to authenticated user on the server.")
	c.Flags.Int64Var(&c.expiration, "expiration", 6*60*60, "Seconds to allow the task to be pending for a bot to run before this task request expires.")
	c.Flags.BoolVar(&c.enableResultDB, "enable-resultdb", false, "Enable ResultDB for this task.")
	c.Flags.StringVar(&c.realm, "realm", "", "Realm name for this task.")

	// Other.
	c.Flags.BoolVar(&c.rawCmd, "raw-cmd", false, "When set, the command after -- is run on the bot. Note that this overrides any command in the .isolated file.")
	c.Flags.StringVar(&c.dumpJSON, "dump-json", "", "Dump details about the triggered task(s) to this file as json.")
}

func (c *triggerRun) Parse(args []string) error {
	var err error
	if err := c.commonFlags.Parse(); err != nil {
		return err
	}

	// Validate options and args.
	if c.dimensions == nil {
		return errors.Reason("please at least specify one dimension").Err()
	}

	if c.rawCmd && len(args) == 0 {
		return errors.Reason("arguments with -raw-cmd should be passed after -- as command delimiter").Err()
	} else if !c.rawCmd && len(c.isolated) == 0 {
		return errors.Reason("please use -isolated to specify hash or -raw-cmd").Err()
	}

	if len(c.user) == 0 {
		c.user = os.Getenv("USER")
	}

	return err
}

func (c *triggerRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.Parse(args); err != nil {
		printError(a, err)
		return 1
	}
	if err := c.main(a, args, env); err != nil {
		printError(a, err)
		return 1
	}
	return 0
}

func (c *triggerRun) main(a subcommands.Application, args []string, env subcommands.Env) error {
	start := time.Now()
	ctx, cancel := context.WithCancel(c.defaultFlags.MakeLoggingContext(os.Stderr))
	signals.HandleInterrupt(cancel)

	request, err := c.processTriggerOptions(args, env)
	if err != nil {
		return errors.Annotate(err, "failed to process trigger options").Err()
	}

	service, err := c.createSwarmingClient(ctx)
	if err != nil {
		return err
	}

	result, err := service.NewTask(ctx, request)
	if err != nil {
		return err
	}

	if c.dumpJSON != "" {
		dump, err := os.Create(c.dumpJSON)
		if err != nil {
			return err
		}
		defer dump.Close()

		data := triggerResults{Tasks: []*swarming.SwarmingRpcsTaskRequestMetadata{result}}
		b, err := json.MarshalIndent(&data, "", "  ")
		if err != nil {
			return errors.Annotate(err, "marshalling trigger result").Err()
		}

		_, err = dump.Write(b)
		if err != nil {
			return errors.Annotate(err, "writing json dump").Err()
		}

		if !c.defaultFlags.Quiet {
			fmt.Println("To collect results use:")
			fmt.Printf("  swarming collect -server %s -requests-json %s\n", c.serverURL, c.dumpJSON)
		}
	} else if !c.defaultFlags.Quiet {
		fmt.Println("To collect results use:")
		fmt.Printf("  swarming collect -server %s %s", c.serverURL, result.TaskId)
		fmt.Println()
	}

	if !c.defaultFlags.Quiet {
		fmt.Println("You can also see the task status in")
		fmt.Printf("  %s/task?id=%s\n", c.serverURL, result.TaskId)
	}

	duration := time.Since(start)
	log.Printf("Duration: %s\n", duration.Round(time.Millisecond))
	return nil
}

func (c *triggerRun) processTriggerOptions(args []string, env subcommands.Env) (*swarming.SwarmingRpcsNewTaskRequest, error) {
	var inputsRefs *swarming.SwarmingRpcsFilesRef
	var commands []string
	var extraArgs []string

	if c.rawCmd {
		commands = args
	} else {
		extraArgs = args
	}

	if c.taskName == "" {
		c.taskName = fmt.Sprintf("%s/%s", c.user, namePartFromDimensions(c.dimensions))
	}

	if c.isolated != "" {
		if len(c.taskName) == 0 {
			c.taskName = fmt.Sprintf("%s/%s", c.taskName, c.isolated)
		}
		inputsRefs = &swarming.SwarmingRpcsFilesRef{
			Isolated:       c.isolated,
			Isolatedserver: c.isolateServer,
			Namespace:      c.namespace,
		}
	}

	properties := swarming.SwarmingRpcsTaskProperties{
		Command:              commands,
		RelativeCwd:          c.relativeCwd,
		Dimensions:           mapToArray(c.dimensions),
		Env:                  mapToArray(c.env),
		EnvPrefixes:          listToStringListPairArray(c.envPrefix),
		ExecutionTimeoutSecs: c.hardTimeout,
		ExtraArgs:            extraArgs,
		GracePeriodSecs:      30,
		Idempotent:           c.idempotent,
		InputsRef:            inputsRefs,
		Outputs:              c.outputs,
		IoTimeoutSecs:        c.ioTimeout,
		Containment: &swarming.SwarmingRpcsContainment{
			LowerPriority:             c.lowerPriority,
			ContainmentType:           string(c.containmentType),
			LimitProcesses:            c.limitProcesses,
			LimitTotalCommittedMemory: c.limitTotalCommittedMemory,
		},
	}

	if len(c.cipdPackage) > 0 {
		pkgs := []*swarming.SwarmingRpcsCipdPackage{}
		for k, v := range c.cipdPackage {
			s := strings.SplitN(k, ":", 2)
			pkg := swarming.SwarmingRpcsCipdPackage{
				PackageName: s[len(s)-1],
				Version:     v,
			}
			if len(s) > 1 {
				pkg.Path = s[0]
			}
			pkgs = append(pkgs, &pkg)
		}
		properties.CipdInput = &swarming.SwarmingRpcsCipdInput{Packages: pkgs}
	}

	for name, path := range c.namedCache {
		properties.Caches = append(properties.Caches,
			&swarming.SwarmingRpcsCacheEntry{
				Name: name,
				Path: path,
			},
		)
	}

	randomUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Annotate(err, "failed to get random UUID").Err()
	}

	return &swarming.SwarmingRpcsNewTaskRequest{
		ExpirationSecs: c.expiration,
		Name:           c.taskName,
		ParentTaskId:   env["SWARMING_TASK_ID"].Value,
		Priority:       c.priority,
		Properties:     &properties,
		ServiceAccount: c.serviceAccount,
		Tags:           c.tags,
		User:           c.user,
		RequestUuid:    randomUUID.String(),
		Resultdb: &swarming.SwarmingRpcsResultDBCfg{
			Enable: c.enableResultDB,
		},
		Realm: c.realm,
	}, nil
}
