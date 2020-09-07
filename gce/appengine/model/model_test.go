// Copyright 2018 The LUCI Authors.
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

package model

import (
	"context"
	"testing"

	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/gce/api/config/v1"
	"github.com/tetrafolium/luci-go/gce/api/projects/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	Convey("Config", t, func() {
		c := memory.Use(context.Background())
		cfg := &Config{ID: "id"}
		err := datastore.Get(c, cfg)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		err = datastore.Put(c, &Config{
			ID: "id",
			Config: config.Config{
				Attributes: &config.VM{
					Disk: []*config.Disk{
						{
							Image: "image",
						},
					},
					Project: "project",
				},
				Prefix: "prefix",
			},
		})
		So(err, ShouldBeNil)

		err = datastore.Get(c, cfg)
		So(err, ShouldBeNil)
		So(cfg, ShouldResemble, &Config{
			ID: "id",
			Config: config.Config{
				Attributes: &config.VM{
					Disk: []*config.Disk{
						{
							Image: "image",
						},
					},
					Project: "project",
				},
				Prefix: "prefix",
			},
		})
	})
}

func TestProject(t *testing.T) {
	t.Parallel()

	Convey("Project", t, func() {
		c := memory.Use(context.Background())
		p := &Project{ID: "id"}
		err := datastore.Get(c, p)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		err = datastore.Put(c, &Project{
			ID: "id",
			Config: projects.Config{
				Metric: []string{
					"metric-1",
					"metric-2",
				},
				Project: "project",
				Region: []string{
					"region-1",
					"region-2",
				},
			},
		})
		So(err, ShouldBeNil)

		err = datastore.Get(c, p)
		So(err, ShouldBeNil)
		So(p.Config, ShouldResemble, projects.Config{
			Metric: []string{
				"metric-1",
				"metric-2",
			},
			Project: "project",
			Region: []string{
				"region-1",
				"region-2",
			},
		})
	})
}

