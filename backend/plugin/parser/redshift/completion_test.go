package redshift

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type completionWant struct {
	text string
	typ  base.CandidateType
}

type completionCase struct {
	name       string
	input      string
	want       []completionWant
	absentText []string
}

func TestCompletionDoesNotDependOnANTLR(t *testing.T) {
	content, err := os.ReadFile("completion.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/redshift")
	require.NotContains(t, source, "CodeCompletionCore")
}

func TestCompletionRedshiftSlots(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       []completionWant
		absentText []string
	}{
		{
			name:  "statement start",
			input: "|",
			want: []completionWant{
				{"SELECT", base.CandidateTypeKeyword},
				{"COPY", base.CandidateTypeKeyword},
				{"UNLOAD", base.CandidateTypeKeyword},
				{"SHOW", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "prefix filtering",
			input: "SH|",
			want: []completionWant{
				{"SHOW", base.CandidateTypeKeyword},
			},
			absentText: []string{"SELECT", "COPY"},
		},
		{
			name:  "create dispatch",
			input: "CREATE |",
			want: []completionWant{
				{"TABLE", base.CandidateTypeKeyword},
				{"VIEW", base.CandidateTypeKeyword},
				{"SCHEMA", base.CandidateTypeKeyword},
				{"DATABASE", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "alter dispatch",
			input: "ALTER |",
			want: []completionWant{
				{"TABLE", base.CandidateTypeKeyword},
				{"DATABASE", base.CandidateTypeKeyword},
				{"ROLE", base.CandidateTypeKeyword},
				{"SCHEMA", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "drop dispatch",
			input: "DROP |",
			want: []completionWant{
				{"TABLE", base.CandidateTypeKeyword},
				{"VIEW", base.CandidateTypeKeyword},
				{"INDEX", base.CandidateTypeKeyword},
				{"FUNCTION", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "create table options",
			input: "CREATE TABLE t (id INT) |",
			want: []completionWant{
				{"DISTSTYLE", base.CandidateTypeKeyword},
				{"DISTKEY", base.CandidateTypeKeyword},
				{"SORTKEY", base.CandidateTypeKeyword},
				{"ENCODE", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "create table option prefix",
			input: "CREATE TABLE t (id INT) DIS|",
			want: []completionWant{
				{"DISTSTYLE", base.CandidateTypeKeyword},
				{"DISTKEY", base.CandidateTypeKeyword},
			},
			absentText: []string{"SORTKEY", "ENCODE"},
		},
		{
			name:  "copy options",
			input: "COPY t FROM 's3://bucket/file' |",
			want: []completionWant{
				{"IAM_ROLE", base.CandidateTypeKeyword},
				{"CREDENTIALS", base.CandidateTypeKeyword},
				{"FORMAT", base.CandidateTypeKeyword},
				{"MANIFEST", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "copy option prefix",
			input: "COPY t FROM 's3://bucket/file' G|",
			want: []completionWant{
				{"GZIP", base.CandidateTypeKeyword},
			},
			absentText: []string{"BZIP2", "MANIFEST"},
		},
		{
			name:  "unload options",
			input: "UNLOAD ('SELECT * FROM t') TO 's3://bucket/out' |",
			want: []completionWant{
				{"IAM_ROLE", base.CandidateTypeKeyword},
				{"FORMAT", base.CandidateTypeKeyword},
				{"MANIFEST", base.CandidateTypeKeyword},
				{"HEADER", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "unload option prefix",
			input: "UNLOAD ('SELECT * FROM t') TO 's3://bucket/out' H|",
			want: []completionWant{
				{"HEADER", base.CandidateTypeKeyword},
			},
			absentText: []string{"MANIFEST", "FORMAT"},
		},
		{
			name:  "show subcommands",
			input: "SHOW |",
			want: []completionWant{
				{"DATABASES", base.CandidateTypeKeyword},
				{"SCHEMAS", base.CandidateTypeKeyword},
				{"TABLES", base.CandidateTypeKeyword},
				{"DATASHARES", base.CandidateTypeKeyword},
			},
		},
		{
			name:  "show subcommand prefix",
			input: "SHOW DATA|",
			want: []completionWant{
				{"DATABASES", base.CandidateTypeKeyword},
				{"DATASHARES", base.CandidateTypeKeyword},
			},
			absentText: []string{"TABLES", "COLUMNS"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statement, caretLine, caretOffset := catchCompletionCaret(t, tc.input)
			candidates, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, base.CompletionContext{
				Scene:           base.SceneTypeAll,
				DefaultDatabase: "db",
				DefaultSchema:   "public",
			}, statement, caretLine, caretOffset)
			require.NoError(t, err)
			for _, want := range tc.want {
				require.Truef(t, hasCompletionCandidate(candidates, want.text, want.typ), "missing candidate {%s, %s} in %v", want.text, want.typ, candidateSummary(candidates))
			}
			for _, text := range tc.absentText {
				require.Falsef(t, hasCompletionCandidateText(candidates, text), "unexpected candidate %q in %v", text, candidateSummary(candidates))
			}
		})
	}
}

func TestCompletionMetadataCoverageMatrix(t *testing.T) {
	completionContext := redshiftCompletionContextForTest()

	tests := []completionCase{
		{
			name:  "FROM offers schemas and current-schema relations",
			input: "SELECT 1 FROM |",
			want: []completionWant{
				{"public", base.CandidateTypeSchema},
				{"analytics", base.CandidateTypeSchema},
				{"orders", base.CandidateTypeTable},
				{"spectrum_orders", base.CandidateTypeTable},
				{"active_orders", base.CandidateTypeView},
				{"orders_summary", base.CandidateTypeMaterializedView},
				{"order_seq", base.CandidateTypeSequence},
			},
		},
		{
			name:  "schema-qualified FROM offers that schema's tables",
			input: "SELECT 1 FROM analytics.|",
			want: []completionWant{
				{"events", base.CandidateTypeTable},
			},
			absentText: []string{"orders", "active_orders"},
		},
		{
			name:  "projection offers table columns",
			input: "SELECT | FROM orders",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
		},
		{
			name:  "WHERE offers table columns",
			input: "SELECT * FROM orders WHERE |",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
		},
		{
			name:  "JOIN ON offers both sides' columns",
			input: "SELECT * FROM orders o JOIN analytics.events e ON |",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"event_id", base.CandidateTypeColumn},
			},
		},
		{
			name:  "alias-qualified columns",
			input: "SELECT o.| FROM orders AS o",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
			absentText: []string{"event_id"},
		},
		{
			name:  "schema table-qualified columns",
			input: "SELECT events.| FROM analytics.events",
			want: []completionWant{
				{"event_id", base.CandidateTypeColumn},
			},
			absentText: []string{"amount"},
		},
		{
			name:       "unknown qualifier does not broaden column scope",
			input:      "SELECT missing.| FROM orders",
			absentText: []string{"id", "amount", "event_id"},
		},
		{
			name:  "view columns are available",
			input: "SELECT | FROM active_orders",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
		},
		{
			name:  "materialized view columns are available",
			input: "SELECT | FROM orders_summary",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
		},
		{
			name:  "external table columns are available",
			input: "SELECT | FROM spectrum_orders",
			want: []completionWant{
				{"external_id", base.CandidateTypeColumn},
			},
		},
		{
			name:  "CTE relation is offered after FROM",
			input: "WITH recent_orders AS (SELECT * FROM orders) SELECT * FROM |",
			want: []completionWant{
				{"recent_orders", base.CandidateTypeTable},
			},
		},
		{
			name:  "CTE columns are offered in projection",
			input: "WITH recent_orders(id_alias, amount_alias) AS (SELECT id, amount FROM orders) SELECT | FROM recent_orders",
			want: []completionWant{
				{"id_alias", base.CandidateTypeColumn},
				{"amount_alias", base.CandidateTypeColumn},
			},
		},
		{
			name:  "INSERT INTO offers tables",
			input: "INSERT INTO |",
			want: []completionWant{
				{"orders", base.CandidateTypeTable},
				{"events", base.CandidateTypeTable},
			},
		},
		{
			name:  "UPDATE table context offers tables",
			input: "UPDATE |",
			want: []completionWant{
				{"orders", base.CandidateTypeTable},
			},
		},
		{
			name:  "DELETE FROM offers tables",
			input: "DELETE FROM |",
			want: []completionWant{
				{"orders", base.CandidateTypeTable},
			},
		},
		{
			name:  "quoted relation names are surfaced quoted",
			input: "SELECT 1 FROM |",
			want: []completionWant{
				{`"Order Items"`, base.CandidateTypeTable},
			},
		},
		{
			name:  "quoted column names are surfaced quoted",
			input: `SELECT | FROM "Order Items"`,
			want: []completionWant{
				{`"Item ID"`, base.CandidateTypeColumn},
				{`"Order ID"`, base.CandidateTypeColumn},
			},
		},
		{
			name:  "typed prefix filters object candidates",
			input: "SELECT 1 FROM ord|",
			want: []completionWant{
				{"orders", base.CandidateTypeTable},
				{"orders_summary", base.CandidateTypeMaterializedView},
			},
			absentText: []string{"events", "active_orders"},
		},
		{
			name:  "UTF-16 caret before projection resolves columns",
			input: "SELECT '😀', | FROM orders",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
		},
		{
			name:  "statement at caret scopes columns to current statement",
			input: "SELECT * FROM orders; SELECT | FROM analytics.events",
			want: []completionWant{
				{"event_id", base.CandidateTypeColumn},
			},
			absentText: []string{"amount"},
		},
	}

	runCompletionCases(t, completionContext, tests)
}

func TestCompletionMySQLPGScaleCoverageMatrix(t *testing.T) {
	completionContext := redshiftCompletionContextForTest()
	tests := make([]completionCase, 0, 220)

	relationCases := []completionCase{
		{name: "select from", input: "SELECT * FROM |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "select join", input: "SELECT * FROM orders JOIN |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "select comma join", input: "SELECT * FROM orders, |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "left join", input: "SELECT * FROM orders LEFT JOIN |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "right join", input: "SELECT * FROM orders RIGHT JOIN |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "full join", input: "SELECT * FROM orders FULL JOIN |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "cross join", input: "SELECT * FROM orders CROSS JOIN |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "insert into", input: "INSERT INTO |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "copy relation", input: "COPY | FROM 's3://bucket/file'", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "update relation", input: "UPDATE |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "delete relation", input: "DELETE FROM |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "truncate relation", input: "TRUNCATE TABLE |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "create table as from", input: "CREATE TABLE new_orders AS SELECT * FROM |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "create view as from", input: "CREATE VIEW v AS SELECT * FROM |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "create materialized view as from", input: "CREATE MATERIALIZED VIEW mv AS SELECT * FROM |", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "subquery from", input: "SELECT * FROM (SELECT * FROM |) s", want: []completionWant{{"orders", base.CandidateTypeTable}}},
		{name: "exists subquery from", input: "SELECT * FROM orders WHERE EXISTS (SELECT 1 FROM |)", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "in subquery from", input: "SELECT * FROM orders WHERE id IN (SELECT event_id FROM |)", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "union right from", input: "SELECT * FROM orders UNION ALL SELECT * FROM |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "with body from", input: "WITH x AS (SELECT * FROM |) SELECT * FROM x", want: []completionWant{{"orders", base.CandidateTypeTable}}},
	}
	tests = append(tests, relationCases...)

	schemaQualified := []struct {
		schema string
		want   string
		absent []string
	}{
		{"public", "orders", []string{"events"}},
		{"analytics", "events", []string{"orders", "active_orders"}},
	}
	qualifiedTemplates := []string{
		"SELECT * FROM %s.|",
		"SELECT * FROM orders JOIN %s.|",
		"INSERT INTO %s.|",
		"UPDATE %s.|",
		"DELETE FROM %s.|",
		"COPY %s.| FROM 's3://bucket/file'",
		"CREATE TABLE copy AS SELECT * FROM %s.|",
		"CREATE VIEW copy_v AS SELECT * FROM %s.|",
		"SELECT * FROM (SELECT * FROM %s.|) s",
		"WITH x AS (SELECT * FROM %s.|) SELECT * FROM x",
	}
	for _, schema := range schemaQualified {
		for i, tmpl := range qualifiedTemplates {
			tests = append(tests, completionCase{
				name:       fmt.Sprintf("schema qualified %s %02d", schema.schema, i),
				input:      fmt.Sprintf(tmpl, schema.schema),
				want:       []completionWant{{schema.want, base.CandidateTypeTable}},
				absentText: schema.absent,
			})
		}
	}

	columnTemplates := []completionCase{
		{name: "projection", input: "SELECT | FROM orders", want: []completionWant{{"id", base.CandidateTypeColumn}, {"amount", base.CandidateTypeColumn}}},
		{name: "projection after expression", input: "SELECT count(*), | FROM orders", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "where", input: "SELECT * FROM orders WHERE |", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "where after predicate", input: "SELECT * FROM orders WHERE amount > 10 AND |", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "group by", input: "SELECT id, sum(amount) FROM orders GROUP BY |", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "order by", input: "SELECT id, amount FROM orders ORDER BY |", want: []completionWant{{"id", base.CandidateTypeColumn}, {"amount", base.CandidateTypeColumn}}},
		{name: "having", input: "SELECT id, sum(amount) FROM orders GROUP BY id HAVING |", want: []completionWant{{"id", base.CandidateTypeColumn}, {"amount", base.CandidateTypeColumn}}},
		{name: "join on left", input: "SELECT * FROM orders o JOIN analytics.events e ON |", want: []completionWant{{"id", base.CandidateTypeColumn}, {"event_id", base.CandidateTypeColumn}}},
		{name: "join on right predicate", input: "SELECT * FROM orders o JOIN analytics.events e ON o.id = e.event_id AND |", want: []completionWant{{"id", base.CandidateTypeColumn}, {"event_id", base.CandidateTypeColumn}}},
		{name: "subquery projection", input: "SELECT * FROM (SELECT | FROM orders) s", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "exists projection", input: "SELECT * FROM orders WHERE EXISTS (SELECT | FROM analytics.events)", want: []completionWant{{"event_id", base.CandidateTypeColumn}}},
		{name: "with body projection", input: "WITH x AS (SELECT | FROM orders) SELECT * FROM x", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "view projection", input: "SELECT | FROM active_orders", want: []completionWant{{"id", base.CandidateTypeColumn}, {"amount", base.CandidateTypeColumn}}},
		{name: "external table projection", input: "SELECT | FROM spectrum_orders", want: []completionWant{{"external_id", base.CandidateTypeColumn}}},
		{name: "quoted table projection", input: `SELECT | FROM "Order Items"`, want: []completionWant{{`"Item ID"`, base.CandidateTypeColumn}, {`"Order ID"`, base.CandidateTypeColumn}}},
	}
	tests = append(tests, columnTemplates...)

	qualifierCases := []completionCase{
		{name: "alias o", input: "SELECT o.| FROM orders AS o", want: []completionWant{{"id", base.CandidateTypeColumn}, {"amount", base.CandidateTypeColumn}}, absentText: []string{"event_id"}},
		{name: "alias e", input: "SELECT e.| FROM orders o JOIN analytics.events e ON o.id = e.event_id", want: []completionWant{{"event_id", base.CandidateTypeColumn}}},
		{name: "table alias named orders", input: "SELECT orders.| FROM orders AS orders", want: []completionWant{{"id", base.CandidateTypeColumn}}},
		{name: "table events", input: "SELECT events.| FROM analytics.events", want: []completionWant{{"event_id", base.CandidateTypeColumn}}, absentText: []string{"amount"}},
		{name: "cte alias", input: "WITH x(a, b) AS (SELECT id, amount FROM orders) SELECT x.| FROM x", want: []completionWant{{"a", base.CandidateTypeColumn}, {"b", base.CandidateTypeColumn}}},
		{name: "unknown qualifier", input: "SELECT nope.| FROM orders", absentText: []string{"id", "amount", "event_id"}},
	}
	tests = append(tests, qualifierCases...)

	prefixCases := []completionCase{
		{name: "relation prefix ord", input: "SELECT * FROM ord|", want: []completionWant{{"orders", base.CandidateTypeTable}, {"orders_summary", base.CandidateTypeMaterializedView}}, absentText: []string{"events"}},
		{name: "relation prefix act", input: "SELECT * FROM act|", want: []completionWant{{"active_orders", base.CandidateTypeView}}, absentText: []string{"orders"}},
		{name: "relation prefix spec", input: "SELECT * FROM spec|", want: []completionWant{{"spectrum_orders", base.CandidateTypeTable}}, absentText: []string{"active_orders"}},
		{name: "schema prefix ana", input: "SELECT * FROM ana|", want: []completionWant{{"analytics", base.CandidateTypeSchema}}, absentText: []string{"public"}},
		{name: "column prefix am", input: "SELECT am| FROM orders", want: []completionWant{{"amount", base.CandidateTypeColumn}}, absentText: []string{"id"}},
		{name: "column prefix event", input: "SELECT event| FROM analytics.events", want: []completionWant{{"event_id", base.CandidateTypeColumn}}, absentText: []string{"amount"}},
	}
	tests = append(tests, prefixCases...)

	cteCases := []completionCase{
		{name: "cte from", input: "WITH x AS (SELECT * FROM orders) SELECT * FROM |", want: []completionWant{{"x", base.CandidateTypeTable}}},
		{name: "cte projection alias cols", input: "WITH x(a, b) AS (SELECT id, amount FROM orders) SELECT | FROM x", want: []completionWant{{"a", base.CandidateTypeColumn}, {"b", base.CandidateTypeColumn}}},
		{name: "cte qualified alias cols", input: "WITH x(a, b) AS (SELECT id, amount FROM orders) SELECT x.| FROM x", want: []completionWant{{"a", base.CandidateTypeColumn}, {"b", base.CandidateTypeColumn}}},
		{name: "recursive-looking cte name", input: "WITH recent_orders AS (SELECT * FROM orders) SELECT * FROM recent_|", want: []completionWant{{"recent_orders", base.CandidateTypeTable}}},
		{name: "cte join relation", input: "WITH x AS (SELECT * FROM orders) SELECT * FROM x JOIN |", want: []completionWant{{"events", base.CandidateTypeTable}}},
		{name: "cte join columns", input: "WITH x(a, b) AS (SELECT id, amount FROM orders) SELECT | FROM x JOIN analytics.events e ON x.a = e.event_id", want: []completionWant{{"a", base.CandidateTypeColumn}, {"event_id", base.CandidateTypeColumn}}},
	}
	tests = append(tests, cteCases...)

	for i := 0; i < 25; i++ {
		tests = append(tests, completionCase{
			name:  fmt.Sprintf("multi statement current scope %02d", i),
			input: fmt.Sprintf("SELECT * FROM orders WHERE id = %d; SELECT | FROM analytics.events", i),
			want:  []completionWant{{"event_id", base.CandidateTypeColumn}},
			absentText: []string{
				"amount",
			},
		})
	}

	slotInputs := []struct {
		prefix string
		want   string
		absent []string
	}{
		{"", "DISTSTYLE", nil},
		{"D", "DISTSTYLE", []string{"ENCODE"}},
		{"DI", "DISTKEY", []string{"ENCODE"}},
		{"S", "SORTKEY", []string{"ENCODE"}},
		{"E", "ENCODE", []string{"SORTKEY"}},
	}
	for i, item := range slotInputs {
		tests = append(tests, completionCase{
			name:       fmt.Sprintf("create table slot prefix %02d", i),
			input:      fmt.Sprintf("CREATE TABLE t (id INT) %s|", item.prefix),
			want:       []completionWant{{item.want, base.CandidateTypeKeyword}},
			absentText: item.absent,
		})
	}
	for i, item := range []struct {
		prefix string
		want   string
		absent []string
	}{
		{"", "IAM_ROLE", nil},
		{"C", "CREDENTIALS", []string{"FORMAT"}},
		{"F", "FORMAT", []string{"MANIFEST"}},
		{"M", "MANIFEST", []string{"FORMAT"}},
		{"G", "GZIP", []string{"BZIP2"}},
		{"B", "BZIP2", []string{"GZIP"}},
	} {
		tests = append(tests, completionCase{
			name:       fmt.Sprintf("copy slot prefix %02d", i),
			input:      fmt.Sprintf("COPY orders FROM 's3://bucket/file' %s|", item.prefix),
			want:       []completionWant{{item.want, base.CandidateTypeKeyword}},
			absentText: item.absent,
		})
	}
	for i, item := range []struct {
		prefix string
		want   string
		absent []string
	}{
		{"", "IAM_ROLE", nil},
		{"F", "FORMAT", []string{"MANIFEST"}},
		{"M", "MANIFEST", []string{"FORMAT"}},
		{"G", "GZIP", []string{"BZIP2"}},
		{"B", "BZIP2", []string{"GZIP"}},
		{"H", "HEADER", []string{"FORMAT"}},
	} {
		tests = append(tests, completionCase{
			name:       fmt.Sprintf("unload slot prefix %02d", i),
			input:      fmt.Sprintf("UNLOAD ('SELECT * FROM orders') TO 's3://bucket/out' %s|", item.prefix),
			want:       []completionWant{{item.want, base.CandidateTypeKeyword}},
			absentText: item.absent,
		})
	}
	for i, item := range []struct {
		prefix string
		want   string
		absent []string
	}{
		{"", "DATABASES", nil},
		{"D", "DATASHARES", []string{"TABLES"}},
		{"S", "SCHEMAS", []string{"TABLES"}},
		{"T", "TABLES", []string{"SCHEMAS"}},
		{"C", "COLUMNS", []string{"DATASHARES"}},
		{"G", "GRANTS", []string{"TABLES"}},
	} {
		tests = append(tests, completionCase{
			name:       fmt.Sprintf("show slot prefix %02d", i),
			input:      fmt.Sprintf("SHOW %s|", item.prefix),
			want:       []completionWant{{item.want, base.CandidateTypeKeyword}},
			absentText: item.absent,
		})
	}

	for len(tests) < 205 {
		i := len(tests)
		tests = append(tests, completionCase{
			name:  fmt.Sprintf("prefix stress relation %03d", i),
			input: fmt.Sprintf("SELECT * FROM %s|", []string{"ord", "act", "spec", "ana"}[i%4]),
			want: []completionWant{
				[]completionWant{
					{"orders", base.CandidateTypeTable},
					{"active_orders", base.CandidateTypeView},
					{"spectrum_orders", base.CandidateTypeTable},
					{"analytics", base.CandidateTypeSchema},
				}[i%4],
			},
		})
	}

	require.GreaterOrEqual(t, len(tests), 200)
	runCompletionCases(t, completionContext, tests)
}

func redshiftCompletionContextForTest() base.CompletionContext {
	metadata := model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "amount", Type: "numeric"},
						},
					},
					{
						Name: "Order Items",
						Columns: []*storepb.ColumnMetadata{
							{Name: "Item ID", Type: "bigint"},
							{Name: "Order ID", Type: "integer"},
						},
					},
				},
				ExternalTables: []*storepb.ExternalTableMetadata{
					{
						Name: "spectrum_orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "external_id", Type: "varchar"},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name: "active_orders",
						Columns: []*storepb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "amount", Type: "numeric"},
						},
					},
				},
				MaterializedViews: []*storepb.MaterializedViewMetadata{
					{Name: "orders_summary", Definition: "SELECT id, amount FROM orders"},
				},
				Sequences: []*storepb.SequenceMetadata{
					{Name: "order_seq"},
				},
			},
			{
				Name: "analytics",
				Tables: []*storepb.TableMetadata{
					{
						Name: "events",
						Columns: []*storepb.ColumnMetadata{
							{Name: "event_id", Type: "bigint"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_REDSHIFT, false)
	return base.CompletionContext{
		Scene:           base.SceneTypeAll,
		InstanceID:      "instance",
		DefaultDatabase: "db",
		DefaultSchema:   "public",
		Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
			return "db", metadata, nil
		},
	}
}

func runCompletionCases(t *testing.T, completionContext base.CompletionContext, tests []completionCase) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statement, caretLine, caretOffset := catchCompletionCaret(t, tc.input)
			candidates, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, completionContext, statement, caretLine, caretOffset)
			require.NoError(t, err)
			for _, want := range tc.want {
				require.Truef(t, hasCompletionCandidate(candidates, want.text, want.typ), "missing candidate {%s, %s} in %v", want.text, want.typ, candidateSummary(candidates))
			}
			for _, text := range tc.absentText {
				require.Falsef(t, hasCompletionCandidateText(candidates, text), "unexpected candidate %q in %v", text, candidateSummary(candidates))
			}
		})
	}
}

func TestCompletionNilOrFailedMetadataDoesNotReturnObjects(t *testing.T) {
	tests := []struct {
		name string
		cCtx base.CompletionContext
	}{
		{
			name: "nil metadata",
			cCtx: base.CompletionContext{
				Scene:           base.SceneTypeAll,
				DefaultDatabase: "db",
				DefaultSchema:   "public",
			},
		},
		{
			name: "metadata error",
			cCtx: base.CompletionContext{
				Scene:           base.SceneTypeAll,
				DefaultDatabase: "db",
				DefaultSchema:   "public",
				Metadata: func(context.Context, string, string) (string, *model.DatabaseMetadata, error) {
					return "", nil, assertAnError{}
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statement, caretLine, caretOffset := catchCompletionCaret(t, "SELECT 1 FROM |")
			candidates, err := base.Completion(context.Background(), storepb.Engine_REDSHIFT, tc.cCtx, statement, caretLine, caretOffset)
			require.NoError(t, err)
			for _, candidate := range candidates {
				switch candidate.Type {
				case base.CandidateTypeSchema, base.CandidateTypeTable, base.CandidateTypeView, base.CandidateTypeMaterializedView, base.CandidateTypeColumn, base.CandidateTypeSequence:
					t.Fatalf("metadata-less completion returned object candidate {%s, %s}; all candidates: %v", candidate.Text, candidate.Type, candidateSummary(candidates))
				default:
				}
			}
		})
	}
}

type assertAnError struct{}

func (assertAnError) Error() string { return "assert an error" }

func catchCompletionCaret(t *testing.T, input string) (string, int, int) {
	t.Helper()
	line := 1
	offset := 0
	for i, r := range input {
		if r == '|' {
			return input[:i] + input[i+1:], line, offset
		}
		if r == '\n' {
			line++
			offset = 0
			continue
		}
		if r <= 0xFFFF {
			offset++
		} else {
			offset += 2
		}
	}
	t.Fatalf("missing caret marker in %q", input)
	return "", 0, 0
}

func hasCompletionCandidateText(candidates []base.Candidate, text string) bool {
	for _, candidate := range candidates {
		if candidate.Text == text {
			return true
		}
	}
	return false
}

func hasCompletionCandidate(candidates []base.Candidate, text string, typ base.CandidateType) bool {
	for _, candidate := range candidates {
		if candidate.Type == typ && candidate.Text == text {
			return true
		}
	}
	return false
}

func candidateSummary(candidates []base.Candidate) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate.String())
	}
	return result
}
