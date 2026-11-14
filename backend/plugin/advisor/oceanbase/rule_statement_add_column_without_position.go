package oceanbase

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oceanbase/ast"
	oceanbasev1review "github.com/bytebase/omni/oceanbase/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

var (
	_ advisor.Advisor = (*StatementAddColumnWithoutPositionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION, &StatementAddColumnWithoutPositionAdvisor{})
}

// StatementAddColumnWithoutPositionAdvisor is the advisor checking for no position in ADD COLUMN clause.
// The judgment lives in omni (oceanbase/v1review.StatementAddColumnWithoutPosition); this adapter
// renders its findings as advice.
type StatementAddColumnWithoutPositionAdvisor struct {
}

// Check checks for no position in ADD COLUMN clause.
func (*StatementAddColumnWithoutPositionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := oceanbaseparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		for _, f := range oceanbasev1review.StatementAddColumnWithoutPosition([]ast.Node{node}) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementAddColumnWithPosition.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("add column with position \"%s\"", f.Name),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
			})
		}
	}
	return adviceList, nil
}
