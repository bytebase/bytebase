package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestOmniQuerySpanScaffold_QueryTypesAndAccessTables(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		wantType      base.QueryType
		wantResources []base.ColumnResource
	}{
		{
			name:      "explain_analyze_select_accesses_table",
			statement: "EXPLAIN ANALYZE SELECT * FROM t",
			wantType:  base.Select,
			wantResources: []base.ColumnResource{
				{Database: "db", Table: "t"},
			},
		},
		{
			name:      "set_is_select_with_no_access_tables",
			statement: "SET CHARSET DEFAULT",
			wantType:  base.Select,
		},
		{
			name:      "show_is_select_info_schema",
			statement: "SHOW DATABASES",
			wantType:  base.SelectInfoSchema,
		},
		{
			name:      "plain_explain_is_explain",
			statement: "EXPLAIN SELECT * FROM t",
			wantType:  base.Explain,
		},
		{
			name:      "ddl",
			statement: "CREATE TABLE t(a INT)",
			wantType:  base.DDL,
		},
		{
			name:      "dml",
			statement: "INSERT INTO t VALUES(1)",
			wantType:  base.DML,
		},
		{
			name:      "system_select",
			statement: "SELECT * FROM mysql.user",
			wantType:  base.SelectInfoSchema,
		},
		{
			name:      "standard_select_accesses_table",
			statement: "SELECT * FROM t WHERE a > 0",
			wantType:  base.Select,
			wantResources: []base.ColumnResource{
				{Database: "db", Table: "t"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("db", newOmniTestQuerySpanContext(), false).getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.wantType, span.Type)
			require.Equal(t, sourceColumnSetFromResources(tc.wantResources), span.SourceColumns)
			if tc.wantType != base.Select {
				require.Empty(t, span.Results)
			}
			require.Empty(t, span.PredicateColumns)
		})
	}
}

func TestOmniQuerySpanPhase1_SimpleSelectResultColumns(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "constant",
			statement: "SELECT 1",
			want: []base.QuerySpanResult{
				{Name: "1", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
			},
		},
		{
			name:      "bare_column",
			statement: "SELECT a FROM t",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			},
		},
		{
			name:      "alias",
			statement: "SELECT a AS x FROM t",
			want: []base.QuerySpanResult{
				{Name: "x", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			},
		},
		{
			name:      "qualified_columns",
			statement: "SELECT a, t.b, db.t.c FROM t",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
			},
		},
		{
			name:      "star",
			statement: "SELECT * FROM t",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
			},
		},
		{
			name:      "star_and_column",
			statement: "SELECT *, a FROM t",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			},
		},
		{
			name:      "table_star",
			statement: "SELECT t.* FROM t",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
			},
		},
	}

	gCtx := newOmniTestQuerySpanContext()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, base.Select, span.Type)
			require.Equal(t, tc.want, span.Results)
		})
	}
}

func TestOmniQuerySpanPhase2_ExpressionSourceMerging(t *testing.T) {
	span, err := newOmniQuerySpanExtractor("db", newOmniTestQuerySpanContext(), false).getOmniQuerySpan(
		context.Background(),
		"SELECT max(a), a-b AS c1, a=b AS c2, a>b, b in (a, c) FROM t",
	)
	require.NoError(t, err)
	require.Equal(t, []base.QuerySpanResult{
		{
			Name:          "max(a)",
			SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}),
			IsPlainField:  false,
		},
		{
			Name: "c1",
			SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
				{Database: "db", Table: "t", Column: "a"},
				{Database: "db", Table: "t", Column: "b"},
			}),
			IsPlainField: false,
		},
		{
			Name: "c2",
			SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
				{Database: "db", Table: "t", Column: "a"},
				{Database: "db", Table: "t", Column: "b"},
			}),
			IsPlainField: false,
		},
		{
			Name: "a>b",
			SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
				{Database: "db", Table: "t", Column: "a"},
				{Database: "db", Table: "t", Column: "b"},
			}),
			IsPlainField: false,
		},
		{
			Name: "b in (a, c)",
			SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
				{Database: "db", Table: "t", Column: "a"},
				{Database: "db", Table: "t", Column: "b"},
				{Database: "db", Table: "t", Column: "c"},
			}),
			IsPlainField: false,
		},
	}, span.Results)
}

func TestOmniQuerySpanPhase3_FromJoinAliasAndScope(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "table_alias",
			statement: "SELECT x.a FROM t AS x",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			},
		},
		{
			name:      "join_on_with_qualified_star_and_column",
			statement: "SELECT t1.*, t2.c, 0 FROM t1 JOIN t2 ON 1 = 1",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "c"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "c"}}), IsPlainField: true},
				{Name: "0", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
			},
		},
		{
			name:      "join_using_merges_star_columns",
			statement: "SELECT * FROM t AS t1 JOIN t AS t2 USING(a)",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
			},
		},
	}

	gCtx := newOmniTestQuerySpanContext()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.want, span.Results)
		})
	}
}

