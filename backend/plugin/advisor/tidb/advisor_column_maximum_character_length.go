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
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for max character length.
// The judgment lives in omni (tidb/v1review.ColumnMaximumCharacterLength).
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		return nil, errors.New("number_payload is required for column maximum character length rule")
	}
	maximum := int(numberPayload.Number)
	if maximum <= 0 {
		return nil, nil
	}
	title := checkCtx.Rule.Type.String()

	var adviceList []*storepb.Advice
	for _, stmt := range stmts {
		for _, f := range tidbv1review.ColumnMaximumCharacterLength([]ast.Node{stmt.Node}, maximum) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.CharLengthExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The length of the CHAR column `%s` is %s, bigger than %d, please use VARCHAR instead", f.Name, f.Value, maximum),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.AbsoluteLine(f.Start)),
			})
		}
	}
	return adviceList, nil
}
