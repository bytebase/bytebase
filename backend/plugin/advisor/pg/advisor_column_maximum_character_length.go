package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for maximum
// character length. The judgment lives in omni
// (pg/v1review.ColumnMaximumCharacterLength); this adapter passes the maximum
// and renders the findings as advice.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	var adviceList []*storepb.Advice
	for i, stmt := range stmts {
		for _, f := range pgv1review.ColumnMaximumCharacterLength([]ast.Node{nodes[i]}, maximum) {
			tableName := fmt.Sprintf("%q", f.Table)
			if f.Value != "" {
				tableName = fmt.Sprintf("%q.%q", f.Value, f.Table)
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.CharLengthExceedsLimit.Int32(),
				Title:   checkCtx.Rule.Type.String(),
				Content: fmt.Sprintf("The length of the CHAR column %q in table %s is bigger than %d, please use VARCHAR instead", f.Name, tableName, maximum),
				StartPosition: &storepb.Position{
					Line:   findingLine(stmt, f),
					Column: 0,
				},
			})
		}
	}
	return adviceList, nil
}