func TestOmniQuerySpanPhase4_SubqueryExpressions(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("scalar_subquery_results", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT 1 AS col_1, (SELECT(2)) AS col_2, (SELECT AVG(a + b * c) FROM t) AS avg_a_b_c FROM t",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "col_1", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
			{Name: "col_2", SourceColumns: base.SourceColumnSet{}, IsPlainField: false},
			{
				Name: "avg_a_b_c",
				SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
					{Database: "db", Table: "t", Column: "a"},
					{Database: "db", Table: "t", Column: "b"},
					{Database: "db", Table: "t", Column: "c"},
				}),
				IsPlainField: false,
			},
		}, span.Results)
		require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t"}}), span.SourceColumns)
	})

	t.Run("correlated_subquery_access_tables", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT city, (SELECT COUNT(*) FROM paintings p WHERE g.id = p.gallery_id) AS total_paintings FROM galleries g",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "city", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "galleries", Column: "city"}}), IsPlainField: true},
			{Name: "total_paintings", SourceColumns: base.SourceColumnSet{}, IsPlainField: false},
		}, span.Results)
		require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{
			{Database: "db", Table: "galleries"},
			{Database: "db", Table: "paintings"},
		}), span.SourceColumns)
	})
}

func TestOmniQuerySpanPhase5And6_CTEAndSetOperations(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "simple_cte",
			statement: "WITH t1 AS (SELECT * FROM t) SELECT * FROM t1",
			want: []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
			},
		},
		{
			name:      "cte_column_aliases",
			statement: "WITH t1(x, y, z) AS (SELECT * FROM t) SELECT * FROM t1",
			want: []base.QuerySpanResult{
				{Name: "x", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				{Name: "y", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
				{Name: "z", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
			},
		},
		{
			name:      "union_merges_positionally",
			statement: "SELECT 1 AS c1 UNION SELECT a FROM t",
			want: []base.QuerySpanResult{
				{Name: "c1", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: false},
			},
		},
		{
			name:      "recursive_keyword_non_recursive_cte",
			statement: "WITH RECURSIVE t1 AS (SELECT 1 AS c1 UNION SELECT a FROM t) SELECT * FROM t1",
			want: []base.QuerySpanResult{
				{Name: "c1", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: false},
			},
		},
	}

	gCtx := newOmniTestQuerySpanContext()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.want, span.Results)
		})
	}
}

func TestOmniQuerySpanPhase7_JSONTable(t *testing.T) {
	span, err := newOmniQuerySpanExtractor("db", newOmniTestQuerySpanContext(), false).getOmniQuerySpan(
		context.Background(),
		`SELECT *
FROM products,
JSON_TABLE(product_info, '$' COLUMNS (
  product_id INT PATH '$.id',
  product_name VARCHAR(50) PATH '$.name'
)) AS jt`,
	)
	require.NoError(t, err)
	require.Equal(t, []base.QuerySpanResult{
		{Name: "product_info", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
		{Name: "product_id", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
		{Name: "product_name", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
	}, span.Results)
}

func TestOmniQuerySpanPhase9_ResourceNotFoundFailOpen(t *testing.T) {
	metadata := &storepb.DatabaseSchemaMetadata{Name: "db"}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	span, err := newOmniQuerySpanExtractor("db", base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
		Engine:                  storepb.Engine_MYSQL,
	}, false).getOmniQuerySpan(context.Background(), "SELECT * FROM t WHERE a > 0")
	require.NoError(t, err)
	require.Equal(t, base.Select, span.Type)
	require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t"}}), span.SourceColumns)
	require.Empty(t, span.Results)
	require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))
}

