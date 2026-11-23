package oceanbase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

// checkTestRowLimitRule runs one OceanBase row limit rule with a limit of 5
// against a ProbeRecorder that answers every EXPLAIN with plan.
func checkTestRowLimitRule(t *testing.T, ruleType storepb.SQLReviewRule_Type, level storepb.SQLReviewRule_Level, plan, statement string) ([]*storepb.Advice, *advisor.ProbeRecorder) {
	t.Helper()
	prober := &advisor.ProbeRecorder{
		Result: db.ProbeResult{
			Columns: []string{"Query Plan"},
			Rows:    [][]any{{plan}},
		},
	}

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), statement, []*storepb.SQLReviewRule{
		{
			Type:    ruleType,
			Level:   level,
			Engine:  storepb.Engine_OCEANBASE,
			Payload: &storepb.SQLReviewRule_NumberPayload{NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{Number: 5}},
		},
	}, advisor.Context{
		DBType:          storepb.Engine_OCEANBASE,
		Prober:          prober,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	return adviceList, prober
}

func TestEstimatedRowsFromPlan(t *testing.T) {
	// OceanBase splits the JSON plan across rows, and EST.ROWS may be fractional.
	explain := func(res db.ProbeResult) (int64, string) {
		probe, err := advisor.ExplainPlanRows(
			context.Background(),
			advisor.Context{DBType: storepb.Engine_OCEANBASE, Prober: &advisor.ProbeRecorder{Result: res}},
			nil,
			"EXPLAIN format=json UPDATE t SET a = 1",
			oceanbaseparser.EstimateAffectedRowsFromExplainJSON,
		)
		require.NoError(t, err)
		return probe.Rows, probe.CountError
	}

	rows, countErr := explain(db.ProbeResult{
		Columns:     []string{"Query Plan"},
		ColumnTypes: []string{"VARCHAR"},
		Rows: [][]any{
			{`{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE",`},
			{`"NAME":"","EST.ROWS":41.6,"output":""}`},
		},
	})
	require.Empty(t, countErr)
	require.Equal(t, int64(42), rows)

	_, countErr = explain(db.ProbeResult{
		Columns:     []string{"Query Plan"},
		ColumnTypes: []string{"VARCHAR"},
		Rows:        [][]any{{`{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE","NAME":""}`}},
	})
	require.Contains(t, countErr, `operator "DISTRIBUTED UPDATE" has no EST.ROWS`)

	_, countErr = explain(db.ProbeResult{})
	require.NotEmpty(t, countErr)
}
