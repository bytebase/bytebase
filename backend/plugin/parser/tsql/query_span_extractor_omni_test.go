package tsql

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// omniTestMetadata is the default mock catalog used by these tests.
// db.dbo has tables t(a,b,c), t1(a,b,c), t2(a,b), and a view vw.
var omniTestMetadata = []*storepb.DatabaseSchemaMetadata{
	{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{Name: "t", Columns: []*storepb.ColumnMetadata{{Name: "a"}, {Name: "b"}, {Name: "c"}}},
					{Name: "t1", Columns: []*storepb.ColumnMetadata{{Name: "a"}, {Name: "b"}, {Name: "c"}}},
					{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "a"}, {Name: "b"}}},
				},
				Views: []*storepb.ViewMetadata{
					{Name: "vw", Definition: "CREATE VIEW [dbo].[vw] AS SELECT a, b FROM t"},
				},
			},
		},
	},
	{
		Name: "db2",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{Name: "t2", Columns: []*storepb.ColumnMetadata{{Name: "x"}, {Name: "y"}}},
				},
			},
		},
	},
}

// newOmniTestExtractor builds an extractor wired to the default mock catalog.
// Case-insensitive matching is on, matching T-SQL's default collation semantics.
func newOmniTestExtractor(t *testing.T, defaultDatabase string) *omniQuerySpanExtractor {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter(omniTestMetadata)
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		TempTables:              make(map[string]*base.PhysicalTable),
	}
	return newOmniQuerySpanExtractor(defaultDatabase, "dbo", gCtx, true)
}

type expectedColumn struct {
	name    string
	sources []base.ColumnResource // empty means isPlainField=false / no sources
}

func sortedSources(set base.SourceColumnSet) []base.ColumnResource {
	out := make([]base.ColumnResource, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b base.ColumnResource) int { return cmp.Compare(a.String(), b.String()) })
	return out
}

