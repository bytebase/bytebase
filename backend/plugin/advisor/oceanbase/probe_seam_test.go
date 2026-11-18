package oceanbase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestLiveRulesIssueTheSameStatements proves each live OceanBase rule asks the
// seam for the statements it issued through the *sql.DB handle.
func TestLiveRulesIssueTheSameStatements(t *testing.T) {
	const (
		update = "UPDATE tech_book SET name = 'xz' WHERE id = 1;"
		insert = "INSERT INTO tech_book SELECT * FROM tech_book;"
		plan   = `{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE","NAME":"","EST.ROWS":1000,"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t","EST.ROWS":1000}}`
	)
	for _, tc := range []struct {
		name           string
		ruleType       storepb.SQLReviewRule_Type
		number         int32
		statement      string
		currentDB      string
		prober         *advisor.ProbeRecorder
		wantProbes     []advisor.ProbeCall
		wantLookups    []advisor.ColumnLookupCall
		wantAdvice     []string
		wantAdviceCode int32
	}{
		{
			name:       "affected row limit",
			ruleType:   storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
			number:     5,
			statement:  update,
			prober:     &advisor.ProbeRecorder{Result: db.ProbeResult{Rows: [][]any{{plan}}}},
			wantProbes: []advisor.ProbeCall{{Statement: "EXPLAIN format=json " + update}},
			wantAdvice: []string{`"` + update + `" affected 1000 rows (estimated). The count exceeds 5.`},
		},
		{
			name:       "insert row limit",
			ruleType:   storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
			number:     5,
			statement:  insert,
			prober:     &advisor.ProbeRecorder{Result: db.ProbeResult{Rows: [][]any{{plan}}}},
			wantProbes: []advisor.ProbeCall{{Statement: "EXPLAIN format=json " + insert}},
			wantAdvice: []string{`"` + insert + `" inserts 1000 rows. The count exceeds 5.`},
		},
		{
			name:       "dml dry run",
			ruleType:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
			statement:  update,
			prober:     &advisor.ProbeRecorder{},
			wantProbes: []advisor.ProbeCall{{Statement: "EXPLAIN " + update}},
		},
		{
			name:        "offline ddl reads the dropped column's extra",
			ruleType:    storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL,
			statement:   "ALTER TABLE tech_book DROP COLUMN name;",
			currentDB:   "test",
			prober:      &advisor.ProbeRecorder{Extras: map[string]string{"test.tech_book.name": "STORED GENERATED"}},
			wantLookups: []advisor.ColumnLookupCall{{Database: "test", Table: "tech_book", Column: "name"}},
			wantAdvice:  []string{"Dropping stored generated columns is an offline DDL operation."},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &storepb.SQLReviewRule{Type: tc.ruleType, Level: storepb.SQLReviewRule_WARNING, Engine: storepb.Engine_OCEANBASE}
			if tc.number > 0 {
				rule.Payload = &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: tc.number}}
			}
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{rule}, advisor.Context{
				DBType:          storepb.Engine_OCEANBASE,
				CurrentDatabase: tc.currentDB,
				Prober:          tc.prober,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantProbes, tc.prober.Probes)
			require.Equal(t, tc.wantLookups, tc.prober.Lookups)
			require.Len(t, adviceList, len(tc.wantAdvice))
			for i := range tc.wantAdvice {
				require.Equal(t, tc.wantAdvice[i], adviceList[i].Content)
			}
		})
	}
}

// TestOfflineDDLWithoutTheLookupDoesNotPass covers the one rule that needs the
// catalogue lookup: a connection that cannot bind the three values must say so.
func TestOfflineDDLWithoutTheLookupDoesNotPass(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prober db.Prober
		want   string
	}{
		{
			name: "no connection",
			want: "OCEANBASE: the live connection cannot read a column's catalogue extra, so this rule did not run",
		},
		{
			name:   "no capability",
			prober: advisor.ProbeOnly{Recorder: &advisor.ProbeRecorder{}},
			want:   "OCEANBASE: the live connection cannot read a column's catalogue extra, so this rule did not run",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), "ALTER TABLE tech_book DROP COLUMN name;", []*storepb.SQLReviewRule{{
				Type:   storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: storepb.Engine_OCEANBASE,
			}}, advisor.Context{
				DBType:          storepb.Engine_OCEANBASE,
				CurrentDatabase: "test",
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

// TestLiveRulesReportAnAbsentProbe makes sure a row limit rule that cannot
// estimate says so instead of passing silently.
func TestLiveRulesReportAnAbsentProbe(t *testing.T) {
	for _, tc := range []struct {
		ruleType  storepb.SQLReviewRule_Type
		statement string
	}{
		{ruleType: storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, statement: "UPDATE tech_book SET name = 'xz' WHERE id = 1;"},
		{ruleType: storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, statement: "INSERT INTO tech_book SELECT * FROM tech_book;"},
		{ruleType: storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, statement: "UPDATE tech_book SET name = 'xz' WHERE id = 1;"},
	} {
		t.Run(tc.ruleType.String(), func(t *testing.T) {
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:    tc.ruleType,
				Level:   storepb.SQLReviewRule_WARNING,
				Engine:  storepb.Engine_OCEANBASE,
				Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
			}}, advisor.Context{
				DBType:          storepb.Engine_OCEANBASE,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Len(t, adviceList, 1)
			require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
			require.Equal(t, "Rule check failed: OCEANBASE: the live connection cannot probe the database, so this rule did not run", adviceList[0].Content)
		})
	}
}
