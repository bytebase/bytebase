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

func TestQuerySpan_SelectStarColumnOrder(t *testing.T) {
	// SELECT * must report result columns in the table's metadata column order,
	// not Go map order. The masker applies per-result maskers positionally
	// against the executed result's column order (query_result_masker.go), so a
	// nondeterministic order here could apply a column's masker to a different
	// column and leak sensitive data.
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "catalog1",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "accounts",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "secret", Type: "varchar"},
							{Name: "name", Type: "varchar"},
							{Name: "balance", Type: "double"},
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
	want := []string{"id", "secret", "name", "balance"}

	// Repeat to catch nondeterministic map iteration: a single pass can pass by
	// luck, so assert a stable order across many runs.
	for i := 0; i < 50; i++ {
		extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)
		span, err := extractor.getQuerySpan(context.Background(), "SELECT * FROM accounts")
		require.NoError(t, err)
		var got []string
		for _, r := range span.Results {
			got = append(got, r.Name)
		}
		assert.Equal(t, want, got, "SELECT * results must be in metadata column order (iteration %d)", i)
	}
}

func TestQuerySpan_AliasedColumnSourceColumns(t *testing.T) {
	// An aliased, alias-qualified select item (u.email over "users u") must
	// resolve back to the physical column users.email. Otherwise the per-result
	// SourceColumns are empty, the masker treats the column as a constant
	// expression, and a sensitive aliased column is returned unmasked.
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
							{Name: "email", Type: "varchar"},
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
	span, err := extractor.getQuerySpan(context.Background(), "SELECT u.email FROM users u")
	require.NoError(t, err)

	require.Len(t, span.Results, 1)
	res := span.Results[0]
	assert.Equal(t, "email", res.Name)
	_, ok := res.SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "email"}]
	assert.True(t, ok, "aliased column u.email must resolve to physical users.email; got %v", res.SourceColumns)
}

func TestQuerySpan_SystemSchemaNoNotFoundError(t *testing.T) {
	// A system / information_schema query must NOT surface a
	// ResourceNotFoundError. sql_service turns a span's NotFoundError into a hard
	// "failed to mask data" rejection of an otherwise-successful result, so
	// resolving Trino's pseudo-catalogs (which are not Bytebase-tracked
	// databases) as metadata would fail valid SQL-info queries after execution.
	metadata := &storepb.DatabaseSchemaMetadata{
		Name:    "mydb",
		Schemas: []*storepb.SchemaMetadata{{Name: "public"}},
	}
	getter, lister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		Engine:                  storepb.Engine_TRINO,
	}
	extractor := newQuerySpanExtractor("mydb", "public", gCtx, false)

	for _, stmt := range []string{
		"SELECT * FROM system.runtime.nodes",
		"SELECT table_name FROM information_schema.tables",
	} {
		span, err := extractor.getQuerySpan(context.Background(), stmt)
		require.NoError(t, err, "stmt: %s", stmt)
		assert.Equal(t, base.SelectInfoSchema, span.Type, "stmt %q should classify as SelectInfoSchema", stmt)
		assert.Nil(t, span.NotFoundError, "system query %q must not set NotFoundError", stmt)
	}
}

func TestQuerySpan_SystemSubstringDoesNotBypassMasking(t *testing.T) {
	// A real, maskable table must still be column-expanded even when the
	// statement text contains "system." in a string literal (which trips the
	// coarse containsSystemSchema classifier). System detection is per resolved
	// table, so the literal must not suppress lineage for users.email.
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
							{Name: "email", Type: "varchar"},
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
	span, err := extractor.getQuerySpan(context.Background(), "SELECT email FROM users WHERE 'system.' = 'system.'")
	require.NoError(t, err)

	require.Len(t, span.Results, 1)
	_, ok := span.Results[0].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "email"}]
	assert.True(t, ok, "users.email must stay masked despite the 'system.' literal; got %v", span.Results[0].SourceColumns)
}

func TestQuerySpan_SelfJoinStarColumnCount(t *testing.T) {
	// SELECT * over a self-join returns one set of columns per table instance
	// (2N for a 2-way self-join). The positional masker needs a span result per
	// output column, so the expansion must NOT collapse the duplicate physical
	// columns or the trailing instance's columns would get no masker.
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
							{Name: "email", Type: "varchar"},
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
	span, err := extractor.getQuerySpan(context.Background(), "SELECT * FROM users u1 JOIN users u2 ON u1.id = u2.id")
	require.NoError(t, err)

	require.Len(t, span.Results, 4, "self-join SELECT * must yield 2N=4 result columns")
	var ids, emails int
	for _, r := range span.Results {
		switch r.Name {
		case "id":
			ids++
		case "email":
			emails++
		default:
		}
	}
	assert.Equal(t, 2, ids, "expected two 'id' result columns (one per instance)")
	assert.Equal(t, 2, emails, "expected two 'email' result columns (one per instance)")
}

