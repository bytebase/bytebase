package oracle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

// TestLiveRulesIssueTheSameStatements proves the Oracle dry run asks the seam
// for EXPLAIN PLAN FOR and runs it outside a transaction, where BEGIN would open
// a PL/SQL block instead.
func TestLiveRulesIssueTheSameStatements(t *testing.T) {
	const update = "UPDATE TECH_BOOK SET NAME = 'xz' WHERE ID = 1;"
	prober := &advisor.ProbeRecorder{}
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), update, []*storepb.SQLReviewRule{{
		Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_ORACLE,
	}}, advisor.Context{
		DBType:          storepb.Engine_ORACLE,
		Prober:          prober,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	require.Empty(t, adviceList)
	require.Equal(t, []advisor.ProbeCall{{
		Autocommit: true,
		Statement:  "EXPLAIN PLAN FOR UPDATE TECH_BOOK SET NAME = 'xz' WHERE ID = 1",
	}}, prober.Probes)
}

// TestLiveRulesReportAnAbsentProbe makes sure the dry run says it did not run
// instead of passing silently.
func TestLiveRulesReportAnAbsentProbe(t *testing.T) {
	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), "UPDATE TECH_BOOK SET NAME = 'xz' WHERE ID = 1;", []*storepb.SQLReviewRule{{
		Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_ORACLE,
	}}, advisor.Context{
		DBType:          storepb.Engine_ORACLE,
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_ERROR, adviceList[0].Status)
	require.Equal(t, "Rule check failed: ORACLE: the live connection cannot probe the database, so this rule did not run", adviceList[0].Content)
}
