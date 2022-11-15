package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/api"
)

func TestDeparseDatabaseEdit(t *testing.T) {
	var defaultValue = "0"

	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "create table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t1",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `id` INT NOT NULL\n);",
		},
		{
			name: "create table t1&t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name:   "t1",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
					{
						Name:   "t2",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `id` INT NOT NULL\n) ENGINE=InnoDB;\nCREATE TABLE `t2` (\n  `id` INT NOT NULL\n) ENGINE=InnoDB;",
		},
		{
			name: "create table t1 with name",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name:   "t1",
						Type:   "BASE TABLE",
						Engine: "InnoDB",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Comment:  "ID",
								Default:  &defaultValue,
								Nullable: true,
							},
							{
								Name:    "name",
								Type:    "varchar",
								Comment: "Name",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `id` INT COMMENT 'ID' DEFAULT '0',\n  `name` VARCHAR CHARACTER SET UTF8MB4 COLLATE utf8mb4_bin COMMENT 'Name' NOT NULL\n) ENGINE=InnoDB;",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}
