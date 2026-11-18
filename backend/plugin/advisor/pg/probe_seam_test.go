package pg

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestLiveRulesIssueTheSameStatements proves each live PostgreSQL rule asks the
// seam for the statements it issued through the *sql.DB handle, with the same
// session pre-executions and the same plan text.
func TestLiveRulesIssueTheSameStatements(t *testing.T) {
	const (
		update = "UPDATE tech_book SET name = 'xz' WHERE id = 1;"
		insert = "INSERT INTO tech_book SELECT * FROM tech_book;"
	)
	for _, tc := range []struct {
		name       string
		ruleType   storepb.SQLReviewRule_Type
		number     int32
		statement  string
		prober     *advisor.ProbeRecorder
		wantProbes []advisor.ProbeCall
	}{
		{
			name:      "affected row limit",
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			number:    5,
			statement: "SET search_path = app;\n" + update,
			prober:    &advisor.ProbeRecorder{Result: db.ProbeResult{Rows: [][]any{{modifyTablePlan(1)}}}},
			wantProbes: []advisor.ProbeCall{{
				PreExecutions: []string{"SET search_path = app"},
				Statement:     "EXPLAIN (FORMAT JSON) UPDATE tech_book SET name = 'xz' WHERE id = 1",
			}},
		},
		{
			name:      "insert row limit",
			ruleType:  storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			number:    5,
			statement: insert,
			prober:    &advisor.ProbeRecorder{Result: db.ProbeResult{Rows: [][]any{{modifyTablePlan(1)}}}},
			wantProbes: []advisor.ProbeCall{{
				Statement: "EXPLAIN (FORMAT JSON) INSERT INTO tech_book SELECT * FROM tech_book",
			}},
		},
		{
			name:      "dml dry run",
			ruleType:  storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
			statement: update,
			prober:    &advisor.ProbeRecorder{},
			wantProbes: []advisor.ProbeCall{{
				Statement: "EXPLAIN UPDATE tech_book SET name = 'xz' WHERE id = 1",
			}},
		},
		{
			name:      "object owner check",
			ruleType:  storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK,
			statement: update,
			prober:    &advisor.ProbeRecorder{Result: db.ProbeResult{Rows: [][]any{{"alice"}}}},
			wantProbes: []advisor.ProbeCall{{
				Statement: "SELECT CURRENT_USER",
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &storepb.SQLReviewRule{Type: tc.ruleType, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_POSTGRES}
			if tc.number > 0 {
				rule.Payload = &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: tc.number}}
			}
			_, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{rule}, advisor.Context{
				DBType:          storepb.Engine_POSTGRES,
				DBSchema:        &metadatapb.DatabaseSchemaMetadata{},
				Prober:          tc.prober,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantProbes, tc.prober.Probes)
		})
	}
}

// TestLiveRulesReportAnAbsentProbe makes sure a live rule that cannot probe says
// so instead of passing silently.
func TestLiveRulesReportAnAbsentProbe(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ruleType  storepb.SQLReviewRule_Type
		number    int32
		statement string
		want      string
	}{
		{
			name:      "affected row limit",
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			number:    5,
			statement: "UPDATE tech_book SET name = 'xz' WHERE id = 1;",
			want:      "POSTGRES: the live connection cannot probe the database, so this rule did not run",
		},
		{
			name:      "insert row limit",
			ruleType:  storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			number:    5,
			statement: "INSERT INTO tech_book SELECT * FROM tech_book;",
			want:      "POSTGRES: the live connection cannot probe the database, so this rule did not run",
		},
		{
			name:      "dml dry run",
			ruleType:  storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
			statement: "UPDATE tech_book SET name = 'xz' WHERE id = 1;",
			want:      "POSTGRES: the live connection cannot probe the database, so this rule did not run",
		},
		{
			name:      "object owner check",
			ruleType:  storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK,
			statement: "UPDATE tech_book SET name = 'xz' WHERE id = 1;",
			want:      "POSTGRES: the live connection cannot probe the database, so this rule did not run",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &storepb.SQLReviewRule{Type: tc.ruleType, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_POSTGRES}
			if tc.number > 0 {
				rule.Payload = &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: tc.number}}
			}
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{rule}, advisor.Context{
				DBType:          storepb.Engine_POSTGRES,
				DBSchema:        &metadatapb.DatabaseSchemaMetadata{},
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Len(t, adviceList, 1)
			require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
			require.Equal(t, "Rule check failed: "+tc.want, adviceList[0].Content)
		})
	}
}
