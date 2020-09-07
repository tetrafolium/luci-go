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

package isolated

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tetrafolium/luci-go/common/errors"
)

// ScatterGather represents a mapping of working directories to relative paths.
//
// The purpose is to represent some notion of "local" vs. "archived" paths.
// All relative paths are relative to both their corresponding working
// directories as well as the root of an archive.
//
// filepath.Join(working dir, relative path) == location of file or directory
// on the system.
//
// relative path == location of file or directory in an archive.
//
// Notably, in such a design, we may not have more than one copy of a relative
// path in the archive, because there is a conflict. In order to efficiently
// check this case at the expense of extra memory, ScatterGather actually
// stores a mapping of relative paths to working directories.
type ScatterGather map[string]string

// Add adds a (working directory, relative path) pair to the ScatterGather.
//
// Add returns an error if the relative path was already added.
func (sc ScatterGather) Add(wd string, rel string) error {
	cleaned := filepath.Clean(rel)
	if _, ok := sc[cleaned]; ok {
		return errors.Reason("name conflict %q", rel).Err()
	}
	sc[cleaned] = wd
	return nil
}

// Set implements the flags.Var interface.
func (sc *ScatterGather) Set(value string) error {
	colon := strings.LastIndexByte(value, ':')
	if colon == -1 {
		return errors.Reason("malformed input %q", value).Err()
	}
	if *sc == nil {
		*sc = ScatterGather{}
	}
	return sc.Add(value[:colon], value[colon+1:])
}

// String implements the Stringer interface.
func (sc *ScatterGather) String() string {
	mapping := make(map[string][]string, len(*sc))
	for item, wd := range *sc {
		mapping[wd] = append(mapping[wd], item)
	}
	return fmt.Sprintf("%v", mapping)
}
