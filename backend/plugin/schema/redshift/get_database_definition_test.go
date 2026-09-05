package redshift

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

func TestGetObjectDefinition(t *testing.T) {
	tests := []struct {
		name string
		get  func() (string, error)
		want string
	}{
		{
			name: "table with comments",
			get: func() (string, error) {
				return GetTableDefinition("public", &storepb.TableMetadata{
					Name:    "orders",
					Comment: "one row per order",
					Columns: []*storepb.ColumnMetadata{
						{Name: "id", Type: "integer", Comment: "surrogate key"},
						{Name: "total", Type: "numeric(10,2)", Nullable: true},
						{Name: "label", Type: "character varying(50)", Nullable: true, Comment: "display name"},
					},
				}, nil)
			},
			want: `CREATE TABLE public.orders (
  id integer NOT NULL,
  total numeric(10,2),
  label character varying(50)
);
COMMENT ON TABLE public.orders IS 'one row per order';
COMMENT ON COLUMN public.orders.id IS 'surrogate key';
COMMENT ON COLUMN public.orders.label IS 'display name';
`,
		},
		{
			name: "view with comment",
			get: func() (string, error) {
				return GetViewDefinition("public", &storepb.ViewMetadata{
					Name:       "recent_orders",
					Definition: "SELECT id, total FROM orders",
					Comment:    "last 30 days",
				})
			},
			want: `CREATE OR REPLACE VIEW public.recent_orders AS SELECT id, total FROM orders;
COMMENT ON VIEW public.recent_orders IS 'last 30 days';
`,
		},
		{
			// GetSchemaString passes the requested schema, and a table outside
			// the search path has to carry it or the DDL lands somewhere else.
			name: "table outside the default schema keeps its qualifier",
			get: func() (string, error) {
				return GetTableDefinition("analytics", &storepb.TableMetadata{
					Name:    "orders",
					Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}},
				}, nil)
			},
			want: `CREATE TABLE analytics.orders (
  id integer NOT NULL
);
`,
		},
		{
			// An apostrophe would close the literal and produce DDL that will
			// not replay.
			name: "comments with apostrophes stay valid",
			get: func() (string, error) {
				return GetTableDefinition("public", &storepb.TableMetadata{
					Name:    "orders",
					Comment: "owner's orders",
					Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer", Comment: "don't drop"}},
				}, nil)
			},
			want: `CREATE TABLE public.orders (
  id integer NOT NULL
);
COMMENT ON TABLE public.orders IS 'owner''s orders';
COMMENT ON COLUMN public.orders.id IS 'don''t drop';
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestColumnCommentOrderIsStable pins the ordering of the trailing COMMENT ON
// COLUMN statements. They used to come from ranging over a map, so the dump
// differed between runs for any table with more than one commented column.
func TestColumnCommentOrderIsStable(t *testing.T) {
	table := &storepb.TableMetadata{Name: "wide"}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		table.Columns = append(table.Columns, &storepb.ColumnMetadata{
			Name:     name,
			Type:     "integer",
			Nullable: true,
			Comment:  "note " + name,
		})
	}

	first, err := GetTableDefinition("public", table, nil)
	require.NoError(t, err)
	require.Equal(t, 8, strings.Count(first, "COMMENT ON COLUMN"))

	for i := 0; i < 200; i++ {
		got, err := GetTableDefinition("public", table, nil)
		require.NoError(t, err)
		require.Equalf(t, first, got, "column comment order changed on iteration %d", i)
	}
}

// TestObjectDefinitionsRegisteredForRedshift goes through the schema registry
// rather than calling the functions directly, because the registry lookup is
// what was missing: GetSchemaString served only DATABASE for Redshift and
// returned "engine REDSHIFT is not supported" for TABLE and VIEW, both of which
// the SQL editor offers because supportGetStringSchema lists REDSHIFT.
func TestObjectDefinitionsRegisteredForRedshift(t *testing.T) {
	tests := []struct {
		name string
		get  func() (string, error)
	}{
		{
			name: "table",
			get: func() (string, error) {
				return schema.GetTableDefinition(storepb.Engine_REDSHIFT, "public", &storepb.TableMetadata{
					Name:    "t",
					Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}},
				}, nil)
			},
		},
		{
			name: "view",
			get: func() (string, error) {
				return schema.GetViewDefinition(storepb.Engine_REDSHIFT, "public", &storepb.ViewMetadata{
					Name:       "v",
					Definition: "SELECT 1",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get()
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

// TestGetDatabaseDefinitionKeepsSections guards the extraction above: the
// whole-database output still frames each object with its banner and renders
// the same body the single-object entry points return.
func TestGetDatabaseDefinitionKeepsSections(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name:    "orders",
						Comment: "one row per order",
						Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer", Comment: "surrogate key"}},
					},
				},
				Views: []*storepb.ViewMetadata{
					{Name: "recent_orders", Definition: "SELECT id FROM orders"},
				},
			},
		},
	}

	got, err := GetDatabaseDefinition(schema.GetDefinitionContext{}, metadata)
	require.NoError(t, err)

	require.Equal(t, `--
-- Table structure for `+"`orders`"+`
--
CREATE TABLE public.orders (
  id integer NOT NULL
);
COMMENT ON TABLE public.orders IS 'one row per order';
COMMENT ON COLUMN public.orders.id IS 'surrogate key';

--
-- View structure for `+"`recent_orders`"+`
--
CREATE OR REPLACE VIEW public.recent_orders AS SELECT id FROM orders;
`, got)

	table, err := GetTableDefinition("public", metadata.Schemas[0].Tables[0], nil)
	require.NoError(t, err)
	require.Contains(t, got, table)
}