func TestOmniQuerySpanSystematicMigrationRegressions(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("derived_table_column_aliases_are_applied", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT x.c1 FROM (SELECT a, b FROM t) AS x(c1, c2)",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "c1", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("in_subquery_sources_are_part_of_result_lineage", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT a IN (SELECT t2.c FROM t2) AS matched FROM t1",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{
				Name: "matched",
				SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
					{Database: "db", Table: "t1", Column: "a"},
					{Database: "db", Table: "t2", Column: "c"},
				}),
				IsPlainField: false,
			},
		}, span.Results)
	})

	t.Run("explicit_expression_nodes_do_not_drop_lineage", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			`SELECT
  EXISTS (SELECT t2.a FROM t2 WHERE t2.a = t.a) AS exists_field,
  CONVERT(a USING utf8mb4) AS converted,
  a COLLATE utf8mb4_bin AS collated,
  MATCH(a) AGAINST (b) AS matched,
  ROW(a, b) AS row_field,
  a MEMBER OF(c) AS member_field,
  a + INTERVAL b DAY AS interval_field
FROM t`,
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{
				Name: "exists_field",
				SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{
					{Database: "db", Table: "t2", Column: "a"},
				}),
				IsPlainField: false,
			},
			{Name: "converted", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: false},
			{Name: "collated", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: false},
			{Name: "matched", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t", Column: "b"}}), IsPlainField: false},
			{Name: "row_field", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t", Column: "b"}}), IsPlainField: false},
			{Name: "member_field", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t", Column: "c"}}), IsPlainField: false},
			{Name: "interval_field", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t", Column: "b"}}), IsPlainField: false},
		}, span.Results)
	})

	t.Run("legacy_dml_roots_stay_dml", func(t *testing.T) {
		tests := []string{
			"CALL p()",
			"DO 1",
			"HANDLER t OPEN",
			"HANDLER t READ FIRST",
			"HANDLER t CLOSE",
		}
		for _, statement := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			require.Equal(t, base.DML, span.Type, statement)
			require.Empty(t, span.Results, statement)
		}
	})

	t.Run("table_and_values_roots_return_select_results", func(t *testing.T) {
		tableSpan, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "TABLE t")
		require.NoError(t, err)
		require.Equal(t, base.Select, tableSpan.Type)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
			{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
		}, tableSpan.Results)

		valuesSpan, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "VALUES ROW(1, 2 + 3)")
		require.NoError(t, err)
		require.Equal(t, base.Select, valuesSpan.Type)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "1", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
			{Name: "2+3", SourceColumns: base.SourceColumnSet{}, IsPlainField: false},
		}, valuesSpan.Results)
	})
}

func TestOmniQuerySpanScenarioBatch1_RootAndQueryTypeCoverage(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("select_family_roots", func(t *testing.T) {
		tests := []struct {
			name          string
			statement     string
			wantType      base.QueryType
			wantResults   []base.QuerySpanResult
			wantResources []base.ColumnResource
		}{
			{
				name:      "values_default",
				statement: "VALUES ROW(DEFAULT)",
				wantType:  base.Select,
				wantResults: []base.QuerySpanResult{
					{Name: "DEFAULT", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
				},
			},
			{
				name:      "explain_analyze_table",
				statement: "EXPLAIN ANALYZE TABLE t",
				wantType:  base.Select,
				wantResults: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
					{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
					{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
				},
				wantResources: []base.ColumnResource{{Database: "db", Table: "t"}},
			},
			{
				name:      "explain_analyze_insert",
				statement: "EXPLAIN ANALYZE INSERT INTO t VALUES(1, 2, 3)",
				wantType:  base.DML,
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
				require.NoError(t, err)
				require.Equal(t, tc.wantType, span.Type)
				require.Equal(t, tc.wantResults, span.Results)
				require.Equal(t, sourceColumnSetFromResources(tc.wantResources), span.SourceColumns)
			})
		}
	})

	t.Run("multiple_statements_are_rejected", func(t *testing.T) {
		_, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT 1; SELECT 2;")
		require.ErrorContains(t, err, "expected exactly 1 statement")
	})

	t.Run("ddl_bucket_roots", func(t *testing.T) {
		tests := []string{
			"CREATE DATABASE d",
			"CREATE VIEW v AS SELECT a FROM t",
			"ALTER TABLE t ADD COLUMN d INT",
			"DROP TABLE t",
			"RENAME TABLE t TO t_new",
			"TRUNCATE TABLE t",
		}
		for _, statement := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			require.Equal(t, base.DDL, span.Type, statement)
			require.Empty(t, span.Results, statement)
		}
	})

	t.Run("dml_bucket_roots", func(t *testing.T) {
		tests := []string{
			"UPDATE t SET a = 1",
			"DELETE FROM t WHERE a = 1",
			"BEGIN",
			"COMMIT",
			"ROLLBACK",
			"SAVEPOINT s",
			"LOCK TABLES t READ",
			"UNLOCK TABLES",
			"PREPARE stmt FROM 'SELECT 1'",
			"EXECUTE stmt",
			"DEALLOCATE PREPARE stmt",
		}
		for _, statement := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			require.Equal(t, base.DML, span.Type, statement)
			require.Empty(t, span.Results, statement)
		}
	})
}

