package tidb

import (
	"context"
	"fmt"

	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// The limits tidbv1review.ColumnCurrentTimeCountLimit judges, as the advice
// prints them.
const (
	maxDefaultCurrentTimeColumCount   = 2
	maxOnUpdateCurrentTimeColumnCount = 1
)

var (
	_ advisor.Advisor = (*ColumnCurrentTimeCountLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT, &ColumnCurrentTimeCountLimitAdvisor{})
}

// ColumnCurrentTimeCountLimitAdvisor is the advisor checking for current time column count limit.
// The judgment lives in omni (tidb/v1review.ColumnCurrentTimeCountLimit).
type ColumnCurrentTimeCountLimitAdvisor struct {
}

// Check checks for current time column count limit.
func (*ColumnCurrentTimeCountLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, f := range tidbv1review.ColumnCurrentTimeCountLimit(stmtNodes(stmts)) {
		advice := &storepb.Advice{
			Status:        level,
			Title:         title,
			StartPosition: base.ConvertANTLRLineToPosition(stmts[f.Statement].AbsoluteLine(f.Start)),
		}
		switch f.Code {
		case v1review.ColumnDefaultCurrentTimeCount:
			advice.Code = code.DefaultCurrentTimeColumnCountExceedsLimit.Int32()
			advice.Content = fmt.Sprintf("Table `%s` has %s DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, maxDefaultCurrentTimeColumCount)
		case v1review.ColumnOnUpdateCurrentTimeCount:
			advice.Code = code.OnUpdateCurrentTimeColumnCountExceedsLimit.Int32()
			advice.Content = fmt.Sprintf("Table `%s` has %s ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, maxOnUpdateCurrentTimeColumnCount)
		default:
			continue
		}
		adviceList = append(adviceList, advice)
	}
	return adviceList, nil
}
