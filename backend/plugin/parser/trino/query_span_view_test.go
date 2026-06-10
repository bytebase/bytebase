package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
					{
						// Explicit view column list renames the projection: the
						// definition outputs "phone" but the view column is "ph".
						Name:       "renamed_v",
						Definition: "SELECT phone FROM customer",
						Columns:    []*storepb.ColumnMetadata{{Name: "ph"}},
					},
					{
						// A composed view column derives from two base columns.
						Name:       "comp_v",
						Definition: "SELECT phone || name AS token FROM customer",
						Columns:    []*storepb.ColumnMetadata{{Name: "token"}},
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
// through a view must carry the underlying base-table column (where masking
// config lives) in its lineage. omni's resolution is additive — the view column
// ref is retained beside the base column — which is safe: the view column has
// no masking catalog and contributes nothing, while the base column's masker
// applies.
func TestQuerySpan_ViewColumnResolvesToBase(t *testing.T) {
	span := viewLineageSpan(t, "SELECT phone FROM customer_v")
	srcs := resultSources(t, span, "phone")
	require.True(t, srcs[customerCol("phone")], "phone should resolve to base customer.phone; got %+v", srcs)
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

// TestQuerySpan_ViewExplicitColumnListResolvesToBase covers a view whose column
// list renames the projection (CREATE VIEW renamed_v (ph) AS SELECT phone ...):
// the outer ph must map positionally to base customer.phone even though the
// definition's output name is "phone".
func TestQuerySpan_ViewExplicitColumnListResolvesToBase(t *testing.T) {
	span := viewLineageSpan(t, "SELECT ph FROM renamed_v")
	srcs := resultSources(t, span, "ph")
	require.True(t, srcs[customerCol("phone")], "ph should resolve positionally to base customer.phone; got %+v", srcs)
}

// TestQuerySpan_ComposedViewColumnNotPlain covers a view column composed from two
// base columns: it must resolve to both, and must not stay a plain field.
func TestQuerySpan_ComposedViewColumnNotPlain(t *testing.T) {
	span := viewLineageSpan(t, "SELECT token FROM comp_v")
	for _, r := range span.Results {
		if r.Name != "token" {
			continue
		}
		require.True(t, r.SourceColumns[customerCol("phone")], "token should include customer.phone; got %+v", r.SourceColumns)
		require.True(t, r.SourceColumns[customerCol("name")], "token should include customer.name; got %+v", r.SourceColumns)
		require.False(t, r.IsPlainField, "a composed view column must not be a plain field")
		return
	}
	t.Fatalf("no result column named token in %+v", span.Results)
}

// TestQuerySpan_CrossCatalogViewResolvesToBase covers a view defined over a
// DIFFERENT catalog's table (with a star body, so resolution needs that
// catalog's column metadata): the outer statement never names catalog2, so the
// span-catalog builder must follow the view definition transitively to load it.
func TestQuerySpan_CrossCatalogViewResolvesToBase(t *testing.T) {
	cat1 := &storepb.DatabaseSchemaMetadata{
		Name: "catalog1",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Views: []*storepb.ViewMetadata{
					{
						Name:       "xcat_v",
						Definition: "SELECT * FROM catalog2.public.customer",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id"}, {Name: "phone"}, {Name: "name"},
						},
					},
				},
			},
		},
	}
	cat2 := &storepb.DatabaseSchemaMetadata{
		Name: "catalog2",
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
			},
		},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{cat1, cat2})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)
	span, err := extractor.getQuerySpan(context.Background(), "SELECT phone FROM xcat_v")
	require.NoError(t, err)
	srcs := resultSources(t, span, "phone")
	want := base.ColumnResource{Database: "catalog2", Schema: "public", Table: "customer", Column: "phone"}
	require.True(t, srcs[want], "phone should resolve through xcat_v to catalog2.public.customer.phone; got %+v", srcs)
}

// TestQuerySpan_BaseTableUnaffectedByViewResolution guards that a plain base
// table query is unchanged by the view post-pass.
func TestQuerySpan_BaseTableUnaffectedByViewResolution(t *testing.T) {
	span := viewLineageSpan(t, "SELECT phone FROM customer")
	srcs := resultSources(t, span, "phone")
	require.True(t, srcs[customerCol("phone")], "base customer.phone expected; got %+v", srcs)
	require.Len(t, srcs, 1, "base-table lineage should be exactly customer.phone; got %+v", srcs)
}
