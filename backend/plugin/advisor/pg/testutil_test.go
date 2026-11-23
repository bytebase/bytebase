package pg

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// newExplainProber returns a ProbeRecorder that answers `EXPLAIN (FORMAT JSON)`
// or `EXPLAIN` with the plan registered for the statement and fails to explain
// any other statement.
func newExplainProber(plans map[string]string) *advisor.ProbeRecorder {
	return &advisor.ProbeRecorder{
		ProbeFunc: func(_ db.ProbeContext, query string) (db.ProbeResult, error) {
			statement, ok := stripExplainPrefix(query)
			if !ok {
				return db.ProbeResult{}, errors.Errorf("unexpected query %q", query)
			}
			plan, ok := plans[statement]
			if !ok {
				return db.ProbeResult{}, errors.Errorf("relation in %q does not exist", statement)
			}
			return db.ProbeResult{
				Columns: []string{"QUERY PLAN"},
				Rows:    [][]any{{plan}},
			}, nil
		},
	}
}

func stripExplainPrefix(query string) (string, bool) {
	if statement, ok := strings.CutPrefix(query, "EXPLAIN (FORMAT JSON) "); ok {
		return statement, true
	}
	return strings.CutPrefix(query, "EXPLAIN ")
}

func explainedStatements(prober *advisor.ProbeRecorder) []string {
	if len(prober.Probes) == 0 {
		return nil
	}
	explained := make([]string, 0, len(prober.Probes))
	for _, call := range prober.Probes {
		statement, _ := stripExplainPrefix(call.Statement)
		explained = append(explained, statement)
	}
	return explained
}

// modifyTablePlan returns a minimal `EXPLAIN (FORMAT JSON)` plan whose ModifyTable node is fed rows.
func modifyTablePlan(rows int) string {
	return fmt.Sprintf(`[{"Plan": {"Node Type": "ModifyTable", "Plan Rows": 0, "Plans": [{"Node Type": "Seq Scan", "Parent Relationship": "Outer", "Plan Rows": %d}]}}]`, rows)
}

func runRowLimitRule(t *testing.T, prober db.Prober, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, limit int32, statement string) []*storepb.Advice {
	t.Helper()
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{{
		Type:    ruleType,
		Level:   level,
		Engine:  storepb.Engine_POSTGRES,
		Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: limit}},
	}}, advisor.Context{
		DBType:          storepb.Engine_POSTGRES,
		Prober:          prober,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList
}