func TestVM(t *testing.T) {
	t.Parallel()

	Convey("VM", t, func() {
		c := memory.Use(context.Background())
		v := &VM{
			ID: "id",
		}
		err := datastore.Get(c, v)
		So(err, ShouldEqual, datastore.ErrNoSuchEntity)

		err = datastore.Put(c, &VM{
			ID: "id",
			Attributes: config.VM{
				Project: "project",
			},
		})
		So(err, ShouldBeNil)

		So(datastore.Get(c, v), ShouldBeNil)
		So(v, ShouldResemble, &VM{
			ID: "id",
			Attributes: config.VM{
				Project: "project",
			},
		})

		Convey("IndexAttributes", func() {
			v.IndexAttributes()
			So(v.AttributesIndexed, ShouldBeEmpty)

			v := &VM{
				ID: "id",
				Attributes: config.VM{
					Disk: []*config.Disk{
						{
							Image: "global/images/image-1",
						},
						{
							Image: "projects/project/global/images/image-2",
						},
					},
				},
			}
			v.IndexAttributes()
			So(v.AttributesIndexed, ShouldResemble, []string{"disk.image:image-1", "disk.image:image-2"})
		})
	})

	Convey("getDisks", t, func() {
		Convey("zero", func() {
			Convey("nil", func() {
				v := &VM{}
				d := v.getDisks()
				So(d, ShouldHaveLength, 0)
			})

			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						Disk: []*config.Disk{},
					},
				}
				d := v.getDisks()
				So(d, ShouldHaveLength, 0)
			})
		})

		Convey("non-zero", func() {
			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						Disk: []*config.Disk{
							{},
						},
					},
				}
				d := v.getDisks()
				So(d, ShouldHaveLength, 1)
				So(d[0].AutoDelete, ShouldBeTrue)
				So(d[0].Boot, ShouldBeTrue)
				So(d[0].InitializeParams.DiskSizeGb, ShouldEqual, 0)
			})

			Convey("non-empty", func() {
				v := &VM{
					Attributes: config.VM{
						Disk: []*config.Disk{
							{
								Image: "image",
							},
						},
					},
				}
				d := v.getDisks()
				So(d, ShouldHaveLength, 1)
				So(d[0].InitializeParams.SourceImage, ShouldEqual, "image")
			})

			Convey("multi", func() {
				v := &VM{
					Attributes: config.VM{
						Disk: []*config.Disk{
							{},
							{},
						},
					},
				}
				d := v.getDisks()
				So(d, ShouldHaveLength, 2)
				So(d[0].Boot, ShouldBeTrue)
				So(d[1].Boot, ShouldBeFalse)
			})

			Convey("scratch", func() {
				v := &VM{
					Attributes: config.VM{
						Disk: []*config.Disk{
							{
								Type: "zones/zone/diskTypes/pd-ssd",
							},
							{
								Type: "zones/zone/diskTypes/local-ssd",
							},
							{
								Type: "zones/zone/diskTypes/pd-standard",
							},
						},
					},
				}
				d := v.getDisks()
				So(d, ShouldHaveLength, 3)
				So(d[0].Type, ShouldEqual, "")
				So(d[1].Type, ShouldEqual, "SCRATCH")
				So(d[2].Type, ShouldEqual, "")
			})
		})
	})

	Convey("getMetadata", t, func() {
		Convey("nil", func() {
			v := &VM{}
			m := v.getMetadata()
			So(m, ShouldBeNil)
		})

		Convey("empty", func() {
			v := &VM{
				Attributes: config.VM{
					Metadata: []*config.Metadata{},
				},
			}
			m := v.getMetadata()
			So(m, ShouldBeNil)
		})

		Convey("non-empty", func() {
			Convey("empty-nil", func() {
				v := &VM{
					Attributes: config.VM{
						Metadata: []*config.Metadata{
							{},
						},
					},
				}
				m := v.getMetadata()
				So(m.Items, ShouldHaveLength, 1)
				So(m.Items[0].Key, ShouldEqual, "")
				So(m.Items[0].Value, ShouldBeNil)
			})

			Convey("key-nil", func() {
				v := &VM{
					Attributes: config.VM{
						Metadata: []*config.Metadata{
							{
								Metadata: &config.Metadata_FromText{
									FromText: "key",
								},
							},
						},
					},
				}
				m := v.getMetadata()
				So(m.Items, ShouldHaveLength, 1)
				So(m.Items[0].Key, ShouldEqual, "key")
				So(m.Items[0].Value, ShouldBeNil)
			})

			Convey("key-empty", func() {
				v := &VM{
					Attributes: config.VM{
						Metadata: []*config.Metadata{
							{
								Metadata: &config.Metadata_FromText{
									FromText: "key:",
								},
							},
						},
					},
				}
				m := v.getMetadata()
				So(m.Items, ShouldHaveLength, 1)
				So(m.Items[0].Key, ShouldEqual, "key")
				So(*m.Items[0].Value, ShouldEqual, "")
			})

			Convey("key-value", func() {
				v := &VM{
					Attributes: config.VM{
						Metadata: []*config.Metadata{
							{
								Metadata: &config.Metadata_FromText{
									FromText: "key:value",
								},
							},
						},
					},
				}
				m := v.getMetadata()
				So(m.Items, ShouldHaveLength, 1)
				So(m.Items[0].Key, ShouldEqual, "key")
				So(*m.Items[0].Value, ShouldEqual, "value")
			})

			Convey("empty-value", func() {
				v := &VM{
					Attributes: config.VM{
						Metadata: []*config.Metadata{
							{
								Metadata: &config.Metadata_FromText{
									FromText: ":value",
								},
							},
						},
					},
				}
				m := v.getMetadata()
				So(m.Items, ShouldHaveLength, 1)
				So(m.Items[0].Key, ShouldEqual, "")
				So(*m.Items[0].Value, ShouldEqual, "value")
			})

			Convey("from file", func() {
				v := &VM{
					Attributes: config.VM{
						Metadata: []*config.Metadata{
							{
								Metadata: &config.Metadata_FromFile{
									FromFile: "key:file",
								},
							},
						},
					},
				}
				m := v.getMetadata()
				So(m.Items, ShouldHaveLength, 1)
				So(m.Items[0].Key, ShouldEqual, "")
				So(m.Items[0].Value, ShouldBeNil)
			})
		})
	})

	Convey("getNetworkInterfaces", t, func() {
		Convey("zero", func() {
			Convey("nil", func() {
				v := &VM{}
				n := v.getNetworkInterfaces()
				So(n, ShouldHaveLength, 0)
			})

			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						NetworkInterface: []*config.NetworkInterface{},
					},
				}
				n := v.getNetworkInterfaces()
				So(n, ShouldHaveLength, 0)
			})
		})

		Convey("non-zero", func() {
			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						NetworkInterface: []*config.NetworkInterface{
							{},
						},
					},
				}
				n := v.getNetworkInterfaces()
				So(n, ShouldHaveLength, 1)
				So(n[0].AccessConfigs, ShouldHaveLength, 0)
				So(n[0].Network, ShouldEqual, "")
			})

			Convey("non-empty", func() {
				Convey("network", func() {
					v := &VM{
						Attributes: config.VM{
							NetworkInterface: []*config.NetworkInterface{
								{
									AccessConfig: []*config.AccessConfig{},
									Network:      "network",
								},
							},
						},
					}
					n := v.getNetworkInterfaces()
					So(n, ShouldHaveLength, 1)
					So(n[0].AccessConfigs, ShouldBeNil)
					So(n[0].Network, ShouldEqual, "network")
				})

				Convey("access configs", func() {
					v := &VM{
						Attributes: config.VM{
							NetworkInterface: []*config.NetworkInterface{
								{
									AccessConfig: []*config.AccessConfig{
										{
											Type: config.AccessConfigType_ONE_TO_ONE_NAT,
										},
									},
								},
							},
						},
					}
					n := v.getNetworkInterfaces()
					So(n, ShouldHaveLength, 1)
					So(n[0].AccessConfigs, ShouldHaveLength, 1)
					So(n[0].AccessConfigs[0].Type, ShouldEqual, "ONE_TO_ONE_NAT")
				})
			})
		})
	})

	Convey("getServiceAccounts", t, func() {
		Convey("zero", func() {
			Convey("nil", func() {
				v := &VM{}
				s := v.getServiceAccounts()
				So(s, ShouldHaveLength, 0)
			})

			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						ServiceAccount: []*config.ServiceAccount{},
					},
				}
				s := v.getServiceAccounts()
				So(s, ShouldHaveLength, 0)
			})
		})

		Convey("non-zero", func() {
			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						ServiceAccount: []*config.ServiceAccount{
							{},
						},
					},
				}
				s := v.getServiceAccounts()
				So(s, ShouldHaveLength, 1)
				So(s[0].Email, ShouldEqual, "")
				So(s[0].Scopes, ShouldHaveLength, 0)
			})

			Convey("non-empty", func() {
				Convey("email", func() {
					v := &VM{
						Attributes: config.VM{
							ServiceAccount: []*config.ServiceAccount{
								{
									Email: "email",
									Scope: []string{},
								},
							},
						},
					}
					s := v.getServiceAccounts()
					So(s, ShouldHaveLength, 1)
					So(s[0].Email, ShouldEqual, "email")
					So(s[0].Scopes, ShouldHaveLength, 0)
				})

				Convey("scopes", func() {
					v := &VM{
						Attributes: config.VM{
							ServiceAccount: []*config.ServiceAccount{
								{
									Scope: []string{
										"scope",
									},
								},
							},
						},
					}
					s := v.getServiceAccounts()
					So(s, ShouldHaveLength, 1)
					So(s[0].Email, ShouldEqual, "")
					So(s[0].Scopes, ShouldHaveLength, 1)
					So(s[0].Scopes[0], ShouldEqual, "scope")
				})
			})
		})

	})

	Convey("getTags", t, func() {
		Convey("zero", func() {
			Convey("nil", func() {
				v := &VM{}
				t := v.getTags()
				So(t, ShouldBeNil)
			})

			Convey("empty", func() {
				v := &VM{
					Attributes: config.VM{
						Tag: []string{},
					},
				}
				t := v.getTags()
				So(t, ShouldBeNil)
			})
		})

		Convey("non-zero", func() {
			v := &VM{
				Attributes: config.VM{
					Tag: []string{
						"tag",
					},
				},
			}
			t := v.getTags()
			So(t.Items, ShouldHaveLength, 1)
			So(t.Items[0], ShouldEqual, "tag")
		})
	})

	Convey("GetInstance", t, func() {
		Convey("empty", func() {
			v := &VM{}
			i := v.GetInstance()
			So(i.Disks, ShouldHaveLength, 0)
			So(i.MachineType, ShouldEqual, "")
			So(i.Metadata, ShouldBeNil)
			So(i.MinCpuPlatform, ShouldEqual, "")
			So(i.NetworkInterfaces, ShouldHaveLength, 0)
			So(i.ServiceAccounts, ShouldBeNil)
			So(i.Tags, ShouldBeNil)
		})

		Convey("non-empty", func() {
			v := &VM{
				Attributes: config.VM{
					Disk: []*config.Disk{
						{
							Image: "image",
							Size:  100,
						},
					},
					MachineType:    "type",
					MinCpuPlatform: "plat",
					NetworkInterface: []*config.NetworkInterface{
						{
							AccessConfig: []*config.AccessConfig{
								{},
							},
							Network: "network",
						},
					},
				},
			}
			i := v.GetInstance()
			So(i.Disks, ShouldHaveLength, 1)
			So(i.MachineType, ShouldEqual, "type")
			So(i.Metadata, ShouldBeNil)
			So(i.MinCpuPlatform, ShouldEqual, "plat")
			So(i.NetworkInterfaces, ShouldHaveLength, 1)
			So(i.ServiceAccounts, ShouldBeNil)
			So(i.Tags, ShouldBeNil)
		})
	})
}
