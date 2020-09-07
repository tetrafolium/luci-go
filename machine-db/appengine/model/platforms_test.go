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

	"github.com/tetrafolium/luci-go/machine-db/api/config/v1"
	"github.com/tetrafolium/luci-go/machine-db/appengine/database"

	. "github.com/smartystreets/goconvey/convey"
	. "github.com/tetrafolium/luci-go/common/testing/assertions"
)

func TestPlatforms(t *testing.T) {
	Convey("fetch", t, func() {
		db, m, _ := sqlmock.New()
		defer db.Close()
		c := database.With(context.Background(), db)
		selectStmt := `^SELECT id, name, description, manufacturer FROM platforms$`
		columns := []string{"id", "name", "description", "manufacturer"}
		rows := sqlmock.NewRows(columns)
		table := &PlatformsTable{}

		Convey("query failed", func() {
			m.ExpectQuery(selectStmt).WillReturnError(fmt.Errorf("error"))
			So(table.fetch(c), ShouldErrLike, "failed to select platforms")
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
			rows.AddRow(1, "platform 1", "description 1", "manufacturer 1")
			rows.AddRow(2, "platform 2", "description 2", "manufacturer 2")
			m.ExpectQuery(selectStmt).WillReturnRows(rows)
			So(table.fetch(c), ShouldBeNil)
			So(table.current, ShouldResemble, []*Platform{
				{
					Platform: config.Platform{
						Name:         "platform 1",
						Description:  "description 1",
						Manufacturer: "manufacturer 1",
					},
					Id: 1,
				},
				{
					Platform: config.Platform{
						Name:         "platform 2",
						Description:  "description 2",
						Manufacturer: "manufacturer 2",
					},
					Id: 2,
				},
			})
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})
	})

	Convey("computeChanges", t, func() {
		table := &PlatformsTable{}
		c := context.Background()

		Convey("empty", func() {
			table.computeChanges(c, nil)
			So(table.additions, ShouldBeEmpty)
			So(table.updates, ShouldBeEmpty)
			So(table.removals, ShouldBeEmpty)
		})

		Convey("addition", func() {
			platforms := []*config.Platform{
				{
					Name:         "platform 1",
					Description:  "description 1",
					Manufacturer: "manufacturer 1",
				},
				{
					Name:         "platform 2",
					Description:  "description 2",
					Manufacturer: "manufacturer 2",
				},
			}
			table.computeChanges(c, platforms)
			So(table.additions, ShouldResemble, []*Platform{
				{
					Platform: config.Platform{
						Name:         "platform 1",
						Description:  "description 1",
						Manufacturer: "manufacturer 1",
					},
				},
				{
					Platform: config.Platform{
						Name:         "platform 2",
						Description:  "description 2",
						Manufacturer: "manufacturer 2",
					},
				},
			})
			So(table.updates, ShouldBeEmpty)
			So(table.removals, ShouldBeEmpty)
		})

		Convey("update", func() {
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name:         "platform",
					Description:  "old description",
					Manufacturer: "old manufacturer",
				},
				Id: 1,
			})
			platforms := []*config.Platform{
				{
					Name:         table.current[0].Name,
					Description:  "new description",
					Manufacturer: "new manufacturer",
				},
			}
			table.computeChanges(c, platforms)
			So(table.additions, ShouldBeEmpty)
			So(table.updates, ShouldHaveLength, 1)
			So(table.updates, ShouldResemble, []*Platform{
				{
					Platform: config.Platform{
						Name:         table.current[0].Name,
						Description:  platforms[0].Description,
						Manufacturer: platforms[0].Manufacturer,
					},
					Id: table.current[0].Id,
				},
			})
			So(table.removals, ShouldBeEmpty)
		})

		Convey("removal", func() {
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name: "platform",
				},
				Id: 1,
			})
			table.computeChanges(c, nil)
			So(table.additions, ShouldBeEmpty)
			So(table.updates, ShouldBeEmpty)
			So(table.removals, ShouldResemble, []*Platform{
				{
					Platform: config.Platform{
						Name: table.current[0].Name,
					},
					Id: 1,
				},
			})
		})
	})

	Convey("add", t, func() {
		db, m, _ := sqlmock.New()
		defer db.Close()
		c := database.With(context.Background(), db)
		insertStmt := `^INSERT INTO platforms \(name, description, manufacturer\) VALUES \(\?, \?, \?\)$`
		table := &PlatformsTable{}

		Convey("empty", func() {
			So(table.add(c), ShouldBeNil)
			So(table.additions, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("prepare failed", func() {
			table.additions = append(table.additions, &Platform{
				Platform: config.Platform{
					Name: "platform",
				},
			})
			m.ExpectPrepare(insertStmt).WillReturnError(fmt.Errorf("error"))
			So(table.add(c), ShouldErrLike, "failed to prepare statement")
			So(table.additions, ShouldHaveLength, 1)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("exec failed", func() {
			table.additions = append(table.additions, &Platform{
				Platform: config.Platform{
					Name: "platform",
				},
			})
			m.ExpectPrepare(insertStmt)
			m.ExpectExec(insertStmt).WillReturnError(fmt.Errorf("error"))
			So(table.add(c), ShouldErrLike, "failed to add platform")
			So(table.additions, ShouldHaveLength, 1)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			table.additions = append(table.additions, &Platform{
				Platform: config.Platform{
					Name: "platform",
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
		deleteStmt := `^DELETE FROM platforms WHERE id = \?$`
		table := &PlatformsTable{}

		Convey("empty", func() {
			So(table.remove(c), ShouldBeNil)
			So(table.removals, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("prepare failed", func() {
			table.removals = append(table.removals, &Platform{
				Platform: config.Platform{
					Name: "platform",
				},
				Id: 1,
			})
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name: table.removals[0].Name,
				},
				Id: table.removals[0].Id,
			})
			m.ExpectPrepare(deleteStmt).WillReturnError(fmt.Errorf("error"))
			So(table.remove(c), ShouldErrLike, "failed to prepare statement")
			So(table.removals, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("exec failed", func() {
			table.removals = append(table.removals, &Platform{
				Platform: config.Platform{
					Name: "platform",
				},
				Id: 1,
			})
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name: table.removals[0].Name,
				},
				Id: table.removals[0].Id,
			})
			m.ExpectPrepare(deleteStmt)
			m.ExpectExec(deleteStmt).WillReturnError(fmt.Errorf("error"))
			So(table.remove(c), ShouldErrLike, "failed to remove platform")
			So(table.removals, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			table.removals = append(table.removals, &Platform{
				Platform: config.Platform{
					Name: "platform",
				},
				Id: 1,
			})
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name: table.removals[0].Name,
				},
				Id: table.removals[0].Id,
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
		updateStmt := `^UPDATE platforms SET description = \?, manufacturer = \? WHERE id = \?$`
		table := &PlatformsTable{}

		Convey("empty", func() {
			So(table.update(c), ShouldBeNil)
			So(table.updates, ShouldBeEmpty)
			So(table.current, ShouldBeEmpty)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("prepare failed", func() {
			table.updates = append(table.updates, &Platform{
				Platform: config.Platform{
					Name:         "platform",
					Description:  "new description",
					Manufacturer: "new manufacturer",
				},
				Id: 1,
			})
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name:         table.updates[0].Name,
					Description:  "old description",
					Manufacturer: "old manufacturer",
				},
				Id: table.updates[0].Id,
			})
			m.ExpectPrepare(updateStmt).WillReturnError(fmt.Errorf("error"))
			So(table.update(c), ShouldErrLike, "failed to prepare statement")
			So(table.updates, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("exec failed", func() {
			table.updates = append(table.updates, &Platform{
				Platform: config.Platform{
					Name:         "platform",
					Description:  "new description",
					Manufacturer: "new manufacturer",
				},
				Id: 1,
			})
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name:         table.updates[0].Name,
					Description:  "old description",
					Manufacturer: "old manufacturer",
				},
				Id: table.updates[0].Id,
			})
			m.ExpectPrepare(updateStmt)
			m.ExpectExec(updateStmt).WillReturnError(fmt.Errorf("error"))
			So(table.update(c), ShouldErrLike, "failed to update platform")
			So(table.updates, ShouldHaveLength, 1)
			So(table.current, ShouldHaveLength, 1)
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})

		Convey("ok", func() {
			table.updates = append(table.updates, &Platform{
				Platform: config.Platform{
					Name:         "platform",
					Description:  "new description",
					Manufacturer: "new manufacturer",
				},
				Id: 1,
			})
			table.current = append(table.current, &Platform{
				Platform: config.Platform{
					Name:         table.updates[0].Name,
					Description:  "old description",
					Manufacturer: "old manufacturer",
				},
				Id: table.updates[0].Id,
			})
			m.ExpectPrepare(updateStmt)
			m.ExpectExec(updateStmt).WillReturnResult(sqlmock.NewResult(1, 1))
			So(table.update(c), ShouldBeNil)
			So(table.updates, ShouldBeEmpty)
			So(table.current, ShouldHaveLength, 1)
			So(table.current[0].Description, ShouldEqual, "new description")
			So(table.current[0].Manufacturer, ShouldEqual, "new manufacturer")
			So(m.ExpectationsWereMet(), ShouldBeNil)
		})
	})
}
