package redshift

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestRedshiftOmniQuerySpanSupportedShapes(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"analytics", "public"}, redshiftOmniQuerySpanContext(t))

	tests := []struct {
		name             string
		statement        string
		wantResults      []base.QuerySpanResult
		wantSource       []base.ColumnResource
		wantPredicate    []base.ColumnResource
		wantNotFoundText string
	}{
		{
			name:      "plain columns and expression",
			statement: "SELECT id, amount AS total, amount + tax AS gross FROM orders WHERE status = 'open'",
			wantResults: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "total", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}}), IsPlainField: true},
				{Name: "gross", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{
					{Database: "db", Schema: "public", Table: "orders", Column: "amount"},
					{Database: "db", Schema: "public", Table: "orders", Column: "tax"},
				})},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
			},
			wantPredicate: []base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "status"}},
		},
		{
			name:      "schema-qualified external table",
			statement: "SELECT external_id FROM spectrum.spectrum_orders",
			wantResults: []base.QuerySpanResult{
				{Name: "external_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "spectrum", Table: "spectrum_orders", Column: "external_id"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{{Database: "db", Schema: "spectrum", Table: "spectrum_orders"}},
		},
		{
			name:      "cte lineage",
			statement: "WITH q(x) AS (SELECT id FROM orders) SELECT x FROM q",
			wantResults: []base.QuerySpanResult{
				{Name: "x", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
			},
		},
		{
			name:      "view definition lineage",
			statement: "SELECT id, amount FROM active_orders",
			wantResults: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "amount", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "active_orders"},
			},
		},
		{
			name:             "missing relation returns NotFoundError span",
			statement:        "SELECT id FROM missing_orders",
			wantResults:      []base.QuerySpanResult{},
			wantSource:       []base.ColumnResource{{Database: "db"}},
			wantNotFoundText: "missing_orders",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := q.getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Equal(t, base.Select, span.Type)
			require.Equal(t, test.wantResults, span.Results)
			require.Equal(t, redshiftSourceColumnSetFromList(test.wantSource), span.SourceColumns)
			require.Equal(t, redshiftSourceColumnSetFromList(test.wantPredicate), span.PredicateColumns)
			if test.wantNotFoundText != "" {
				require.Error(t, span.NotFoundError)
				require.ErrorContains(t, span.NotFoundError, test.wantNotFoundText)
			} else {
				require.NoError(t, span.NotFoundError)
			}
		})
	}
}

func TestRedshiftQuerySpanEntrypointDoesNotDependOnANTLR(t *testing.T) {
	for _, path := range []string{"query_span.go", "query_span_extractor_omni.go"} {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		source := string(content)
		require.NotContains(t, source, "github.com/antlr4-go/antlr/v4", path)
		require.NotContains(t, source, "github.com/bytebase/parser/redshift", path)
		require.NotContains(t, source, "ParseRedshift(", path)
	}
}

func TestRedshiftOmniQuerySpanUsesLazyRelationResolver(t *testing.T) {
	content, err := os.ReadFile("query_span_extractor_omni.go")
	require.NoError(t, err)
	source := string(content)
	require.Contains(t, source, "SetRelationResolver")
	require.NotContains(t, source, "pendingOmniCatalogView")
	require.NotContains(t, source, "buildOmniQuerySpanCatalog")
	require.NotContains(t, source, "orderPendingOmniCatalogViews")
	require.NotContains(t, source, "createQuerySpanViewDDL")
}

