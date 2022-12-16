package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bytebase/bytebase/api"
)

func TestDeparseCreateTable(t *testing.T) {
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
			want: "CREATE TABLE `t1` (\n  `id` INT NOT NULL\n);\n",
		},
		{
			name: "create table t1 with column name of SQL keyword",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t1",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "type",
								Type: "int",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `type` INT NOT NULL\n);\n",
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
			want: "CREATE TABLE `t1` (\n  `id` INT NOT NULL\n) ENGINE=InnoDB;\n\nCREATE TABLE `t2` (\n  `id` INT NOT NULL\n) ENGINE=InnoDB;\n",
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
								Name:       "id",
								Type:       "int",
								Comment:    "ID",
								HasDefault: true,
								Default:    defaultValue,
								Nullable:   true,
							},
							{
								Name:    "name",
								Type:    "varchar(32)",
								Comment: "Name",
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `id` INT COMMENT 'ID' DEFAULT '0',\n  `name` VARCHAR(32) COMMENT 'Name' NOT NULL\n) ENGINE=InnoDB;\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseAlterTable(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "alter table t1 and add column id, id_card",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "id_card",
								Type: "int",
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t1` ADD COLUMN (`id` INT NOT NULL, `id_card` INT NOT NULL);\n",
		},
		{
			name: "alter table t1 and modify column id",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						ChangeColumnList: []*api.ChangeColumnContext{
							{
								OldName:  "id",
								NewName:  "id",
								Type:     "int",
								Comment:  "Name",
								Nullable: true,
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t1` CHANGE COLUMN `id` `id` INT COMMENT 'Name';\n",
		},
		{
			name: "alter table t1 and drop column id",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						DropColumnList: []*api.DropColumnContext{
							{
								Name: "id",
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t1` DROP COLUMN `id`;\n",
		},
		{
			name: "alter table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
						},
						ChangeColumnList: []*api.ChangeColumnContext{
							{
								OldName: "id_card",
								NewName: "id_card2",
								Type:    "int",
								Comment: "ID Card",
							},
						},
						DropColumnList: []*api.DropColumnContext{
							{
								Name: "email",
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t1` DROP COLUMN `email`, ADD COLUMN (`id` INT NOT NULL), CHANGE COLUMN `id_card` `id_card2` INT COMMENT 'ID Card' NOT NULL;\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseRenameTable(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "rename table name t1 to t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				RenameTableList: []*api.RenameTableContext{
					{
						OldName: "t1",
						NewName: "t2",
					},
				},
			},
			want: "RENAME TABLE `t1` TO `t2`;\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseAlterAndRenameTable(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "alter table t1 and rename to t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "id_card",
								Type: "int",
							},
						},
					},
				},
				RenameTableList: []*api.RenameTableContext{
					{
						OldName: "t1",
						NewName: "t2",
					},
				},
			},
			want: "ALTER TABLE `t1` ADD COLUMN (`id` INT NOT NULL, `id_card` INT NOT NULL);\n\nRENAME TABLE `t1` TO `t2`;\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseDropTable(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "drop table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				DropTableList: []*api.DropTableContext{
					{
						Name: "t1",
					},
				},
			},
			want: "DROP TABLE IF EXISTS `t1`;\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}
