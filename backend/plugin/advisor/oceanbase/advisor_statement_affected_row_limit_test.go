package oceanbase

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestStatementAffectedRowLimitAdvisor(t *testing.T) {
	const updatePlan = `{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE","NAME":"","EST.ROWS":1000,"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t","EST.ROWS":1000}}`

	t.Run("estimate over the limit", func(t *testing.T) {
		adviceList := checkTestRowLimitRule(t, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_WARNING, updatePlan, "UPDATE t SET c = 1;")
		require.Len(t, adviceList, 1)
		require.Equal(t, code.StatementAffectedRowExceedsLimit.Int32(), adviceList[0].Code)
		require.Equal(t, `"UPDATE t SET c = 1;" affected 1000 rows (estimated). The count exceeds 5.`, adviceList[0].Content)
	})

	t.Run("missing estimate", func(t *testing.T) {
		adviceList := checkTestRowLimitRule(t, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_WARNING, `{"ID":0,"OPERATOR":"DISTRIBUTED UPDATE","NAME":""}`, "UPDATE t SET c = 1;")
		require.Len(t, adviceList, 1)
		require.Equal(t, code.Internal.Int32(), adviceList[0].Code)
		require.Contains(t, adviceList[0].Content, `operator "DISTRIBUTED UPDATE" has no EST.ROWS`)
	})

	t.Run("statements beyond the EXPLAIN limit", func(t *testing.T) {
		statement := strings.Repeat("UPDATE t SET c = 1;\n", common.MaximumLintExplainSize+2)
		adviceList := checkTestRowLimitRule(t, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, storepb.SQLReviewRule_ERROR, updatePlan, statement)
		require.Len(t, adviceList, common.MaximumLintExplainSize+1)
		warning := adviceList[common.MaximumLintExplainSize]
		require.Equal(t, storepb.Advice_WARNING, warning.Status)
		require.Equal(t, code.StatementAffectedRowExceedsLimit.Int32(), warning.Code)
		require.Equal(t, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT.String(), warning.Title)
		require.Equal(t, "Only the first 10 statements were estimated; 2 more were not checked against the row limit.", warning.Content)
		require.Equal(t, int32(common.MaximumLintExplainSize+1), warning.StartPosition.GetLine())
		require.Equal(t, common.MaximumLintExplainSize, testOceanBaseExplainCount)
	})
}
