package pg

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
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
	require.NotContains(t, source, "github.com/bytebase/parser/postgresql")
	require.NotContains(t, source, "CodeCompletionCore")
}

func TestCompletionRegisteredEngines(t *testing.T) {
	cCtx := pgCompletionContextForTest()
	for _, engine := range []storepb.Engine{
		storepb.Engine_POSTGRES,
		storepb.Engine_SNOWFLAKE,
		storepb.Engine_COCKROACHDB,
	} {
		stmt, line, col := catchCompletionCaret(t, "SELECT * FROM |")
		candidates, err := base.Completion(context.Background(), engine, cCtx, stmt, line, col)
		require.NoError(t, err, "engine %v", engine)
		require.True(t, hasCompletionCandidate(candidates, "orders", base.CandidateTypeTable),
			"engine %v should offer table 'orders'; got %v", engine, candidateSummary(candidates))
	}
}

func TestCompletionMetadataCoverageMatrix(t *testing.T) {
	completionContext := pgCompletionContextForTest()

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
				{`"order"`, base.CandidateTypeTable},
				{`"Order Items"`, base.CandidateTypeTable},
				{`"my-table"`, base.CandidateTypeTable},
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
			name:  "dollar-signed qualifier columns",
			input: "SELECT t$1.| FROM orders AS t$1",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
			},
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
			name:  "foreign table columns are available",
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
			name:  "set-operation CTE infers output columns from left branch",
			input: "WITH combined AS (SELECT id, amount FROM orders UNION SELECT event_id, event_id FROM analytics.events) SELECT combined.| FROM combined",
			want: []completionWant{
				{"id", base.CandidateTypeColumn},
				{"amount", base.CandidateTypeColumn},
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
			name:  "quoted column names are surfaced quoted",
			input: `SELECT | FROM "Order Items"`,
			want: []completionWant{
				{`"Item ID"`, base.CandidateTypeColumn},
				{`"Order ID"`, base.CandidateTypeColumn},
			},
		},
		{
			name:  "caret inside double quotes omits outer quotes",
			input: `SELECT * FROM "ord|"`,
			want: []completionWant{
				{"order", base.CandidateTypeTable},
				{"orders", base.CandidateTypeTable},
			},
		},
		{
			name:  "quote inside comment does not suppress outer quotes",
			input: `SELECT * FROM /* " */ |`,
			want: []completionWant{
				{`"Order Items"`, base.CandidateTypeTable},
			},
		},
		{
			name:  "quote inside dollar-quoted string does not suppress outer quotes",
			input: `SELECT $tag$"$tag$, | FROM "Order Items"`,
			want: []completionWant{
				{`"Item ID"`, base.CandidateTypeColumn},
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statement, caretLine, caretOffset := catchCompletionCaret(t, tc.input)
			candidates, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, completionContext, statement, caretLine, caretOffset)
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

// Completion cost must be bounded by the caret's statement, not the whole
// sheet: a broken statement anywhere after the caret must not make completion
// scale with the entire trailing document (BYT-9886).
func TestCompletionWithBrokenTrailingStatementScalesLinearly(t *testing.T) {
	a := require.New(t)

	var sheet strings.Builder
	sheet.WriteString("SELECT  FROM orders JOIN analytics.events a2 ON orders.id = a2.event_id;\n")
	sheet.WriteString("SELEC broken FROM oops;\n")
	for i := range 2000 {
		fmt.Fprintf(&sheet, "SELECT col_a, col_b, col_c FROM table_%04d WHERE col_a = %d AND col_b LIKE 'pattern%%' ORDER BY col_c LIMIT 100;\n", i, i)
	}

	started := time.Now()
	result, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, pgCompletionContextForTest(), sheet.String(), 1, 7)
	elapsed := time.Since(started)

	a.NoError(err)
	a.True(hasCompletionCandidate(result, "id", base.CandidateTypeColumn))
	a.True(hasCompletionCandidate(result, "event_id", base.CandidateTypeColumn))
	a.Less(elapsed, 2*time.Second)
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
			candidates, err := base.Completion(context.Background(), storepb.Engine_POSTGRES, tc.cCtx, statement, caretLine, caretOffset)
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

func pgCompletionContextForTest() base.CompletionContext {
	metadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name: "db",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "amount", Type: "numeric"},
						},
					},
					{
						Name: "order",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
						},
					},
					{
						Name: "Order Items",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "Item ID", Type: "bigint"},
							{Name: "Order ID", Type: "integer"},
						},
					},
					{
						Name: "my-table",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "col1", Type: "integer"},
						},
					},
				},
				ExternalTables: []*metadatapb.ExternalTableMetadata{
					{
						Name: "spectrum_orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "external_id", Type: "varchar"},
						},
					},
				},
				Views: []*metadatapb.ViewMetadata{
					{
						Name: "active_orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer"},
							{Name: "amount", Type: "numeric"},
						},
					},
				},
				MaterializedViews: []*metadatapb.MaterializedViewMetadata{
					{Name: "orders_summary", Definition: "SELECT id, amount FROM orders"},
				},
				Sequences: []*metadatapb.SequenceMetadata{
					{Name: "order_seq"},
				},
			},
			{
				Name: "analytics",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "events",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "event_id", Type: "bigint"},
						},
					},
				},
			},
		},
	}, nil, nil, storepb.Engine_POSTGRES, true)
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