func TestRedshiftOmniQuerySpanNonSelectTypes(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"public"}, redshiftOmniQuerySpanContext(t))

	tests := []struct {
		name      string
		statement string
		want      base.QueryType
	}{
		{
			name:      "copy is dml",
			statement: "COPY orders FROM 's3://bucket/orders.csv' IAM_ROLE 'arn:aws:iam::123456789012:role/redshift';",
			want:      base.DML,
		},
		{
			name:      "call is dml",
			statement: "CALL refresh_orders();",
			want:      base.DML,
		},
		{
			name:      "create table is ddl",
			statement: "CREATE TABLE created_orders(id int);",
			want:      base.DDL,
		},
		{
			name:      "comment is ddl",
			statement: "COMMENT ON TABLE orders IS 'hello';",
			want:      base.DDL,
		},
		{
			name:      "create user is ddl",
			statement: "CREATE USER report_user PASSWORD 'Password1';",
			want:      base.DDL,
		},
		{
			name:      "alter user is ddl",
			statement: "ALTER USER report_user RENAME TO report_user2;",
			want:      base.DDL,
		},
		{
			name:      "alter user options is ddl",
			statement: "ALTER USER report_user PASSWORD 'Password2';",
			want:      base.DDL,
		},
		{
			name:      "alter user set is ddl",
			statement: "ALTER USER report_user SET search_path TO public;",
			want:      base.DDL,
		},
		{
			name:      "drop user is ddl",
			statement: "DROP USER report_user;",
			want:      base.DDL,
		},
		{
			name:      "grant role is ddl",
			statement: "GRANT report_role TO report_user;",
			want:      base.DDL,
		},
		{
			name:      "show tables is information schema",
			statement: "SHOW TABLES;",
			want:      base.SelectInfoSchema,
		},
		{
			name:      "desc table is information schema",
			statement: "DESC TABLE orders;",
			want:      base.SelectInfoSchema,
		},
		{
			name:      "system table query is information schema",
			statement: "SELECT oid FROM pg_class;",
			want:      base.SelectInfoSchema,
		},
		{
			name:      "explain select is explain",
			statement: "EXPLAIN SELECT id FROM orders;",
			want:      base.Explain,
		},
		{
			name:      "explain analyze select has sources but no result columns",
			statement: "EXPLAIN ANALYZE SELECT id FROM orders;",
			want:      base.Select,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := q.getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.NotNil(t, span)
			require.Equal(t, test.want, span.Type)
			require.Empty(t, span.Results)
			if test.name == "explain analyze select has sources but no result columns" {
				require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders"}}), span.SourceColumns)
			} else {
				require.Empty(t, span.SourceColumns)
			}
		})
	}

	t.Run("unload is dml export with inner sources", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "UNLOAD ('SELECT id FROM orders WHERE status = ''open''') TO 's3://bucket/out' IAM_ROLE DEFAULT;")
		require.NoError(t, err)
		require.NotNil(t, span)
		require.Equal(t, base.DML, span.Type)
		require.Empty(t, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders"}}), span.SourceColumns)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "status"}}), span.PredicateColumns)
	})
}

func TestRedshiftQuerySpanUsesOmniPath(t *testing.T) {
	span, err := GetQuerySpan(context.Background(), redshiftOmniQuerySpanContext(t), base.Statement{Text: "SELECT id FROM orders"}, "db", "public", false)
	require.NoError(t, err)
	require.NotNil(t, span)
	require.Equal(t, []base.QuerySpanResult{
		{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
	}, span.Results)
}

func TestRedshiftOmniQuerySpanDefaultSearchPath(t *testing.T) {
	span, err := newOmniQuerySpanExtractor("db", nil, redshiftOmniQuerySpanContext(t)).getOmniQuerySpan(context.Background(), "SELECT id FROM orders")
	require.NoError(t, err)
	require.Equal(t, []base.QuerySpanResult{
		{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
	}, span.Results)
	require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders"}}), span.SourceColumns)
}

func TestRedshiftOmniQuerySpanRootAndQueryTypeCoverage(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"public"}, redshiftOmniQuerySpanContext(t))

	t.Run("empty statement returns empty select span", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "")
		require.NoError(t, err)
		require.Equal(t, base.Select, span.Type)
		require.Empty(t, span.Results)
		require.Empty(t, span.SourceColumns)
	})

	t.Run("multiple statements are rejected", func(t *testing.T) {
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT 1; SELECT 2;")
		require.ErrorContains(t, err, "expected exactly 1 statement")
	})

	tests := []struct {
		name      string
		statement string
		want      base.QueryType
	}{
		{name: "set is select", statement: "SET search_path TO public;", want: base.Select},
		{name: "select into is ddl", statement: "SELECT id INTO new_orders FROM orders;", want: base.DDL},
		{name: "set operation select into is ddl", statement: "SELECT id INTO new_orders FROM orders UNION SELECT id FROM orders;", want: base.DDL},
		{name: "create table as is ddl", statement: "CREATE TABLE copied_orders AS SELECT id FROM orders;", want: base.DDL},
		{name: "drop table is ddl", statement: "DROP TABLE orders;", want: base.DDL},
		{name: "alter table is ddl", statement: "ALTER TABLE orders ADD COLUMN note varchar(100);", want: base.DDL},
		{name: "truncate is ddl", statement: "TRUNCATE TABLE orders;", want: base.DDL},
		{name: "insert is dml", statement: "INSERT INTO orders(id) SELECT event_id FROM analytics.events;", want: base.DML},
		{name: "update is dml", statement: "UPDATE orders SET amount = amount + 1 WHERE id = 1;", want: base.DML},
		{name: "delete is dml", statement: "DELETE FROM orders WHERE id = 1;", want: base.DML},
		{name: "refresh materialized view is dml", statement: "REFRESH MATERIALIZED VIEW order_totals_mv;", want: base.DML},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := q.getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.Equal(t, test.want, span.Type)
			require.Empty(t, span.Results)
		})
	}
}

