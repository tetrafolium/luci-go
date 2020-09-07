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

package model

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/tetrafolium/luci-go/machine-db/api/common/v1"
	"github.com/tetrafolium/luci-go/machine-db/api/config/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/database"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestVLANs(t *testing.T) {
	Convey("fetch", t, func() {
		db, m, _ := sqlmock.New()
		defer db.Close()
		c := database.With(context.Background(), db)
		selectStmt := `^SELECT id, alias, state, cidr_block FROM vlans$`
		columns := []string{"id", "alias", "state", "cidr_block"}
		rows := sqlmock.NewRows(columns)
		table := &VLANsTable{}

		Convey("query failed", func() {
			m.ExpectQuery(selectStmt).WillReturnError(fmt.Errorf("error"))
			So(table.fetch(c), ShouldErrLike, "failed to select VLANs")
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("empty", func() {
			m.ExpectQuery(selectStmt).WillReturnRows(rows)
			So(table.fetch(c), ShouldBeNil)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			rows.AddRow(1, "vlan 1", common.State_FREE, "127.0.0.1/20")
			rows.AddRow(2, "vlan 2", common.State_SERVING, "192.168.0.1/20")
			m.ExpectQuery(selectStmt).WillReturnRows(rows)
			So(table.fetch(c), ShouldBeNil)
			So(table.current, ShouldResemble, []*VLAN{
				{
					VLAN: config.VLAN{
						Id:        1,
						Alias:     "vlan 1",
						State:     common.State_FREE,
						CidrBlock: "127.0.0.1/20",
					},
				},
				{
					VLAN: config.VLAN{
						Id:        2,
						Alias:     "vlan 2",
						State:     common.State_SERVING,
						CidrBlock: "192.168.0.1/20",
					},
				},
			})
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})
	})

	Convey("computeChanges", t, func() {
		table := &VLANsTable{}
		c := context.Background()

		Convey("empty", func() {
			table.computeChanges(c, nil)
			So(table.additions, ShouldBeEmpty)
			So(table.updates, ShouldBeEmpty)
			So(table.removals, ShouldBeEmpty)
		})

		Convey("addition", func() {
			vlans := []*config.VLAN{
				{
					Id:        1,
					Alias:     "vlan 1",
					State:     common.State_FREE,
					CidrBlock: "127.0.0.1/20",
				},
				{
					Id:        2,
					Alias:     "vlan 2",
					State:     common.State_SERVING,
					CidrBlock: "192.168.0.1/20",
				},
			}
			table.computeChanges(c, vlans)
			So(table.additions, ShouldResemble, []*VLAN{
				{
					VLAN: config.VLAN{
						Id:        1,
						Alias:     "vlan 1",
						State:     common.State_FREE,
						CidrBlock: "127.0.0.1/20",
					},
				},
				{
					VLAN: config.VLAN{
						Id:        2,
						Alias:     "vlan 2",
						State:     common.State_SERVING,
						CidrBlock: "192.168.0.1/20",
					},
				},
			})
			So(table.updates, ShouldBeEmpty)
			So(table.removals, ShouldBeEmpty)
		})

		Convey("update", func() {
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id:        1,
					Alias:     "old alias",
					State:     common.State_FREE,
					CidrBlock: "127.0.0.1/20",
				},
			})
			vlans := []*config.VLAN{
				{
					Id:        table.current[0].Id,
					Alias:     "new alias",
					State:     common.State_SERVING,
					CidrBlock: "192.168.0.1/20",
				},
			}
			table.computeChanges(c, vlans)
			So(table.additions, ShouldBeEmpty)
			So(table.updates, ShouldHaveLength, 1)
			So(table.updates, ShouldResemble, []*VLAN{
				{
					VLAN: config.VLAN{
						Id:        table.current[0].Id,
						Alias:     vlans[0].Alias,
						State:     vlans[0].State,
						CidrBlock: vlans[0].CidrBlock,
					},
				},
			})
			So(table.removals, ShouldBeEmpty)
		})

		Convey("removal", func() {
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			table.computeChanges(c, nil)
			So(table.additions, ShouldBeEmpty)
			So(table.updates, ShouldBeEmpty)
			So(table.removals, ShouldResemble, []*VLAN{
				{
					VLAN: config.VLAN{
						Id: table.current[0].Id,
					},
				},
			})
		})
	})

	Convey("add", t, func() {
		db, m, _ := sqlmock.New()
		defer db.Close()
		c := database.With(context.Background(), db)
		insertStmt := `^INSERT INTO vlans \(id, alias, state, cidr_block\) VALUES \(\?, \?, \?, \?\)$`
		table := &VLANsTable{}

		Convey("empty", func() {
			So(table.add(c), ShouldBeNil)
			So(table.additions, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("prepare failed", func() {
			table.additions = append(table.additions, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			m.ExpectPrepare(insertStmt).WillReturnError(fmt.Errorf("error"))
			So(table.add(c), ShouldErrLike, "failed to prepare statement")
			So(table.additions, ShouldHaveLength, 1)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("exec failed", func() {
			table.additions = append(table.additions, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			m.ExpectPrepare(insertStmt)
			m.ExpectExec(insertStmt).WillReturnError(fmt.Errorf("error"))
			So(table.add(c), ShouldErrLike, "failed to add VLAN")
			So(table.additions, ShouldHaveLength, 1)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			table.additions = append(table.additions, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			m.ExpectPrepare(insertStmt)
			m.ExpectExec(insertStmt).WillReturnResult(sqlmock.NewResult(1, 1))
			So(table.add(c), ShouldBeNil)
			So(table.additions, ShouldBeEmpty)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})
	})

	Convey("remove", t, func() {
		db, m, _ := sqlmock.New()
		defer db.Close()
		c := database.With(context.Background(), db)
		deleteStmt := `^DELETE FROM vlans WHERE id = \?$`
		table := &VLANsTable{}

		Convey("empty", func() {
			So(table.remove(c), ShouldBeNil)
			So(table.removals, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("prepare failed", func() {
			table.removals = append(table.removals, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id: table.removals[0].Id,
				},
			})
			m.ExpectPrepare(deleteStmt).WillReturnError(fmt.Errorf("error"))
			So(table.remove(c), ShouldErrLike, "failed to prepare statement")
			So(table.removals, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("exec failed", func() {
			table.removals = append(table.removals, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id: table.removals[0].Id,
				},
			})
			m.ExpectPrepare(deleteStmt)
			m.ExpectExec(deleteStmt).WillReturnError(fmt.Errorf("error"))
			So(table.remove(c), ShouldErrLike, "failed to remove VLAN")
			So(table.removals, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			table.removals = append(table.removals, &VLAN{
				VLAN: config.VLAN{
					Id: 1,
				},
			})
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id: table.removals[0].Id,
				},
			})
			m.ExpectPrepare(deleteStmt)
			m.ExpectExec(deleteStmt).WillReturnResult(sqlmock.NewResult(1, 1))
			So(table.remove(c), ShouldBeNil)
			So(table.removals, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})
	})

	Convey("update", t, func() {
		db, m, _ := sqlmock.New()
		defer db.Close()
		c := database.With(context.Background(), db)
		updateStmt := `^UPDATE vlans SET alias = \?, state = \?, cidr_block = \? WHERE id = \?$`
		table := &VLANsTable{}

		Convey("empty", func() {
			So(table.update(c), ShouldBeNil)
			So(table.updates, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("prepare failed", func() {
			table.updates = append(table.updates, &VLAN{
				VLAN: config.VLAN{
					Id:        1,
					Alias:     "new alias",
					State:     common.State_SERVING,
					CidrBlock: "192.168.0.1/20",
				},
			})
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id:        table.updates[0].Id,
					Alias:     "old alias",
					State:     common.State_FREE,
					CidrBlock: "127.0.0.1/20",
				},
			})
			m.ExpectPrepare(updateStmt).WillReturnError(fmt.Errorf("error"))
			So(table.update(c), ShouldErrLike, "failed to prepare statement")
			So(table.updates, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("exec failed", func() {
			table.updates = append(table.updates, &VLAN{
				VLAN: config.VLAN{
					Id:        1,
					Alias:     "new alias",
					State:     common.State_SERVING,
					CidrBlock: "192.168.0.1/20",
				},
			})
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id:        table.updates[0].Id,
					Alias:     "old alias",
					State:     common.State_FREE,
					CidrBlock: "127.0.0.1/20",
				},
			})
			m.ExpectPrepare(updateStmt)
			m.ExpectExec(updateStmt).WillReturnError(fmt.Errorf("error"))
			So(table.update(c), ShouldErrLike, "failed to update VLAN")
			So(table.updates, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			table.updates = append(table.updates, &VLAN{
				VLAN: config.VLAN{
					Id:        1,
					Alias:     "new alias",
					State:     common.State_SERVING,
					CidrBlock: "192.168.0.1/20",
				},
			})
			table.current = append(table.current, &VLAN{
				VLAN: config.VLAN{
					Id:        table.updates[0].Id,
					Alias:     "old alias",
					State:     common.State_FREE,
					CidrBlock: "127.0.0.1/20",
				},
			})
			m.ExpectPrepare(updateStmt)
			m.ExpectExec(updateStmt).WillReturnResult(sqlmock.NewResult(1, 1))
			So(table.update(c), ShouldBeNil)
			So(table.updates, ShouldBeEmpty)
			So(table.current, ShouldHaveLength, 1)
			So(table.current[0].Alias, ShouldEqual, "new alias")
			So(table.current[0].State, ShouldEqual, common.State_SERVING)
			So(table.current[0].CidrBlock, ShouldEqual, "192.168.0.1/20")
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})
	})
}
