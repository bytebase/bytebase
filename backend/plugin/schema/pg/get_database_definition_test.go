package pg

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinitionSeam(t *testing.T) {
	backupSchema := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
	meta := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Extensions: []*metadatapb.ExtensionMetadata{
			{Name: "pg_trgm", Schema: "public", Version: "1.6"},
		},
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{{
					Name:    "t",
					Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer", Nullable: false}},
				}},
			},
			{
				Name: backupSchema,
				Tables: []*metadatapb.TableMetadata{{
					Name:    "archived",
					Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer", Nullable: false}},
				}},
			},
		},
	}

	t.Run("SkipBackupSchema filters backup schema without mutating caller metadata", func(t *testing.T) {
		filtered, err := schema.GetDatabaseDefinition(storepb.Engine_POSTGRES, schema.GetDefinitionContext{
			PrintHeader:      true,
			SkipBackupSchema: true,
		}, meta)
		require.NoError(t, err)
		require.Contains(t, filtered, `CREATE EXTENSION IF NOT EXISTS "pg_trgm"`)
		require.Contains(t, filtered, `CREATE TABLE "public"."t"`)
		require.NotContains(t, filtered, backupSchema)
		require.Len(t, meta.Schemas, 2, "caller metadata must not be mutated")

		unfiltered, err := schema.GetDatabaseDefinition(storepb.Engine_POSTGRES, schema.GetDefinitionContext{
			SkipBackupSchema: false,
		}, meta)
		require.NoError(t, err)
		require.Contains(t, unfiltered, backupSchema)

		multiFile, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_POSTGRES, schema.GetDefinitionContext{
			SkipBackupSchema: true,
		}, meta)
		require.NoError(t, err)
		require.NotEmpty(t, multiFile.Files)
		for _, f := range multiFile.Files {
			require.NotContains(t, f.Name, backupSchema)
			require.NotContains(t, f.Content, backupSchema)
		}
	})

	t.Run("CockroachDB routes through PostgreSQL definition registry", func(t *testing.T) {
		entry := newPostgresEntryPoint()
		def, err := schema.GetDatabaseDefinition(storepb.Engine_COCKROACHDB, schema.GetDefinitionContext{PrintHeader: true}, entry.metadata)
		require.NoError(t, err)
		require.Contains(t, def, `"public"."t"`)

		tblDef, err := schema.GetTableDefinition(storepb.Engine_COCKROACHDB, "public", entry.table, nil)
		require.NoError(t, err)
		require.Contains(t, tblDef, `CREATE TABLE`)

		multiFile, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_COCKROACHDB, schema.GetDefinitionContext{}, entry.metadata)
		require.NoError(t, err)
		require.NotEmpty(t, multiFile.Files)
	})
}

func TestParseQualifiedTypeIdent(t *testing.T) {
	testCases := []struct {
		input  string
		schema string
		name   string
		ok     bool
	}{
		{"integer", "", "", false},
		{"character varying(50)", "", "", false},
		{"numeric(9,6)", "", "", false},
		{"text[]", "", "", false},
		{"timestamp(6) with time zone", "", "", false},
		{"public.addr", "public", "addr", true},
		{"public.addr[]", "public", "addr", true},
		{"public.addr(5)", "public", "addr", true},
		{`"My Schema"."Weird Type"`, "My Schema", "Weird Type", true},
		{`"has""quote".t`, `has"quote`, "t", true},
		{`other.geo`, "other", "geo", true},
	}
	for _, tc := range testCases {
		schemaName, typeName, ok := parseQualifiedTypeIdent(tc.input)
		assert.Equal(t, tc.ok, ok, "input %q", tc.input)
		assert.Equal(t, tc.schema, schemaName, "input %q", tc.input)
		assert.Equal(t, tc.name, typeName, "input %q", tc.input)
	}
}
