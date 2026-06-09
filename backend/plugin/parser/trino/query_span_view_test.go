package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// viewLineageMetadata builds a catalog with a base table customer(id, phone,
// name), a pass-through view customer_v, a renaming view cust_masked
// (phone AS ph), and a view-over-view v2.
func viewLineageMetadata() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name: "catalog1",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "customer",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "phone", Type: "varchar"},
							{Name: "name", Type: "varchar"},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "customer_v",
						Definition: "SELECT id, phone, name FROM customer",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id"}, {Name: "phone"}, {Name: "name"},
						},
					},
					{
						Name:       "cust_masked",
						Definition: "SELECT phone AS ph FROM customer",
						Columns:    []*storepb.ColumnMetadata{{Name: "ph"}},
					},
					{
						Name:       "v2",
						Definition: "SELECT phone FROM customer_v",
						Columns:    []*storepb.ColumnMetadata{{Name: "phone"}},
					},
				},
			},
		},
	}
}

func viewLineageSpan(t *testing.T, sql string) *base.QuerySpan {
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{viewLineageMetadata()})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)
	span, err := extractor.getQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	return span
}

func resultSources(t *testing.T, span *base.QuerySpan, name string) base.SourceColumnSet {
	t.Helper()
	for _, r := range span.Results {
		if r.Name == name {
			return r.SourceColumns
		}
	}
	t.Fatalf("no result column named %q in %+v", name, span.Results)
	return nil
}

func customerCol(col string) base.ColumnResource {
	return base.ColumnResource{Database: "catalog1", Schema: "public", Table: "customer", Column: col}
}

// TestQuerySpan_ViewColumnResolvesToBase covers BYT-9679: a column selected
// through a view must resolve to the underlying base-table column (where masking
// config lives), not the view column.
func TestQuerySpan_ViewColumnResolvesToBase(t *testing.T) {
	span := viewLineageSpan(t, "SELECT phone FROM customer_v")
	srcs := resultSources(t, span, "phone")
	require.True(t, srcs[customerCol("phone")], "phone should resolve to base customer.phone; got %+v", srcs)
	// The view column must not survive as the (unmaskable) lineage.
	require.False(t, srcs[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "customer_v", Column: "phone"}],
		"view column customer_v.phone must be rewritten to the base column; got %+v", srcs)
}

// TestQuerySpan_ViewStarResolvesToBase resolves every column of SELECT * over a
// view to its base columns.
func TestQuerySpan_ViewStarResolvesToBase(t *testing.T) {
	span := viewLineageSpan(t, "SELECT * FROM customer_v")
	require.True(t, resultSources(t, span, "phone")[customerCol("phone")], "SELECT * phone should resolve to customer.phone")
	require.True(t, resultSources(t, span, "id")[customerCol("id")], "SELECT * id should resolve to customer.id")
	require.True(t, resultSources(t, span, "name")[customerCol("name")], "SELECT * name should resolve to customer.name")
}

// TestQuerySpan_RenamingViewResolvesToBase resolves a view that renames the
// projected column (phone AS ph): the outer ph must map to base customer.phone.
func TestQuerySpan_RenamingViewResolvesToBase(t *testing.T) {
	span := viewLineageSpan(t, "SELECT ph FROM cust_masked")
	srcs := resultSources(t, span, "ph")
	require.True(t, srcs[customerCol("phone")], "ph should resolve to base customer.phone; got %+v", srcs)
}

// TestQuerySpan_ViewOverViewResolvesToBase resolves a view defined over another
// view transitively to the base column.
func TestQuerySpan_ViewOverViewResolvesToBase(t *testing.T) {
	span := viewLineageSpan(t, "SELECT phone FROM v2")
	srcs := resultSources(t, span, "phone")
	require.True(t, srcs[customerCol("phone")], "phone should resolve through v2 -> customer_v -> customer.phone; got %+v", srcs)
}

// TestQuerySpan_BaseTableUnaffectedByViewResolution guards that a plain base
// table query is unchanged by the view post-pass.
func TestQuerySpan_BaseTableUnaffectedByViewResolution(t *testing.T) {
	span := viewLineageSpan(t, "SELECT phone FROM customer")
	srcs := resultSources(t, span, "phone")
	require.True(t, srcs[customerCol("phone")], "base customer.phone expected; got %+v", srcs)
	require.Len(t, srcs, 1, "base-table lineage should be exactly customer.phone; got %+v", srcs)
}
