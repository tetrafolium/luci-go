// Copyright 2017 The LUCI Authors.
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

package buildbucket

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/auth/identity"
	"go.chromium.org/luci/common/data/strpair"
	"go.chromium.org/luci/common/errors"
)

// TagBuilder is the key of builder name tag.
const TagBuilder = "builder"

// Build is a buildbucket build.
// It is a more type-safe version of buildbucket.ApiCommonBuildMessage.
type Build struct {
	// fields set at the build creation time, immutable.

	ID int64
	// Address is an alternative identifier of the build.
	// If the build has build number, it is "<bucket>/<builder>/<number>".
	// Otherwise it is "<id>".
	Address      string
	CreationTime time.Time
	CreatedBy    identity.Identity
	Bucket       string
	Builder      string
	// Number identifies the build within the builder.
	// Build numbers are monotonically increasing, mostly contiguous.
	//
	// The type is *int to prevent accidental confusion
	// of valid build number 0 with absence of the number (zero value).
	Number *int
	// BuildSets is parsed "buildset" tag values.
	//
	// If a buildset is present in tags, but not recognized
	// it won't be included here.
	BuildSets        []BuildSet
	Tags             strpair.Map
	Input            Input
	CanaryPreference CanaryPreference

	// fields that can change during build lifetime

	Status           Status
	StatusChangeTime time.Time
	URL              string
	StartTime        time.Time
	UpdateTime       time.Time
	Canary           bool

	// fields set on build completion

	CompletionTime time.Time
	Output         Output
}

// RunDuration returns duration between build start and completion.
func (b *Build) RunDuration() (duration time.Duration, ok bool) {
	if b.StartTime.IsZero() || b.CompletionTime.IsZero() {
		return 0, false
	}
	return b.CompletionTime.Sub(b.StartTime), true
}

// SchedulingDuration returns duration between build creation and start.
func (b *Build) SchedulingDuration() (duration time.Duration, ok bool) {
	if b.CompletionTime.IsZero() || b.StartTime.IsZero() {
		return 0, false
	}
	return b.StartTime.Sub(b.CompletionTime), true
}

// Properties is data provided by users, opaque to LUCI services.
// The value must be JSON marshalable/unmarshalable into/out from a
// JSON object.
//
// When using an unmarshaling function, such as (*Build).ParseMessage,
// if the user knows the properties they need, they may set the
// value to a json-compatible struct. The unmarshaling function will try
// to unmarshal the properties into the struct. Otherwise, the unmarshaling
// function will use a generic type, e.g. map[string]interface{}.
//
// Example:
//
//   var props struct {
//     A string
//   }
//   var build buildbucket.Build
//   build.Input.Properties = &props
//   if err := build.ParseMessage(msg); err != nil {
//     return err
//   }
//   println(props.A)
type Properties interface{}

// Input is the input to the builder.
type Input struct {
	// Properties is opaque data passed to the build.
	// For recipe-based builds, this is build properties.
	Properties Properties
}

// Output is build output.
type Output struct {
	Properties Properties

	// TODO(nodir, iannucci): replace type "error" with a new type that
	// represents a stack of errors emitted by different layers of the system,
	// where each error has
	// - domain string, e.g. "kitchen"
	// - reason string, e.g. kitchen-specific error code
	// - message string: human readable error
	// - meta: a proto.Struct with random data provided by the layer
	// The new type must implement error so that the change is
	// backward compatible.
	Err error // populated in builds with status StatusError
}