func TestOmniQuerySpanScenarioBatch4_QueryTypeEdges(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("legacy_query_type_buckets", func(t *testing.T) {
		tests := []struct {
			statement string
			want      base.QueryType
		}{
			{statement: "SET PASSWORD = 'new_password'", want: base.QueryTypeUnknown},
			{statement: "IMPORT TABLE FROM '/tmp/t.sdi'", want: base.DDL},
			{statement: "REPLACE INTO t VALUES(1, 2, 3)", want: base.DML},
			{statement: "LOAD DATA INFILE '/tmp/t.csv' INTO TABLE t", want: base.DML},
			{statement: "CHANGE REPLICATION SOURCE TO SOURCE_HOST = '127.0.0.1'", want: base.DML},
			{statement: "START REPLICA", want: base.DML},
			{statement: "STOP REPLICA", want: base.DML},
			{statement: "RESET REPLICA", want: base.DML},
			{statement: "PURGE BINARY LOGS TO 'mysql-bin.010'", want: base.DML},
			{statement: "RESET MASTER", want: base.DML},
			{statement: "START GROUP_REPLICATION", want: base.DML},
			{statement: "STOP GROUP_REPLICATION", want: base.DML},
			{statement: "HELP 'contents'", want: base.QueryTypeUnknown},
		}
		for _, tc := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err, tc.statement)
			require.Equal(t, tc.want, span.Type, tc.statement)
			require.Empty(t, span.Results, tc.statement)
		}
	})
}

func TestOmniQuerySpanScenarioBatch1_ExpressionCoverage(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("quoted_identifier_names", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT `select`, `Camel` AS `ExactCase` FROM keyword_table",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "select", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "keyword_table", Column: "select"}}), IsPlainField: true},
			{Name: "ExactCase", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "keyword_table", Column: "Camel"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("variable_refs_have_empty_lineage", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT @x AS user_var, @@version AS system_var",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "user_var", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
			{Name: "system_var", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
		}, span.Results)
	})

	t.Run("json_extract_operator_uses_object_lineage", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT product_info->'$.id' AS id FROM products",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "id", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: false},
		}, span.Results)
	})

	t.Run("group_concat_separator_contributes_lineage", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT GROUP_CONCAT(a SEPARATOR b) AS gc FROM t",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "gc", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t", Column: "b"}}), IsPlainField: false},
		}, span.Results)
	})

	t.Run("duplicate_output_names_are_preserved", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT a AS x, b AS x FROM t",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "x", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			{Name: "x", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
		}, span.Results)
	})
}

