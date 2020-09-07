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

package config

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/config/validation"
	"github.com/tetrafolium/luci-go/machine-db/api/common/v1"
	"github.com/tetrafolium/luci-go/machine-db/api/config/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestValidateDatacentersCfg(t *testing.T) {
	t.Parallel()

	Convey("validateDatacentersCfg", t, func() {
		context := &validation.Context{Context: context.Background()}

		Convey("empty config", func() {
			cfg := &config.Datacenters{}
			validateDatacentersCfg(context, cfg)
			So(context.Finalize(), ShouldBeNil)
		})

		Convey("unnamed file", func() {
			cfg := &config.Datacenters{
				Datacenter: []string{""},
			}
			validateDatacentersCfg(context, cfg)
			So(context.Finalize(), ShouldErrLike, "datacenter filenames are required and must be non-empty")
		})

		Convey("duplicate", func() {
			cfg := &config.Datacenters{
				Datacenter: []string{
					"duplicate.cfg",
					"datacenter.cfg",
					"duplicate.cfg",
				},
			}
			validateDatacentersCfg(context, cfg)
			So(context.Finalize(), ShouldErrLike, "duplicate filename")
		})

		Convey("ok", func() {
			cfg := &config.Datacenters{
				Datacenter: []string{
					"datacenter1.cfg",
					"datacenter2.cfg",
				},
			}
			validateDatacentersCfg(context, cfg)
			So(context.Finalize(), ShouldBeNil)
		})
	})
}

