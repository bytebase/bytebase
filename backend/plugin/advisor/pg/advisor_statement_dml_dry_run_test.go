package pg

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestStatementDMLDryRun(t *testing.T) {
	for _, tc := range []struct {
		name          string
		statement     string
		plans         map[string]string
		wantExplained []string
		want          []*storepb.Advice
	}{
		{
			name:      "insert, update, delete, and merge",
			statement: "INSERT INTO t VALUES (1);\nUPDATE t SET v = 1;\nDELETE FROM t;\nMERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE;",
			plans: map[string]string{
				"INSERT INTO t VALUES (1)": modifyTablePlan(1),
				"UPDATE t SET v = 1":       modifyTablePlan(1),
				"DELETE FROM t":            modifyTablePlan(1),
				"MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE": modifyTablePlan(1),
			},
			wantExplained: []string{
				"INSERT INTO t VALUES (1)",
				"UPDATE t SET v = 1",
				"DELETE FROM t",
				"MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
			},
		},
		{
			name:          "data-modifying CTE",
			statement:     "WITH d AS (DELETE FROM t RETURNING *) SELECT count(*) FROM d;",
			plans:         map[string]string{"WITH d AS (DELETE FROM t RETURNING *) SELECT count(*) FROM d": modifyTablePlan(1)},
			wantExplained: []string{"WITH d AS (DELETE FROM t RETURNING *) SELECT count(*) FROM d"},
		},
		{
			name:          "explain analyze runs the statement it explains",
			statement:     "EXPLAIN (ANALYZE, BUFFERS) INSERT INTO t VALUES (1);",
			plans:         map[string]string{"INSERT INTO t VALUES (1)": modifyTablePlan(1)},
			wantExplained: []string{"INSERT INTO t VALUES (1)"},
		},
		{
			name:      "select and explain without analyze are not dry run",
			statement: "SELECT * FROM t;\nEXPLAIN DELETE FROM t;",
		},
		{
			name:          "dry run failure",
			statement:     "MERGE INTO missing USING s ON missing.id = s.id WHEN MATCHED THEN DELETE;",
			wantExplained: []string{"MERGE INTO missing USING s ON missing.id = s.id WHEN MATCHED THEN DELETE"},
			want: []*storepb.Advice{{
				Status:        storepb.Advice_WARNING,
				Code:          code.StatementDMLDryRunFailed.Int32(),
				Title:         "STATEMENT_DML_DRY_RUN",
				Content:       `"MERGE INTO missing USING s ON missing.id = s.id WHEN MATCHED THEN DELETE" dry runs failed: relation in "MERGE INTO missing USING s ON missing.id = s.id WHEN MATCHED THEN DELETE" does not exist`,
				StartPosition: &storepb.Position{Line: 1},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, fake := newExplainFakeDB(t, tc.plans)
			got, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: storepb.Engine_POSTGRES,
			}}, advisor.Context{
				DBType:          storepb.Engine_POSTGRES,
				Driver:          db,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantExplained, fake.explainedStatements())
			require.Equal(t, tc.want, got)
		})
	}
}
