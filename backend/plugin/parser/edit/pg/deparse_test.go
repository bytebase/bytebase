package pg

import (
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/bytebase/bytebase/backend/legacyapi"

	// Register PostgreSQL parser engine.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/engine/pg"
)

func TestDeparseRenameSchema(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "renmae schema public",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				RenameSchemaList: []*api.RenameSchemaContext{
					{
						OldName: "public",
						NewName: "protected",
					},
				},
			},
			want: "ALTER SCHEMA \"public\" RENAME TO \"protected\";",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseDropSchema(t *testing.T) {
	tests := []struct {
		name         string
		databaseEdit *api.DatabaseEdit
		want         string
	}{
		{
			name: "drop schema public",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				DropSchemaList: []*api.DropSchemaContext{
					{
						Schema: "public",
					},
				},
			},
			want: "DROP SCHEMA IF EXISTS \"public\" CASCADE;",
		},
		{
			name: "drop schema public and t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				DropSchemaList: []*api.DropSchemaContext{
					{
						Schema: "public",
					},
					{
						Schema: "t1",
					},
				},
			},
			want: "DROP SCHEMA IF EXISTS \"public\" CASCADE;\nDROP SCHEMA IF EXISTS \"t1\" CASCADE;",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

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
				CreateSchemaList: []*api.CreateSchemaContext{
					{
						Schema: "public",
					},
				},
				CreateTableList: []*api.CreateTableContext{
					{
						Schema: "public",
						Name:   "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:    "id",
								Type:    "int",
								Comment: "ID",
								Default: &defaultValue,
							},
						},
					},
				},
			},
			want: "CREATE SCHEMA IF NOT EXISTS \"public\";\nCREATE TABLE \"public\".\"t1\" (\n    \"id\" integer DEFAULT 0 NOT NULL\n);\nCOMMENT ON COLUMN \"public\".\"t1\".\"id\" IS 'ID';",
		},
		{
			name: "create table t1",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				CreateTableList: []*api.CreateTableContext{
					{
						Schema: "public",
						Name:   "t1",
						AddColumnList: []*api.AddColumnContext{
							{
								Name:    "id",
								Type:    "int",
								Default: &defaultValue,
							},
							{
								Name:    "name",
								Type:    "int",
								Default: &defaultValue,
							},
						},
						PrimaryKeyList: []string{"id", "name"},
					},
				},
			},
			want: "CREATE TABLE \"public\".\"t1\" (\n    \"id\" integer DEFAULT 0 NOT NULL,\n    \"name\" integer DEFAULT 0 NOT NULL,\n    PRIMARY KEY (\"id\", \"name\")\n);",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}

func TestDeparseAlterTable(t *testing.T) {
	var defaultType = "TEXT"

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
						AlterColumnList: []*api.AlterColumnContext{
							{
								OldName: "address",
								NewName: "address",
								Type:    &defaultType,
							},
						},
					},
				},
			},
			want: "ALTER TABLE \"t1\"\n    ALTER COLUMN \"address\" SET DATA TYPE text;",
		},
		{
			name: "alter table t2",
			databaseEdit: &api.DatabaseEdit{
				DatabaseID: api.UnknownID,
				AlterTableList: []*api.AlterTableContext{
					{
						Name: "t2",
						AlterColumnList: []*api.AlterColumnContext{
							{
								OldName: "address",
								NewName: "home_address",
							},
						},
					},
				},
			},
			want: "ALTER TABLE \"t2\"\n    RENAME COLUMN \"address\" TO \"home_address\";",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
						Schema:  "public",
						OldName: "t1",
						NewName: "t2",
					},
				},
			},
			want: "ALTER TABLE \"public\".\"t1\"\n    RENAME TO \"t2\";",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
						Schema: "public",
						Name:   "t2",
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
						Schema:  "public",
						OldName: "t1",
						NewName: "t2",
					},
				},
			},
			want: "ALTER TABLE \"public\".\"t1\"\n    RENAME TO \"t2\";\nALTER TABLE \"public\".\"t2\"\n    ADD COLUMN \"id\" integer NOT NULL,\n    ADD COLUMN \"id_card\" integer NOT NULL;",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
						Schema: "public",
						Name:   "t1",
					},
				},
			},
			want: "DROP TABLE IF EXISTS \"public\".\"t1\";",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
			want: "CREATE TABLE \"t1\" (\n    \"id\" integer NOT NULL,\n    PRIMARY KEY (\"id\")\n);",
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
			want: "CREATE TABLE \"t2\" (\n    \"id\" integer NOT NULL,\n    \"name\" character varying(255) NOT NULL,\n    PRIMARY KEY (\"id\", \"name\")\n);",
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
			want: "CREATE TABLE \"t3\" (\n    \"id\" integer NOT NULL,\n    \"name\" character varying(255) NOT NULL,\n    PRIMARY KEY (\"id\")\n);",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
			want: "ALTER TABLE \"t1\"\n    ADD COLUMN \"id\" integer NOT NULL,\n    ADD PRIMARY KEY (\"id\");",
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
						DropPrimaryKeyList: []string{"_pk_1"},
					},
				},
			},
			want: "ALTER TABLE \"t2\"\n    ADD COLUMN \"id\" integer NOT NULL,\n    DROP CONSTRAINT IF EXISTS \"_pk_1\";",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
						Schema: "public",
						Name:   "t1",
						Type:   "BASE TABLE",
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
								ReferencedSchema:     "public",
								ReferencedTable:      "t2",
								ReferencedColumnList: []string{"name"},
							},
						},
					},
				},
			},
			want: "CREATE TABLE \"public\".\"t1\" (\n    \"id\" integer NOT NULL,\n    \"name\" integer NOT NULL,\n    FOREIGN KEY (\"name\") REFERENCES \"public\".\"t2\" (\"name\")\n);",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
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
			want: "ALTER TABLE \"t1\"\n    ADD COLUMN \"id\" integer NOT NULL,\n    ADD COLUMN \"name\" integer NOT NULL,\n    DROP CONSTRAINT IF EXISTS \"t1_ibkf_1\",\n    DROP CONSTRAINT IF EXISTS \"t1_ibkf_2\",\n    ADD FOREIGN KEY (\"name\") REFERENCES \"t2\" (\"name\");",
		},
	}

	postgresEditor := &SchemaEditor{}
	for _, test := range tests {
		stmt, err := postgresEditor.DeparseDatabaseEdit(test.databaseEdit)
		assert.NoError(t, err)
		assert.Equal(t, test.want, stmt)
	}
}
