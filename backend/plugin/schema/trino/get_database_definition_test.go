package trino

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetDatabaseDefinition(t *testing.T) {
	tests := []struct {
		name     string
		catalog  string
		schema   string
		table    *storepb.TableMetadata
		expected string
	}{
		{
			name:    "Simple table",
			catalog: "testcatalog",
			schema:  "testschema",
			table: &storepb.TableMetadata{
				Name: "testtable",
				Columns: []*storepb.ColumnMetadata{
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
			table: &storepb.TableMetadata{
				Name:    "empty_table",
				Columns: []*storepb.ColumnMetadata{},
			},
			expected: `CREATE TABLE IF NOT EXISTS "testschema"."empty_table" (

);

`,
		},
		{
			name:    "Special characters in identifiers",
			catalog: "test-catalog",
			schema:  "test_schema",
			table: &storepb.TableMetadata{
				Name: "test.table",
				Columns: []*storepb.ColumnMetadata{
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
			metadata := &storepb.DatabaseSchemaMetadata{
				Name: tt.catalog,
				Schemas: []*storepb.SchemaMetadata{
					{
						Name:   tt.schema,
						Tables: []*storepb.TableMetadata{tt.table},
					},
				},
			}

			result, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
			a.NoError(err)
			a.Equal(tt.expected, result, tt.name)
		})
	}
}

func TestWriteCreateTable(t *testing.T) {
	tests := []struct {
		name     string
		catalog  string
		schema   string
		table    *storepb.TableMetadata
		expected string
	}{
		{
			name:    "Simple table",
			catalog: "catalog",
			schema:  "schema",
			table: &storepb.TableMetadata{
				Name: "table",
				Columns: []*storepb.ColumnMetadata{
					{
						Name:     "col1",
						Type:     "integer",
						Nullable: false,
					},
				},
			},
			expected: `CREATE TABLE IF NOT EXISTS "schema"."table" (
    "col1" integer NOT NULL
);`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := writeCreateTable(&buf, tt.schema, tt.table)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}
