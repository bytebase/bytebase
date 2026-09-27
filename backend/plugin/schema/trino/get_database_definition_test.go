package trino

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinition(t *testing.T) {
	tests := []struct {
		name     string
		catalog  string
		schema   string
		table    *metadatapb.TableMetadata
		expected string
	}{
		{
			name:    "Simple table",
			catalog: "testcatalog",
			schema:  "testschema",
			table: &metadatapb.TableMetadata{
				Name: "testtable",
				Columns: []*metadatapb.ColumnMetadata{
					{
						Name:     "id",
						Type:     "bigint",
						Nullable: false,
					},
					{
						Name:     "name",
						Type:     "varchar",
						Nullable: true,
					},
				},
			},
			expected: `CREATE TABLE IF NOT EXISTS "testschema"."testtable" (
    "id" bigint NOT NULL,
    "name" varchar
);

`,
		},
		{
			name:    "Empty columns",
			catalog: "testcatalog",
			schema:  "testschema",
			table: &metadatapb.TableMetadata{
				Name:    "empty_table",
				Columns: []*metadatapb.ColumnMetadata{},
			},
			expected: `CREATE TABLE IF NOT EXISTS "testschema"."empty_table" (

);

`,
		},
		{
			name:    "Special characters in identifiers",
			catalog: "test-catalog",
			schema:  "test_schema",
			table: &metadatapb.TableMetadata{
				Name: "test.table",
				Columns: []*metadatapb.ColumnMetadata{
					{
						Name:     "id-field",
						Type:     "bigint",
						Nullable: false,
					},
				},
			},
			expected: `CREATE TABLE IF NOT EXISTS "test_schema"."test.table" (
    "id-field" bigint NOT NULL
);

`,
		},
	}
	a := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			metadata := &metadatapb.DatabaseSchemaMetadata{
				Name: tt.catalog,
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name:   tt.schema,
						Tables: []*metadatapb.TableMetadata{tt.table},
					},
				},
			}

			result, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
			a.NoError(err)
			a.Equal(tt.expected, result, tt.name)
		})
	}
}

// TestWriteCreateTable exercises the single-table CREATE TABLE rendering through
// the public GetDatabaseDefinition entry point (the omni deparser owns the
// per-table writer internally now, so it is no longer a directly callable
// helper). The emitted statement is followed by the standard inter-statement
// "\n\n" separator the deparser appends.
func TestWriteCreateTable(t *testing.T) {
	tests := []struct {
		name     string
		catalog  string
		schema   string
		table    *metadatapb.TableMetadata
		expected string
	}{
		{
			name:    "Simple table",
			catalog: "catalog",
			schema:  "schema",
			table: &metadatapb.TableMetadata{
				Name: "table",
				Columns: []*metadatapb.ColumnMetadata{
					{
						Name:     "col1",
						Type:     "integer",
						Nullable: false,
					},
				},
			},
			expected: `CREATE TABLE IF NOT EXISTS "schema"."table" (
    "col1" integer NOT NULL
);

`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &metadatapb.DatabaseSchemaMetadata{
				Name: tt.catalog,
				Schemas: []*metadatapb.SchemaMetadata{
					{
						Name:   tt.schema,
						Tables: []*metadatapb.TableMetadata{tt.table},
					},
				},
			}
			result, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