func TestOmniQuerySpanScenarioBatch2_FromJoinAndScopeCoverage(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("table_source_variants", func(t *testing.T) {
		tests := []struct {
			name      string
			statement string
			want      []base.QuerySpanResult
			wantSrc   []base.ColumnResource
		}{
			{
				name:      "database_qualified_table",
				statement: "SELECT db.t.a FROM db.t",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				},
				wantSrc: []base.ColumnResource{{Database: "db", Table: "t"}},
			},
			{
				name:      "parenthesized_single_table",
				statement: "SELECT a FROM (t)",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
				},
				wantSrc: []base.ColumnResource{{Database: "db", Table: "t"}},
			},
			{
				name:      "comma_separated_tables",
				statement: "SELECT t1.a, t2.b FROM t1, t2",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
					{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "b"}}), IsPlainField: true},
				},
				wantSrc: []base.ColumnResource{{Database: "db", Table: "t1"}, {Database: "db", Table: "t2"}},
			},
			{
				name:      "dual",
				statement: "SELECT 1 FROM DUAL",
				want: []base.QuerySpanResult{
					{Name: "1", SourceColumns: base.SourceColumnSet{}, IsPlainField: true},
				},
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
				require.NoError(t, err)
				require.Equal(t, tc.want, span.Results)
				require.Equal(t, sourceColumnSetFromResources(tc.wantSrc), span.SourceColumns)
			})
		}
	})

	t.Run("join_variants_expose_columns", func(t *testing.T) {
		tests := []string{
			"SELECT * FROM t1 STRAIGHT_JOIN t2",
			"SELECT * FROM t1 LEFT JOIN t2 ON t1.a = t2.a",
			"SELECT * FROM t1 RIGHT JOIN t2 ON t1.a = t2.a",
			"SELECT * FROM t1 NATURAL LEFT JOIN t2",
			"SELECT * FROM t1 NATURAL RIGHT JOIN t2",
		}
		want := []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
			{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "b"}}), IsPlainField: true},
			{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "c"}}), IsPlainField: true},
		}
		wantNonNatural := append([]base.QuerySpanResult{}, want...)
		wantNonNatural = append(wantNonNatural,
			base.QuerySpanResult{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "a"}}), IsPlainField: true},
			base.QuerySpanResult{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "b"}}), IsPlainField: true},
			base.QuerySpanResult{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "c"}}), IsPlainField: true},
		)
		for i, statement := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			if i < 3 {
				require.Equal(t, wantNonNatural, span.Results, statement)
			} else {
				require.Equal(t, want, span.Results, statement)
			}
			require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1"}, {Database: "db", Table: "t2"}}), span.SourceColumns, statement)
		}
	})

	t.Run("derived_table_edges", func(t *testing.T) {
		_, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT * FROM (SELECT a, b FROM t) AS x(c1)",
		)
		require.ErrorContains(t, err, "derived table column list length")

		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT y.c1 FROM (SELECT x.c1 FROM (SELECT a FROM t) AS x(c1)) AS y",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "c1", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)

		span, err = newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT x.a FROM t, LATERAL (SELECT t.a) AS x",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("join_dependency_edges", func(t *testing.T) {
		tests := []struct {
			name      string
			statement string
			want      []base.QuerySpanResult
			wantSrc   []base.ColumnResource
		}{
			{
				name:      "join_on_subquery_contributes_access_table",
				statement: "SELECT t1.a FROM t1 JOIN t2 ON EXISTS (SELECT t.a FROM t WHERE t.a = t2.a)",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
				},
				wantSrc: []base.ColumnResource{
					{Database: "db", Table: "t1"},
					{Database: "db", Table: "t2"},
					{Database: "db", Table: "t"},
				},
			},
			{
				name:      "nested_join_tree_preserves_visible_order",
				statement: "SELECT * FROM t1 JOIN (t2 JOIN t ON t2.a = t.a) ON t1.a = t2.a",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
					{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "b"}}), IsPlainField: true},
					{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "c"}}), IsPlainField: true},
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "a"}}), IsPlainField: true},
					{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "b"}}), IsPlainField: true},
					{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "c"}}), IsPlainField: true},
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
					{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
					{Name: "c", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "c"}}), IsPlainField: true},
				},
				wantSrc: []base.ColumnResource{
					{Database: "db", Table: "t1"},
					{Database: "db", Table: "t2"},
					{Database: "db", Table: "t"},
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
				require.NoError(t, err)
				require.Equal(t, tc.want, span.Results)
				require.Equal(t, sourceColumnSetFromResources(tc.wantSrc), span.SourceColumns)
			})
		}
	})
}

