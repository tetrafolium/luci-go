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

package dm

import (
	"github.com/golang/protobuf/proto"

	"github.com/tetrafolium/luci-go/gae/service/datastore"
)

var _ datastore.PropertyConverter = (*AbnormalFinish)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'AbnormalFinish' to serialize to an unindexed '[]byte' when used with the
// "github.com/tetrafolium/luci-go/gae" library.
func (p *AbnormalFinish) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'AbnormalFinish' when used with the "github.com/tetrafolium/luci-go/gae" library.
func (p *AbnormalFinish) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}

var _ datastore.PropertyConverter = (*Execution_Auth)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'Execution_Auth' to serialize to an unindexed '[]byte' when used with the
// "github.com/tetrafolium/luci-go/gae" library.
func (p *Execution_Auth) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'Execution_Auth' when used with the "github.com/tetrafolium/luci-go/gae" library.
func (p *Execution_Auth) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}

var _ datastore.PropertyConverter = (*JsonResult)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'JsonResult' to serialize to an unindexed '[]byte' when used with the
// "github.com/tetrafolium/luci-go/gae" library.
func (p *JsonResult) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'JsonResult' when used with the "github.com/tetrafolium/luci-go/gae" library.
func (p *JsonResult) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}

var _ datastore.PropertyConverter = (*Quest_Desc)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'Quest_Desc' to serialize to an unindexed '[]byte' when used with the
// "github.com/tetrafolium/luci-go/gae" library.
func (p *Quest_Desc) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'Quest_Desc' when used with the "github.com/tetrafolium/luci-go/gae" library.
func (p *Quest_Desc) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}

var _ datastore.PropertyConverter = (*Quest_TemplateSpec)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'Quest_TemplateSpec' to serialize to an unindexed '[]byte' when used with the
// "github.com/tetrafolium/luci-go/gae" library.
func (p *Quest_TemplateSpec) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'Quest_TemplateSpec' when used with the "github.com/tetrafolium/luci-go/gae" library.
func (p *Quest_TemplateSpec) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}

var _ datastore.PropertyConverter = (*Result)(nil)

// ToProperty implements datastore.PropertyConverter. It causes an embedded
// 'Result' to serialize to an unindexed '[]byte' when used with the
// "github.com/tetrafolium/luci-go/gae" library.
func (p *Result) ToProperty() (prop datastore.Property, err error) {
	data, err := proto.Marshal(p)
	if err == nil {
		prop.SetValue(data, datastore.NoIndex)
	}
	return
}

// FromProperty implements datastore.PropertyConverter. It parses a '[]byte'
// into an embedded 'Result' when used with the "github.com/tetrafolium/luci-go/gae" library.
func (p *Result) FromProperty(prop datastore.Property) error {
	data, err := prop.Project(datastore.PTBytes)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data.([]byte), p)
}
