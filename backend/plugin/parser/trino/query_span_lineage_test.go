package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// These tests pin the seven under-masking vectors from the Trino masking audit
// (BYT-9674..BYT-9680, sub-issues of BYT-9142) at the consumer level: each
// result column backed by a sensitive base column must carry that base column
// in its lineage, so the result masker attaches the configured masker instead
// of falling through to NoneMasker. The lineage resolution itself lives in
// omni's trino/analysis (catalog-aware since GetQuerySpanWithCatalog); this
// exercises the full extractor path metadata -> catalog -> omni -> resource
// mapping. The view vectors are covered in query_span_view_test.go.

func lineageAuditMetadata() *storepb.DatabaseSchemaMetadata {
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
							{Name: "phones", Type: "array(varchar)"},
						},
					},
				},
			},
		},
	}
}

func lineageAuditSpan(t *testing.T, sql string) *base.QuerySpan {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{lineageAuditMetadata()})
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

func auditCol(col string) base.ColumnResource {
	return base.ColumnResource{Database: "catalog1", Schema: "public", Table: "customer", Column: col}
}

// BYT-9674: a column projected through a derived table and re-aliased.
func TestLineageAudit_DerivedTableAlias(t *testing.T) {
	span := lineageAuditSpan(t, "SELECT d.x FROM (SELECT phone AS x FROM public.customer) d")
	srcs := resultSources(t, span, "x")
	require.True(t, srcs[auditCol("phone")], "x should reach customer.phone; got %+v", srcs)
}

// BYT-9675: a column projected through a CTE and re-aliased.
func TestLineageAudit_CTEAlias(t *testing.T) {
	span := lineageAuditSpan(t, "WITH w AS (SELECT phone AS pp FROM public.customer) SELECT pp FROM w")
	srcs := resultSources(t, span, "pp")
	require.True(t, srcs[auditCol("phone")], "pp should reach customer.phone; got %+v", srcs)
}

// BYT-9676: a scalar subquery used as a SELECT value.
func TestLineageAudit_ScalarSubquery(t *testing.T) {
	span := lineageAuditSpan(t, "SELECT (SELECT phone FROM public.customer LIMIT 1) AS sp FROM public.customer")
	srcs := resultSources(t, span, "sp")
	require.True(t, srcs[auditCol("phone")], "sp should reach customer.phone; got %+v", srcs)
}

// BYT-9677: a set operation's output column carries every arm's lineage, not
// just the first arm's.
func TestLineageAudit_SetOpArmMerge(t *testing.T) {
	span := lineageAuditSpan(t, "SELECT name FROM public.customer UNION SELECT phone FROM public.customer")
	srcs := resultSources(t, span, "name")
	require.True(t, srcs[auditCol("name")], "left arm lineage expected; got %+v", srcs)
	require.True(t, srcs[auditCol("phone")], "right arm (sensitive) lineage must be merged; got %+v", srcs)
}

// BYT-9678: SELECT * over a derived table expands to the derived projection
// (width and order), not the base table's column set — the original
// positional-masker repro: the span used to report 4 columns where Trino
// returns 2, sliding phone under id's (None) masker.
func TestLineageAudit_StarOverDerived(t *testing.T) {
	span := lineageAuditSpan(t, "SELECT * FROM (SELECT phone, name FROM public.customer) d")
	require.Len(t, span.Results, 2, "span must match the executed width [phone name]; got %+v", span.Results)
	require.Equal(t, "phone", span.Results[0].Name)
	require.Equal(t, "name", span.Results[1].Name)
	require.True(t, span.Results[0].SourceColumns[auditCol("phone")], "Results[0] -> customer.phone; got %+v", span.Results[0].SourceColumns)
	require.True(t, span.Results[1].SourceColumns[auditCol("name")], "Results[1] -> customer.name; got %+v", span.Results[1].SourceColumns)
}

// BYT-9680: a column produced by UNNEST of a sensitive array column.
func TestLineageAudit_UnnestAlias(t *testing.T) {
	span := lineageAuditSpan(t, "SELECT t.p FROM public.customer CROSS JOIN UNNEST(phones) AS t(p)")
	srcs := resultSources(t, span, "p")
	require.True(t, srcs[auditCol("phones")], "p should reach customer.phones; got %+v", srcs)
}

// Star over a base table still expands positionally against metadata (sanity
// that the catalog-aware path keeps the executed order).
func TestLineageAudit_StarOverBase(t *testing.T) {
	span := lineageAuditSpan(t, "SELECT * FROM public.customer")
	require.Len(t, span.Results, 4)
	require.Equal(t, "id", span.Results[0].Name)
	require.Equal(t, "phone", span.Results[1].Name)
	require.True(t, span.Results[1].SourceColumns[auditCol("phone")], "Results[1] -> customer.phone; got %+v", span.Results[1].SourceColumns)
}