// ParseMessage parses a build message to Build.
//
// Numeric values in JSON-formatted fields, e.g. property values, are parsed as
// json.Number.
//
// If an error is returned, the state of b is undefined.
func (b *Build) ParseMessage(msg *buildbucket.ApiCommonBuildMessage) error {
	status, err := ParseStatus(msg)
	if err != nil {
		return err
	}

	createdBy, err := identity.MakeIdentity(msg.CreatedBy)
	if err != nil {
		return err
	}

	tags := strpair.ParseMap(msg.Tags)
	builder := tags.Get(TagBuilder)

	canaryPref, err := parseEndpointsCanaryPreference(msg.CanaryPreference)
	if err != nil {
		return err
	}

	address := tags.Get("build_address")
	var number *int
	if address == "" {
		address = strconv.FormatInt(msg.Id, 10)
	} else {
		parts := strings.Split(address, "/")
		if len(parts) != 3 || parts[0] != msg.Bucket || parts[1] != builder {
			return fmt.Errorf("invalid build_address %q", address)
		}
		num, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("invalid build_address %q", address)
		}
		number = &num
	}

	input := struct{ Properties interface{} }{b.Input.Properties}
	if err := parseJSON(msg.ParametersJson, &input); err != nil {
		return errors.Annotate(err, "invalid msg.ParametersJson").Err()
	}

	output := struct {
		Properties interface{}
		Error      struct {
			Message string
		}
	}{Properties: b.Output.Properties}
	if err := parseJSON(msg.ResultDetailsJson, &output); err != nil {
		return errors.Annotate(err, "invalid msg.ResultDetailsJson").Err()
	}
	var outErr error
	if output.Error.Message != "" {
		outErr = errors.New(output.Error.Message)
	}

	var bs []BuildSet
	for _, t := range tags[TagBuildSet] {
		if parsed := ParseBuildSet(t); parsed != nil {
			bs = append(bs, parsed)
		}
	}

	*b = Build{
		ID:               msg.Id,
		Address:          address,
		CreationTime:     ParseTimestamp(msg.CreatedTs),
		CreatedBy:        createdBy,
		Bucket:           msg.Bucket,
		Builder:          builder,
		Number:           number,
		BuildSets:        bs,
		Tags:             tags,
		CanaryPreference: canaryPref,
		Input: Input{
			Properties: input.Properties,
		},

		Status:           status,
		StatusChangeTime: ParseTimestamp(msg.StatusChangedTs),
		URL:              msg.Url,
		StartTime:        ParseTimestamp(msg.StartedTs),
		UpdateTime:       ParseTimestamp(msg.UpdatedTs),
		Canary:           msg.Canary,

		CompletionTime: ParseTimestamp(msg.CompletedTs),
		Output: Output{
			Properties: output.Properties,
			Err:        outErr,
		},
	}
	return nil
}

// PutRequest converts b to a build creation request.
//
// If a buildset is present in both b.BuildSets and b.Map, it is deduped.
// Returned value has zero ClientOperationId.
// Returns an error if properties could not be marshaled to JSON.
func (b *Build) PutRequest() (*buildbucket.ApiPutRequestMessage, error) {
	tags := b.Tags.Copy()
	tags.Del(TagBuilder) // buildbucket adds it automatically
	for _, bs := range b.BuildSets {
		s := bs.String()
		if !tags.Contains(TagBuildSet, s) {
			tags.Add(TagBuildSet, s)
		}
	}

	canaryPref, err := b.CanaryPreference.endpointsString()
	if err != nil {
		return nil, err
	}
	msg := &buildbucket.ApiPutRequestMessage{
		Bucket:           b.Bucket,
		Tags:             tags.Format(),
		CanaryPreference: canaryPref,
	}

	parameters := map[string]interface{}{
		"builder_name": b.Builder,
		"properties":   b.Input.Properties,
		// keep this synced with marshaling error annotation
	}
	if data, err := json.Marshal(parameters); err != nil {
		// realistically, only properties may cause this.
		return nil, errors.Annotate(err, "marshaling properties").Err()
	} else {
		msg.ParametersJson = string(data)
	}

	return msg, nil
}

func parseJSON(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	dec := json.NewDecoder(strings.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}

// ParseTimestamp parses a buildbucket timestamp.
func ParseTimestamp(usec int64) time.Time {
	if usec == 0 {
		return time.Time{}
	}
	// TODO(nodir): will start to overflow in year ~2250+, fix it then.
	return time.Unix(0, usec*1000).UTC()
}

// FormatTimestamp converts t to a buildbucket timestamp.
func FormatTimestamp(t time.Time) int64 {
	return t.UnixNano() / 1000
}
