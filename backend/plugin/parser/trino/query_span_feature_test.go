package trino

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// These tests exercise the omni-backed query-span extractor end-to-end through
// the public extractor API (newQuerySpanExtractor + getQuerySpan). They replace
// the legacy feature tests that poked at ANTLR-listener internals
// (newTrinoQuerySpanListener, extractor.sourceColumns, addPredicateColumn,
// extractor.tableSourcesFrom, …) which no longer exist after the cutover. The
// behaviour under test — CTE table resolution, predicate-column extraction,
// UNNEST/LATERAL handling, and Trino's 3-part naming — is preserved.

// withoutMetadata builds an extractor with no metadata getter, so source tables
// stay at the table level (Column empty); useful for asserting which physical
// tables a query reads.
func tablesFromSpan(t *testing.T, sql, defaultDatabase string) base.SourceColumnSet {
	t.Helper()
	extractor := newQuerySpanExtractor(defaultDatabase, "public", base.GetQuerySpanContext{}, false)
	span, err := extractor.getQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	return span.SourceColumns
}

func TestQuerySpan_PredicateExtraction(t *testing.T) {
	// A WHERE predicate's columns are reported as predicate columns once they
	// resolve against the (metadata-expanded) source columns.
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "catalog1",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "users",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "name", Type: "varchar"},
						},
					},
				},
			},
		},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)
	span, err := extractor.getQuerySpan(context.Background(), "SELECT id, name FROM users WHERE id > 10 AND name LIKE 'A%'")
	require.NoError(t, err)

	var foundID, foundName bool
	for col := range span.PredicateColumns {
		switch col.Column {
		case "id":
			foundID = true
		case "name":
			foundName = true
		default:
		}
	}
	assert.True(t, foundID, "expected 'id' predicate column")
	assert.True(t, foundName, "expected 'name' predicate column")
}

func TestQuerySpan_CTEHandling(t *testing.T) {
	// A CTE name is not reported as a source table; the physical table the CTE
	// reads from is.
	sources := tablesFromSpan(t, "WITH temp_cte AS (SELECT id FROM users) SELECT id FROM temp_cte", "catalog1")

	assert.Equal(t, 1, len(sources), "expected exactly one physical source table")
	_, hasUsers := sources[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users"}]
	assert.True(t, hasUsers, "expected physical table catalog1.public.users; got %v", sources)
	_, hasCTE := sources[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "temp_cte"}]
	assert.False(t, hasCTE, "CTE name temp_cte must not appear as a source table")
}

func TestQuerySpan_UnnestAndLateral(t *testing.T) {
	// UNNEST over a column of a base table still reports the base table.
	unnest := tablesFromSpan(t, "SELECT id, t.name FROM users CROSS JOIN UNNEST(names) AS t(name)", "catalog1")
	_, hasUsers := unnest[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users"}]
	assert.True(t, hasUsers, "UNNEST query should still report base table users; got %v", unnest)

	// A LATERAL derived table that selects only computed expressions reports the
	// outer base table.
	lateral := tablesFromSpan(t, "SELECT u.id, t.x FROM users u, LATERAL (SELECT id + 1 AS x) t", "catalog1")
	_, hasUsersLateral := lateral[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users"}]
	assert.True(t, hasUsersLateral, "LATERAL query should report base table users; got %v", lateral)
}

func TestQuerySpan_ThreePartNaming(t *testing.T) {
	// Trino's catalog.schema.table maps onto Database/Schema/Table, and the
	// default database/schema are NOT applied when the reference is fully
	// qualified.
	sources := tablesFromSpan(t,
		"SELECT a.id, b.id FROM catalog1.public.users a JOIN catalog2.public.orders b ON a.id = b.user_id",
		"catalog3")

	_, hasUsers := sources[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users"}]
	_, hasOrders := sources[base.ColumnResource{Database: "catalog2", Schema: "public", Table: "orders"}]
	assert.True(t, hasUsers, "expected catalog1.public.users; got %v", sources)
	assert.True(t, hasOrders, "expected catalog2.public.orders; got %v", sources)
}
