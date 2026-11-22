package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestLiveRulesIssueTheSameStatements proves each live MySQL rule asks the seam
// for the statements it issued through the *sql.DB handle.
func TestLiveRulesIssueTheSameStatements(t *testing.T) {
	const (
		update = "UPDATE tech_book SET name = 'xz' WHERE id = 1;"
		insert = "INSERT INTO tech_book SELECT * FROM tech_book"
	)
	for _, tc := range []struct {
		name          string
		ruleType      storepb.SQLReviewRule_Type
		number        int32
		statement     string
		prober        *advisor.ProbeRecorder
		wantProbes    []advisor.ProbeCall
		wantEstimates []string
	}{
		{
			name:          "affected row limit",
			ruleType:      storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			number:        5,
			statement:     update,
			prober:        &advisor.ProbeRecorder{},
			wantEstimates: []string{update},
		},
		{
			name:          "insert row limit",
			ruleType:      storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			number:        5,
			statement:     insert,
			prober:        &advisor.ProbeRecorder{},
			wantEstimates: []string{insert + ";"},
		},
		{
			name:       "dml dry run",
			ruleType:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
			statement:  update,
			prober:     &advisor.ProbeRecorder{},
			wantProbes: []advisor.ProbeCall{{Statement: "EXPLAIN " + update}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &storepb.SQLReviewRule{Type: tc.ruleType, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_MYSQL}
			if tc.number > 0 {
				rule.Payload = &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: tc.number}}
			}
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{rule}, advisor.Context{
				DBType:          storepb.Engine_MYSQL,
				Prober:          tc.prober,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Empty(t, adviceList)
			require.Equal(t, tc.wantProbes, tc.prober.Probes)
			require.Equal(t, tc.wantEstimates, tc.prober.Estimates)
		})
	}
}

// TestLiveRulesReportAnAbsentCapability makes sure a live rule that cannot answer
// says so instead of passing silently.
func TestLiveRulesReportAnAbsentCapability(t *testing.T) {
	const (
		update = "UPDATE tech_book SET name = 'xz' WHERE id = 1;"
		insert = "INSERT INTO tech_book SELECT * FROM tech_book;"
	)
	for _, tc := range []struct {
		name      string
		ruleType  storepb.SQLReviewRule_Type
		statement string
		prober    db.Prober
		want      string
	}{
		{
			name:      "affected row limit without a connection",
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			statement: update,
			want:      "MYSQL: the live connection cannot probe the database, so this rule did not run",
		},
		{
			name:      "affected row limit without the capability",
			ruleType:  storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			statement: update,
			prober:    advisor.ProbeOnly{Recorder: &advisor.ProbeRecorder{}},
			want:      "MYSQL: the live connection cannot estimate affected rows, so this rule did not run",
		},
		{
			name:      "insert row limit without a connection",
			ruleType:  storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			statement: insert,
			want:      "MYSQL: the live connection cannot probe the database, so this rule did not run",
		},
		{
			name:      "insert row limit without the capability",
			ruleType:  storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			statement: insert,
			prober:    advisor.ProbeOnly{Recorder: &advisor.ProbeRecorder{}},
			want:      "MYSQL: the live connection cannot estimate affected rows, so this rule did not run",
		},
		{
			name:      "dml dry run without a connection",
			ruleType:  storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
			statement: update,
			want:      "MYSQL: the live connection cannot probe the database, so this rule did not run",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:    tc.ruleType,
				Level:   storepb.SQLReviewRule_WARNING,
				Engine:  storepb.Engine_MYSQL,
				Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
			}}, advisor.Context{
				DBType:          storepb.Engine_MYSQL,
				Prober:          tc.prober,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Len(t, adviceList, 1)
			require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
			require.Equal(t, "Rule check failed: "+tc.want, adviceList[0].Content)
		})
	}
}