func TestQuerySpan_ShadowedAliasStillMasked(t *testing.T) {
	// An alias reused across scopes ("u" for users outside, for orders inside an
	// EXISTS subquery) must not drop lineage for the outer u.email. Additive
	// alias resolution keeps users.email so the sensitive column stays masked.
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
							{Name: "email", Type: "varchar"},
						},
					},
					{
						Name: "orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
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
	span, err := extractor.getQuerySpan(context.Background(), "SELECT u.email FROM users u WHERE EXISTS (SELECT 1 FROM orders u)")
	require.NoError(t, err)

	require.Len(t, span.Results, 1)
	_, ok := span.Results[0].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "email"}]
	assert.True(t, ok, "u.email must resolve to users.email despite the shadowing inner alias; got %v", span.Results[0].SourceColumns)
}

func TestQuerySpan_QualifiedStarSourceColumns(t *testing.T) {
	// A qualified star (u.*) must expand to the aliased relation's columns, each
	// carrying its physical source column so they stay maskable. omni names the
	// result "u.*"; only u's table (users) is expanded, not the joined orders.
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
							{Name: "email", Type: "varchar"},
						},
					},
					{
						Name: "orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "amount", Type: "double"},
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
	// Genuine qualified stars in several written forms must all expand to users'
	// columns (id, email), each mapped to its physical source column, and never
	// the joined orders. Covers alias (u.*), bare table (users.*) and
	// schema-qualified (public.users.*) qualifiers.
	for _, stmt := range []string{
		"SELECT u.* FROM users u JOIN orders o ON u.id = o.id",
		"SELECT users.* FROM users JOIN orders o ON users.id = o.id",
		"SELECT public.users.* FROM users JOIN orders o ON users.id = o.id",
	} {
		extractor := newQuerySpanExtractor("catalog1", "public", gCtx, false)
		span, err := extractor.getQuerySpan(context.Background(), stmt)
		require.NoError(t, err, "stmt: %s", stmt)

		require.Lenf(t, span.Results, 2, "qualified star must yield exactly 2 result columns; stmt: %s", stmt)
		assert.Equalf(t, "id", span.Results[0].Name, "stmt: %s", stmt)
		assert.Equalf(t, "email", span.Results[1].Name, "stmt: %s", stmt)
		_, ok := span.Results[0].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "id"}]
		assert.Truef(t, ok, "result id must map to users.id; stmt: %s got %v", stmt, span.Results[0].SourceColumns)
		_, ok = span.Results[1].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "email"}]
		assert.Truef(t, ok, "result email must map to users.email; stmt: %s got %v", stmt, span.Results[1].SourceColumns)
		for _, r := range span.Results {
			for sc := range r.SourceColumns {
				assert.NotEqualf(t, "orders", sc.Table, "qualified star must not include orders columns; stmt: %s got %v", stmt, r.SourceColumns)
			}
		}
	}
}

func TestQuerySpan_QualifiedStarDistinguishesSameNamedTables(t *testing.T) {
	// Two tables named "users" in different schemas. A qualified star s1.users.*
	// must expand to ONLY s1.users' columns, not s2.users' — matching on the full
	// (database, schema, table), not just the table name. Otherwise the positional
	// masker could apply s2's policies to s1's columns and leak.
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "catalog1",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "s1",
				Tables: []*storepb.TableMetadata{
					{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}, {Name: "secret", Type: "varchar"}}},
				},
			},
			{
				Name: "s2",
				Tables: []*storepb.TableMetadata{
					{Name: "users", Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "integer"}, {Name: "name", Type: "varchar"}}},
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
	extractor := newQuerySpanExtractor("catalog1", "s1", gCtx, false)
	span, err := extractor.getQuerySpan(context.Background(), "SELECT s1.users.* FROM s1.users JOIN s2.users ON s1.users.id = s2.users.id")
	require.NoError(t, err)

	require.Len(t, span.Results, 2, "s1.users.* must expand to exactly s1.users' two columns")
	_, hasSecret := span.Results[1].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "s1", Table: "users", Column: "secret"}]
	assert.True(t, hasSecret, "expected s1.users.secret; got %v", span.Results[1].SourceColumns)
	// No column may resolve to schema s2.
	for _, r := range span.Results {
		for sc := range r.SourceColumns {
			assert.NotEqual(t, "s2", sc.Schema, "s1.users.* must not include s2.users columns; got %v", r.SourceColumns)
		}
	}
}

func TestQuerySpan_AliasEndingInStarIsNotExpanded(t *testing.T) {
	// A column aliased to a quoted identifier that ends in ".*" (SELECT email AS
	// "u.*") must NOT be treated as a qualified star. It is one output column
	// (email); star-expanding it would emit several results and misalign the
	// positional masker, leaking the value. Detection keys on the source-ref
	// shape, so this resolves to a single email result.
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
							{Name: "email", Type: "varchar"},
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
	span, err := extractor.getQuerySpan(context.Background(), `SELECT email AS "u.*" FROM users u`)
	require.NoError(t, err)

	require.Len(t, span.Results, 1, `SELECT email AS "u.*" is one column, not a star expansion`)
	_, ok := span.Results[0].SourceColumns[base.ColumnResource{Database: "catalog1", Schema: "public", Table: "users", Column: "email"}]
	assert.True(t, ok, "the aliased column must still resolve to users.email; got %v", span.Results[0].SourceColumns)
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
