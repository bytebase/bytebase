package plancheck

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestTagPlanCheckResultsCopiesTargetMetadata(t *testing.T) {
	results := []*storepb.PlanCheckRunResult_Result{
		{
			Status: storepb.Advice_SUCCESS,
			Title:  "ok",
		},
	}
	target := &CheckTarget{
		Target:      "instances/prod/databases/app",
		SheetSha256: "abc123",
	}

	tagPlanCheckResults(results, target, storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT)

	require.Len(t, results, 1)
	require.Equal(t, "instances/prod/databases/app", results[0].Target)
	require.Equal(t, "abc123", results[0].SheetSha256)
	require.Equal(t, storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT, results[0].Type)
}
