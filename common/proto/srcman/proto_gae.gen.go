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

// AUTOGENERATED: Do not edit

package srcman

import (
	"github.com/golang/protobuf/proto"

	"go.chromium.org/luci/gae/service/datastore"
)

var _ datastore.PropertyConverter = (*Manifest)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'Manifest' to serialize to an unindexed '[]byte' when used with the
// "go.chromium.org/luci/gae" library.
func (p *Manifest) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'Manifest' when used with the "go.chromium.org/luci/gae" library.
func (p *Manifest) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}

var _ datastore.PropertyConverter = (*ManifestDiff)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'ManifestDiff' to serialize to an unindexed '[]byte' when used with the
// "go.chromium.org/luci/gae" library.
func (p *ManifestDiff) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'ManifestDiff' when used with the "go.chromium.org/luci/gae" library.
func (p *ManifestDiff) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}
