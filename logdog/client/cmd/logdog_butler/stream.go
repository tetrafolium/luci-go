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

package main

import (
	"flag"
	"fmt"

	"github.com/tetrafolium/luci-go/logdog/api/logpb"
	"github.com/tetrafolium/luci-go/logdog/client/butlerlib/streamproto"
	"github.com/tetrafolium/luci-go/logdog/common/types"
)

// Holds common command-line stream configuration parameters.
type streamConfig struct {
	streamproto.Flags
}

func (s *streamConfig) addFlags(fs *flag.FlagSet) {
	// Set defaults.
	if s.ContentType == "" {
		s.ContentType = string(types.ContentTypeText)
	}
	s.Type = streamproto.StreamType(logpb.StreamType_TEXT)

	fs.Var(&s.Name, "name", "The name of the stream")
	fs.StringVar(&s.ContentType, "content-type", s.ContentType,
		"The stream content type.")
	fs.Var(&s.Type, "type",
		fmt.Sprintf("Input stream type. Choices are: %s",
			streamproto.StreamTypeFlagEnum.Choices()))
	fs.Var(&s.Tags, "tag", "Add a key[=value] tag.")
}

// Converts command-line parameters into a stream.Config.
func (s streamConfig) properties() *logpb.LogStreamDescriptor {
	if s.ContentType == "" {
		// Choose content type based on format.
		s.ContentType = string(s.Type.DefaultContentType())
	}
	return s.Descriptor()
}