func TestOmniQuerySpanScenarioBatch2_SubqueryCTEAndSetCoverage(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("subquery_access_locations", func(t *testing.T) {
		tests := []string{
			"SELECT a FROM t WHERE EXISTS (SELECT t2.a FROM t2)",
			"SELECT a FROM t GROUP BY a HAVING EXISTS (SELECT t2.a FROM t2)",
			"SELECT a FROM t ORDER BY (SELECT t2.a FROM t2)",
		}
		for _, statement := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t"}, {Database: "db", Table: "t2"}}), span.SourceColumns, statement)
		}
	})

	t.Run("cte_and_set_error_edges", func(t *testing.T) {
		_, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"WITH c(x) AS (SELECT a, b FROM t) SELECT * FROM c",
		)
		require.ErrorContains(t, err, "CTE column list")

		_, err = newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"WITH RECURSIVE c(x) AS (SELECT a, b FROM t UNION SELECT a, b FROM t2) SELECT * FROM c",
		)
		require.ErrorContains(t, err, "different column counts")

		_, err = newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT a FROM t UNION SELECT a, b FROM t2",
		)
		require.ErrorContains(t, err, "UNION operator left has 1 fields, right has 2 fields")

		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"WITH c1 AS (SELECT a FROM c2), c2 AS (SELECT a FROM t) SELECT * FROM c1",
		)
		require.NoError(t, err)
		require.Empty(t, span.Results)
		require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))
	})

	t.Run("set_operation_in_derived_and_cte", func(t *testing.T) {
		tests := []string{
			"SELECT x.a FROM (SELECT a FROM t UNION SELECT a FROM t2) AS x",
			"WITH c AS (SELECT a FROM t UNION SELECT a FROM t2) SELECT a FROM c",
		}
		for _, statement := range tests {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), statement)
			require.NoError(t, err, statement)
			require.Equal(t, []base.QuerySpanResult{
				{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t2", Column: "a"}}), IsPlainField: true},
			}, span.Results, statement)
		}
	})

	t.Run("additional_set_operation_edges", func(t *testing.T) {
		tests := []struct {
			name      string
			statement string
			want      []base.QuerySpanResult
		}{
			{
				name:      "intersect",
				statement: "SELECT a FROM t INTERSECT SELECT a FROM t2",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t2", Column: "a"}}), IsPlainField: false},
				},
			},
			{
				name:      "except",
				statement: "SELECT a FROM t EXCEPT SELECT a FROM t2",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t2", Column: "a"}}), IsPlainField: false},
				},
			},
			{
				name:      "parenthesized_grouping",
				statement: "SELECT a FROM t UNION (SELECT a FROM t2 UNION SELECT b FROM t1)",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t2", Column: "a"}, {Database: "db", Table: "t1", Column: "b"}}), IsPlainField: false},
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
				require.NoError(t, err)
				require.Equal(t, tc.want, span.Results)
			})
		}
	})

	t.Run("scope_resolution_edges", func(t *testing.T) {
		tests := []struct {
			name      string
			statement string
			want      []base.QuerySpanResult
		}{
			{
				name:      "nested_subquery_nearest_scope_alias",
				statement: "SELECT (SELECT x.a FROM t2 AS x WHERE x.b = outer_t.b) AS inner_a FROM t AS outer_t",
				want: []base.QuerySpanResult{
					{Name: "inner_a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "a"}}), IsPlainField: true},
				},
			},
			{
				name:      "unqualified_inner_column_uses_legacy_resolution_order",
				statement: "SELECT (SELECT a FROM t2) AS inner_a FROM t1",
				want: []base.QuerySpanResult{
					{Name: "inner_a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
				},
			},
			{
				name:      "inner_alias_shadow_keeps_legacy_outer_first_lookup",
				statement: "SELECT (SELECT x.a FROM t2 AS x) AS inner_a FROM t1 AS x",
				want: []base.QuerySpanResult{
					{Name: "inner_a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t1", Column: "a"}}), IsPlainField: true},
				},
			},
			{
				name:      "cte_name_shadows_physical_table",
				statement: "WITH t AS (SELECT b FROM t2) SELECT * FROM t",
				want: []base.QuerySpanResult{
					{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t2", Column: "b"}}), IsPlainField: true},
				},
			},
			{
				name:      "nested_cte_visibility_order",
				statement: "WITH c1 AS (SELECT a FROM t), c2 AS (SELECT a FROM c1 UNION SELECT b FROM t2) SELECT * FROM c2",
				want: []base.QuerySpanResult{
					{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t2", Column: "b"}}), IsPlainField: false},
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
				require.NoError(t, err)
				require.Equal(t, tc.want, span.Results)
			})
		}
	})

	t.Run("recursive_cte_reaches_stable_source_closure", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"WITH RECURSIVE c(x) AS (SELECT a FROM t UNION SELECT t2.b FROM t2 JOIN c ON c.x = t2.a) SELECT * FROM c",
		)
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "x", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}, {Database: "db", Table: "t2", Column: "b"}}), IsPlainField: false},
		}, span.Results)
	})
}

