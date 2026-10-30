package oceanbase

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestGetDatabaseDefinition pins the DDL this package writes for each kind of object,
// and that OceanBase reaches it through the registry. The input is hand-written
// metadata rather than a schema text, so no generator branch is capped by what the
// parser happens to extract; parsing is covered in get_database_metadata_test.go.
func TestGetDatabaseDefinition(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name: "t",
		Columns: []*metadatapb.ColumnMetadata{
			{Name: "id", Type: "int"},
			{Name: "name", Type: "varchar(64)", Nullable: true, Default: "'anon'"},
		},
		Indexes: []*metadatapb.IndexMetadata{
			{Name: "PRIMARY", Expressions: []string{"id"}, Primary: true, Unique: true},
		},
	}
	const tableDDL = "--\n" +
		"-- Table structure for `t`\n" +
		"--\n" +
		"CREATE TABLE `t` (\n" +
		"  `id` int NOT NULL,\n" +
		"  `name` varchar(64) DEFAULT 'anon',\n" +
		"  PRIMARY KEY (`id`)\n" +
		");\n\n"

	for _, tc := range []struct {
		name string
		want string
		get  func() (string, error)
	}{
		{
			name: "table",
			want: tableDDL,
			get: func() (string, error) {
				return schema.GetTableDefinition(storepb.Engine_OCEANBASE, "", table, nil)
			},
		},
		{
			name: "view",
			want: "--\n-- View structure for `v`\n--\nCREATE OR REPLACE VIEW `v` AS SELECT `id` FROM `t`;\n\n",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_OCEANBASE, "", &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT `id` FROM `t`"})
			},
		},
		{
			name: "function",
			want: "--\n-- Function structure for `f`\n--\nCREATE FUNCTION `f`() RETURNS int RETURN 1;;\nDELIMITER ;\n\n",
			get: func() (string, error) {
				return schema.GetFunctionDefinition(storepb.Engine_OCEANBASE, "", &metadatapb.FunctionMetadata{Name: "f", Definition: "CREATE FUNCTION `f`() RETURNS int RETURN 1"})
			},
		},
		{
			name: "procedure",
			want: "--\n-- Procedure structure for `p`\n--\nDELIMITER ;;\nCREATE PROCEDURE `p`() BEGIN END;;\nDELIMITER ;\n\n",
			get: func() (string, error) {
				return schema.GetProcedureDefinition(storepb.Engine_OCEANBASE, "", &metadatapb.ProcedureMetadata{Name: "p", Definition: "CREATE PROCEDURE `p`() BEGIN END"})
			},
		},
		{
			name: "database",
			want: tableDDL,
			get: func() (string, error) {
				return schema.GetDatabaseDefinition(storepb.Engine_OCEANBASE, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{
					Name:    "db",
					Schemas: []*metadatapb.SchemaMetadata{{Name: "", Tables: []*metadatapb.TableMetadata{table}}},
				})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, err := tc.get()
			require.NoError(t, err)
			require.Equal(t, tc.want, definition)
		})
	}
}
