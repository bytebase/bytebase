package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/bytebase/bytebase/backend/legacyapi"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
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
								Name:     "id",
								Type:     "int",
								Comment:  "ID",
								Default:  &defaultValue,
								Nullable: true,
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
	var defaultValue = "0"

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
						DropColumnList: []*api.DropColumnContext{
							{
								Name: "name",
							},
						},
						ChangeColumnList: []*api.ChangeColumnContext{
							{
								OldName:  "address",
								NewName:  "address",
								Type:     "int",
								Nullable: false,
								Default:  &defaultValue,
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t1` DROP COLUMN `name`, ADD COLUMN (`id` INT NOT NULL, `id_card` INT NOT NULL), CHANGE COLUMN `address` `address` INT NOT NULL DEFAULT '0';\n",
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
						Name: "t2",
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
			want: "RENAME TABLE `t1` TO `t2`;\n\nALTER TABLE `t2` ADD COLUMN (`id` INT NOT NULL, `id_card` INT NOT NULL);\n",
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

func TestDeparseCreateTableWithPrimaryKey(t *testing.T) {
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
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
						},
						PrimaryKeyList: []string{"id"},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `id` INT NOT NULL,\n  PRIMARY KEY (`id`)\n);\n",
		},
		{
			name: "create table t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t2",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "varchar(255)",
								Nullable: false,
							},
						},
						PrimaryKeyList: []string{"id", "name"},
					},
				},
			},
			want: "CREATE TABLE `t2` (\n  `id` INT NOT NULL,\n  `name` VARCHAR(255) NOT NULL,\n  PRIMARY KEY (`id`, `name`)\n);\n",
		},
		{
			name: "create table t3",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Name: "t3",
						Type: "BASE TABLE",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "varchar(255)",
								Nullable: false,
							},
						},
						PrimaryKeyList: []string{"id"},
					},
				},
			},
			want: "CREATE TABLE `t3` (\n  `id` INT NOT NULL,\n  `name` VARCHAR(255) NOT NULL,\n  PRIMARY KEY (`id`)\n);\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseAlterTableWithPrimaryKey(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "alter table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
						},
						PrimaryKeyList: &[]string{"id"},
					},
				},
			},
			want: "ALTER TABLE `t1` ADD COLUMN (`id` INT NOT NULL), ADD PRIMARY KEY (`id`);\n",
		},
		{
			name: "alter table t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t2",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
						},
						DropPrimaryKey: true,
						PrimaryKeyList: &[]string{},
					},
				},
			},
			want: "ALTER TABLE `t2` ADD COLUMN (`id` INT NOT NULL), DROP PRIMARY KEY;\n",
		},
		{
			name: "alter table t3",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t3",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t3` ADD COLUMN (`id` INT NOT NULL);\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseCreateTableWithForeignKey(t *testing.T) {
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
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "int",
								Nullable: false,
							},
						},
						AddForeignKeyList: []*api.AddForeignKeyContext{
							{
								ColumnList:           []string{"name"},
								ReferencedTable:      "t2",
								ReferencedColumnList: []string{"name"},
							},
						},
					},
				},
			},
			want: "CREATE TABLE `t1` (\n  `id` INT NOT NULL,\n  `name` INT NOT NULL,\n  CONSTRAINT FOREIGN KEY (`name`) REFERENCES `t2` (`name`)\n);\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseAlterTableWithForeignKey(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "create table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "int",
								Nullable: false,
							},
						},
						DropForeignKeyList: []string{"t1_ibkf_1", "t1_ibkf_2"},
						AddForeignKeyList: []*api.AddForeignKeyContext{
							{
								ColumnList:           []string{"name"},
								ReferencedTable:      "t2",
								ReferencedColumnList: []string{"name"},
							},
						},
					},
				},
			},
			want: "ALTER TABLE `t1` ADD COLUMN (`id` INT NOT NULL, `name` INT NOT NULL), DROP FOREIGN KEY `t1_ibkf_1`, DROP FOREIGN KEY `t1_ibkf_2`, ADD CONSTRAINT FOREIGN KEY (`name`) REFERENCES `t2` (`name`);\n",
		},
	}

	mysqlEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := mysqlEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}