func TestRedshiftOmniQuerySpanExpressionCoverage(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"public"}, redshiftOmniQuerySpanContext(t))

	tests := []struct {
		name      string
		statement string
		want      []base.QuerySpanResult
	}{
		{
			name:      "star expands table columns",
			statement: "SELECT * FROM orders",
			want: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "amount", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}}), IsPlainField: true},
				{Name: "tax", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "tax"}}), IsPlainField: true},
				{Name: "status", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "status"}}), IsPlainField: true},
				{Name: "customer_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "customer_id"}}), IsPlainField: true},
			},
		},
		{
			name:      "table alias star expands aliased table",
			statement: "SELECT o.* FROM orders AS o",
			want: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "amount", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}}), IsPlainField: true},
				{Name: "tax", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "tax"}}), IsPlainField: true},
				{Name: "status", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "status"}}), IsPlainField: true},
				{Name: "customer_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "customer_id"}}), IsPlainField: true},
			},
		},
		{
			name:      "duplicate output names are preserved",
			statement: "SELECT amount AS value, tax AS value FROM orders",
			want: []base.QuerySpanResult{
				{Name: "value", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}}), IsPlainField: true},
				{Name: "value", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "tax"}}), IsPlainField: true},
			},
		},
		{
			name:      "quoted identifiers preserve exact output names",
			statement: `SELECT "select", "Camel" AS "ExactCase" FROM keyword_table`,
			want: []base.QuerySpanResult{
				{Name: "select", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "keyword_table", Column: "select"}}), IsPlainField: true},
				{Name: "ExactCase", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "keyword_table", Column: "Camel"}}), IsPlainField: true},
			},
		},
		{
			name:      "case cast and coalesce merge expression sources",
			statement: "SELECT CASE WHEN amount > 0 THEN tax ELSE customer_id END AS case_value, CAST(amount AS decimal(10,2)) AS cast_value, COALESCE(amount, tax) AS coalesced FROM orders",
			want: []base.QuerySpanResult{
				{Name: "case_value", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{
					{Database: "db", Schema: "public", Table: "orders", Column: "amount"},
					{Database: "db", Schema: "public", Table: "orders", Column: "tax"},
					{Database: "db", Schema: "public", Table: "orders", Column: "customer_id"},
				})},
				{Name: "cast_value", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}})},
				{Name: "coalesced", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{
					{Database: "db", Schema: "public", Table: "orders", Column: "amount"},
					{Database: "db", Schema: "public", Table: "orders", Column: "tax"},
				})},
			},
		},
		{
			name:      "aggregate expressions keep argument sources",
			statement: "SELECT SUM(amount) AS total, AVG(tax) AS avg_tax FROM orders",
			want: []base.QuerySpanResult{
				{Name: "total", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}})},
				{Name: "avg_tax", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "tax"}})},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := q.getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.Equal(t, test.want, span.Results)
		})
	}
}

