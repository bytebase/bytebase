package mariadb

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

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
				return schema.GetTableDefinition(storepb.Engine_MARIADB, "", table, nil)
			},
		},
		{
			name: "view",
			want: "--\n-- View structure for `v`\n--\nCREATE OR REPLACE VIEW `v` AS SELECT `id` FROM `t`;\n\n",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_MARIADB, "", &metadatapb.ViewMetadata{Name: "v", Definition: "SELECT `id` FROM `t`"})
			},
		},
		{
			name: "function",
			want: "--\n-- Function structure for `f`\n--\nDELIMITER ;;\nCREATE FUNCTION `f`() RETURNS int RETURN 1;;\nDELIMITER ;\n\n",
			get: func() (string, error) {
				return schema.GetFunctionDefinition(storepb.Engine_MARIADB, "", &metadatapb.FunctionMetadata{Name: "f", Definition: "CREATE FUNCTION `f`() RETURNS int RETURN 1"})
			},
		},
		{
			name: "procedure",
			want: "--\n-- Procedure structure for `p`\n--\nDELIMITER ;;\nCREATE PROCEDURE `p`() BEGIN END;;\nDELIMITER ;\n\n",
			get: func() (string, error) {
				return schema.GetProcedureDefinition(storepb.Engine_MARIADB, "", &metadatapb.ProcedureMetadata{Name: "p", Definition: "CREATE PROCEDURE `p`() BEGIN END"})
			},
		},
		{
			name: "sequence",
			want: "--\n-- Sequence structure for `s`\n--\nCREATE SEQUENCE `s` start with 10 minvalue 1 maxvalue 100 increment by 2 cache 5 cycle;\n\n",
			get: func() (string, error) {
				return schema.GetSequenceDefinition(storepb.Engine_MARIADB, "", &metadatapb.SequenceMetadata{
					Name:      "s",
					Start:     "10",
					Increment: "2",
					MinValue:  "1",
					MaxValue:  "100",
					CacheSize: "5",
					Cycle:     true,
				})
			},
		},
		{
			name: "database",
			want: tableDDL,
			get: func() (string, error) {
				return schema.GetDatabaseDefinition(storepb.Engine_MARIADB, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{
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