func TestValidateDatacenters(t *testing.T) {
	t.Parallel()

	Convey("validateDatacenters", t, func() {
		context := &validation.Context{Context: context.Background()}

		Convey("empty datacenters config", func() {
			datacenters := map[string]*config.Datacenter{}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldBeNil)
		})

		Convey("unnamed datacenter", func() {
			datacenters := map[string]*config.Datacenter{
				"unnamed.cfg": {},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "datacenter names are required and must be non-empty")
		})

		Convey("duplicate datacenters", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter1.cfg": {
					Name: "duplicate",
				},
				"datacenter2.cfg": {
					Name: "unique",
				},
				"datacenter3.cfg": {
					Name: "duplicate",
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate datacenter")
		})

		Convey("unnamed rack", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "rack names are required and must be non-empty")
		})

		Convey("duplicate racks in the same datacenter", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "duplicate",
						},
						{
							Name: "unique",
						},
						{
							Name: "duplicate",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate rack")
		})

		Convey("duplicate racks in different datacenters", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter1.cfg": {
					Name: "datacenter 1",
					Rack: []*config.Rack{
						{
							Name: "duplicate",
						},
						{
							Name: "unique",
						},
					},
				},
				"datacenter2.cfg": {
					Name: "datacenter 2",
					Rack: []*config.Rack{
						{
							Name: "duplicate",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate rack")
		})

		Convey("unnamed switch", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Switch: []*config.Switch{
								{},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "switch names are required and must be non-empty")
		})

		Convey("duplicate switches in the same rack", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Switch: []*config.Switch{
								{
									Name:  "duplicate",
									Ports: 2,
								},
								{
									Name:  "unique",
									Ports: 4,
								},
								{
									Name:  "duplicate",
									Ports: 8,
								},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate switch")
		})

		Convey("duplicate switches in different racks", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter1.cfg": {
					Name: "datacenter 1",
					Rack: []*config.Rack{
						{
							Name: "rack 1",
							Switch: []*config.Switch{
								{
									Name:  "duplicate",
									Ports: 2,
								},
								{
									Name:  "unique",
									Ports: 4,
								},
							},
						},
					},
				},
				"datacenter2.cfg": {
					Name: "datacenter 2",
					Rack: []*config.Rack{
						{
							Name: "rack 2",
							Switch: []*config.Switch{
								{
									Name:  "duplicate",
									Ports: 8,
								},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate switch")
		})

		Convey("missing switch ports", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Switch: []*config.Switch{
								{
									Name: "switch",
								},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "switches must have at least one port")
		})

		Convey("negative switch ports", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Switch: []*config.Switch{
								{
									Name:  "switch",
									Ports: -1,
								},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "switches must have at least one port")
		})

		Convey("zero switch ports", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Switch: []*config.Switch{
								{
									Name:  "switch",
									Ports: 0,
								},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "switches must have at least one port")
		})

		Convey("excessive switch ports", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Switch: []*config.Switch{
								{
									Name:  "switch",
									Ports: 65536,
								},
							},
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "switches must have at most 65535 ports")
		})

		Convey("unnamed KVM", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
						},
					},
					Kvm: []*config.KVM{
						{
							Rack:       "rack",
							MacAddress: "00:00:00:00:00:01",
							Ipv4:       "0.0.0.1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "KVM names are required and must be non-empty")
		})

		Convey("duplicate KVMs in the same datacenter", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "duplicate",
							Rack:       "rack",
							MacAddress: "01:02:03:04:05:06",
							Ipv4:       "127.0.0.1",
						},
						{
							Name:       "unique",
							Rack:       "rack",
							MacAddress: "11:22:33:44:55:66",
							Ipv4:       "127.0.0.2",
						},
						{
							Name:       "duplicate",
							Rack:       "rack",
							MacAddress: "00:00:00:00:00:01",
							Ipv4:       "0.0.0.1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate KVM")
		})

		Convey("duplicate KVMs in different datacenters", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter1.cfg": {
					Name: "datacenter 1",
					Rack: []*config.Rack{
						{
							Name: "rack 1",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "duplicate",
							Rack:       "rack 1",
							MacAddress: "01:02:03:04:05:06",
							Ipv4:       "127.0.0.1",
						},
						{
							Name:       "unique",
							Rack:       "rack 1",
							MacAddress: "11:22:33:44:55:66",
							Ipv4:       "127.0.0.2",
						},
					},
				},
				"datacenter2.cfg": {
					Name: "datacenter 2",
					Rack: []*config.Rack{
						{
							Name: "rack 2",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "duplicate",
							Rack:       "rack 2",
							MacAddress: "00:00:00:00:00:01",
							Ipv4:       "0.0.0.1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "duplicate KVM")
		})

		Convey("invalid rack", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "kvm",
							MacAddress: "00:00:00:00:00:01",
							Ipv4:       "0.0.0.1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "does not exist")
		})

		Convey("invalid KVM", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
							Kvm:  "asdf",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "kvm",
							Rack:       "rack",
							MacAddress: "00:00:00:00:00:01",
							Ipv4:       "0.0.0.1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "does not exist")
		})

		Convey("invalid MAC-48 address", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "kvm",
							Rack:       "rack",
							MacAddress: "01:02:03:04:05:06:07:08",
							Ipv4:       "127.0.0.1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "invalid MAC-48 address")
		})

		Convey("invalid IPv4 address", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter.cfg": {
					Name: "datacenter",
					Rack: []*config.Rack{
						{
							Name: "rack",
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "kvm",
							Rack:       "rack",
							MacAddress: "01:02:03:04:05:06",
							Ipv4:       "2001:db8:a0b:12f0::1",
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldErrLike, "invalid IPv4 address")
		})

		Convey("ok", func() {
			datacenters := map[string]*config.Datacenter{
				"datacenter1.cfg": {
					Name:        "datacenter 1",
					Description: "A description of datacenter 1",
					State:       common.State_SERVING,
					Rack: []*config.Rack{
						{
							Name:        "rack 1",
							Description: "A description of rack 1",
							State:       common.State_SERVING,
							Switch: []*config.Switch{
								{
									Name:        "switch 1",
									Description: "A description of switch 1",
									Ports:       4,
									State:       common.State_SERVING,
								},
							},
						},
						{
							Name:  "rack 2",
							State: common.State_DECOMMISSIONED,
						},
					},
					Kvm: []*config.KVM{
						{
							Name:        "kvm 1",
							Rack:        "rack 1",
							Description: "A description of kvm 1",
							MacAddress:  "01:02:03:04:05:06",
							Ipv4:        "127.0.0.1",
							State:       common.State_PRERELEASE,
						},
						{
							Name:       "kvm 2",
							Rack:       "rack 2",
							MacAddress: "11:22:33:44:55:66",
							Ipv4:       "127.0.0.2",
						},
					},
				},
				"datacenter2.cfg": {
					Name:  "datacenter 2",
					State: common.State_PRERELEASE,
				},
				"datacenter3.cfg": {
					Name:  "datacenter 3",
					State: common.State_SERVING,
					Rack: []*config.Rack{
						{
							Name:  "rack 3",
							State: common.State_SERVING,
							Switch: []*config.Switch{
								{
									Name:  "switch 2",
									Ports: 8,
									State: common.State_SERVING,
								},
								{
									Name:  "switch 3",
									Ports: 16,
									State: common.State_SERVING,
								},
							},
						},
					},
					Kvm: []*config.KVM{
						{
							Name:       "kvm 3",
							Rack:       "rack 3",
							MacAddress: "00:00:00:00:00:01",
							Ipv4:       "0.0.0.1",
							State:      common.State_SERVING,
						},
					},
				},
			}
			validateDatacenters(context, datacenters)
			So(context.Finalize(), ShouldBeNil)
		})
	})
}
