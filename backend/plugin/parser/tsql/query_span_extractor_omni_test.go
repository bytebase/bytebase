package tsql

import (
	"cmp"
	"context"
	"errors"
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

// TestOmniQuerySpan_UnsupportedTableSources verifies that table sources whose
// output shape is not modeled fail explicitly instead of returning partial
// lineage.
func TestOmniQuerySpan_UnsupportedTableSources(t *testing.T) {
	cases := []struct {
		name                 string
		sql                  string
		wantTypeNotSupported bool
	}{
		{name: "pivot", sql: "SELECT * FROM t PIVOT (SUM(a) FOR b IN ([1],[2])) AS p", wantTypeNotSupported: true},
		{name: "unpivot", sql: "SELECT * FROM t UNPIVOT (v FOR c IN ([1],[2])) AS u", wantTypeNotSupported: true},
		{name: "cross_apply_tvf", sql: "SELECT * FROM t1 CROSS APPLY fn(t1.a) AS x(v)", wantTypeNotSupported: true},
		{name: "aliased_tvf", sql: "SELECT * FROM fn(t1.a) AS x(v)", wantTypeNotSupported: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			_, err := q.getOmniQuerySpan(context.Background(), tc.sql)
			require.Error(t, err)
			if tc.wantTypeNotSupported {
				var typeNotSupported *base.TypeNotSupportedError
				require.True(t, errors.As(err, &typeNotSupported), "expected TypeNotSupportedError, got %T: %v", err, err)
			}
		})
	}
}

func TestOmniQuerySpan_SystemTableValuedFunctions(t *testing.T) {
	cases := []struct {
		name        string
		sql         string
		wantName    string
		wantSources []base.ColumnResource
	}{
		{
			name:     "string_split_value_flows_from_input",
			sql:      "SELECT s.value FROM t1 CROSS APPLY STRING_SPLIT(t1.a, ',') AS s",
			wantName: "value",
			wantSources: []base.ColumnResource{
				{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
			},
		},
		{
			name:     "openjson_value_flows_from_input",
			sql:      "SELECT j.value FROM t1 CROSS APPLY OPENJSON(t1.a) AS j",
			wantName: "value",
			wantSources: []base.ColumnResource{
				{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
			},
		},
		{
			name:     "tvf_column_alias_list",
			sql:      "SELECT s.v FROM t1 CROSS APPLY STRING_SPLIT(t1.a, ',', 1) AS s(v, ordinal)",
			wantName: "v",
			wantSources: []base.ColumnResource{
				{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
			},
		},
		{
			name:     "string_split_third_arg_zero_has_no_ordinal",
			sql:      "SELECT s.value FROM t1 CROSS APPLY STRING_SPLIT(t1.a, ',', 0) AS s(value)",
			wantName: "value",
			wantSources: []base.ColumnResource{
				{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
			},
		},
		{
			name:     "string_split_third_arg_null_has_no_ordinal",
			sql:      "SELECT s.value FROM t1 CROSS APPLY STRING_SPLIT(t1.a, ',', NULL) AS s(value)",
			wantName: "value",
			wantSources: []base.ColumnResource{
				{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			span, err := q.getOmniQuerySpan(context.Background(), tc.sql)
			require.NoError(t, err)
			require.Len(t, span.Results, 1)
			require.Equal(t, tc.wantName, span.Results[0].Name)
			require.ElementsMatch(t, tc.wantSources, sortedSources(span.Results[0].SourceColumns))
		})
	}
}

func TestOmniQuerySpan_TempTableDefinitions(t *testing.T) {
	getter, lister := buildMockDatabaseMetadataGetter(omniTestMetadata)
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		TempTables:              make(map[string]*base.PhysicalTable),
	}

	q1 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span1, err := q1.getOmniQuerySpan(context.Background(), "CREATE TABLE #tmp(id INT, name NVARCHAR(50))")
	require.NoError(t, err)
	require.Equal(t, base.DDL, span1.Type)
	require.Contains(t, gCtx.TempTables, "#tmp")
	require.Equal(t, []string{"id", "name"}, gCtx.TempTables["#tmp"].Columns)

	q2 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span2, err := q2.getOmniQuerySpan(context.Background(), "SELECT id, name FROM #tmp")
	require.NoError(t, err)
	require.Len(t, span2.Results, 2)
	require.Equal(t, "id", span2.Results[0].Name)
	require.Equal(t, "name", span2.Results[1].Name)
}

func TestOmniQuerySpan_TempTableDefinitionsCaseInsensitive(t *testing.T) {
	getter, lister := buildMockDatabaseMetadataGetter(omniTestMetadata)
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		TempTables:              make(map[string]*base.PhysicalTable),
	}

	q1 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	_, err := q1.getOmniQuerySpan(context.Background(), "CREATE TABLE #Tmp(id INT, name NVARCHAR(50))")
	require.NoError(t, err)

	q2 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span2, err := q2.getOmniQuerySpan(context.Background(), "SELECT id, name FROM #tmp")
	require.NoError(t, err)
	require.Len(t, span2.Results, 2)
	require.Equal(t, "id", span2.Results[0].Name)
	require.Equal(t, "name", span2.Results[1].Name)
}

func TestOmniQuerySpan_SelectIntoTempTableRegistersColumns(t *testing.T) {
	getter, lister := buildMockDatabaseMetadataGetter(omniTestMetadata)
	gCtx := base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
		TempTables:              make(map[string]*base.PhysicalTable),
	}

	q1 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span1, err := q1.getOmniQuerySpan(context.Background(), "SELECT a AS id, b INTO #selected FROM t")
	require.NoError(t, err)
	require.Equal(t, base.Select, span1.Type)
	require.Contains(t, gCtx.TempTables, "#selected")
	require.Equal(t, []string{"id", "b"}, gCtx.TempTables["#selected"].Columns)

	q2 := newOmniQuerySpanExtractor("db", "dbo", gCtx, true)
	span2, err := q2.getOmniQuerySpan(context.Background(), "SELECT * FROM #selected")
	require.NoError(t, err)
	require.Len(t, span2.Results, 2)
	require.Equal(t, "id", span2.Results[0].Name)
	require.Equal(t, "b", span2.Results[1].Name)
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

func TestOmniQuerySpan_ApplyRightSideCanReferenceLeft(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT x FROM t1 CROSS APPLY (VALUES (t1.a)) AS v(x)"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.Equal(t, "x", span.Results[0].Name)
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
	}, sortedSources(span.Results[0].SourceColumns))
}

func TestOmniQuerySpan_ValuesClauseMergesAllRows(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT x FROM t1 CROSS APPLY (VALUES (t1.a), (t1.b)) AS v(x)"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.Equal(t, "x", span.Results[0].Name)
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
		{Database: "db", Schema: "dbo", Table: "t1", Column: "b"},
	}, sortedSources(span.Results[0].SourceColumns))
}