func TestOmniQuerySpanScenarioBatch4_JSONTableCoverage(t *testing.T) {
	span, err := newOmniQuerySpanExtractor("db", newOmniTestQuerySpanContext(), false).getOmniQuerySpan(
		context.Background(),
		`SELECT *
FROM products,
JSON_TABLE(product_info, '$' COLUMNS (
  product_id INT PATH '$.id',
  NESTED PATH '$.variants[*]' COLUMNS (
    variant_id INT PATH '$.id',
    variant_name VARCHAR(50) PATH '$.name'
  )
)) AS jt`,
	)
	require.NoError(t, err)
	require.Equal(t, []base.QuerySpanResult{
		{Name: "product_info", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
		{Name: "product_id", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
		{Name: "variant_id", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
		{Name: "variant_name", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
	}, span.Results)
}

func TestOmniQuerySpanScenarioBatch4_AccessTableExpressions(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	tests := []struct {
		name      string
		statement string
		want      []base.ColumnResource
	}{
		{
			name:      "function_argument_subquery",
			statement: "SELECT COALESCE((SELECT t2.a FROM t2), a) FROM t",
			want: []base.ColumnResource{
				{Database: "db", Table: "t"},
				{Database: "db", Table: "t2"},
			},
		},
		{
			name:      "values_subquery",
			statement: "VALUES ROW((SELECT a FROM t))",
			want: []base.ColumnResource{
				{Database: "db", Table: "t"},
			},
		},
		{
			name:      "call_argument_subquery",
			statement: "CALL p((SELECT a FROM t))",
			want: []base.ColumnResource{
				{Database: "db", Table: "t"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), tc.statement)
			require.NoError(t, err)
			require.Equal(t, sourceColumnSetFromResources(tc.want), span.SourceColumns)
		})
	}
}

func TestOmniQuerySpanScenarioBatch3_ErrorAndMetadataCoverage(t *testing.T) {
	t.Run("error_semantics", func(t *testing.T) {
		gCtx := newOmniTestQuerySpanContext()

		_, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT a INTO @x FROM t")
		require.ErrorContains(t, err, "unsupported select statement with into")

		_, err = newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT FROM")
		require.Error(t, err)

		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "")
		require.NoError(t, err)
		require.Equal(t, base.Select, span.Type)
		require.Empty(t, span.Results)
		require.Empty(t, span.SourceColumns)
	})

	t.Run("view_lineage", func(t *testing.T) {
		gCtx := newOmniViewTestQuerySpanContext()

		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT * FROM v")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "va", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
			{Name: "b", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "b"}}), IsPlainField: true},
		}, span.Results)

		span, err = newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT * FROM v2")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "va", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("missing_view_dependency_fails_open", func(t *testing.T) {
		gCtx := newOmniViewTestQuerySpanContext()
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT * FROM broken_v")
		require.NoError(t, err)
		require.Equal(t, base.Select, span.Type)
		require.Empty(t, span.Results)
		require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))
	})

	t.Run("system_schema_detection", func(t *testing.T) {
		tests := []struct {
			name                string
			statement           string
			ignoreCaseSensitive bool
			wantType            base.QueryType
			wantSource          []base.ColumnResource
			wantNotFound        bool
		}{
			{
				name:      "information_schema_lowercase",
				statement: "SELECT * FROM information_schema.tables",
				wantType:  base.SelectInfoSchema,
			},
			{
				name:                "information_schema_uppercase_ignore_case",
				statement:           "SELECT * FROM INFORMATION_SCHEMA.tables",
				ignoreCaseSensitive: true,
				wantType:            base.SelectInfoSchema,
			},
			{
				name:      "information_schema_uppercase_case_sensitive",
				statement: "SELECT * FROM INFORMATION_SCHEMA.tables",
				wantType:  base.SelectInfoSchema,
			},
			{
				name:      "performance_schema_lowercase",
				statement: "SELECT * FROM performance_schema.events_statements_current",
				wantType:  base.SelectInfoSchema,
			},
			{
				name:      "mysql_lowercase",
				statement: "SELECT * FROM mysql.user",
				wantType:  base.SelectInfoSchema,
			},
			{
				name:                "mysql_uppercase_ignore_case",
				statement:           "SELECT * FROM MYSQL.user",
				ignoreCaseSensitive: true,
				wantType:            base.SelectInfoSchema,
			},
			{
				name:         "mysql_uppercase_case_sensitive",
				statement:    "SELECT * FROM MYSQL.user",
				wantType:     base.Select,
				wantNotFound: true,
				wantSource: []base.ColumnResource{
					{Database: "MYSQL", Table: "user"},
				},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				span, err := newOmniQuerySpanExtractor("db", newOmniTestQuerySpanContext(), tc.ignoreCaseSensitive).getOmniQuerySpan(context.Background(), tc.statement)
				require.NoError(t, err)
				require.Equal(t, tc.wantType, span.Type)
				require.Equal(t, sourceColumnSetFromResources(tc.wantSource), span.SourceColumns)
				if tc.wantNotFound {
					require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))
				} else {
					require.NoError(t, span.NotFoundError)
				}
			})
		}
	})
}

func TestOmniQuerySpanScenarioBatch4_CaseSensitivityCoverage(t *testing.T) {
	gCtx := newOmniTestQuerySpanContext()

	t.Run("table_alias_case_sensitivity", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT X.a FROM t AS x")
		require.NoError(t, err)
		require.Empty(t, span.Results)
		require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))

		span, err = newOmniQuerySpanExtractor("db", gCtx, true).getOmniQuerySpan(context.Background(), "SELECT X.a FROM t AS x")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("derived_table_alias_case_sensitivity", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT X.a FROM (SELECT a FROM t) AS x")
		require.NoError(t, err)
		require.Empty(t, span.Results)
		require.ErrorAs(t, span.NotFoundError, new(*base.ResourceNotFoundError))

		span, err = newOmniQuerySpanExtractor("db", gCtx, true).getOmniQuerySpan(context.Background(), "SELECT X.a FROM (SELECT a FROM t) AS x")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("cte_name_case_sensitivity", func(t *testing.T) {
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "WITH c AS (SELECT a FROM t) SELECT C.a FROM c AS C")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
	})
}

