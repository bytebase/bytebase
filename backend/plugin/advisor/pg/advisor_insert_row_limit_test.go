package pg

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestInsertRowLimit(t *testing.T) {
	const title = "STATEMENT_INSERT_ROW_LIMIT"
	for _, tc := range []struct {
		name          string
		statement     string
		plans         map[string]string
		wantExplained []string
		want          []*storepb.Advice
	}{
		{
			name:          "exceeds limit",
			statement:     "INSERT INTO t2 SELECT * FROM t;",
			plans:         map[string]string{"INSERT INTO t2 SELECT * FROM t": modifyTablePlan(1000)},
			wantExplained: []string{"INSERT INTO t2 SELECT * FROM t"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.InsertTooManyRows.Int32(),
				Title:         title,
				Content:       `The statement "INSERT INTO t2 SELECT * FROM t" inserts 1000 rows. The count exceeds 100.`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:          "within limit",
			statement:     "INSERT INTO t2 SELECT * FROM t LIMIT 30;",
			plans:         map[string]string{"INSERT INTO t2 SELECT * FROM t LIMIT 30": modifyTablePlan(30)},
			wantExplained: []string{"INSERT INTO t2 SELECT * FROM t LIMIT 30"},
		},
		{
			name:          "dry run failure",
			statement:     "INSERT INTO missing SELECT * FROM t;",
			wantExplained: []string{"INSERT INTO missing SELECT * FROM t"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.InsertTooManyRows.Int32(),
				Title:         title,
				Content:       `"INSERT INTO missing SELECT * FROM t" dry runs failed: relation in "INSERT INTO missing SELECT * FROM t" does not exist`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:          "unreadable plan reports internal count error",
			statement:     "INSERT INTO t2 SELECT * FROM t;",
			plans:         map[string]string{"INSERT INTO t2 SELECT * FROM t": "Insert on t2  (cost=0.00..17.00 rows=0 width=0)"},
			wantExplained: []string{"INSERT INTO t2 SELECT * FROM t"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.Internal.Int32(),
				Title:         title,
				Content:       `failed to get row count for "INSERT INTO t2 SELECT * FROM t": failed to parse the EXPLAIN (FORMAT JSON) output: invalid character 'I' looking for beginning of value`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prober := newExplainProber(tc.plans)
			got := runRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_WARNING, 100, tc.statement)
			require.Equal(t, tc.wantExplained, explainedStatements(prober))
			require.Equal(t, tc.want, got)
		})
	}
}

func TestInsertRowLimitReportsStatementsBeyondExplainLimit(t *testing.T) {
	plans := map[string]string{}
	var statements []string
	for i := range 11 {
		statement := fmt.Sprintf("INSERT INTO t2 SELECT * FROM t WHERE id = %d", i)
		plans[statement] = modifyTablePlan(1)
		statements = append(statements, statement+";")
	}
	statements = append(statements, "INSERT INTO t2 VALUES (1), (2), (3);", "INSERT INTO t2 SELECT * FROM t;")
	prober := newExplainProber(plans)

	got := runRowLimitRule(t, prober, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_ERROR, 2, strings.Join(statements, "\n"))
	require.Len(t, explainedStatements(prober), 10)
	require.Equal(t, []*storepb.Advice{
		{
			Status:        storepb.Advice_ERROR,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         "STATEMENT_INSERT_ROW_LIMIT",
			Content:       `The statement "INSERT INTO t2 VALUES (1), (2), (3)" inserts 3 rows. The count exceeds 2.`,
			StartPosition: &storepb.Position{Line: 12},
		},
		{
			Status:        storepb.Advice_WARNING,
			Code:          code.InsertTooManyRows.Int32(),
			Title:         "STATEMENT_INSERT_ROW_LIMIT",
			Content:       "Only the first 10 statements were estimated; 2 more were not checked against the row limit.",
			StartPosition: &storepb.Position{Line: 11},
		},
	}, got)
}
