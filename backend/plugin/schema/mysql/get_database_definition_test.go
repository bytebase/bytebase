package mysql

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinitionRegistry(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name:      "t",
		Engine:    "InnoDB",
		Charset:   "utf8mb4",
		Collation: "utf8mb4_0900_ai_ci",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "varchar(64)", Nullable: true, Default: "'anon'"},
		},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"id"}, Primary: true, Unique: true, Visible: true},
		},
	}
	dbMeta := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{table},
			Views: []*metadatapb.ViewMetadata{
				{Name: "v", Definition: "select `app`.`t`.`id` AS `id` from `app`.`t`"},
			},
		}},
	}

	for _, tc := range []struct {
		name     string
		contains string
		get      func() (string, error)
	}{
		{
			name:     "database with header",
			contains: "FOREIGN_KEY_CHECKS=0;",
			get: func() (string, error) {
				return schema.GetDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{PrintHeader: true}, dbMeta)
			},
		},
		{
			name:     "database in SDL format",
			contains: "CREATE TABLE `t`",
			get: func() (string, error) {
				return schema.GetDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{SDLFormat: true}, dbMeta)
			},
		},
		{
			name:     "table",
			contains: "CREATE TABLE `t`",
			get: func() (string, error) {
				return schema.GetTableDefinition(storepb.Engine_MYSQL, "", table, nil)
			},
		},
		{
			name:     "view",
			contains: "CREATE OR REPLACE VIEW `v`",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_MYSQL, "", &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT `id` FROM `t`"})
			},
		},
		{
			name:     "function",
			contains: "CREATE FUNCTION `f`() RETURNS int RETURN 1;;",
			get: func() (string, error) {
				return schema.GetFunctionDefinition(storepb.Engine_MYSQL, "", &metadatapb.FunctionMetadata{Name: "f", Definition: "CREATE FUNCTION `f`() RETURNS int RETURN 1"})
			},
		},
		{
			name:     "procedure",
			contains: "CREATE PROCEDURE `p`() BEGIN END;;",
			get: func() (string, error) {
				return schema.GetProcedureDefinition(storepb.Engine_MYSQL, "", &metadatapb.ProcedureMetadata{Name: "p", Definition: "CREATE PROCEDURE `p`() BEGIN END"})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			require.Contains(t, got, tc.contains)
		})
	}
}