func TestOmniQuerySpanScenarioBatch5_StarRocksAndMetadataCoverage(t *testing.T) {
	t.Run("starrocks_cluster_qualified_tables", func(t *testing.T) {
		gCtx := newOmniStarRocksTestQuerySpanContext()
		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT `cluster_name:db`.products.product_info FROM `cluster_name:db`.products",
		)
		require.NoError(t, err)
		require.Equal(t, base.Select, span.Type)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "product_info", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products", Column: "product_info"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "products"}}), span.SourceColumns)
	})

	t.Run("starrocks_system_user_mixing_uses_normalized_database", func(t *testing.T) {
		gCtx := newOmniStarRocksTestQuerySpanContext()
		_, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(
			context.Background(),
			"SELECT * FROM products, `cluster_name:information_schema`.tables",
		)
		require.ErrorIs(t, err, base.MixUserSystemTablesError)
	})

	t.Run("mysql_and_starrocks_lineage_match_without_cluster_prefix", func(t *testing.T) {
		mysqlSpan, err := newOmniQuerySpanExtractor("db", newOmniTestQuerySpanContext(), false).getOmniQuerySpan(
			context.Background(),
			"SELECT product_info FROM products",
		)
		require.NoError(t, err)

		starrocksSpan, err := newOmniQuerySpanExtractor("db", newOmniStarRocksTestQuerySpanContext(), false).getOmniQuerySpan(
			context.Background(),
			"SELECT product_info FROM products",
		)
		require.NoError(t, err)
		require.Equal(t, mysqlSpan.Results, starrocksSpan.Results)
		require.Equal(t, mysqlSpan.SourceColumns, starrocksSpan.SourceColumns)
	})

	t.Run("duplicate_database_case_uses_ignore_case_setting", func(t *testing.T) {
		gCtx := newOmniCaseCollisionTestQuerySpanContext()

		span, err := newOmniQuerySpanExtractor("db", gCtx, false).getOmniQuerySpan(context.Background(), "SELECT DB.t.a FROM DB.t")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "DB", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "DB", Table: "t"}}), span.SourceColumns)

		span, err = newOmniQuerySpanExtractor("db", gCtx, true).getOmniQuerySpan(context.Background(), "SELECT DB.t.a FROM DB.t")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "a", SourceColumns: sourceColumnSetFromResources([]base.ColumnResource{{Database: "db", Table: "t", Column: "a"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, sourceColumnSetFromResources([]base.ColumnResource{{Database: "DB", Table: "t"}}), span.SourceColumns)
	})
}

func sourceColumnSetFromResources(resources []base.ColumnResource) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	for _, resource := range resources {
		result[resource] = true
	}
	return result
}

func newOmniViewTestQuerySpanContext() base.GetQuerySpanContext {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a"},
							{Name: "b"},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v",
						Definition: "SELECT a AS va, b FROM t",
					},
					{
						Name:       "v2",
						Definition: "SELECT va FROM v",
					},
					{
						Name:       "broken_v",
						Definition: "SELECT missing FROM t",
					},
				},
			},
		},
	}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	return base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
		Engine:                  storepb.Engine_MYSQL,
	}
}

func newOmniTestQuerySpanContext() base.GetQuerySpanContext {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a"},
							{Name: "b"},
							{Name: "c"},
						},
					},
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a"},
							{Name: "b"},
							{Name: "c"},
						},
					},
					{
						Name: "t2",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a"},
							{Name: "b"},
							{Name: "c"},
						},
					},
					{
						Name: "galleries",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id"},
							{Name: "city"},
						},
					},
					{
						Name: "paintings",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id"},
							{Name: "gallery_id"},
						},
					},
					{
						Name: "products",
						Columns: []*storepb.ColumnMetadata{
							{Name: "product_info"},
						},
					},
					{
						Name: "keyword_table",
						Columns: []*storepb.ColumnMetadata{
							{Name: "select"},
							{Name: "Camel"},
						},
					},
				},
			},
		},
	}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
	return base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
		Engine:                  storepb.Engine_MYSQL,
	}
}

func newOmniStarRocksTestQuerySpanContext() base.GetQuerySpanContext {
	gCtx := newOmniTestQuerySpanContext()
	gCtx.Engine = storepb.Engine_STARROCKS
	return gCtx
}

func newOmniCaseCollisionTestQuerySpanContext() base.GetQuerySpanContext {
	dbMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a"},
						},
					},
				},
			},
		},
	}
	upperMetadata := &storepb.DatabaseSchemaMetadata{
		Name: "DB",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t",
						Columns: []*storepb.ColumnMetadata{
							{Name: "a"},
						},
					},
				},
			},
		},
	}
	databaseMetadataGetter, databaseNameLister := buildMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{dbMetadata, upperMetadata})
	return base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: databaseMetadataGetter,
		ListDatabaseNamesFunc:   databaseNameLister,
		Engine:                  storepb.Engine_MYSQL,
	}
}
