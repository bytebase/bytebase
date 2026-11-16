package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"
	tidbv1review "github.com/bytebase/omni/tidb/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementInitialValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, &ColumnAutoIncrementInitialValueAdvisor{})
}

// ColumnAutoIncrementInitialValueAdvisor is the advisor checking for auto-increment column initial value.
// The judgment lives in omni (tidb/v1review.ColumnAutoIncrementInitialValue).
type ColumnAutoIncrementInitialValueAdvisor struct {
}

// Check checks for auto-increment column initial value.
func (*ColumnAutoIncrementInitialValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for column auto increment initial value rule")
	}
	value := int(numberPayload.Number)
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.ColumnAutoIncrementInitialValue([]ast.Node{stmt.Node}, value) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AutoIncrementColumnInitialValueNotMatch.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The initial auto-increment value in table `%s` is %s, which doesn't equal %v", f.Table, f.Value, value),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}