func TestOmniQuerySpan_WindowOrderBySources(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT SUM(a) OVER (ORDER BY b) FROM t"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t", Column: "a"},
		{Database: "db", Schema: "dbo", Table: "t", Column: "b"},
	}, sortedSources(span.Results[0].SourceColumns))
}

func TestOmniQuerySpan_WithinGroupOrderBySources(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY b) FROM t"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t", Column: "b"},
	}, sortedSources(span.Results[0].SourceColumns))
}

func TestOmniQuerySpan_InListAccessTables(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := "SELECT a FROM t WHERE a IN (ABS((SELECT x FROM db2.dbo.t2)))"
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t"},
		{Database: "db2", Schema: "dbo", Table: "t2"},
	}, sortedSources(span.SourceColumns))
}

func TestOmniQuerySpan_CorrelatedExistsWithDateaddDatepart(t *testing.T) {
	q := newOmniTestExtractor(t, "db")
	sql := `
SELECT outer_t.a
FROM t AS outer_t
WHERE outer_t.c > DATEADD(HOUR, 2, GETUTCDATE())
  AND NOT EXISTS (
    SELECT 1
    FROM t1 AS inner_t
    WHERE inner_t.a = outer_t.a
      AND inner_t.b > outer_t.b
  )
ORDER BY outer_t.a
`
	span, err := q.getOmniQuerySpan(context.Background(), sql)
	require.NoError(t, err)
	require.Len(t, span.Results, 1)
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t", Column: "a"},
	}, sortedSources(span.Results[0].SourceColumns))
	require.ElementsMatch(t, []base.ColumnResource{
		{Database: "db", Schema: "dbo", Table: "t", Column: "a"},
		{Database: "db", Schema: "dbo", Table: "t", Column: "b"},
		{Database: "db", Schema: "dbo", Table: "t", Column: "c"},
		{Database: "db", Schema: "dbo", Table: "t1", Column: "a"},
		{Database: "db", Schema: "dbo", Table: "t1", Column: "b"},
	}, sortedSources(span.PredicateColumns))
}

// TestOmniQuerySpan_UnresolvedColumnErrors verifies that an unresolvable
// column reference is surfaced as an error (both from SELECT list and from
// WHERE predicates) rather than producing a silently-partial span. Matches
// the legacy ANTLR extractor's strict behavior.
func TestOmniQuerySpan_UnresolvedColumnErrors(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"select_list", "SELECT no_such_col FROM t"},
		{"qualified_select_list", "SELECT t.no_such_col FROM t"},
		{"where_predicate", "SELECT a FROM t WHERE no_such_col = 1"},
		{"window_partition", "SELECT SUM(a) OVER (PARTITION BY no_such_col) FROM t"},
		{"window_order", "SELECT SUM(a) OVER (ORDER BY no_such_col) FROM t"},
		{"within_group_order", "SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY no_such_col) FROM t"},
		{"full_text_predicate", "SELECT CASE WHEN CONTAINS(no_such_col, 'x') THEN 1 ELSE 0 END FROM t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newOmniTestExtractor(t, "db")
			_, err := q.getOmniQuerySpan(context.Background(), tc.sql)
			require.Errorf(t, err, "unresolved column %q should surface as error", tc.sql)
		})
	}
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
