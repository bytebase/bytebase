package oceanbase

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestInsertRowLimitAdvisorKeepsCountingValuesBeyondExplainLimit(t *testing.T) {
	const insertPlan = `{"ID":0,"OPERATOR":"DISTRIBUTED INSERT","NAME":"","EST.ROWS":100,"CHILD_1":{"ID":1,"OPERATOR":"TABLE FULL SCAN","NAME":"t","EST.ROWS":100}}`
	statement := strings.Repeat("INSERT INTO t2 SELECT * FROM t;\n", common.MaximumLintExplainSize+1) +
		"INSERT INTO t2 (id) VALUES (1), (2), (3), (4), (5), (6);"

	adviceList := checkTestRowLimitRule(t, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_WARNING, insertPlan, statement)
	require.Len(t, adviceList, common.MaximumLintExplainSize+2)
	for _, advice := range adviceList[:common.MaximumLintExplainSize] {
		require.Contains(t, advice.Content, "inserts 100 rows. The count exceeds 5.")
	}
	require.Contains(t, adviceList[common.MaximumLintExplainSize].Content, "inserts 6 rows. The count exceeds 5.")
	warning := adviceList[common.MaximumLintExplainSize+1]
	require.Equal(t, storepb.Advice_WARNING, warning.Status)
	require.Equal(t, code.InsertTooManyRows.Int32(), warning.Code)
	require.Equal(t, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT.String(), warning.Title)
	require.Equal(t, "Only the first 10 statements were estimated; 1 more were not checked against the row limit.", warning.Content)
	require.Equal(t, int32(common.MaximumLintExplainSize+1), warning.StartPosition.GetLine())
	require.Equal(t, common.MaximumLintExplainSize, testOceanBaseExplainCount)
}
