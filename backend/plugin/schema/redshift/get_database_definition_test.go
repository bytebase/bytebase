package redshift

import (
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// TestObjectDefinitionsRegisteredForRedshift pins that Redshift reaches
// omni's database, table, and view definition generators through the schema
// registry.
func TestObjectDefinitionsRegisteredForRedshift(t *testing.T) {
	table := &metadatapb.TableMetadata{
		Name:    "orders",
		Columns: []*metadatapb.ColumnMetadata{{Name: "id", Type: "integer"}},
	}
	view := &metadatapb.ViewMetadata{
		Name:       "recent_orders",
		Definition: "SELECT id FROM orders",
	}

	tests := []struct {
		name     string
		get      func() (string, error)
		contains []string
	}{
		{
			name: "table",
			get: func() (string, error) {
				return schema.GetTableDefinition(storepb.Engine_REDSHIFT, "public", table, nil)
			},
			contains: []string{`CREATE TABLE "public"."orders"`, `"id" integer NOT NULL`},
		},
		{
			name: "view",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_REDSHIFT, "public", view)
			},
			contains: []string{`CREATE OR REPLACE VIEW "public"."recent_orders" AS SELECT id FROM orders;`},
		},
		{
			name: "database",
			get: func() (string, error) {
				return schema.GetDatabaseDefinition(storepb.Engine_REDSHIFT, schema.GetDefinitionContext{}, &metadatapb.DatabaseSchemaMetadata{
					Schemas: []*metadatapb.SchemaMetadata{{
						Name:   "public",
						Tables: []*metadatapb.TableMetadata{table},
						Views:  []*metadatapb.ViewMetadata{view},
					}},
				})
			},
			contains: []string{
				`CREATE TABLE "public"."orders"`,
				`CREATE OR REPLACE VIEW "public"."recent_orders" AS SELECT id FROM orders;`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get()
			require.NoError(t, err)
			for _, want := range tt.contains {
				require.Contains(t, got, want)
			}
		})
	}
}