func TestRedshiftOmniQuerySpanJoinAndScopeCoverage(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"analytics", "public"}, redshiftOmniQuerySpanContext(t))

	tests := []struct {
		name          string
		statement     string
		wantResults   []base.QuerySpanResult
		wantSource    []base.ColumnResource
		wantPredicate []base.ColumnResource
	}{
		{
			name:      "join aliases resolve both sides",
			statement: "SELECT o.id, c.name FROM orders AS o JOIN customers AS c ON o.customer_id = c.id WHERE c.region = 'apac'",
			wantResults: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "name", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "customers", Column: "name"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
				{Database: "db", Schema: "public", Table: "customers"},
			},
			wantPredicate: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders", Column: "customer_id"},
				{Database: "db", Schema: "public", Table: "customers", Column: "id"},
				{Database: "db", Schema: "public", Table: "customers", Column: "region"},
			},
		},
		{
			name:      "comma separated tables preserve access tables",
			statement: "SELECT o.id, c.name FROM orders o, customers c WHERE o.customer_id = c.id",
			wantResults: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "name", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "customers", Column: "name"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
				{Database: "db", Schema: "public", Table: "customers"},
			},
			wantPredicate: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders", Column: "customer_id"},
				{Database: "db", Schema: "public", Table: "customers", Column: "id"},
			},
		},
		{
			name:      "derived table aliases flow source columns",
			statement: "SELECT d.order_id, d.gross FROM (SELECT id AS order_id, amount + tax AS gross FROM orders) AS d",
			wantResults: []base.QuerySpanResult{
				{Name: "order_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "gross", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{
					{Database: "db", Schema: "public", Table: "orders", Column: "amount"},
					{Database: "db", Schema: "public", Table: "orders", Column: "tax"},
				}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{{Database: "db", Schema: "public", Table: "orders"}},
		},
		{
			name:      "search path resolves analytics before public",
			statement: "SELECT event_id FROM events",
			wantResults: []base.QuerySpanResult{
				{Name: "event_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "analytics", Table: "events", Column: "event_id"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{{Database: "db", Schema: "analytics", Table: "events"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := q.getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.Equal(t, test.wantResults, span.Results)
			require.Equal(t, redshiftSourceColumnSetFromList(test.wantSource), span.SourceColumns)
			require.Equal(t, redshiftSourceColumnSetFromList(test.wantPredicate), span.PredicateColumns)
		})
	}
}