func TestOmniQuerySpan_SupportedShapes(t *testing.T) {
	type testCase struct {
		name             string
		sql              string
		defaultDatabase  string
		wantResults      []expectedColumn
		wantPredicate    []base.ColumnResource
		wantAccessTables []base.ColumnResource
	}

	src := func(db, schema, table, col string) base.ColumnResource {
		return base.ColumnResource{Database: db, Schema: schema, Table: table, Column: col}
	}
	access := func(db, schema, table string) base.ColumnResource {
		return base.ColumnResource{Database: db, Schema: schema, Table: table}
	}

	cases := []testCase{
		{
			name:            "bare_column",
			sql:             "SELECT a FROM t",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "qualified_column",
			sql:             "SELECT t.a FROM t",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "alias_as",
			sql:             "SELECT a AS x FROM t",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"x", []base.ColumnResource{src("db", "dbo", "t", "a")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "multi_columns",
			sql:             "SELECT a, b FROM t",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
				{"b", []base.ColumnResource{src("db", "dbo", "t", "b")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "star_unqualified",
			sql:             "SELECT * FROM t",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
				{"b", []base.ColumnResource{src("db", "dbo", "t", "b")}},
				{"c", []base.ColumnResource{src("db", "dbo", "t", "c")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "table_alias_qualified",
			sql:             "SELECT x.a FROM t AS x",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "table_alias_star",
			sql:             "SELECT x.* FROM t AS x",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
				{"b", []base.ColumnResource{src("db", "dbo", "t", "b")}},
				{"c", []base.ColumnResource{src("db", "dbo", "t", "c")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "where_predicate_columns",
			sql:             "SELECT a FROM t WHERE b > 1 AND c = 2",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
			},
			wantPredicate: []base.ColumnResource{
				src("db", "dbo", "t", "b"),
				src("db", "dbo", "t", "c"),
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "cross_database_table",
			sql:             "SELECT x FROM db2.dbo.t2",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"x", []base.ColumnResource{src("db2", "dbo", "t2", "x")}},
			},
			wantAccessTables: []base.ColumnResource{access("db2", "dbo", "t2")},
		},
		{
			name:            "explicit_schema",
			sql:             "SELECT a FROM dbo.t",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t", "a")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t")},
		},
		{
			name:            "star_with_alias",
			sql:             "SELECT y.* FROM t2 AS y",
			defaultDatabase: "db",
			wantResults: []expectedColumn{
				{"a", []base.ColumnResource{src("db", "dbo", "t2", "a")}},
				{"b", []base.ColumnResource{src("db", "dbo", "t2", "b")}},
			},
			wantAccessTables: []base.ColumnResource{access("db", "dbo", "t2")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newOmniTestExtractor(t, tc.defaultDatabase)
			span, err := q.getOmniQuerySpan(context.Background(), tc.sql)
			require.NoError(t, err, "sql: %s", tc.sql)
			require.NotNil(t, span)

			require.Equalf(t, base.Select, span.Type, "query type")
			require.Lenf(t, span.Results, len(tc.wantResults), "result count; got: %+v", span.Results)
			for i, want := range tc.wantResults {
				require.Equalf(t, want.name, span.Results[i].Name, "result[%d].Name", i)
				gotSources := sortedSources(span.Results[i].SourceColumns)
				wantSources := append([]base.ColumnResource{}, want.sources...)
				slices.SortFunc(wantSources, func(a, b base.ColumnResource) int { return cmp.Compare(a.String(), b.String()) })
				require.ElementsMatchf(t, wantSources, gotSources, "result[%d].SourceColumns", i)
			}

			wantPred := append([]base.ColumnResource{}, tc.wantPredicate...)
			slices.SortFunc(wantPred, func(a, b base.ColumnResource) int { return cmp.Compare(a.String(), b.String()) })
			require.ElementsMatch(t, wantPred, sortedSources(span.PredicateColumns), "PredicateColumns")

			wantAccess := append([]base.ColumnResource{}, tc.wantAccessTables...)
			slices.SortFunc(wantAccess, func(a, b base.ColumnResource) int { return cmp.Compare(a.String(), b.String()) })
			require.ElementsMatch(t, wantAccess, sortedSources(span.SourceColumns), "AccessTables")
		})
	}
}

func TestOmniQuerySpan_NotFound(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	span, err := q.getOmniQuerySpan(context.Background(), "SELECT a FROM no_such_table")
	require.NoError(t, err)
	require.NotNil(t, span.NotFoundError)
	require.Empty(t, span.Results)
}

// TestOmniQuerySpan_NonSelectQueryType verifies that non-SELECT statements
// still return an empty span with the correct query type (matching the ANTLR
// early-return behavior).
func TestOmniQuerySpan_NonSelectQueryType(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		typ  base.QueryType
	}{
		{"insert", "INSERT INTO t(a) VALUES (1)", base.DML},
		{"update", "UPDATE t SET a = 1", base.DML},
		{"delete", "DELETE FROM t", base.DML},
		{"create_table", "CREATE TABLE t2 (id INT)", base.DDL},
		{"alter_table", "ALTER TABLE t ADD c INT", base.DDL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			span, err := q.getOmniQuerySpan(context.Background(), tc.sql)
			require.NoError(t, err)
			require.Equal(t, tc.typ, span.Type)
			require.Empty(t, span.Results)
		})
	}
}

// TestOmniQuerySpan_DeclareTableVariableRoundTrip verifies that a DECLARE @t
// TABLE(...) call populates gCtx.TempTables so a subsequent SELECT from @t
// (in the same session, with the same gCtx) can resolve its columns. This
// matches the pre-migration ANTLR side-effect.
func TestOmniQuerySpan_DeclareTableVariableRoundTrip(t *testing.T) {
	getter, lister := buildMockDatabaseMetadataGetter(omniTestMetadata)
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		TempTables:              make(map[string]*base.PhysicalTable),
	}

	// Step 1: DECLARE populates TempTables.
	q1 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span1, err := q1.getOmniQuerySpan(context.Background(), "DECLARE @t TABLE(id INT, name NVARCHAR(50))")
	require.NoError(t, err)
	require.Equal(t, base.Select, span1.Type)
	require.Empty(t, span1.Results)
	require.Contains(t, gCtx.TempTables, "@t", "DECLARE should register @t in gCtx.TempTables")
	require.Equal(t, []string{"id", "name"}, gCtx.TempTables["@t"].Columns)

	// Step 2: SELECT from @t resolves via TempTables.
	q2 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span2, err := q2.getOmniQuerySpan(context.Background(), "SELECT id, name FROM @t")
	require.NoError(t, err)
	require.Equal(t, base.Select, span2.Type)
	require.Len(t, span2.Results, 2)
	require.Equal(t, "id", span2.Results[0].Name)
	require.Equal(t, "name", span2.Results[1].Name)
}

