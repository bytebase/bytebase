package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for max varchar length.
// The judgment lives in omni (mysql/v1review.ColumnMaximumVarcharLength).
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for column maximum varchar length rule")
	}
	maximum := int(numberPayload.Number)
	if maximum <= 0 {
		return nil, nil
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range mysqlv1review.ColumnMaximumVarcharLength([]ast.Node{nodes[i]}, maximum) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.VarcharLengthExceedsLimit.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", f.Table, f.Name, maximum),
				StartPosition: columnFindingPosition(stmt, f),
			})
		}
	}
	return adviceList, nil
}
