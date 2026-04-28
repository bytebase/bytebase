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

func sourceColumnSetFromResources(resources []base.ColumnResource) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	for _, resource := range resources {
		result[resource] = true
	}
	return result
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