// TestOmniQuerySpan_PivotPassThrough verifies that PIVOT/UNPIVOT pass the
// source table's columns through to the result, matching the pre-migration
// ANTLR extractor's behavior (which ignored the pivot transformation).
// Real PIVOT column inference is a separate project.
func TestOmniQuerySpan_PivotPassThrough(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"pivot", "SELECT * FROM t PIVOT (SUM(a) FOR b IN ([1],[2])) AS p"},
		{"unpivot", "SELECT * FROM t UNPIVOT (v FOR c IN ([1],[2])) AS u"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			span, err := q.getOmniQuerySpan(context.Background(), tc.sql)
			require.NoError(t, err)
			require.Equal(t, base.Select, span.Type)
			// t has columns a,b,c; PIVOT/UNPIVOT pass them all through.
			require.Len(t, span.Results, 3, "expected source columns to pass through")
		})
	}
}

// TestOmniQuerySpan_CTEVisibleInSetOpArms verifies that a CTE defined on a
// set-op SELECT root (the outer SelectStmt with Op != None) is visible inside
// both arms. Regression test for the P1 comment where the set-op early
// return fired before processWithClause.
func TestOmniQuerySpan_CTEVisibleInSetOpArms(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "WITH cte AS (SELECT a FROM t) SELECT a FROM cte UNION SELECT a FROM cte"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Nil(t, span.NotFoundError, "CTE should resolve inside both UNION arms")
	require.Len(t, span.Results, 1)
	require.Equal(t, "a", span.Results[0].Name)
}

// TestOmniQuerySpan_SubqueryNotFoundPropagates verifies that a predicate
// subquery referencing a missing table surfaces the ResourceNotFoundError at
// the top-level QuerySpan instead of silently succeeding. Regression test
// for the P1 comment on mergeSubqueryIntoPredicates silently dropping errors.
func TestOmniQuerySpan_SubqueryNotFoundPropagates(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT a FROM t WHERE EXISTS (SELECT 1 FROM no_such_table)"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.NotNil(t, span.NotFoundError, "missing subquery table should surface NotFoundError")
	require.Empty(t, span.Results)
}

// TestOmniQuerySpan_RecursiveCTEArmNotFoundPropagates verifies that a
// recursive-CTE arm that can't resolve (e.g. references a missing table)
// surfaces NotFoundError at the top level instead of silently falling back
// to anchor-only columns.
func TestOmniQuerySpan_RecursiveCTEArmNotFoundPropagates(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "WITH rec AS (SELECT a FROM t UNION ALL SELECT a FROM rec JOIN no_such_table nx ON rec.a = nx.a) SELECT * FROM rec"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.NotNil(t, span.NotFoundError, "recursive arm hitting a missing table should surface NotFoundError")
	require.Empty(t, span.Results)
}

// TestOmniQuerySpan_SubqueryComparisonNotFoundPropagates verifies that a
// subquery-comparison expression (x > ANY/ALL (SELECT ...)) whose subquery
// references a missing table surfaces NotFoundError.
func TestOmniQuerySpan_SubqueryComparisonNotFoundPropagates(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT a FROM t WHERE a > ANY (SELECT x FROM no_such_table)"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.NotNil(t, span.NotFoundError, "missing subquery table should surface NotFoundError via SubqueryComparisonExpr")
	require.Empty(t, span.Results)
}

// TestOmniQuerySpan_PivotPreservesAllSourceColumns verifies that PIVOT's
// pass-through keeps all source columns. T-SQL PIVOT only allows a single
// table source in the grammar, so in practice extractTableSource returns
// one element, but the extractor is defensive against multi-source inputs
// (e.g. a future grammar relaxation); this guards that the single-source
// case still emits the source columns under the pivot alias.
func TestOmniQuerySpan_PivotPreservesAllSourceColumns(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT * FROM t PIVOT (SUM(a) FOR b IN ([1],[2])) AS p"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Equal(t, base.Select, span.Type)
	require.Len(t, span.Results, 3, "PIVOT should pass source-table columns through")
}

// TestOmniQuerySpan_EmptyAndGo verifies that empty statements and bare GO
// produce a SELECT-typed empty span, matching ANTLR's zero-AST behavior.
func TestOmniQuerySpan_EmptyAndGo(t *testing.T) {
	cases := []string{"", "GO", ";"}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			span, err := q.getOmniQuerySpan(context.Background(), sql)
			require.NoError(t, err)
			require.Equal(t, base.Select, span.Type)
			require.Empty(t, span.Results)
		})
	}
}
