package tidb

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
		Collation: "utf8mb4_bin",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "bigint", Default: "AUTO_RANDOM(5)"},
		},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"id"}, Primary: true, Unique: true, Visible: true},
		},
	}
	dbMeta := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{{
			Tables: []*metadatapb.TableMetadata{table},
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
				return schema.GetDatabaseDefinition(storepb.Engine_TIDB, schema.GetDefinitionContext{PrintHeader: true}, dbMeta)
			},
		},
		{
			name:     "table",
			contains: "CREATE TABLE `t`",
			get: func() (string, error) {
				return schema.GetTableDefinition(storepb.Engine_TIDB, "", table, nil)
			},
		},
		{
			name:     "view",
			contains: "CREATE VIEW `v`",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_TIDB, "", &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT `id` FROM `t`"})
			},
		},
		{
			name:     "function",
			contains: "CREATE FUNCTION `f`() RETURNS int RETURN 1;;",
			get: func() (string, error) {
				return schema.GetFunctionDefinition(storepb.Engine_TIDB, "", &metadatapb.FunctionMetadata{Name: "f", Definition: "CREATE FUNCTION `f`() RETURNS int RETURN 1"})
			},
		},
		{
			name:     "procedure",
			contains: "CREATE PROCEDURE `p`() BEGIN END;;",
			get: func() (string, error) {
				return schema.GetProcedureDefinition(storepb.Engine_TIDB, "", &metadatapb.ProcedureMetadata{Name: "p", Definition: "CREATE PROCEDURE `p`() BEGIN END"})
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
