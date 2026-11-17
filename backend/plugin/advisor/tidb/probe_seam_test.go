package tidb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

// TestLiveRulesIssueTheSameStatements proves the TiDB dry run asks the seam for
// the statements it issued through the *sql.DB handle, including the BATCH DRY
// RUN that must run outside a transaction.
func TestLiveRulesIssueTheSameStatements(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statement  string
		wantProbes []advisor.ProbeCall
	}{
		{
			name:      "plain dml is explained in a transaction",
			statement: "DELETE FROM t WHERE id = 1;",
			wantProbes: []advisor.ProbeCall{
				{Statement: "EXPLAIN DELETE FROM t WHERE id = 1"},
			},
		},
		{
			name:      "batch dry runs in autocommit then explains the inner dml",
			statement: "BATCH ON id LIMIT 2 DELETE FROM t WHERE id > 0",
			wantProbes: []advisor.ProbeCall{
				{Autocommit: true, Statement: "BATCH ON id LIMIT 2 DRY RUN DELETE FROM t WHERE id > 0"},
				{Statement: "EXPLAIN DELETE FROM t WHERE id > 0"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prober := &advisor.ProbeRecorder{}
			adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: storepb.Engine_TIDB,
			}}, advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Prober:          prober,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			require.Empty(t, adviceList)
			require.Equal(t, tc.wantProbes, prober.Probes)
		})
	}
}

// TestLiveRulesReportAnAbsentProbe makes sure the dry run says it did not run
// instead of passing silently.
func TestLiveRulesReportAnAbsentProbe(t *testing.T) {
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), "DELETE FROM t WHERE id = 1;", []*storepb.SQLReviewRule{{
		Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_TIDB,
	}}, advisor.Context{
		DBType:          storepb.Engine_TIDB,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
	require.Equal(t, "Rule check failed: TIDB: the live connection cannot probe the database, so this rule did not run", adviceList[0].Content)
}