func TestRedshiftOmniQuerySpanSubqueryAndCTECoverage(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"public"}, redshiftOmniQuerySpanContext(t))

	tests := []struct {
		name       string
		statement  string
		want       []base.QuerySpanResult
		wantSource []base.ColumnResource
	}{
		{
			name:      "target subquery carries inner lineage",
			statement: "SELECT (SELECT name FROM customers WHERE customers.id = orders.customer_id) AS customer_name FROM orders",
			want: []base.QuerySpanResult{
				{Name: "customer_name", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "customers", Column: "name"}})},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
				{Database: "db", Schema: "public", Table: "customers"},
			},
		},
		{
			name:      "where exists contributes access tables",
			statement: "SELECT id FROM orders WHERE EXISTS (SELECT 1 FROM line_items WHERE line_items.order_id = orders.id)",
			want: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
				{Database: "db", Schema: "public", Table: "line_items"},
			},
		},
		{
			name:      "cte name shadows physical table",
			statement: "WITH orders AS (SELECT id FROM line_items) SELECT id FROM orders",
			want: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "line_items", Column: "id"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{{Database: "db", Schema: "public", Table: "line_items"}},
		},
		{
			name:      "multiple independent ctes join",
			statement: "WITH base AS (SELECT id FROM orders), named AS (SELECT name FROM customers) SELECT base.id, named.name FROM base JOIN named ON true",
			want: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
				{Name: "name", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "customers", Column: "name"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{
				{Database: "db", Schema: "public", Table: "orders"},
				{Database: "db", Schema: "public", Table: "customers"},
			},
		},
		{
			name:      "nested cte shadow does not hide outer physical table",
			statement: "SELECT id FROM orders WHERE EXISTS (WITH orders AS (SELECT 1) SELECT 1 FROM orders)",
			want: []base.QuerySpanResult{
				{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
			},
			wantSource: []base.ColumnResource{{Database: "db", Schema: "public", Table: "orders"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			span, err := q.getOmniQuerySpan(context.Background(), test.statement)
			require.NoError(t, err)
			require.Equal(t, test.want, span.Results)
			require.Equal(t, redshiftSourceColumnSetFromList(test.wantSource), span.SourceColumns)
		})
	}
}

func TestRedshiftOmniQuerySpanMetadataAndErrorCoverage(t *testing.T) {
	q := newOmniQuerySpanExtractor("db", []string{"public"}, redshiftOmniQuerySpanContext(t))

	t.Run("materialized view definition lineage", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT customer_id, total_amount FROM order_totals_mv")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "customer_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "customer_id"}}), IsPlainField: true},
			{Name: "total_amount", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "amount"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "order_totals_mv"}}), span.SourceColumns)
	})

	t.Run("view definition waits for later dependency", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT id FROM a_view")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "a_view"}}), span.SourceColumns)
	})

	t.Run("view definition resolves unqualified dependencies from view schema", func(t *testing.T) {
		metadata := &storepb.DatabaseSchemaMetadata{
			Name:       "db",
			SearchPath: "analytics, public",
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "public",
					Tables: []*storepb.TableMetadata{
						{
							Name:    "orders",
							Columns: []*storepb.ColumnMetadata{{Name: "id", Type: "int"}},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name:       "unqualified_orders_view",
							Definition: "CREATE VIEW public.unqualified_orders_view AS SELECT id FROM orders;",
							Columns:    []*storepb.ColumnMetadata{{Name: "id", Type: "int"}},
						},
					},
				},
				{
					Name: "analytics",
					Tables: []*storepb.TableMetadata{
						{
							Name: "orders",
							Columns: []*storepb.ColumnMetadata{
								{Name: "id", Type: "int"},
								{Name: "event_id", Type: "int"},
							},
						},
					},
				},
			},
		}
		getter, lister := redshiftMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{metadata})
		localQ := newOmniQuerySpanExtractor("db", []string{"analytics", "public"}, base.GetQuerySpanContext{
			GetDatabaseMetadataFunc: getter,
			ListDatabaseNamesFunc:   lister,
		})
		span, err := localQ.getOmniQuerySpan(context.Background(), "SELECT id FROM unqualified_orders_view")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "unqualified_orders_view"}}), span.SourceColumns)
	})

	t.Run("full create view definition applies column aliases", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT order_id FROM aliased_orders")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "order_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "aliased_orders"}}), span.SourceColumns)
	})

	t.Run("full create materialized view definition applies column aliases", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT order_id FROM aliased_orders_mv")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "order_id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
		}, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "aliased_orders_mv"}}), span.SourceColumns)
	})

	t.Run("sql udf body contributes source columns", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT reporting_fn()")
		require.NoError(t, err)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders"}}), span.SourceColumns)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "reporting_fn", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}})},
		}, span.Results)
	})

	t.Run("schema qualified sql udf body contributes source columns", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT analytics.reporting_fn()")
		require.NoError(t, err)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "analytics", Table: "events"}}), span.SourceColumns)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "reporting_fn", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "analytics", Table: "events", Column: "event_id"}})},
		}, span.Results)
	})

	t.Run("sql udf body result sources propagate through cte", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "WITH c AS (SELECT reporting_fn() AS x) SELECT x FROM c")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "x", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("sql udf body result sources propagate through subquery", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT c.x FROM (SELECT reporting_fn() AS x) AS c")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "x", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "orders", Column: "id"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("sql udf body missing relation returns not found span", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT stale_reporting_fn()")
		require.NoError(t, err)
		require.Error(t, span.NotFoundError)
		require.ErrorContains(t, span.NotFoundError, "missing_orders")
		require.Equal(t, base.Select, span.Type)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db"}}), span.SourceColumns)
	})

	t.Run("unsupported sql udf body returns function not supported span", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT unsupported_reporting_fn()")
		require.NoError(t, err)
		require.Error(t, span.FunctionNotSupportedError)
		require.ErrorContains(t, span.FunctionNotSupportedError, "public.unsupported_reporting_fn")
		require.Equal(t, base.Select, span.Type)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db"}}), span.SourceColumns)
	})

	t.Run("sequence read uses default sequence columns", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT last_value, log_cnt, is_called FROM order_id_seq")
		require.NoError(t, err)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "order_id_seq"}}), span.SourceColumns)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "last_value", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "order_id_seq", Column: "last_value"}}), IsPlainField: true},
			{Name: "log_cnt", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "order_id_seq", Column: "log_cnt"}}), IsPlainField: true},
			{Name: "is_called", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "order_id_seq", Column: "is_called"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("view fallback uses declared columns when definition is missing", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT id, amount FROM fallback_view")
		require.NoError(t, err)
		require.Equal(t, []base.QuerySpanResult{
			{Name: "id", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "fallback_view", Column: "id"}}), IsPlainField: true},
			{Name: "amount", SourceColumns: redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db", Schema: "public", Table: "fallback_view", Column: "amount"}}), IsPlainField: true},
		}, span.Results)
	})

	t.Run("missing relation inside subquery returns not found span", func(t *testing.T) {
		span, err := q.getOmniQuerySpan(context.Background(), "SELECT id FROM orders WHERE EXISTS (SELECT 1 FROM missing_orders)")
		require.NoError(t, err)
		require.Error(t, span.NotFoundError)
		require.ErrorContains(t, span.NotFoundError, "missing_orders")
		require.Empty(t, span.Results)
		require.Equal(t, redshiftSourceColumnSetFromList([]base.ColumnResource{{Database: "db"}}), span.SourceColumns)
	})

	t.Run("mixed system and user tables are rejected", func(t *testing.T) {
		_, err := q.getOmniQuerySpan(context.Background(), "SELECT orders.id FROM orders, pg_class")
		require.ErrorIs(t, err, base.MixUserSystemTablesError)
	})

	t.Run("syntax error propagates through public entrypoint", func(t *testing.T) {
		_, err := GetQuerySpan(context.Background(), redshiftOmniQuerySpanContext(t), base.Statement{
			Text:  "SELECT * FORM orders;",
			Start: &storepb.Position{Line: 7, Column: 1},
		}, "db", "public", false)
		require.Error(t, err)
		var syntaxErr *base.SyntaxError
		require.ErrorAs(t, err, &syntaxErr)
		require.Equal(t, int32(7), syntaxErr.Position.Line)
	})
}

