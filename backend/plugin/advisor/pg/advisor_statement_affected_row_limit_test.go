package pg

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestStatementAffectedRowLimit(t *testing.T) {
	const title = "STATEMENT_AFFECTED_ROW_LIMIT"
	for _, tc := range []struct {
		name          string
		statement     string
		plans         map[string]string
		wantExplained []string
		want          []*storepb.Advice
	}{
		{
			name:          "exceeds limit",
			statement:     "UPDATE big SET v = 1 WHERE s_id = 1;",
			plans:         map[string]string{"UPDATE big SET v = 1 WHERE s_id = 1": modifyTablePlan(3333)},
			wantExplained: []string{"UPDATE big SET v = 1 WHERE s_id = 1"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       `The statement "UPDATE big SET v = 1 WHERE s_id = 1" affected 3333 rows (estimated). The count exceeds 1000.`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:          "within limit",
			statement:     "DELETE FROM big WHERE id = 1;",
			plans:         map[string]string{"DELETE FROM big WHERE id = 1": modifyTablePlan(1)},
			wantExplained: []string{"DELETE FROM big WHERE id = 1"},
		},
		{
			name:          "explain analyze unwraps inner statement",
			statement:     "EXPLAIN (ANALYZE, BUFFERS) DELETE FROM big WHERE s_id <= 50;",
			plans:         map[string]string{"DELETE FROM big WHERE s_id <= 50": modifyTablePlan(5000)},
			wantExplained: []string{"DELETE FROM big WHERE s_id <= 50"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       `The statement "DELETE FROM big WHERE s_id <= 50" affected 5000 rows (estimated). The count exceeds 1000.`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:          "dry run failure",
			statement:     "DELETE FROM missing;",
			wantExplained: []string{"DELETE FROM missing"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       `"DELETE FROM missing" dry runs failed: relation in "DELETE FROM missing" does not exist`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:          "unreadable plan reports internal count error",
			statement:     "DELETE FROM big;",
			plans:         map[string]string{"DELETE FROM big": `[{"Plan": {"Node Type": "Seq Scan", "Plan Rows": 10000}}]`},
			wantExplained: []string{"DELETE FROM big"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.Internal.Int32(),
				Title:         title,
				Content:       `failed to get row count for "DELETE FROM big": the plan has no ModifyTable node`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prober := newExplainProber(tc.plans)
			got := runRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_WARNING, 1000, tc.statement)
			require.Equal(t, tc.wantExplained, explainedStatements(prober))
			require.Equal(t, tc.want, got)
		})
	}
}

func TestStatementAffectedRowLimitReportsStatementsBeyondExplainLimit(t *testing.T) {
	plans := map[string]string{}
	var statements []string
	for i := range 12 {
		statement := fmt.Sprintf("UPDATE t SET v = 'y' WHERE id = %d", i)
		plans[statement] = modifyTablePlan(1)
		statements = append(statements, statement+";")
	}
	prober := newExplainProber(plans)

	got := runRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_ERROR, 1000, strings.Join(statements, "\n"))
	require.Len(t, explainedStatements(prober), 10)
	require.Equal(t, []*storepb.Advice{{
		Status:        storepb.Advice_WARNING,
		Code:          code.StatementAffectedRowExceedsLimit.Int32(),
		Title:         "STATEMENT_AFFECTED_ROW_LIMIT",
		Content:       "Only the first 10 statements were estimated; 2 more were not checked against the row limit.",
		StartPosition: &storepb.Position{Line: 11},
	}}, got)
}
