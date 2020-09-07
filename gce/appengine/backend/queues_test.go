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

package backend

import (
	"context"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/genproto/googleapis/type/dayofweek"

	"github.com/tetrafolium/luci-go/appengine/tq"
	"github.com/tetrafolium/luci-go/appengine/tq/tqtesting"
	"github.com/tetrafolium/luci-go/common/clock/testclock"
	"github.com/tetrafolium/luci-go/common/data/rand/mathrand"
	"github.com/tetrafolium/luci-go/gae/impl/memory"
	"github.com/tetrafolium/luci-go/gae/service/datastore"

	"github.com/tetrafolium/luci-go/gce/api/config/v1"
	"github.com/tetrafolium/luci-go/gce/api/projects/v1"
	"github.com/tetrafolium/luci-go/gce/api/tasks/v1"
	"github.com/tetrafolium/luci-go/gce/appengine/model"
	"github.com/tetrafolium/luci-go/gce/appengine/testing/roundtripper"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestQueues(t *testing.T) {
	t.Parallel()

	Convey("queues", t, func() {
		dsp := &tq.Dispatcher{}
		registerTasks(dsp)
		rt := &roundtripper.JSONRoundTripper{}
		gce, err := compute.New(&http.Client{Transport: rt})
		So(err, ShouldBeNil)
		c := withCompute(withDispatcher(memory.Use(context.Background()), dsp), gce)
		datastore.GetTestable(c).AutoIndex(true)
		datastore.GetTestable(c).Consistent(true)
		tqt := tqtesting.GetTestable(c, dsp)
		tqt.CreateQueues()

		Convey("countVMs", func() {
			Convey("invalid", func() {
				Convey("nil", func() {
					err := countVMs(c, nil)
					So(err, ShouldErrLike, "unexpected payload")
				})

				Convey("empty", func() {
					err := countVMs(c, &tasks.CountVMs{})
					So(err, ShouldErrLike, "ID is required")
				})
			})

			Convey("valid", func() {
				err := countVMs(c, &tasks.CountVMs{
					Id: "id",
				})
				So(err, ShouldBeNil)
			})
		})

		Convey("createVM", func() {
			c, _ = testclock.UseTime(c, testclock.TestTimeUTC)

			Convey("invalid", func() {
				Convey("nil", func() {
					err := createVM(c, nil)
					So(err, ShouldErrLike, "unexpected payload")
				})

				Convey("empty", func() {
					err := createVM(c, &tasks.CreateVM{})
					So(err, ShouldErrLike, "is required")
				})

				Convey("ID", func() {
					err := createVM(c, &tasks.CreateVM{
						Config: "config",
					})
					So(err, ShouldErrLike, "ID is required")
				})

				Convey("config", func() {
					err := createVM(c, &tasks.CreateVM{
						Id: "id",
					})
					So(err, ShouldErrLike, "config is required")
				})
			})

			Convey("valid", func() {
				Convey("nil", func() {
					err := createVM(c, &tasks.CreateVM{
						Id:     "id",
						Index:  2,
						Config: "config",
					})
					So(err, ShouldBeNil)
					v := &model.VM{
						ID: "id",
					}
					So(datastore.Get(c, v), ShouldBeNil)
					So(v.Index, ShouldEqual, 2)
					So(v.Config, ShouldEqual, "config")
				})

				Convey("empty", func() {
					err := createVM(c, &tasks.CreateVM{
						Id:         "id",
						Attributes: &config.VM{},
						Index:      2,
						Config:     "config",
					})
					So(err, ShouldBeNil)
					v := &model.VM{
						ID: "id",
					}
					So(datastore.Get(c, v), ShouldBeNil)
					So(v.Index, ShouldEqual, 2)
				})

				Convey("non-empty", func() {
					c = mathrand.Set(c, rand.New(rand.NewSource(1)))
					err := createVM(c, &tasks.CreateVM{
						Id: "id",
						Attributes: &config.VM{
							Disk: []*config.Disk{
								{
									Image: "image",
								},
							},
						},
						Index:  2,
						Config: "config",
						Prefix: "prefix",
					})
					So(err, ShouldBeNil)
					v := &model.VM{
						ID: "id",
					}
					So(datastore.Get(c, v), ShouldBeNil)
					So(v, ShouldResemble, &model.VM{
						ID: "id",
						Attributes: config.VM{
							Disk: []*config.Disk{
								{
									Image: "image",
								},
							},
						},
						AttributesIndexed: []string{
							"disk.image:image",
						},
						Config:     "config",
						Configured: testclock.TestTimeUTC.Unix(),
						Hostname:   "prefix-2-fpll",
						Index:      2,
						Prefix:     "prefix",
					})
				})

				Convey("not updated", func() {
					datastore.Put(c, &model.VM{
						ID: "id",
						Attributes: config.VM{
							Zone: "zone",
						},
						Drained: true,
					})
					err := createVM(c, &tasks.CreateVM{
						Id: "id",
						Attributes: &config.VM{
							Project: "project",
						},
						Config: "config",
						Index:  2,
					})
					So(err, ShouldBeNil)
					v := &model.VM{
						ID: "id",
					}
					So(datastore.Get(c, v), ShouldBeNil)
					So(v, ShouldResemble, &model.VM{
						ID: "id",
						Attributes: config.VM{
							Zone: "zone",
						},
						Drained: true,
					})
				})

				Convey("sets zone", func() {
					err := createVM(c, &tasks.CreateVM{
						Id: "id",
						Attributes: &config.VM{
							Disk: []*config.Disk{
								{
									Type: "{{.Zone}}/type",
								},
							},
							MachineType: "{{.Zone}}/type",
							Zone:        "zone",
						},
						Config: "config",
						Index:  2,
					})
					So(err, ShouldBeNil)
					v := &model.VM{
						ID: "id",
					}
					So(datastore.Get(c, v), ShouldBeNil)
					So(v.Attributes, ShouldResemble, config.VM{
						Disk: []*config.Disk{
							{
								Type: "zone/type",
							},
						},
						MachineType: "zone/type",
						Zone:        "zone",
					})
				})
			})
		})

		Convey("drainVM", func() {
			Convey("invalid", func() {
				Convey("config", func() {
					err := drainVM(c, &model.VM{
						ID: "id",
					})
					So(err, ShouldErrLike, "failed to fetch config")
				})
			})

			Convey("valid", func() {
				Convey("config", func() {
					Convey("drained", func() {
						datastore.Put(c, &model.Config{
							ID: "config",
							Config: config.Config{
								CurrentAmount: 2,
							},
						})
						v := &model.VM{
							ID:      "id",
							Config:  "config",
							Drained: true,
						}
						So(datastore.Put(c, v), ShouldBeNil)
						So(drainVM(c, v), ShouldBeNil)
						So(v.Drained, ShouldBeTrue)
						So(datastore.Get(c, v), ShouldBeNil)
						So(v.Drained, ShouldBeTrue)
					})

					Convey("deleted", func() {
						v := &model.VM{
							ID:     "id",
							Config: "config",
						}
						So(datastore.Put(c, v), ShouldBeNil)
						So(drainVM(c, v), ShouldBeNil)
						So(v.Drained, ShouldBeTrue)
						So(datastore.Get(c, v), ShouldBeNil)
						So(v.Drained, ShouldBeTrue)
					})

					Convey("amount", func() {
						Convey("unspecified", func() {
							datastore.Put(c, &model.Config{
								ID: "config",
							})
							v := &model.VM{
								ID:     "id",
								Config: "config",
							}
							So(datastore.Put(c, v), ShouldBeNil)
							So(drainVM(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeTrue)
							So(err, ShouldBeNil)
							So(datastore.Get(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeTrue)
						})

						Convey("lesser", func() {
							datastore.Put(c, &model.Config{
								ID: "config",
								Config: config.Config{
									CurrentAmount: 1,
								},
							})
							v := &model.VM{
								ID:     "id",
								Config: "config",
								Index:  2,
							}
							So(datastore.Put(c, v), ShouldBeNil)
							So(drainVM(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeTrue)
							So(datastore.Get(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeTrue)
						})

						Convey("equal", func() {
							datastore.Put(c, &model.Config{
								ID: "config",
								Config: config.Config{
									CurrentAmount: 2,
								},
							})
							v := &model.VM{
								ID:     "id",
								Config: "config",
								Index:  2,
							}
							So(datastore.Put(c, v), ShouldBeNil)
							So(drainVM(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeTrue)
							So(datastore.Get(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeTrue)
						})

						Convey("greater", func() {
							datastore.Put(c, &model.Config{
								ID: "config",
								Config: config.Config{
									CurrentAmount: 3,
								},
							})
							v := &model.VM{
								ID:     "id",
								Config: "config",
								Index:  2,
							}
							So(datastore.Put(c, v), ShouldBeNil)
							So(drainVM(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeFalse)
							So(datastore.Get(c, v), ShouldBeNil)
							So(v.Drained, ShouldBeFalse)
						})
					})
				})

				Convey("deleted", func() {
					v := &model.VM{
						ID:     "id",
						Config: "config",
					}
					So(drainVM(c, v), ShouldBeNil)
					So(v.Drained, ShouldBeTrue)
					So(datastore.Get(c, v), ShouldEqual, datastore.ErrNoSuchEntity)
				})
			})
		})

		Convey("expandConfig", func() {
			Convey("invalid", func() {
				Convey("nil", func() {
					err := expandConfig(c, nil)
					So(err, ShouldErrLike, "unexpected payload")
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
				})

				Convey("empty", func() {
					err := expandConfig(c, &tasks.ExpandConfig{})
					So(err, ShouldErrLike, "ID is required")
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
				})

				Convey("missing", func() {
					err := expandConfig(c, &tasks.ExpandConfig{
						Id: "id",
					})
					So(err, ShouldErrLike, "failed to fetch config")
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
					cfg := &model.Config{
						ID: "id",
					}
					So(datastore.Get(c, cfg), ShouldEqual, datastore.ErrNoSuchEntity)
				})
			})

			Convey("valid", func() {
				Convey("none", func() {
					So(datastore.Put(c, &model.Config{
						ID: "id",
						Config: config.Config{
							Attributes: &config.VM{
								Project: "project",
							},
							Prefix: "prefix",
						},
					}), ShouldBeNil)
					err := expandConfig(c, &tasks.ExpandConfig{
						Id: "id",
					})
					So(err, ShouldBeNil)
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
					cfg := &model.Config{
						ID: "id",
					}
					So(datastore.Get(c, cfg), ShouldBeNil)
					So(cfg.Config.CurrentAmount, ShouldEqual, 0)
				})

				Convey("default", func() {
					So(datastore.Put(c, &model.Config{
						ID: "id",
						Config: config.Config{
							Attributes: &config.VM{
								Project: "project",
							},
							Amount: &config.Amount{
								Min: 3,
								Max: 3,
							},
							Prefix: "prefix",
						},
					}), ShouldBeNil)
					err := expandConfig(c, &tasks.ExpandConfig{
						Id: "id",
					})
					So(err, ShouldBeNil)
					So(tqt.GetScheduledTasks(), ShouldHaveLength, 3)
					cfg := &model.Config{
						ID: "id",
					}
					So(datastore.Get(c, cfg), ShouldBeNil)
					So(cfg.Config.CurrentAmount, ShouldEqual, 3)
				})

				Convey("schedule", func() {
					So(datastore.Put(c, &model.Config{
						ID: "id",
						Config: config.Config{
							Attributes: &config.VM{
								Project: "project",
							},
							Amount: &config.Amount{
								Min: 2,
								Max: 2,
								Change: []*config.Schedule{
									{
										Min: 5,
										Max: 5,
										Length: &config.TimePeriod{
											Time: &config.TimePeriod_Duration{
												Duration: "1h",
											},
										},
										Start: &config.TimeOfDay{
											Day:  dayofweek.DayOfWeek_MONDAY,
											Time: "1:00",
										},
									},
								},
							},
							Prefix: "prefix",
						},
					}), ShouldBeNil)

					Convey("default", func() {
						now := time.Time{}
						So(now.Weekday(), ShouldEqual, time.Monday)
						c, _ = testclock.UseTime(c, now)
						err := expandConfig(c, &tasks.ExpandConfig{
							Id: "id",
						})
						So(err, ShouldBeNil)
						So(tqt.GetScheduledTasks(), ShouldHaveLength, 2)
						cfg := &model.Config{
							ID: "id",
						}
						So(datastore.Get(c, cfg), ShouldBeNil)
						So(cfg.Config.CurrentAmount, ShouldEqual, 2)
					})

					Convey("scheduled", func() {
						now := time.Time{}.Add(time.Hour)
						So(now.Weekday(), ShouldEqual, time.Monday)
						So(now.Hour(), ShouldEqual, 1)
						c, _ = testclock.UseTime(c, now)
						err := expandConfig(c, &tasks.ExpandConfig{
							Id: "id",
						})
						So(err, ShouldBeNil)
						So(tqt.GetScheduledTasks(), ShouldHaveLength, 5)
						cfg := &model.Config{
							ID: "id",
						}
						So(datastore.Get(c, cfg), ShouldBeNil)
						So(cfg.Config.CurrentAmount, ShouldEqual, 5)
					})
				})
			})
		})

		Convey("reportQuota", func() {
			Convey("invalid", func() {
				Convey("nil", func() {
					err := reportQuota(c, nil)
					So(err, ShouldErrLike, "unexpected payload")
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
				})

				Convey("empty", func() {
					err := reportQuota(c, &tasks.ReportQuota{})
					So(err, ShouldErrLike, "ID is required")
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
				})

				Convey("missing", func() {
					err := reportQuota(c, &tasks.ReportQuota{
						Id: "id",
					})
					So(err, ShouldErrLike, "failed to fetch project")
					So(tqt.GetScheduledTasks(), ShouldBeEmpty)
				})
			})

			Convey("valid", func() {
				rt.Handler = func(req interface{}) (int, interface{}) {
					return http.StatusOK, &compute.RegionList{
						Items: []*compute.Region{
							{
								Name: "ignore",
							},
							{
								Name: "region",
								Quotas: []*compute.Quota{
									{
										Limit:  100.0,
										Metric: "ignore",
										Usage:  0.0,
									},
									{
										Limit:  100.0,
										Metric: "metric",
										Usage:  25.0,
									},
								},
							},
						},
					}
				}
				datastore.Put(c, &model.Project{
					ID: "id",
					Config: projects.Config{
						Metric:  []string{"metric"},
						Project: "project",
						Region:  []string{"region"},
					},
				})
				err := reportQuota(c, &tasks.ReportQuota{
					Id: "id",
				})
				So(err, ShouldBeNil)
			})
		})
	})
}