func redshiftOmniQuerySpanContext(t *testing.T) base.GetQuerySpanContext {
	t.Helper()
	getter, lister := redshiftMockDatabaseMetadataGetter([]*storepb.DatabaseSchemaMetadata{redshiftOmniQuerySpanMetadata()})
	return base.GetQuerySpanContext{
		GetDatabaseMetadataFunc: getter,
		ListDatabaseNamesFunc:   lister,
	}
}

func redshiftOmniQuerySpanMetadata() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
		Name:       "db",
		SearchPath: "analytics, public",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "amount", Type: "numeric"},
							{Name: "tax", Type: "numeric"},
							{Name: "status", Type: "varchar"},
							{Name: "customer_id", Type: "int"},
						},
					},
					{
						Name: "customers",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "name", Type: "varchar"},
							{Name: "region", Type: "varchar"},
						},
					},
					{
						Name: "line_items",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "order_id", Type: "int"},
							{Name: "sku", Type: "varchar"},
						},
					},
					{
						Name: "keyword_table",
						Columns: []*storepb.ColumnMetadata{
							{Name: "select", Type: "int"},
							{Name: "Camel", Type: "int"},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "active_orders",
						Definition: "CREATE VIEW public.active_orders AS SELECT id, amount FROM public.orders WHERE status = 'open';",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "amount", Type: "numeric"},
						},
					},
					{
						Name:       "a_view",
						Definition: "CREATE VIEW public.a_view AS SELECT id FROM public.z_view;",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
					{
						Name: "fallback_view",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
							{Name: "amount", Type: "numeric"},
						},
					},
					{
						Name:       "z_view",
						Definition: "CREATE VIEW public.z_view AS SELECT id FROM public.orders;",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "int"},
						},
					},
					{
						Name:       "aliased_orders",
						Definition: "CREATE VIEW public.aliased_orders(order_id) AS SELECT id FROM public.orders;",
						Columns: []*storepb.ColumnMetadata{
							{Name: "order_id", Type: "int"},
						},
					},
				},
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{
						Name:       "order_totals_mv",
						Definition: "CREATE MATERIALIZED VIEW public.order_totals_mv AS SELECT customer_id, SUM(amount) AS total_amount FROM public.orders GROUP BY customer_id;",
					},
					{
						Name:       "aliased_orders_mv",
						Definition: "CREATE MATERIALIZED VIEW public.aliased_orders_mv(order_id) AS SELECT id FROM public.orders;",
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name:       "reporting_fn",
						Definition: "CREATE FUNCTION public.reporting_fn() RETURNS int STABLE AS $$ SELECT id FROM public.orders $$ LANGUAGE SQL;",
						Signature:  "reporting_fn()",
					},
					{
						Name:       "stale_reporting_fn",
						Definition: "CREATE FUNCTION public.stale_reporting_fn() RETURNS int STABLE AS $$ SELECT id FROM public.missing_orders $$ LANGUAGE SQL;",
						Signature:  "stale_reporting_fn()",
					},
					{
						Name:       "unsupported_reporting_fn",
						Definition: "CREATE FUNCTION public.unsupported_reporting_fn() RETURNS int STABLE AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql;",
						Signature:  "unsupported_reporting_fn()",
					},
				},
				Sequences: []*storepb.SequenceMetadata{
					{
						Name: "order_id_seq",
					},
				},
			},
			{
				Name: "analytics",
				Tables: []*storepb.TableMetadata{
					{
						Name: "events",
						Columns: []*storepb.ColumnMetadata{
							{Name: "event_id", Type: "int"},
							{Name: "order_id", Type: "int"},
						},
					},
				},
				Functions: []*storepb.FunctionMetadata{
					{
						Name:       "reporting_fn",
						Definition: "CREATE FUNCTION analytics.reporting_fn() RETURNS int STABLE AS $$ SELECT event_id FROM analytics.events $$ LANGUAGE SQL;",
						Signature:  "reporting_fn()",
					},
				},
			},
			{
				Name: "spectrum",
				ExternalTables: []*storepb.ExternalTableMetadata{
					{
						Name: "spectrum_orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "external_id", Type: "varchar"},
						},
					},
				},
			},
		},
	}
}

func redshiftMockDatabaseMetadataGetter(databaseMetadata []*storepb.DatabaseSchemaMetadata) (base.GetDatabaseMetadataFunc, base.ListDatabaseNamesFunc) {
	return func(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
			for _, metadata := range databaseMetadata {
				if metadata.GetName() == databaseName {
					return "", model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_REDSHIFT, true /* isObjectCaseSensitive */), nil
				}
			}
			return "", nil, errors.Errorf("database %q not found", databaseName)
		}, func(_ context.Context, _ string) ([]string, error) {
			names := make([]string, 0, len(databaseMetadata))
			for _, metadata := range databaseMetadata {
				names = append(names, metadata.GetName())
			}
			return names, nil
		}
}

func redshiftSourceColumnSetFromList(columns []base.ColumnResource) base.SourceColumnSet {
	result := make(base.SourceColumnSet)
	for _, column := range columns {
		result[column] = true
	}
	return result
}
