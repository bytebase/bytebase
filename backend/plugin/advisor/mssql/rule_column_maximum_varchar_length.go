package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mssql/ast"
	mssqlv1review "github.com/bytebase/omni/mssql/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for maximum varchar length.
// The judgment lives in omni (mssql/v1review.ColumnMaximumVarcharLength).
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
		return nil, errors.New("number_payload is required for this rule")
	}
	maximum := int(numberPayload.Number)
	if maximum <= 0 {
		return nil, nil
	}

	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := tsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range mssqlv1review.ColumnMaximumVarcharLength([]ast.Node{node}, maximum) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.VarcharLengthExceedsLimit.Int32(),
				Title:         checkCtx.Rule.Type.String(),
				Content:       fmt.Sprintf("The maximum varchar length is %d.", maximum),
				StartPosition: &storepb.Position{Line: int32(stmt.BaseLine()) + findingLine(stmt.Text, f)},
			})
		}
	}
	return adviceList, nil
}
