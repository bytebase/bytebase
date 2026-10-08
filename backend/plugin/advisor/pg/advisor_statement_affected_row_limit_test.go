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
			name:      "estimate from the subplan feeding the update, not its init plan",
			statement: "UPDATE big SET v = 1 WHERE s_id <= (SELECT count(*) FROM s WHERE flag);",
			plans: map[string]string{
				"UPDATE big SET v = 1 WHERE s_id <= (SELECT count(*) FROM s WHERE flag)": `[{"Plan": {"Node Type": "ModifyTable", "Plan Rows": 0, "Plans": [
					{"Node Type": "Aggregate", "Parent Relationship": "InitPlan", "Subplan Name": "InitPlan 1", "Plan Rows": 1},
					{"Node Type": "Seq Scan", "Parent Relationship": "Outer", "Plan Rows": 3333}
				]}}]`,
			},
			wantExplained: []string{"UPDATE big SET v = 1 WHERE s_id <= (SELECT count(*) FROM s WHERE flag)"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       `The statement "UPDATE big SET v = 1 WHERE s_id <= (SELECT count(*) FROM s WHERE flag)" affected 3333 rows (estimated). The count exceeds 1000.`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:      "merge",
			statement: "MERGE INTO big USING s ON big.s_id = s.id WHEN MATCHED THEN UPDATE SET v = 1;",
			plans: map[string]string{
				"MERGE INTO big USING s ON big.s_id = s.id WHEN MATCHED THEN UPDATE SET v = 1": modifyTablePlan(10000),
			},
			wantExplained: []string{"MERGE INTO big USING s ON big.s_id = s.id WHEN MATCHED THEN UPDATE SET v = 1"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementAffectedRowExceedsLimit.Int32(),
				Title:         title,
				Content:       `The statement "MERGE INTO big USING s ON big.s_id = s.id WHEN MATCHED THEN UPDATE SET v = 1" affected 10000 rows (estimated). The count exceeds 1000.`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
		{
			name:      "data-modifying CTEs",
			statement: "WITH d AS (DELETE FROM big WHERE s_id <= 50 RETURNING 1) SELECT count(*) FROM d;\nWITH d AS (DELETE FROM big RETURNING *) INSERT INTO big_archive SELECT * FROM d;\nCREATE TABLE big_moved AS WITH d AS (DELETE FROM big WHERE s_id = 1 RETURNING *) SELECT * FROM d;",
			plans: map[string]string{
				"WITH d AS (DELETE FROM big WHERE s_id <= 50 RETURNING 1) SELECT count(*) FROM d": `[{"Plan": {"Node Type": "Aggregate", "Plan Rows": 1, "Plans": [
					{"Node Type": "ModifyTable", "Parent Relationship": "InitPlan", "Subplan Name": "CTE d", "Plan Rows": 5000, "Plans": [
						{"Node Type": "Seq Scan", "Parent Relationship": "Outer", "Plan Rows": 5000}
					]},
					{"Node Type": "CTE Scan", "Parent Relationship": "Outer", "Plan Rows": 5000}
				]}}]`,
				"WITH d AS (DELETE FROM big RETURNING *) INSERT INTO big_archive SELECT * FROM d":                  modifyTablePlan(20000),
				"CREATE TABLE big_moved AS WITH d AS (DELETE FROM big WHERE s_id = 1 RETURNING *) SELECT * FROM d": modifyTablePlan(100),
			},
			wantExplained: []string{
				"WITH d AS (DELETE FROM big WHERE s_id <= 50 RETURNING 1) SELECT count(*) FROM d",
				"WITH d AS (DELETE FROM big RETURNING *) INSERT INTO big_archive SELECT * FROM d",
				"CREATE TABLE big_moved AS WITH d AS (DELETE FROM big WHERE s_id = 1 RETURNING *) SELECT * FROM d",
			},
			want: []*storepb.Advice{
				{
					Status:        storepb.Advice_WARNING,
					Code:          code.StatementAffectedRowExceedsLimit.Int32(),
					Title:         title,
					Content:       `The statement "WITH d AS (DELETE FROM big WHERE s_id <= 50 RETURNING 1) SELECT count(*) FROM d" affected 5000 rows (estimated). The count exceeds 1000.`,
					StartPosition: &storepb.Position{Line: 1},
				},
				{
					Status:        storepb.Advice_WARNING,
					Code:          code.StatementAffectedRowExceedsLimit.Int32(),
					Title:         title,
					Content:       `The statement "WITH d AS (DELETE FROM big RETURNING *) INSERT INTO big_archive SELECT * FROM d" affected 20000 rows (estimated). The count exceeds 1000.`,
					StartPosition: &storepb.Position{Line: 2},
				},
			},
		},
		{
			name:      "select, read-only CTE, and insert are not explained",
			statement: "SELECT * FROM big;\nWITH x AS (SELECT id FROM s) SELECT * FROM x;\nINSERT INTO big_archive SELECT * FROM big;",
		},
		{
			name:      "explain analyze runs the statement it explains",
			statement: "EXPLAIN (ANALYZE, BUFFERS) DELETE FROM big WHERE s_id <= 50;",
			plans: map[string]string{
				"DELETE FROM big WHERE s_id <= 50": modifyTablePlan(5000),
			},
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
			name:      "explain without analyze is not explained",
			statement: "EXPLAIN (ANALYZE off) DELETE FROM big;",
		},
		{
			name:          "estimate within the limit",
			statement:     "DELETE FROM big WHERE id = 1;",
			plans:         map[string]string{"DELETE FROM big WHERE id = 1": modifyTablePlan(1)},
			wantExplained: []string{"DELETE FROM big WHERE id = 1"},
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
			name:          "plan without a ModifyTable node",
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
			db, fake := newExplainFakeDB(t, tc.plans)
			got := runRowLimitRule(t, db, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_WARNING, 1000, tc.statement)
			require.Equal(t, tc.wantExplained, fake.explainedStatements())
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
	db, fake := newExplainFakeDB(t, plans)

	got := runRowLimitRule(t, db, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_ERROR, 1000, strings.Join(statements, "\n"))
	require.Len(t, fake.explainedStatements(), 10)
	require.Equal(t, []*storepb.Advice{{
		Status:        storepb.Advice_WARNING,
		Code:          code.StatementAffectedRowExceedsLimit.Int32(),
		Title:         "STATEMENT_AFFECTED_ROW_LIMIT",
		Content:       "Only the first 10 statements were estimated; 2 more were not checked against the row limit.",
		StartPosition: &storepb.Position{Line: 11},
	}}, got)
}
