package oceanbase

import (
	"context"
	"fmt"

	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnCurrentTimeCountLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, &ColumnCurrentTimeCountLimitAdvisor{})
}

// ColumnCurrentTimeCountLimitAdvisor is the advisor checking for current time column count limit.
// The judgment lives in omni (oceanbase/v1review.ColumnCurrentTimeCountLimit).
type ColumnCurrentTimeCountLimitAdvisor struct {
}

// Check checks for current time column count limit.
func (*ColumnCurrentTimeCountLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for _, f := range oceanbasev1review.ColumnCurrentTimeCountLimit(nodes) {
		var adviceCode code.Code
		var content string
		switch f.Code {
		case v1review.ColumnDefaultCurrentTimeCount:
			adviceCode = code.DefaultCurrentTimeColumnCountExceedsLimit
			content = fmt.Sprintf("Table `%s` has %s DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, oceanbasev1review.ColumnCurrentTimeDefaultLimit)
		case v1review.ColumnOnUpdateCurrentTimeCount:
			adviceCode = code.OnUpdateCurrentTimeColumnCountExceedsLimit
			content = fmt.Sprintf("Table `%s` has %s ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, oceanbasev1review.ColumnCurrentTimeOnUpdateLimit)
		default:
			continue
		}
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          adviceCode.Int32(),
			Title:         checkCtx.Rule.Type.String(),
			Content:       content,
			StartPosition: findingPosition(stmts[f.Statement], f),
		})
	}
	return adviceList, nil
}
